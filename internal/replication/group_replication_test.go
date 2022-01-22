package replication

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	syncds "github.com/ipfs/go-datastore/sync"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
)

func TestGroupCreation(t *testing.T) {
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
	h, err := libp2p.New(opts...)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	ds := &mockDAGSyncer{}
	sy := &mockSyncer{}
	childCtx, childCancel := context.WithCancel(ctx)
	m, err := New(childCtx, childCancel, r, h, ds, r.Datastore(), "/prefix", "topic", time.Second, sy, h.Peerstore())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	gMeta, err := m.CreateGroup("NewGroup1", h.ID(), priv)
	if err != nil {
		t.Fatal(err)
	}
	groups, err := m.GroupGetAllGroups(h.ID(), priv)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatal("group count mismatch")
	}
	if groups[0].GroupID != gMeta.GroupID {
		t.Fatal("groupID mismatch")
	}
}

func TestGroupAddMember(t *testing.T) {
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
	h, err := libp2p.New(opts...)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	ds := &mockDAGSyncer{}
	sy := &mockSyncer{}
	childCtx, childCancel := context.WithCancel(ctx)
	m, err := New(childCtx, childCancel, r, h, ds, r.Datastore(), "/prefix", "topic", time.Second, sy, h.Peerstore())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	gMeta, err := m.CreateGroup("NewGroup1", h.ID(), priv)
	if err != nil {
		t.Fatal(err)
	}
	privPeerTwo, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		t.Fatal(err)
	}
	optsPeerTwo := []libp2p.Option{
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/4833"),
		libp2p.Identity(privPeerTwo),
		libp2p.DisableRelay(),
	}
	h2, err := libp2p.New(optsPeerTwo...)
	if err != nil {
		t.Fatal(err)
	}
	defer h2.Close()
	groups, err := m.GroupGetAllGroups(h2.ID(), privPeerTwo)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 0 {
		t.Fatal("group count mismatch")
	}

	err = m.GroupAddMember(h.ID(), h2.ID(), gMeta.GroupID, priv, privPeerTwo.GetPublic())
	if err != nil {
		t.Fatal(err)
	}

	groups, err = m.GroupGetAllGroups(h2.ID(), privPeerTwo)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatal("group count mismatch")
	}
	if groups[0].GroupID != gMeta.GroupID {
		t.Fatal("groupID mismatch")
	}
}
