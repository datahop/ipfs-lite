package ipfslite

import (
	"bytes"
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	cbor "github.com/ipfs/go-ipld-cbor"
	peer "github.com/libp2p/go-libp2p-core/peer"
	multihash "github.com/multiformats/go-multihash"
)

var secret = "2cc2c79ea52c9cc85dfd3061961dd8c4230cce0b09f182a0822c1536bf1d5f21"

func setupPeers(t *testing.T) (p1, p2 *Peer, closer func(t *testing.T)) {
	ctx, cancel := context.WithCancel(context.Background())

	root1 := filepath.Join("./test", "root1")
	root2 := filepath.Join("./test", "root2")
	conf1, err := ConfigInit(2048, "0")
	if err != nil {
		t.Fatal(err)
	}
	err = Init(root1, conf1)
	if err != nil {
		t.Fatal(err)
	}
	r1, err := Open(root1)
	if err != nil {
		t.Fatal(err)
	}

	conf2, err := ConfigInit(2048, "4502")
	if err != nil {
		t.Fatal(err)
	}
	err = Init(root2, conf2)
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

func TestDAG(t *testing.T) {
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
	<-time.After(time.Second * 3)
}

func TestSession(t *testing.T) {
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
	<-time.After(time.Second * 3)
}

func TestFiles(t *testing.T) {
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
	<-time.After(time.Second * 3)
}
