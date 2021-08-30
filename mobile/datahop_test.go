package datahop

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/h2non/filetype"
	"github.com/libp2p/go-libp2p-core/peer"
)

type MockConnManager struct{}

func (m MockConnManager) PeerConnected(s string) {
	// do nothing
}

func (m MockConnManager) PeerDisconnected(s string) {
	// do nothing
}

type MockDisDriver struct{}

func (m MockDisDriver) Start(localPID string, scanTime int, interval int) {
	// do nothing
}

func (m MockDisDriver) AddAdvertisingInfo(topic string, info []byte) {
	// do nothing
}

func (m MockDisDriver) Stop() {
	// do nothing
}

type MockAdvDriver struct{}

func (m MockAdvDriver) Start(localPID string) {
	// do nothing
}

func (m MockAdvDriver) AddAdvertisingInfo(topic string, info []byte) {
	// do nothing
}

func (m MockAdvDriver) Stop() {
	// do nothing
}

func (m MockAdvDriver) NotifyNetworkInformation(network string, pass string, info string) {
	// do nothing
}

func (m MockAdvDriver) NotifyEmptyValue() {
	// do nothing
}

type MockWifiConn struct{}

func (m MockWifiConn) Connect(network string, pass string, ip string) {
	// do nothing
}

func (m MockWifiConn) Disconnect() {
	// do nothing
}

type MockWifiHotspot struct{}

func (m MockWifiHotspot) Start() {
	// do nothing
}

func (m MockWifiHotspot) Stop() {
	// do nothing
}

func TestInit(t *testing.T) {
	root := "../test" + string(os.PathSeparator) + repo.Root
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		removeRepo(root, t)
		Close()
	}()
	err = Start(false)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	if ID() != hop.identity.PeerID {
		t.Fatal("ID() returns different id than config identity")
	}
	pInfo := PeerInfo()
	var peerInfo peer.AddrInfo
	err = peerInfo.UnmarshalJSON([]byte(pInfo))
	if err != nil {
		t.Fatal(err)
	}
	if peerInfo.ID.String() != hop.identity.PeerID {
		t.Fatal("peerInfo.ID.String() & hop.identity.PeerID do not match")
	}
}

func TestAddresses(t *testing.T) {
	root := "../test" + string(os.PathSeparator) + repo.Root
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		removeRepo(root, t)
		Close()
	}()

	if _, err := Addrs(); err == ErrNoPeerAddress {
		t.Fatal(err)
	}
	if _, err := InterfaceAddrs(); err == ErrNoPeerAddress {
		t.Fatal(err)
	}

	err = Start(false)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	if _, err := Addrs(); err != nil {
		t.Fatal(err)
	}
	if _, err := InterfaceAddrs(); err != nil {
		t.Fatal(err)
	}
}

func TestNoPeerConnected(t *testing.T) {
	root := "../test" + string(os.PathSeparator) + repo.Root
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		removeRepo(root, t)
		Close()
	}()
	err = Start(false)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	if _, err := Peers(); err != ErrNoPeersConnected {
		t.Fatal(err)
	}
}

func TestStartStopDiscovery(t *testing.T) {
	root := "../test" + string(os.PathSeparator) + repo.Root
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		removeRepo(root, t)
		Close()
	}()
	err = StartDiscovery()
	if err != nil {
		t.Fatal(err)
	}
	err = StopDiscovery()
	if err != nil {
		t.Fatal(err)
	}
}

func TestContentLength(t *testing.T) {
	root := "../test" + string(os.PathSeparator) + repo.Root
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		removeRepo(root, t)
		Close()
	}()
	_, err = DiskUsage()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultipleStart(t *testing.T) {
	<-time.After(time.Second)
	root := "../test" + string(os.PathSeparator) + repo.Root
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		removeRepo(root, t)
		Close()
	}()
	for i := 0; i < 10; i++ {
		err = Start(false)
		if err != nil {
			t.Fatal(err)
		}
		if IsNodeOnline() != true {
			t.Fatal("Node should be running")
		}
		Stop()
		if IsNodeOnline() != false {
			t.Fatal("Node should not be running ")
		}
	}
}

func TestBootstrap(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../test", repo.Root)
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
	err = Start(false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	p := startAnotherNode(secondNode, t)
	pr := peer.AddrInfo{
		ID:    p.Host.ID(),
		Addrs: p.Host.Addrs(),
	}
	prb, err := pr.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	st := string(prb)
	err = Bootstrap(st)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Peers(); err == ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	defer func() {
		p.Cancel()
		p.Repo.Close()
		removeRepo(secondNode, t)
	}()
}

func TestConnectWithAddress(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../test", repo.Root)
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
	err = Start(false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	p := startAnotherNode(secondNode, t)
	for _, v := range p.Host.Addrs() {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + p.Host.ID().String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}

	// test matrix
	if hop.peer.Matrix.NodeMatrix.NodesDiscovered[p.Host.ID().String()].ConnectionSuccessCount != 1 {
		t.Fatal("ConnectionSuccessCount is not 1")
	}
	pi := PeerInfo()
	var peerInfo peer.AddrInfo
	err = peerInfo.UnmarshalJSON([]byte(pi))
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Second * 5)
	err = p.Disconnect(peerInfo)
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 100)
	if hop.peer.Matrix.NodeMatrix.NodesDiscovered[p.Host.ID().String()].LastSuccessfulConnectionDuration != 5 {
		t.Fatal("LastSuccessfulConnectionDuration should be 5")
	}
	defer func() {
		p.Cancel()
		p.Repo.Close()
		removeRepo(secondNode, t)
	}()
}

