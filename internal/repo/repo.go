package repo

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/datahop/ipfs-lite/internal/config"
	"github.com/datahop/ipfs-lite/internal/matrix"
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
	// Root location of the repository
	Root                       = ".datahop"
	lockFile                   = "repo.lock"
	defaultDatastoreFolderName = "datastore"
	// defaultConfigFile is the filename of the configuration file
	defaultConfigFile = "config"
	defaultStateFile  = "state"
)

var (
	packageLock     sync.Mutex
	log             = logging.Logger("repo")
	ErrorRepoClosed = errors.New("cannot access config, repo not open")
)

// Repo exposes basic repo operations
type Repo interface {
	Path() string
	Config() (*config.Config, error)
	Datastore() Datastore
	Close() error
	State() *bloom.BloomFilter
	SetState() error
	Matrix() *matrix.MatrixKeeper
	StateKeeper() *StateKeeper
}

// Datastore is the interface required from a datastore to be
// acceptable to FSRepo.
type Datastore interface {
	ds.Batching // must be thread-safe
}

// FSRepo implements Repo
type FSRepo struct {
	// has Close been called already
	closed bool
	// path is the file-system path
	path string
	// lockfile is the file system lock to prevent others from opening
	// the same fsrepo path concurrently
	lockfile    io.Closer
	config      *config.Config
	ds          Datastore
	state       *bloom.BloomFilter
	mKeeper     *matrix.MatrixKeeper
	stateKeeper *StateKeeper
	io.Closer
}

// Config returns repository config
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

// Path of the repository
func (r *FSRepo) Path() string {
	return r.path
}

// Datastore returns datastore of the node
func (r *FSRepo) Datastore() Datastore {
	packageLock.Lock()
	defer packageLock.Unlock()

	return r.ds
}

// State returns nodes crdt bloom filter state
func (r *FSRepo) State() *bloom.BloomFilter {
	packageLock.Lock()
	defer packageLock.Unlock()

	return r.state
}

// SetState updates local state file with bloom filter
func (r *FSRepo) SetState() error {
	packageLock.Lock()
	defer packageLock.Unlock()

	f, err := os.OpenFile(filepath.Join(r.path, defaultStateFile), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := r.state.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err != nil {
		return err
	}
	return nil
}

// Close repo
func (r *FSRepo) Close() error {
	packageLock.Lock()
	defer packageLock.Unlock()

	return r.close()
}

func (r *FSRepo) close() error {
	err := r.ds.Close()
	if err != nil {
		return err
	}
	r.closed = true
	return r.lockfile.Close()
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
	if err := initState(repoPath); err != nil {
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

	r.lockfile, err = lockfile.Lock(r.path, lockFile)
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
	if err := r.openState(); err != nil {
		r.close()
		return nil, err
	}
	r.mKeeper = matrix.NewMatrixKeeper(r.ds)
	r.stateKeeper, err = LoadStateKeeper(r.path)
	if err != nil {
		return nil, err
	}
	keepLocked = true
	r.closed = false
	return r, nil
}

// Matrix returns nodes matrix
func (r *FSRepo) Matrix() *matrix.MatrixKeeper {
	return r.mKeeper
}

// StateKeeper returns nodes matrix
func (r *FSRepo) StateKeeper() *StateKeeper {
	return r.stateKeeper
}

func (r *FSRepo) openDatastore() error {
	d, err := levelDatastore(filepath.Join(r.Path(), defaultDatastoreFolderName))
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

func initState(path string) error {
	f, err := os.Create(filepath.Join(path, defaultStateFile))
	if err != nil {
		return err
	}
	defer f.Close()
	filter := bloom.New(uint(2000), 5)
	b, err := filter.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return err
}

// ConfigFilename returns the configuration file path given a configuration root
// directory. If the configuration root directory is empty, use the default one
func ConfigFilename(configroot string) (string, error) {
	return filepath.Join(configroot, defaultConfigFile), nil
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

// openState returns an error if the state file is not present.
func (r *FSRepo) openState() error {
	f, err := os.Open(filepath.Join(r.path, defaultStateFile))
	if os.IsNotExist(err) {
		err = initState(r.path)
		if err != nil {
			return err
		}
		f, err = os.Open(filepath.Join(r.path, defaultStateFile))
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	r.state = bloom.New(uint(2000), 5)
	return r.state.UnmarshalJSON(buf)
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
