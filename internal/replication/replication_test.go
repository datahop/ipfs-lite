package replication

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/datahop/ipfs-lite/internal/config"
	"github.com/datahop/ipfs-lite/internal/matrix"
	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/h2non/filetype"
	"github.com/ipfs/go-cid"
	syncds "github.com/ipfs/go-datastore/sync"
	leveldb "github.com/ipfs/go-ds-leveldb"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
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

func (m mockDAGSyncer) HasBlock(cid.Cid) (bool, error) {
	// do something
	return true, nil
}

type mockSyncer struct{}

func (m mockSyncer) ConnectIfNotConnectedUsingRelay(ctx context.Context, ids []peer.ID) {
	// do nothing
}

func (m mockSyncer) Download(ctx context.Context, c cid.Cid) error {
	// do something
	return nil
}

func (m mockSyncer) FindProviders(ctx context.Context, id cid.Cid) []peer.ID {
	// do something
	return []peer.ID{}
}

type mockRepo struct {
	path  string
	state *bloom.BloomFilter
	ds    repo.Datastore
	sk    *repo.StateKeeper
}

func (m *mockRepo) Path() string {
	return m.path
}

func (m *mockRepo) Config() (*config.Config, error) {
	return nil, nil
}

func (m *mockRepo) Matrix() *matrix.MatrixKeeper {
	return &matrix.MatrixKeeper{
		NodeMatrix: &matrix.NodeMatrix{
			TotalUptime:     0,
			NodesDiscovered: map[string]*matrix.DiscoveredNodeMatrix{},
		},
		ContentMatrix: map[string]*matrix.ContentMatrix{},
	}
}

func (m *mockRepo) Datastore() repo.Datastore {
	return m.ds
}

func (m *mockRepo) Close() error {
	return nil
}

func (m *mockRepo) State() *bloom.BloomFilter {
	return m.state
}

func (m *mockRepo) SetState() error {
	return nil
}

func (m *mockRepo) StateKeeper() *repo.StateKeeper {
	return m.sk
}

type mockDownload struct {
	mockName string
}

func (m *mockDownload) Execute(ctx context.Context) error {
	<-time.After(time.Second * 2)
	return nil
}

func (m *mockDownload) Name() string {
	return m.mockName
}

func TestNewManager(t *testing.T) {
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
		state: bloom.New(uint(2000), 5),
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
	childCtx, childCancel := context.WithCancel(ctx)
	m, err := New(childCtx, childCancel, r, h, ds, r.Datastore(), "/prefix", "topic", time.Second, sy, h.Peerstore(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	id, err := cid.Decode("bafybeiclg7ypvgnbumueqcfgarezgsz7af5kmg75nynaeqjxdme5jqmh3e")
	if err != nil {
		t.Fatal(err)
	}
	m.StartContentWatcher()
	meta := &ContentMetatag{
		Size:      0,
		Type:      filetype.Unknown.Extension,
		Name:      "some content",
		Hash:      id,
		Timestamp: time.Now().Unix(),
		Owner:     h.ID(),
	}
	err = m.Tag("mtag", meta)
	if err != nil {
		t.Fatal(err)
	}
	metaFound, err := m.FindTag("mtag")
	if err != nil {
		t.Fatal(err)
	}
	if id != metaFound.Hash {
		t.Fatal("cid mismatch")
	}
}

func TestDownloadManager(t *testing.T) {
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
		state: bloom.New(uint(2000), 5),
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
	childCtx, childCancel := context.WithCancel(ctx)
	m, err := New(childCtx, childCancel, r, h, ds, r.Datastore(), "/prefix", "topic", time.Second, sy, h.Peerstore(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	taskCount := 10
	for i := 0; i < taskCount; i++ {
		dt := &mockDownload{mockName: fmt.Sprintf("some content %d", i)}
		_, err = m.dlManager.Go(dt)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(m.DownloadManagerStatus()) != taskCount {
		t.Fatalf("taskCount should be %d got %d", taskCount, len(m.DownloadManagerStatus()))
	}
	<-time.After(time.Second * 3)
	if len(m.DownloadManagerStatus()) != 0 {
		t.Fatalf("taskCount should be 0 got %d", len(m.DownloadManagerStatus()))
	}
}

func TestGetAllCids(t *testing.T) {
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
		state: bloom.New(uint(2000), 5),
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
	defer func() {
		h.Close()
	}()
	ds := &mockDAGSyncer{}
	sy := &mockSyncer{}
	childCtx, childCancel := context.WithCancel(ctx)
	m, err := New(childCtx, childCancel, r, h, ds, r.Datastore(), "/prefix", "topic", time.Second, sy, h.Peerstore(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		m.Close()
	}()
	id, err := cid.Decode("bafybeiclg7ypvgnbumueqcfgarezgsz7af5kmg75nynaeqjxdme5jqmh3e")
	if err != nil {
		t.Fatal(err)
	}
	id2, err := cid.Decode("bafybeiexhsqapwqpc7y2oegxwyqnt4b4ukdywuy7vvjs35jquepdtmsryu")
	if err != nil {
		t.Fatal(err)
	}

	m.StartContentWatcher()
	meta1 := &ContentMetatag{
		Size:      0,
		Type:      filetype.Unknown.Extension,
		Name:      "mtag1",
		Hash:      id,
		Timestamp: time.Now().Unix(),
		Owner:     h.ID(),
	}
	meta2 := &ContentMetatag{
		Size:      0,
		Type:      filetype.Unknown.Extension,
		Name:      "mtag2",
		Hash:      id2,
		Timestamp: time.Now().Unix(),
		Owner:     h.ID(),
	}
	err = m.Tag("mtag1", meta1)
	if err != nil {
		t.Fatal(err)
	}
	err = m.Tag("mtag2", meta2)
	if err != nil {
		t.Fatal(err)
	}
	cids, err := m.GetAllCids()
	if err != nil {
		t.Fatal(err)
	}
	if len(cids) != 2 {
		t.Fatal("there should be two cids")
	}
	if cids[0] != id && cids[1] != id {
		t.Fatal(id, " is not in cids ", cids)
	}
	if cids[0] != id2 && cids[1] != id2 {
		t.Fatal(id2, " is not in cids ", cids)
	}
}

func removeRepo(repopath string, t *testing.T) {
	err := os.RemoveAll(repopath)
	if err != nil {
		t.Fatal(err)
	}
}
