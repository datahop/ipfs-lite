package ipfslite

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ipfs/go-datastore/query"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"

	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/libp2p/go-libp2p-core/peer"
	multihash "github.com/multiformats/go-multihash"
)

func setupPeers(t *testing.T) (p1, p2 *Peer, closer func(t *testing.T)) {
	ctx, cancel := context.WithCancel(context.Background())

	root1 := filepath.Join("./test", "root1")
	root2 := filepath.Join("./test", "root2")

	_, err := Init(root1, "0")
	if err != nil {
		t.Fatal(err)
	}
	r1, err := Open(root1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Init(root2, "4502")
	if err != nil {
		t.Fatal(err)
	}
	r2, err := Open(root2)
	if err != nil {
		t.Fatal(err)
	}

	closer = func(t *testing.T) {
		cancel()
	}

	p1, err = New(ctx, r1)
	if err != nil {
		closer(t)
		t.Fatal(err)
	}
	p2, err = New(ctx, r2)
	if err != nil {
		closer(t)
		t.Fatal(err)
	}

	pinfo1 := peer.AddrInfo{
		ID:    p1.Host.ID(),
		Addrs: p1.Host.Addrs(),
	}

	pinfo2 := peer.AddrInfo{
		ID:    p2.Host.ID(),
		Addrs: p2.Host.Addrs(),
	}
	p1.Bootstrap([]peer.AddrInfo{pinfo2})
	p2.Bootstrap([]peer.AddrInfo{pinfo1})

	return
}

func TestHost(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second * 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	root1 := filepath.Join("./test", "root1")
	_, err := Init(root1, "0")
	if err != nil {
		t.Fatal(err)
	}
	r1, err := Open(root1)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := New(ctx, r1)
	if err != nil {
		t.Fatal(err)
	}
	cnf, err := r1.Config()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Config id %s, Host Id %s", cnf.Identity.PeerID, p1.Host.ID().Pretty())
	if cnf.Identity.PeerID != p1.Host.ID().Pretty() {
		t.Fatal("Peer id does not match config")
	}
}

func TestDAG(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second * 1)

	ctx := context.Background()
	p1, p2, closer := setupPeers(t)
	defer closer(t)

	m := map[string]string{
		"akey": "avalue",
	}

	codec := uint64(multihash.SHA2_256)
	node, err := cbor.WrapObject(m, codec, multihash.DefaultLengths[codec])
	if err != nil {
		t.Fatal(err)
	}

	t.Log("created node: ", node.Cid())
	err = p1.Add(ctx, node)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p2.Get(ctx, node.Cid())
	if err != nil {
		t.Error(err)
	}

	err = p1.Remove(ctx, node.Cid())
	if err != nil {
		t.Error(err)
	}

	err = p2.Remove(ctx, node.Cid())
	if err != nil {
		t.Error(err)
	}

	if ok, err := p1.BlockStore().Has(node.Cid()); ok || err != nil {
		t.Error("block should have been deleted")
	}

	if ok, err := p2.BlockStore().Has(node.Cid()); ok || err != nil {
		t.Error("block should have been deleted")
	}
}

func TestSession(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second * 1)

	ctx := context.Background()
	p1, p2, closer := setupPeers(t)
	defer closer(t)

	m := map[string]string{
		"akey": "avalue",
	}

	codec := uint64(multihash.SHA2_256)
	node, err := cbor.WrapObject(m, codec, multihash.DefaultLengths[codec])
	if err != nil {
		t.Fatal(err)
	}

	t.Log("created node: ", node.Cid(), p1)
	err = p1.Add(ctx, node)
	if err != nil {
		t.Fatal(err)
	}

	sesGetter := p2.Session(ctx)
	_, err = sesGetter.Get(ctx, node.Cid())
	if err != nil {
		t.Fatal(err)
	}
}

func TestFiles(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second * 1)

	p1, p2, closer := setupPeers(t)
	defer closer(t)

	content := []byte("hola")
	buf := bytes.NewReader(content)
	n, err := p1.AddFile(context.Background(), buf, nil)
	if err != nil {
		t.Fatal(err)
	}

	rsc, err := p2.GetFile(context.Background(), n.Cid())
	if err != nil {
		t.Fatal(err)
	}
	defer rsc.Close()

	content2, err := ioutil.ReadAll(rsc)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(content, content2) {
		t.Error(string(content))
		t.Error(string(content2))
		t.Error("different content put and retrieved")
	}
}

func TestOperations(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second * 1)

	p1, p2, closer := setupPeers(t)
	defer closer(t)

	p1peers := p1.Peers()
	p2peers := p2.Peers()

	t.Logf("P1 peers %v", p1peers)
	t.Logf("P2 peers %v", p2peers)
	if len(p1peers) != len(p2peers) {
		t.Fatal("Peer count should be same")
	}
	if p1peers[0] != p2.Host.ID().Pretty() || p2peers[0] != p1.Host.ID().Pretty() {
		t.Fatal("Peer connection did not happen")
	}
	p2aInfo := peer.AddrInfo{
		ID:    p2.Host.ID(),
		Addrs: p2.Host.Addrs(),
	}

	err := p1.Disconnect(p2aInfo)
	if err != nil {
		t.Fatal(err)
	}
	p1peers = p1.Peers()
	if len(p1peers) != 0 {
		t.Fatal("Peer count should be zero")
	}
	err = p1.Connect(context.Background(), p2aInfo)
	if err != nil {
		t.Fatal(err)
	}
	p1peers = p1.Peers()
	if len(p1peers) != 1 {
		t.Fatal("Peer count should be one")
	}
}

func TestCRDT(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second * 1)

	p1, p2, closer := setupPeers(t)
	defer closer(t)
	myvalue := "myValue"
	key := datastore.NewKey("mykey")
	err := p1.CrdtStore.Put(key, []byte(myvalue))
	if err != nil {
		t.Fatal(err)
	}
	err = p1.CrdtStore.Sync(datastore.NewKey("crdt"))
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Second * 6)
	ok, err := p2.CrdtStore.Has(key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Data not replicated")
	}
}

func TestFilesWithCRDT(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second * 1)

	p1, p2, closer := setupPeers(t)
	defer closer(t)

	content := []byte("hola")
	buf := bytes.NewReader(content)
	b := new(bytes.Buffer)
	_, err := b.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	key := datastore.NewKey("myfile")

	err = p1.CrdtStore.Put(key, b.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	r, err := p1.Store.Query(query.Query{})
	for v:= range r.Next() {
		fmt.Println(v.Key)
	}

	<-time.After(time.Second * 6)
	ok, err := p2.CrdtStore.Has(key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Data not replicated")
	}

	c2, err := p2.CrdtStore.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	b2 := bytes.NewReader(c2)

	content2, err := ioutil.ReadAll(b2)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(content, content2) {
		t.Error(string(content))
		t.Error(string(content2))
		t.Error("different content put and retrieved")
	}
}
