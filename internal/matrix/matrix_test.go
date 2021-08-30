package matrix

import (
	"context"
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
	mKeeper := NewMatrixKeeper(context.Background(), initDatastore(t))
	if mKeeper == nil {
		t.Fatal("Matrix keeper should not be null")
	}
	defer mKeeper.db.Close()
}

func TestNodeMatrix(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	mKeeper := NewMatrixKeeper(context.Background(), initDatastore(t))
	if mKeeper.NodeMatrix == nil {
		t.Fatal("NodeMatrix keeper should not be null")
	}
	defer mKeeper.db.Close()
}

func TestMatrixKeeperFlush(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	mKeeper := NewMatrixKeeper(context.Background(), initDatastore(t))
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
}

func TestMatrixKeeperFlushWithData(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	mKeeper := NewMatrixKeeper(context.Background(), initDatastore(t))
	if mKeeper.NodeMatrix == nil {
		t.Fatal("NodeMatrix keeper should not be null")
	}
	defer mKeeper.db.Close()
	mKeeper.NodeMatrix.TotalUptime = 120
	discoveredNodeOne := DiscoveredNodeMatrix{
		ConnectionSuccessCount:           5,
		ConnectionFailureCount:           4,
		LastSuccessfulConnectionDuration: 120,
		LastConnected:                    time.Now().Unix(),
	}
	mKeeper.NodeMatrix.NodesDiscovered["discoveredNodeOne"] = &discoveredNodeOne
	discoveredNodeTwo := DiscoveredNodeMatrix{
		ConnectionSuccessCount:           3,
		ConnectionFailureCount:           0,
		LastSuccessfulConnectionDuration: 1200,
		LastConnected:                    time.Now().Unix(),
	}
	mKeeper.NodeMatrix.NodesDiscovered["discoveredNodeTwo"] = &discoveredNodeTwo
	err := mKeeper.Flush()
	if err != nil {
		t.Fatal(err)
	}
	mKeeper2 := NewMatrixKeeper(context.Background(), mKeeper.db)
	if mKeeper.NodeMatrix.NodesDiscovered["discoveredNodeTwo"].LastConnected != mKeeper2.NodeMatrix.NodesDiscovered["discoveredNodeTwo"].LastConnected {
		t.Fatal("discoveredNodeTwo LastConnected mismatch")
	}
}

func TestMatrixKeeperTicker(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	mKeeper := NewMatrixKeeper(ctx, initDatastore(t))
	if mKeeper == nil {
		t.Fatal("Matrix keeper should not be null")
	}
	defer mKeeper.db.Close()
	mKeeper.StartTicker()
	<-time.After(time.Second * 15)
	if mKeeper.GetTotalUptime() != 10 {
		t.Fatal("TotalUptime should be 10")
	}
	defer cancel()
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
