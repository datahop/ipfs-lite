package repo

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/datahop/ipfs-lite/internal/config"
	"github.com/facebookgo/atomicfile"
	"github.com/ipfs/go-datastore"
	ds "github.com/ipfs/go-datastore"
	syncds "github.com/ipfs/go-datastore/sync"
	leveldb "github.com/ipfs/go-ds-leveldb"
	lockfile "github.com/ipfs/go-fs-lock"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
)

const (
	Root                       = ".datahop"
	LockFile                   = "repo.lock"
	DefaultDatastoreFolderName = "datastore"
	// DefaultConfigFile is the filename of the configuration file
	DefaultConfigFile = "config"
)

var (
	packageLock     sync.Mutex
	log             = logging.Logger("repo")
	ErrorRepoClosed = errors.New("cannot access config, repo not open")
)

type Repo interface {
	Path() string
	Config() (*config.Config, error)
	Datastore() Datastore
	Close() error
}

// Datastore is the interface required from a datastore to be
// acceptable to FSRepo.
type Datastore interface {
	ds.Batching // must be thread-safe
}

type FSRepo struct {
	// has Close been called already
	closed bool
	// path is the file-system path
	path string
	// lockfile is the file system lock to prevent others from opening
	// the same fsrepo path concurrently
	lockfile io.Closer
	config   *config.Config
	ds       Datastore
	io.Closer
}

func (r *FSRepo) Config() (*config.Config, error) {
	// It is not necessary to hold the package lock since the repo is in an
	// opened state. The package lock is _not_ meant to ensure that the repo is
	// thread-safe. The package lock is only meant to guard against removal and
	// coordinate the lockfile. However, we provide thread-safety to keep
	// things simple.
	packageLock.Lock()
	defer packageLock.Unlock()

	if r.closed {
		return nil, ErrorRepoClosed
	}
	return r.config, nil
}

func (r *FSRepo) Path() string {
	return r.path
}

func (r *FSRepo) Datastore() Datastore {
	packageLock.Lock()
	defer packageLock.Unlock()

	return r.ds
}

func (r *FSRepo) Close() error {
	packageLock.Lock()
	defer packageLock.Unlock()

	r.closed = true
	return r.ds.Close()
}

// Init initialises ipfs persistent repository to a given location
func Init(repoPath, swarmPort string) error {
	// packageLock must be held to ensure that the repo is not initialized more
	// than once.
	packageLock.Lock()
	defer packageLock.Unlock()

	// Check if already initialised
	if isInitializedUnsynced(repoPath) {
		return nil
	}
	conf, err := config.NewConfig(swarmPort)
	if err != nil {
		return err
	}
	if err := initConfig(repoPath, conf); err != nil {
		return err
	}
	return nil
}

// Open the FSRepo at path. Returns an error if the repo is not
// initialized.
func Open(repoPath string) (Repo, error) {
	return open(repoPath)
}

func open(repoPath string) (Repo, error) {
	packageLock.Lock()
	defer packageLock.Unlock()

	r, err := newFSRepo(repoPath)
	if err != nil {
		return nil, err
	}

	r.lockfile, err = lockfile.Lock(r.path, LockFile)
	if err != nil {
		return nil, err
	}
	keepLocked := false
	defer func() {
		// unlock on error, leave it locked on success
		if !keepLocked {
			r.lockfile.Close()
		}
	}()

	if err := r.openConfig(); err != nil {
		return nil, err
	}
	if err := r.openDatastore(); err != nil {
		return nil, err
	}
	r.closed = false
	return r, nil
}

func (r *FSRepo) openDatastore() error {
	d, err := levelDatastore(filepath.Join(r.Path(), DefaultDatastoreFolderName))
	if err != nil {
		return err
	}
	r.ds = syncds.MutexWrap(d)
	return nil
}

// levelDatastore returns a new instance of LevelDB persisting
// to the given path with the default options.
func levelDatastore(path string) (datastore.Batching, error) {
	return leveldb.NewDatastore(path, &leveldb.Options{})
}

func newFSRepo(rpath string) (*FSRepo, error) {
	expPath, err := homedir.Expand(filepath.Clean(rpath))
	if err != nil {
		return nil, err
	}

	return &FSRepo{path: expPath}, nil
}

func initConfig(path string, cfg *config.Config) error {
	configFilename, err := ConfigFilename(path)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(configFilename), 0775)
	if err != nil {
		return err
	}

	f, err := atomicfile.New(configFilename, 0660)
	if err != nil {
		return err
	}
	defer f.Close()

	return encode(f, cfg)
}

// ConfigFilename returns the configuration file path given a configuration root
// directory. If the configuration root directory is empty, use the default one
func ConfigFilename(configroot string) (string, error) {
	return filepath.Join(configroot, DefaultConfigFile), nil
}

// encode configuration with JSON
func encode(w io.Writer, value interface{}) error {
	// need to prettyprint, hence MarshalIndent, instead of Encoder
	buf, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

// openConfig returns an error if the config file is not present.
func (r *FSRepo) openConfig() error {
	conf, err := openConfig(r.path)
	if err != nil {
		return err
	}
	r.config = conf
	return nil
}

func openConfig(path string) (conf *config.Config, err error) {
	configFilename, err := ConfigFilename(path)
	conf = &config.Config{}
	if err != nil {
		return
	}
	f, err := os.Open(configFilename)
	if err != nil {
		return
	}
	defer f.Close()
	if err = json.NewDecoder(f).Decode(&conf); err != nil {
		return
	}
	return
}

// configIsInitialized returns true if the repo is initialized at
// provided |path|.
func configIsInitialized(path string) bool {
	configFilename, err := ConfigFilename(path)
	if err != nil {
		return false
	}

	fi, err := os.Lstat(configFilename)
	if fi != nil || (err != nil && !os.IsNotExist(err)) {
		return true
	}
	return false
}

// isInitializedUnsynced reports whether the repo is initialized. Caller must
// hold the packageLock.
func isInitializedUnsynced(repoPath string) bool {
	return configIsInitialized(repoPath)
}
