package datahop

import (
	"os"
	"testing"
	"time"

	ipfslite "github.com/datahop/ipfs-lite"
)

type MockConnManager struct{}

func (m MockConnManager) PeerConnected(s string) {
	// do nothing
}

func (m MockConnManager) PeerDisconnected(s string) {
	// do nothing
}

func TestContentLength(t *testing.T) {
	root := "/tmp" + string(os.PathSeparator) + ipfslite.Root
	cm := MockConnManager{}
	err := Init(root, cm, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer Close()
	_, err = DiskUsage()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultipleStart(t *testing.T) {
	<-time.After(time.Second * 1)
	root := "/tmp" + string(os.PathSeparator) + ipfslite.Root
	cm := MockConnManager{}
	err := Init(root, cm, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer Close()
	err = Start()
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Second * 1)
	if IsNodeOnline() != true {
		t.Fatal("Node should be running")
	}
	Stop()
	<-time.After(time.Second * 1)
	if IsNodeOnline() != false {
		t.Fatal("Node should not be running ")
	}
	err = Start()
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Second * 1)
	if IsNodeOnline() != true {
		t.Fatal("Node should be running")
	}
}
