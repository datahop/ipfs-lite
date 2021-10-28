package main

import (
	"fmt"
	"os"
	"time"

	"github.com/datahop/ipfs-lite/internal/repo"
	datahop "github.com/datahop/ipfs-lite/mobile"
	logger "github.com/ipfs/go-log/v2"
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

var log = logger.Logger("lite")

func main() {
	logger.SetLogLevel("lite", "Debug")
	root := "/tmp" + string(os.PathSeparator) + repo.Root
	cm := MockConnManager{}
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	err := datahop.Init(root, cm, dd, ad, whs, wc)
	if err != nil {
		panic(err)
	}
	defer datahop.Close()
	err = datahop.Start(true)
	if err != nil {
		panic(err)
	}
	fmt.Println(datahop.PeerInfo())
	<-time.After(time.Second * 30)
	datahop.Stop()
	datahop.Close()
}
