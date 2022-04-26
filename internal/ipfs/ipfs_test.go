package ipfs

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/h2non/filetype"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multihash"
)

func setupPeers(t *testing.T) (p1, p2 *Peer, closer func(t *testing.T)) {
	ctx, cancel := context.WithCancel(context.Background())

	root1 := filepath.Join("./test", "root1")
	root2 := filepath.Join("./test", "root2")

	err := repo.Init(root1, "0")
	if err != nil {
		t.Fatal(err)
	}
	r1, err := repo.Open(root1)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.Init(root2, "4502")
	if err != nil {
		t.Fatal(err)
	}
	r2, err := repo.Open(root2)
	if err != nil {
		t.Fatal(err)
	}

	closer = func(t *testing.T) {
		r1.Close()
		r2.Close()
		cancel()
	}

	p1, err = New(ctx, cancel, r1, nil, WithRebroadcastInterval(time.Second), WithCrdtNamespace("ipfslite"))
	if err != nil {
		closer(t)
		t.Fatal(err)
	}
	p2, err = New(ctx, cancel, r2, nil, WithRebroadcastInterval(time.Second), WithCrdtNamespace("ipfslite"))
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

func cleanup(t *testing.T) {
	root1 := filepath.Join("./test", "root1")
	root2 := filepath.Join("./test", "root2")
	for _, v := range []string{root1, root2} {
		err := os.RemoveAll(v)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestHost(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	root1 := filepath.Join("./test", "root1")
	err := repo.Init(root1, "0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(root1)
		if err != nil {
			t.Fatal(err)
		}
		cancel()
	}()
	r1, err := repo.Open(root1)
	if err != nil {
		t.Fatal(err)
	}
	defer r1.Close()
	p1, err := New(ctx, cancel, r1, nil)
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

func TestRepoClosed(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	root1 := filepath.Join("./test", "root1")
	defer func() {
		err := os.RemoveAll(root1)
		if err != nil {
			t.Fatal(err)
		}
		cancel()
	}()
	err := repo.Init(root1, "0")
	if err != nil {
		t.Fatal(err)
	}
	r1, err := repo.Open(root1)
	if err != nil {
		t.Fatal(err)
	}
	r1.Close()
	_, err = New(ctx, cancel, r1, nil)
	if err != repo.ErrorRepoClosed {
		t.Fatal(err)
	}
}

func TestDAG(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second)

	ctx := context.Background()
	p1, p2, closer := setupPeers(t)
	defer func() {
		closer(t)
		cleanup(t)
	}()

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
	<-time.After(time.Second)

	ctx := context.Background()
	p1, p2, closer := setupPeers(t)
	defer func() {
		closer(t)
		cleanup(t)
	}()

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

func TestAddFile(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second)

	p1, p2, closer := setupPeers(t)
	defer func() {
		closer(t)
		cleanup(t)
	}()

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

func TestDeleteFile(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second)

	p1, p2, closer := setupPeers(t)
	defer func() {
		closer(t)
		cleanup(t)
	}()

	content := []byte("content to be deleted")
	buf := bytes.NewReader(content)
	n, err := p1.AddFile(p1.Ctx, buf, nil)
	if err != nil {
		t.Fatal(err)
	}

	rsc, err := p2.GetFile(p2.Ctx, n.Cid())
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
	has, err := p1.BlockStore().Has(n.Cid())
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("blockstore should have the cid")
	}

	err = p1.DeleteFile(p1.Ctx, n.Cid())
	if err != nil {
		t.Fatal(err)
	}

	has, err = p1.BlockStore().Has(n.Cid())
	if err != nil {
		t.Fatal(err)
	}

	if has {
		t.Fatal("blockstore should not have the cid")
	}
}

func TestState(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	root1 := filepath.Join("./test", "root1")
	err := repo.Init(root1, "0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(root1)
		if err != nil {
			t.Fatal(err)
		}
		cancel()
	}()
	r1, err := repo.Open(root1)
	if err != nil {
		t.Fatal(err)
	}
	defer r1.Close()
	p1, err := New(ctx, cancel, r1, nil)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		content := []byte(fmt.Sprintf("checkState%d", i))
		buf := bytes.NewReader(content)
		n, err := p1.AddFile(context.Background(), buf, nil)
		if err != nil {
			t.Fatal(err)
		}
		meta := &replication.ContentMetatag{
			Size:      int64(len(content)),
			Type:      filetype.Unknown.Extension,
			Name:      fmt.Sprintf("tag%d", i),
			Hash:      n.Cid(),
			Timestamp: time.Now().Unix(),
			Owner:     p1.Host.ID(),
			Tag:       fmt.Sprintf("tag%d", i),
		}
		err = p1.Manager.Tag(fmt.Sprintf("tag%d", i), meta)
		if err != nil {
			t.Fatal(err)
		}
	}
	<-time.After(time.Second * 2)
	for i := 0; i < 10; i++ {
		if !p1.Repo.State().Test([]byte(fmt.Sprintf("tag%d", i))) {
			t.Fatal("tag should in bloom")
		}
	}
}

func TestStateDualPeer(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second)

	p1, p2, closer := setupPeers(t)
	defer func() {
		closer(t)
		cleanup(t)
	}()
	cids := []cid.Cid{}
	content := []byte("hola")
	buf := bytes.NewReader(content)
	n, err := p1.AddFile(p1.Ctx, buf, nil)
	if err != nil {
		t.Fatal(err)
	}
	meta := &replication.ContentMetatag{
		Size:      int64(len(content)),
		Type:      filetype.Unknown.Extension,
		Name:      "tag",
		Hash:      n.Cid(),
		Timestamp: time.Now().Unix(),
		Owner:     p1.Host.ID(),
	}
	err = p1.Manager.Tag("tag", meta)
	if err != nil {
		t.Fatal(err)
	}
	cids = append(cids, n.Cid())
	for i := 0; i < 10; i++ {
		content := []byte(fmt.Sprintf("checkState%d", i))
		buf := bytes.NewReader(content)
		n, err := p1.AddFile(p1.Ctx, buf, nil)
		if err != nil {
			t.Fatal(err)
		}
		meta := &replication.ContentMetatag{
			Size:      int64(len(content)),
			Type:      filetype.Unknown.Extension,
			Name:      fmt.Sprintf("tag%d", i),
			Hash:      n.Cid(),
			Timestamp: time.Now().Unix(),
			Owner:     p1.Host.ID(),
			Tag:       fmt.Sprintf("tag%d", i),
		}
		err = p1.Manager.Tag(fmt.Sprintf("tag%d", i), meta)
		if err != nil {
			t.Fatal(err)
		}
		cids = append(cids, n.Cid())
	}
	<-time.After(time.Second * 5)
	for i := 0; i < 10; i++ {
		if !p2.Repo.State().Test([]byte(fmt.Sprintf("tag%d", i))) {
			t.Fatal("tag should in bloom")
		}
	}
	for _, v := range cids {
		inStore, _ := p2.bstore.Has(v)
		if !inStore {
			t.Fatalf("%s is not in State", v.String())
		}
	}
}

