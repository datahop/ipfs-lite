package datahop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	types "github.com/datahop/ipfs-lite/pb"
	"github.com/golang/protobuf/proto"

	"github.com/datahop/ipfs-lite/internal/repo"
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
		err := os.RemoveAll(root)
		if err != nil {
			t.Fatal(err)
		}
		Close()
	}()
	_, err = DiskUsage()
	if err != nil {
		t.Fatal(err)
	}

}

func TestMultipleStart(t *testing.T) {
	<-time.After(time.Second * 1)
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
		err := os.RemoveAll(root)
		if err != nil {
			t.Fatal(err)
		}
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

func TestReplication(t *testing.T) {
	<-time.After(time.Second * 1)
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
	defer func() {
		err := os.RemoveAll(root)
		if err != nil {
			t.Fatal(err)
		}
	}()
	err = Start()
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Second * 1)
	numberOfNewKeys := 10
	keyPrefix := "/key"
	for i := 0; i < numberOfNewKeys; i++ {
		r := types.Replica{
			Key:   fmt.Sprintf("/%s/%d", keyPrefix, i),
			Value: []byte(fmt.Sprintf("/value/%d", i)),
		}
		b, err := proto.Marshal(&r)
		if err != nil {
			t.Fatal(err)
		}
		err = Replicate(b)
		if err != nil {
			t.Fatal(err)
		}
	}
	content := types.Content{}
	contentB, err := GetReplicatedContent()
	if err != nil {
		t.Fatal(err)
	}
	err = proto.Unmarshal(contentB, &content)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, v := range content.Replicas {
		if strings.HasPrefix(v.Key, keyPrefix) {
			count++
		}
	}
	if count != numberOfNewKeys {
		t.Fatal("content length mismatch ", count, content.Replicas)
	}
	Stop()
	Close()
}
