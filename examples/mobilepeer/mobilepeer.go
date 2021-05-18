package main

import (
	"os"
	"time"

	"github.com/datahop/ipfs-lite/internal/repo"
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
	root := "/tmp" + string(os.PathSeparator) + repo.Root
	cm := MockConnManager{}
	err := datahop.Init(root, cm)
	if err != nil {
		panic(err)
	}
	defer datahop.Close()
	err = datahop.Start()
	if err != nil {
		panic(err)
	}
	<-time.After(time.Second * 10)
	datahop.Stop()
	datahop.Close()
}