func TestReplicationOut(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../test", repo.Root)
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
	err = Start(false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	p := startAnotherNode(secondNode, t)
	for _, v := range p.Host.Addrs() {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + p.Host.ID().String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	defer func() {
		p.Cancel()
		p.Repo.Close()
		removeRepo(secondNode, t)
	}()
	bf1, err := State()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		content := []byte(fmt.Sprintf("checkState%d", i))
		err := Add(fmt.Sprintf("tag%d", i), content)
		if err != nil {
			t.Fatal(err)
		}
	}
	bf2, err := State()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(bf1, bf2) == 0 {
		t.Fatal("bloom filter is same after addition")
	}
	<-time.After(time.Second * 5)
	bf3, err := p.Repo.State().MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(bf3, bf2) != 0 {
		t.Fatal("bloom filter should be same")
	}
}

func TestReplicationGet(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../test", repo.Root)
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
	err = Start(false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	p := startAnotherNode(secondNode, t)
	for _, v := range p.Host.Addrs() {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + p.Host.ID().String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	defer func() {
		p.Cancel()
		p.Repo.Close()
		removeRepo(secondNode, t)
	}()
	content := []byte(fmt.Sprintf("checkState"))
	buf := bytes.NewReader(content)
	n, err := p.AddFile(context.Background(), buf, nil)
	if err != nil {
		t.Fatal(err)
	}
	tag := "tag"
	meta := &replication.Metatag{
		Size:      int64(len(content)),
		Type:      filetype.Unknown.Extension,
		Name:      tag,
		Hash:      n.Cid(),
		Timestamp: time.Now().Unix(),
	}
	err = p.Manager.Tag(tag, meta)
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Second * 6)
	c, err := Get(tag)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(c, content) {
		t.Fatal("contect mismatch")
	}
}

func TestReplicationIn(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../test", repo.Root)
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
	err = Start(false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	p := startAnotherNode(secondNode, t)
	for _, v := range p.Host.Addrs() {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + p.Host.ID().String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	defer func() {
		p.Cancel()
		p.Repo.Close()
		removeRepo(secondNode, t)
	}()
	bf1, err := State()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		content := []byte(fmt.Sprintf("checkState%d", i))
		buf := bytes.NewReader(content)
		n, err := p.AddFile(context.Background(), buf, nil)
		if err != nil {
			t.Fatal(err)
		}
		meta := &replication.Metatag{
			Size:      int64(len(content)),
			Type:      filetype.Unknown.Extension,
			Name:      fmt.Sprintf("tag%d", i),
			Hash:      n.Cid(),
			Timestamp: time.Now().Unix(),
		}
		p.Manager.Tag(fmt.Sprintf("tag%d", i), meta)
	}
	bf2, err := p.Repo.State().MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(bf1, bf2) == 0 {
		t.Fatal("bloom filter are same")
	}
	<-time.After(time.Second * 5)
	bf3, err := State()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(bf3, bf2) != 0 {
		t.Fatal("bloom filter should be same")
	}
}

func TestConnectWithPeerInfo(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../test", repo.Root)
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
	err = Start(false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	p := startAnotherNode(secondNode, t)
	pr := peer.AddrInfo{
		ID:    p.Host.ID(),
		Addrs: p.Host.Addrs(),
	}
	prb, err := pr.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	st := string(prb)
	err = ConnectWithPeerInfo(st)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Peers(); err == ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	defer func() {
		p.Cancel()
		p.Repo.Close()
		removeRepo(secondNode, t)
	}()
}

func removeRepo(repopath string, t *testing.T) {
	err := os.RemoveAll(repopath)
	if err != nil {
		t.Fatal(err)
	}
}

func startAnotherNode(repopath string, t *testing.T) *ipfslite.Peer {
	ctx, cancel := context.WithCancel(context.Background())
	err := repo.Init(repopath, "5000")
	if err != nil {
		t.Fatal(err)
	}
	r1, err := repo.Open(repopath)
	if err != nil {
		t.Fatal(err)
	}
	p, err := ipfslite.New(ctx, cancel, r1, ipfslite.WithmDNS(false))
	if err != nil {
		t.Fatal(err)
	}
	return p
}
