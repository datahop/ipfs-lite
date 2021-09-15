package matrix

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	syncds "github.com/ipfs/go-datastore/sync"
	leveldb "github.com/ipfs/go-ds-leveldb"
)

var root = filepath.Join("./test", "root1")

func TestNewMatrixKeeper(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	mKeeper := NewMatrixKeeper(initDatastore(t))
	if mKeeper == nil {
		t.Fatal("Matrix keeper should not be null")
	}
	defer mKeeper.db.Close()
}

func TestNodeMatrix(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	mKeeper := NewMatrixKeeper(initDatastore(t))
	if mKeeper.NodeMatrix == nil {
		t.Fatal("NodeMatrix keeper should not be null")
	}
	defer mKeeper.db.Close()
}

func TestMatrixKeeperFlush(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	mKeeper := NewMatrixKeeper(initDatastore(t))
	if mKeeper.NodeMatrix == nil {
		t.Fatal("NodeMatrix keeper should not be null")
	}
	defer mKeeper.db.Close()
	has, err := mKeeper.db.Has(nodeMatrixKey)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("db should not have the key")
	}
	has, err = mKeeper.db.Has(contentMatrixKey)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("db should not have the key")
	}
	err = mKeeper.Flush()
	if err != nil {
		t.Fatal(err)
	}
	has, err = mKeeper.db.Has(nodeMatrixKey)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("db should have the key")
	}
	has, err = mKeeper.db.Has(contentMatrixKey)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("db should have the key")
	}
}

func TestMatrixKeeperFlushWithData(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	mKeeper := NewMatrixKeeper(initDatastore(t))
	if mKeeper.NodeMatrix == nil {
		t.Fatal("NodeMatrix keeper should not be null")
	}
	defer mKeeper.db.Close()
	mKeeper.NodeMatrix.TotalUptime = 120
	discoveredNodeOne := DiscoveredNodeMatrix{
		ConnectionSuccessCount:           5,
		ConnectionFailureCount:           4,
		LastSuccessfulConnectionDuration: 120,
		IPFSConnectedAt:                  time.Now().Unix(),
	}
	mKeeper.NodeMatrix.NodesDiscovered["discoveredNodeOne"] = &discoveredNodeOne
	discoveredNodeTwo := DiscoveredNodeMatrix{
		ConnectionSuccessCount:           3,
		ConnectionFailureCount:           0,
		LastSuccessfulConnectionDuration: 1200,
		IPFSConnectedAt:                  time.Now().Unix(),
	}
	mKeeper.NodeMatrix.NodesDiscovered["discoveredNodeTwo"] = &discoveredNodeTwo
	contentOne := &ContentMatrix{
		AvgSpeed:          5,
		DownloadStartedAt: time.Now().Unix() - 150,
	}
	mKeeper.ContentMatrix["contentOne"] = contentOne
	err := mKeeper.Flush()
	if err != nil {
		t.Fatal(err)
	}
	mKeeper2 := NewMatrixKeeper(mKeeper.db)
	if mKeeper.NodeMatrix.NodesDiscovered["discoveredNodeTwo"].IPFSConnectedAt != mKeeper2.NodeMatrix.NodesDiscovered["discoveredNodeTwo"].IPFSConnectedAt {
		t.Fatal("discoveredNodeTwo IPFSConnectedAt mismatch")
	}

	if mKeeper.ContentMatrix["contentOne"].AvgSpeed != mKeeper2.ContentMatrix["contentOne"].AvgSpeed {
		t.Fatal("contentOne avgSpeed mismatch")
	}
}

func TestMatrixKeeperTicker(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	mKeeper := NewMatrixKeeper(initDatastore(t))
	if mKeeper == nil {
		t.Fatal("Matrix keeper should not be null")
	}
	defer mKeeper.db.Close()
	mKeeper.StartTicker()
	<-time.After(time.Second * 31)
	if mKeeper.GetTotalUptime() != 30 {
		t.Fatal("TotalUptime should be 30")
	}
	defer mKeeper.Close()
}

func removeRepo(t *testing.T) {
	err := os.RemoveAll(root)
	if err != nil {
		t.Fatal(err)
	}
}

func initDatastore(t *testing.T) ds.Batching {
	d, err := leveldb.NewDatastore(filepath.Join(root, "datastore"), &leveldb.Options{})
	if err != nil {
		t.Fatal(err)
	}
	datastore := syncds.MutexWrap(d)
	return datastore
}