func TestOperations(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second)

	p1, p2, closer := setupPeers(t)
	defer func() {
		closer(t)
		cleanup(t)
	}()
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
	<-time.After(time.Second)
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
	<-time.After(time.Second)

	p1, p2, closer := setupPeers(t)
	defer func() {
		closer(t)
		cleanup(t)
	}()

	myvalue := "myValue"
	key := datastore.NewKey("mykey")
	err := p1.Manager.Put(key, []byte(myvalue))
	if err != nil {
		t.Fatal(err)
	}

	ok := false
	for i := 0; i < 5; i++ {
		ok, err = p2.Manager.Has(key)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			break
		}
		<-time.After(time.Second)
	}
	if !ok {
		t.Fatal("Data not replicated")
	}
}

func TestFilesWithCRDT(t *testing.T) {
	// Wait one second for the datastore closer by the previous test
	<-time.After(time.Second)

	p1, p2, closer := setupPeers(t)
	defer func() {
		closer(t)
		cleanup(t)
	}()

	content := []byte("hola")
	buf := bytes.NewReader(content)
	b := new(bytes.Buffer)
	_, err := b.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	key := datastore.NewKey("myfile")

	err = p1.Manager.Put(key, b.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Second * 3)
	ok, err := p2.Manager.Has(key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Data not replicated")
	}

	c2, err := p2.Manager.Get(key)
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

var swarmKey = `/key/swarm/psk/1.0.0/
/base16/
e0db576d0215a9e2851b76a59c000a08e519a46e8385fc0b68fba753bd80530a`

func TestPrivateNetwork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	root1 := filepath.Join("./test-p", "root1")
	root2 := filepath.Join("./test-p", "root2")

	err := repo.Init(root1, "0")
	if err != nil {
		t.Fatal(err)
	}
	r1, err := repo.Open(root1)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.Init(root2, "4502")
	if err != nil {
		t.Fatal(err)
	}
	r2, err := repo.Open(root2)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		r1.Close()
		r2.Close()
		cancel()
		cleanupP()
	}()
	sk := []byte(swarmKey)
	p1, err := New(ctx, cancel, r1, sk, WithRebroadcastInterval(time.Second), WithCrdtNamespace("ipfslite"))
	if err != nil {
		t.Fatal(err)
	}
	p2, err := New(ctx, cancel, r2, nil, WithRebroadcastInterval(time.Second), WithCrdtNamespace("ipfslite"))
	if err != nil {
		t.Fatal(err)
	}

	pinfo2 := peer.AddrInfo{
		ID:    p2.Host.ID(),
		Addrs: p2.Host.Addrs(),
	}
	p1.Bootstrap([]peer.AddrInfo{pinfo2})
	if len(p1.Peers()) != 0 {
		t.Fatal("setting up private network failed")
	}
}

func TestPrivateNetworkSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	root1 := filepath.Join("./test-p", "root1")
	root2 := filepath.Join("./test-p", "root2")

	err := repo.Init(root1, "0")
	if err != nil {
		t.Fatal(err)
	}
	r1, err := repo.Open(root1)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.Init(root2, "4502")
	if err != nil {
		t.Fatal(err)
	}
	r2, err := repo.Open(root2)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		r1.Close()
		r2.Close()
		cancel()
		cleanupP()
	}()
	sk := []byte(swarmKey)
	p1, err := New(ctx, cancel, r1, sk, WithRebroadcastInterval(time.Second), WithCrdtNamespace("ipfslite"))
	if err != nil {
		t.Fatal(err)
	}
	p2, err := New(ctx, cancel, r2, sk, WithRebroadcastInterval(time.Second), WithCrdtNamespace("ipfslite"))
	if err != nil {
		t.Fatal(err)
	}

	pinfo2 := peer.AddrInfo{
		ID:    p2.Host.ID(),
		Addrs: p2.Host.Addrs(),
	}
	p1.Bootstrap([]peer.AddrInfo{pinfo2})
	<-time.After(time.Second)
	if len(p1.Peers()) == 0 {
		t.Fatal("setting up private network failed")
	}
}

func cleanupP() {
	root1 := filepath.Join("./test-p", "root1")
	root2 := filepath.Join("./test-p", "root2")
	for _, v := range []string{root1, root2} {
		os.RemoveAll(v)
	}
}
