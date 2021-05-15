package datahop

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/datahop/ipfs-lite/internal/repo"
	types "github.com/datahop/ipfs-lite/pb"
	"github.com/golang/protobuf/proto"
)

type MockConnManager struct{}

func (m MockConnManager) PeerConnected(s string) {
	// do nothing
}

func (m MockConnManager) PeerDisconnected(s string) {
	// do nothing
}

func TestContentLength(t *testing.T) {
	root := "../test" + string(os.PathSeparator) + repo.Root
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
	root := "../test" + string(os.PathSeparator) + repo.Root
	cm := MockConnManager{}
	err := Init(root, cm, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer Close()
	for i := 0; i < 10; i++ {
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
	}
}

func TestReplication(t *testing.T) {
	<-time.After(time.Second * 1)
	root := filepath.Join("../test", "datahop")
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
	defer Stop()
	<-time.After(time.Second * 1)
	for i := 0; i < 10; i++ {
		r := types.Replica{
			Key:   fmt.Sprintf("/key/%d", i),
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
	if len(content.Replicas) != 10 {
		t.Fatal("content length mismatch")
	}
}
