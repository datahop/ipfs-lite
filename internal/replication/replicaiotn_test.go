package replication

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	logger "github.com/ipfs/go-log/v2"

	syncds "github.com/ipfs/go-datastore/sync"
	leveldb "github.com/ipfs/go-ds-leveldb"

	"github.com/datahop/ipfs-lite/internal/config"
	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	ufsio "github.com/ipfs/go-unixfs/io"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
)

type mockDAGSyncer struct{}

func (m mockDAGSyncer) Get(ctx context.Context, cid cid.Cid) (ipld.Node, error) {
	// do something
	return nil, nil
}

func (m mockDAGSyncer) GetMany(ctx context.Context, cids []cid.Cid) <-chan *ipld.NodeOption {
	// do something
	return nil
}

func (m mockDAGSyncer) Add(ctx context.Context, node ipld.Node) error {
	// do something
	return nil
}

func (m mockDAGSyncer) AddMany(ctx context.Context, nodes []ipld.Node) error {
	// do something
	return nil
}

func (m mockDAGSyncer) Remove(ctx context.Context, cid cid.Cid) error {
	// do something
	return nil
}

func (m mockDAGSyncer) RemoveMany(ctx context.Context, cids []cid.Cid) error {
	// do something
	return nil
}

func (m mockDAGSyncer) HasBlock(c cid.Cid) (bool, error) {
	// do something
	return true, nil
}

type mockSyncer struct{}

func (m mockSyncer) GetFile(ctx context.Context, c cid.Cid) (ufsio.ReadSeekCloser, error) {
	// do something
	return nil, nil
}

type mockRepo struct {
	path  string
	state int
	ds    repo.Datastore
}

func (m *mockRepo) Path() string {
	return m.path
}

func (m *mockRepo) Config() (*config.Config, error) {
	return nil, nil
}

func (m *mockRepo) Datastore() repo.Datastore {
	return m.ds
}

func (m *mockRepo) Close() error {
	return nil
}

func (m *mockRepo) State() int {
	return m.state
}

func (m *mockRepo) SetState(i int) error {
	m.state = i
	return nil
}

func TestNewManager(t *testing.T) {
	logger.SetLogLevel("replication", "Debug")
	<-time.After(time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	root := filepath.Join("../../test", "root1")
	d, err := leveldb.NewDatastore(root, &leveldb.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	r := &mockRepo{
		path:  root,
		state: 0,
		ds:    syncds.MutexWrap(d),
	}
	defer r.Close()
	defer removeRepo(root, t)
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		t.Fatal(err)
	}
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/4832"),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}
	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	ds := &mockDAGSyncer{}
	sy := &mockSyncer{}
	m, err := New(ctx, r, h, ds, r.Datastore(), "/prefix", "topic", time.Second, sy)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	id, err := cid.Decode("bafybeiclg7ypvgnbumueqcfgarezgsz7af5kmg75nynaeqjxdme5jqmh3e")
	if err != nil {
		t.Fatal(err)
	}
	m.StartContentWatcher()
	err = m.Tag("mtag", id)
	if err != nil {
		t.Fatal(err)
	}
	nId, err := m.FindTag("mtag")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != nId.String() {
		t.Fatal("cid mismatch")
	}
}

func removeRepo(repopath string, t *testing.T) {
	err := os.RemoveAll(repopath)
	if err != nil {
		t.Fatal(err)
	}
}
