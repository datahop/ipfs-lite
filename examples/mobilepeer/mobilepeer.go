package main

import (
	"os"
	"time"

	ipfslite "github.com/datahop/ipfs-lite"
	datahop "github.com/datahop/ipfs-lite/mobile"
	logger "github.com/ipfs/go-log/v2"
)

type MockConnManager struct{}

func (m MockConnManager) PeerConnected(s string) {
	// do something
}

func (m MockConnManager) PeerDisconnected(s string) {
	// do something
}

var log = logger.Logger("lite")

func main() {
	logger.SetLogLevel("lite", "Debug")
	root := "/tmp" + string(os.PathSeparator) + ipfslite.Root
	cm := MockConnManager{}
	err := datahop.Init(root, cm, nil)
	if err != nil {
		panic(err)
	}
	defer datahop.Close()
	err = datahop.Start()
	if err != nil {
		panic(err)
	}
	<-time.After(time.Second * 1)

	for {
		select {
		case <-time.After(time.Second * 5):
			log.Debugf("Addresses : %v", datahop.Addrs())
		}
	}
}
