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

	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/datahop/ipfs-lite/pkg"
	"github.com/datahop/ipfs-lite/pkg/store"
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

func (m MockDisDriver) Start(localPID, peerInfo string, scanTime int, interval int) {
	// do nothing
}

func (m MockDisDriver) AddAdvertisingInfo(topic string, info string) {
	// do nothing
}

func (m MockDisDriver) Stop() {
	// do nothing
}

type MockAdvDriver struct{}

func (m MockAdvDriver) Start(localPID, peerInfo string) {
	// do nothing
}

func (m MockAdvDriver) AddAdvertisingInfo(topic string, info string) {
	// do nothing
}

func (m MockAdvDriver) Stop() {
	// do nothing
}

func (m MockAdvDriver) NotifyNetworkInformation(network string, pass string) {
	// do nothing
}

func (m MockAdvDriver) NotifyEmptyValue() {
	// do nothing
}

type MockWifiConn struct{}

func (m MockWifiConn) Connect(network, pass, ip, host string) {
	// do nothing
}

func (m MockWifiConn) Disconnect() {
	// do nothing
}

func (m MockWifiConn) Host() string {
	return ""
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
		Close()
		removeRepo(root, t)
	}()
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()

	pInfo := PeerInfo()
	var peerInfo peer.AddrInfo
	err = peerInfo.UnmarshalJSON([]byte(pInfo))
	if err != nil {
		t.Fatal(err)
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

	if _, err := Addrs(); err != pkg.ErrNoPeerAddress {
		t.Fatal(err)
	}

	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	_, err = Addrs()
	if err != nil {
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	if _, err := Peers(); err != pkg.ErrNoPeersConnected {
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
	err = StartDiscovery(true, true, false)
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

func TestStartPrivate(t *testing.T) {
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
		err = StartPrivate("my_secret", true)
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
		err = Start(false, true)
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	comm := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm.Stop()
		comm.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr := comm.Node.AddrInfo()
	prb, err := pr.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	st := string(prb)
	err = BootstrapWithPeerInfo(st)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	comm := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm.Stop()
		comm.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr := comm.Node.AddrInfo()
	for _, v := range pr.Addrs {
		if !strings.HasPrefix(v.String(), "/ip4/127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	// test matrix
	nodeStatSnapshot := hop.comm.Repo.Matrix().GetNodeStat(pr.ID.String())
	if nodeStatSnapshot.ConnectionSuccessCount != 1 {
		t.Fatal("ConnectionSuccessCount is not 1")
	}
	pi := PeerInfo()
	var peerInfo peer.AddrInfo
	err = peerInfo.UnmarshalJSON([]byte(pi))
	if err != nil {
		t.Fatal(err)
	}
}

func TestConnectWithAddressWithGroupKeyFail(t *testing.T) {
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
	err = StartPrivate("my_secret", true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	comm := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm.Stop()
		comm.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr := comm.Node.AddrInfo()
	for _, v := range pr.Addrs {
		if !strings.HasPrefix(v.String(), "/ip4/127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr.ID.String())
			if err == nil {
				t.Fatal("peers should now be able to connect")
			}
		}
	}
}

func TestConnectWithAddressWithGroupKey(t *testing.T) {
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
	err = StartPrivate("my_secret", true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	comm := startAnotherNode(secondNode, "3214", "my_secret", t)
	defer func() {
		comm.Stop()
		comm.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr := comm.Node.AddrInfo()
	for _, v := range pr.Addrs {
		if !strings.HasPrefix(v.String(), "/ip4/127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	// test matrix
	nodeStatSnapshot := hop.comm.Repo.Matrix().GetNodeStat(pr.ID.String())
	if nodeStatSnapshot.ConnectionSuccessCount != 1 {
		t.Fatal("ConnectionSuccessCount is not 1")
	}
	pi := PeerInfo()
	var peerInfo peer.AddrInfo
	err = peerInfo.UnmarshalJSON([]byte(pi))
	if err != nil {
		t.Fatal(err)
	}
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	comm := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm.Stop()
		comm.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr := comm.Node.AddrInfo()
	for _, v := range pr.Addrs {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	bf1, err := State()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		content := []byte(fmt.Sprintf("checkState%d", i))
		err := Add(fmt.Sprintf("tag%d", i), content, "")
		if err != nil {
			t.Fatal(err)
		}
	}
	<-time.After(time.Second * 5)
	bf2, err := State()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(bf1, bf2) {
		t.Fatal("bloom filter is same after addition")
	}
	bf3, err := comm.Repo.State().MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bf3, bf2) {
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	comm := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm.Stop()
		comm.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr := comm.Node.AddrInfo()
	for _, v := range pr.Addrs {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	content := []byte("checkState")
	buf := bytes.NewReader(content)
	tag := "tag"
	info := &store.Info{
		Tag:  tag,
		Type: filetype.Unknown.Extension,
		Name: tag,
		Size: int64(len(content)),
	}
	_, err = comm.Node.Add(context.Background(), buf, info)
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Second * 6)
	c, err := Get(tag, "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(c, content) {
		t.Fatal("content mismatch")
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	comm := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm.Stop()
		comm.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr := comm.Node.AddrInfo()
	for _, v := range pr.Addrs {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	bf1, err := State()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		content := []byte(fmt.Sprintf("checkState%d", i))
		buf := bytes.NewReader(content)

		info := &store.Info{
			Tag:  fmt.Sprintf("tag%d", i),
			Type: filetype.Unknown.Extension,
			Name: fmt.Sprintf("tag%d", i),
			Size: int64(len(content)),
		}
		_, err = comm.Node.Add(context.Background(), buf, info)
		if err != nil {
			t.Fatal(err)
		}

	}
	<-time.After(time.Second * 5)
	bf2, err := comm.Repo.State().MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(bf1, bf2) {
		t.Fatal("bloom filter are same")
	}
	bf3, err := State()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bf3, bf2) {
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	comm := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm.Stop()
		comm.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr := comm.Node.AddrInfo()
	prb, err := pr.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	st := string(prb)
	err = ConnectWithPeerInfo(st)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
}

func TestContentOwner(t *testing.T) {
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	comm := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm.Stop()
		comm.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr := comm.Node.AddrInfo()
	for _, v := range pr.Addrs {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	for i := 0; i < 10; i++ {
		content := []byte(fmt.Sprintf("checkState%d", i))
		err := Add(fmt.Sprintf("tag%d", i), content, "")
		if err != nil {
			t.Fatal(err)
		}
	}
	<-time.After(time.Second * 2)
	for i := 0; i < 10; i++ {
		meta, err := comm.Node.ReplManager().FindTag(fmt.Sprintf("tag%d", i))
		if err != nil {
			t.Fatal(err, fmt.Sprintf("tag%d", i))
		}
		if meta.Owner != hop.comm.Node.AddrInfo().ID {
			t.Fatal("Owner mismatch")
		}
	}
}

func TestContentMatrix(t *testing.T) {
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()
	secondNode := filepath.Join("./test", "root1")
	comm := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm.Stop()
		comm.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr := comm.Node.AddrInfo()
	for _, v := range pr.Addrs {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	for i := 0; i < 10; i++ {
		content := []byte(fmt.Sprintf("checkState%d", i))
		err := Add(fmt.Sprintf("tag%d", i), content, "")
		if err != nil {
			t.Fatal(err)
		}
	}
	<-time.After(time.Second * 2)
	for _, v := range comm.Repo.Matrix().ContentMatrix {
		if len(v.ProvidedBy) > 0 && v.ProvidedBy[0] != hop.comm.Node.AddrInfo().ID {
			t.Fatal("provider info is wrong")
		}
	}
}

func TestContentDistribution(t *testing.T) {
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()

	secondNode := filepath.Join("./test", "root1")
	comm1 := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm1.Stop()
		comm1.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr1 := comm1.Node.AddrInfo()
	for _, v := range pr1.Addrs {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr1.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}
	content := []byte("check_distribution")
	err = Add("tag", content, "")
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Second * 2)
	for _, v := range comm1.Repo.Matrix().ContentMatrix {
		if len(v.ProvidedBy) > 0 && v.ProvidedBy[0] != hop.comm.Node.AddrInfo().ID {
			t.Fatal("provider info is wrong")
		}
	}
	Stop()
	Close()
	thirdNode := filepath.Join("./test", "root2")
	comm2 := startAnotherNode(thirdNode, "5001", "", t)
	defer func() {
		comm2.Stop()
		comm2.Repo.Close()
		removeRepo(thirdNode, t)
	}()
	pr2 := comm2.Node.AddrInfo()
	err = comm1.Node.Connect(*pr2)
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Second * 5)
	for _, v := range comm2.Repo.Matrix().ContentMatrix {
		if len(v.ProvidedBy) > 0 && v.ProvidedBy[0] != pr1.ID {
			t.Fatal("provider info is wrong", len(v.ProvidedBy), v.ProvidedBy[0], v.ProvidedBy[1])
		}
	}
	comm1.Stop()
	comm1.Repo.Close()
	fourthNode := filepath.Join("./test", "root3")
	comm3 := startAnotherNode(fourthNode, "5002", "", t)
	defer func() {
		comm3.Stop()
		comm3.Repo.Close()
		removeRepo(fourthNode, t)
	}()
	pr3 := comm3.Node.AddrInfo()
	err = comm2.Node.Connect(*pr3)
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Second * 5)
	for _, v := range comm3.Repo.Matrix().ContentMatrix {
		if len(v.ProvidedBy) > 0 && v.ProvidedBy[0] != pr2.ID {
			t.Fatal("provider info is wrong")
		}
	}
}

func TestGroupMemberAdd(t *testing.T) {
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()

	secondNode := filepath.Join("./test", "root1")
	comm1 := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm1.Stop()
		comm1.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr1 := comm1.Node.AddrInfo()
	for _, v := range pr1.Addrs {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr1.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}

	group, err := CreateOpenGroup("group1")
	if err != nil {
		t.Fatal(err)
	}
	groupId, err := peer.Decode(group)
	if err != nil {
		t.Fatal(err)
	}

	err = AddMember(comm1.Node.AddrInfo().ID.String(), group)
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Second * 5)
	_, err = comm1.Node.ReplManager().GroupGetInfo(comm1.Node.AddrInfo().ID, groupId, comm1.Node.GetPrivKey())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGroupStateOnContentAdd(t *testing.T) {
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()

	secondNode := filepath.Join("./test", "root1")
	comm1 := startAnotherNode(secondNode, "3214", "", t)
	defer func() {
		comm1.Stop()
		comm1.Repo.Close()
		removeRepo(secondNode, t)
	}()
	pr1 := comm1.Node.AddrInfo()
	for _, v := range pr1.Addrs {
		if !strings.HasPrefix(v.String(), "127") {
			err := ConnectWithAddress(v.String() + "/p2p/" + pr1.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Peers(); err == pkg.ErrNoPeersConnected {
		t.Fatal("Should be connected to at least one peer")
	}

	group, err := CreateOpenGroup("group1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = peer.Decode(group)
	if err != nil {
		t.Fatal(err)
	}
	f1, err := hop.comm.Repo.StateKeeper().GetState(group)
	if err != nil {
		t.Fatal(err)
	}
	if f1.Membership != true {
		t.Fatal("membership of node should be true")
	}
	_, err = comm1.Repo.StateKeeper().GetState(group)
	if err == nil {
		t.Fatal("error should not be nil")
	}

	content := []byte("check_group_distribution")
	err = GroupAdd("tag", content, "", group)
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Second * 5)
	_, err = comm1.Repo.StateKeeper().GetState(group)
	if err == nil {
		t.Fatal("error should be nil")
	}

	err = AddMember(comm1.Node.AddrInfo().ID.String(), group)
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Second * 5)

	f2, err := comm1.Repo.StateKeeper().GetState(group)
	if err != nil {
		t.Fatal("error should not be nil ", err)
	}
	if f2.Membership != true {
		t.Fatal("membership of second node should be true now")
	}
	if !f2.Filter.TestString("tag") {
		t.Fatal("tag should be present")
	}
}

func TestContentEncryption(t *testing.T) {
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
	err = Start(false, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Stop()
		Close()
	}()

	pass := "check_encryption"
	content := []byte(pass)
	tag := "tag"
	err = Add(tag, content, pass)
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Second * 2)
	ct, err := Get(tag, "")
	if err == nil {
		t.Fatal(err)
	}
	if bytes.Equal(content, ct) {
		t.Fatal("encryption did not wor")
	}
	ct, err = Get(tag, pass)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, ct) {
		t.Fatal("encryption did not wor")
	}
}

func removeRepo(repopath string, t *testing.T) {
	err := os.RemoveAll(repopath)
	if err != nil {
		t.Fatal(err)
	}
}

func startAnotherNode(repopath, port, key string, t *testing.T) *pkg.Common {
	err := pkg.Init(repopath, port)
	if err != nil {
		t.Fatal(err)
	}
	comm, err := pkg.New(context.Background(), repopath, port, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = comm.Start(key, true)
	if err != nil {
		t.Fatal(err)
	}
	return comm
}
