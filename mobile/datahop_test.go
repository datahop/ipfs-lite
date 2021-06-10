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
	"github.com/datahop/ipfs-lite/internal/repo"
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
		err = Start()
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
	err = Start()
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
	if Peers() == NoPeersConnected {
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
	err = Start()
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
	if Peers() == NoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
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
	err = Start()
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
	if Peers() == NoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	defer func() {
		p.Cancel()
		p.Repo.Close()
		removeRepo(secondNode, t)
	}()
	for i := 0; i < 10; i++ {
		content := []byte(fmt.Sprintf("checkState%d", i))
		err := Add(fmt.Sprintf("tag%d", i), content)
		if err != nil {
			t.Fatal(err)
		}
	}
	if State() != 10 {
		t.Fatal("State should be 10")
	}
	<-time.After(time.Second * 5)
	if p.Repo.State() != 10 {
		t.Fatal("State should be 10")
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
	err = Start()
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
	if Peers() == NoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	defer func() {
		p.Cancel()
		p.Repo.Close()
		removeRepo(secondNode, t)
	}()
	for i := 0; i < 10; i++ {
		content := []byte(fmt.Sprintf("checkState%d", i))
		buf := bytes.NewReader(content)
		n, err := p.AddFile(context.Background(), buf, nil)
		if err != nil {
			t.Fatal(err)
		}
		p.Manager.Tag(fmt.Sprintf("tag%d", i), n.Cid())
	}
	if p.Repo.State() != 10 {
		t.Fatal("State should be 10")
	}
	<-time.After(time.Second * 5)
	if State() != 10 {
		t.Fatal("State should be 10")
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
	err = Start()
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
	if Peers() == NoPeersConnected {
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
