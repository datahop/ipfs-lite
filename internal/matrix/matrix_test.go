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
	"github.com/libp2p/go-libp2p-core/peer"
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

func TestMatrixKeeperNodeMatrixOps(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	mKeeper := NewMatrixKeeper(initDatastore(t))
	if mKeeper.NodeMatrix == nil {
		t.Fatal("NodeMatrix keeper should not be null")
	}
	defer mKeeper.db.Close()
	address := "discoveredNodeOne"
	mKeeper.BLEDiscovered(address)
	if mKeeper.GetNodeStat(address).BLEDiscoveredAt == 0 {
		t.Fatal("BLEDiscoveredAt not updated")
	}
	mKeeper.WifiConnected(address, 5, 5, 5)
	if mKeeper.GetNodeStat(address).WifiConnectedAt == 0 {
		t.Fatal("WifiConnectedAt not updated")
	}
	mKeeper.NodeConnectionFailed(address)
	if mKeeper.GetNodeStat(address).ConnectionFailureCount != 1 {
		t.Fatal("ConnectionFailureCount not updated")
	}

	mKeeper.BLEDiscovered(address)
	mKeeper.WifiConnected(address, 5, 5, 5)
	mKeeper.NodeConnected(address)
	if mKeeper.GetNodeStat(address).ConnectionSuccessCount != 1 {
		t.Fatal("ConnectionFailureCount not updated")
	}

	mKeeper.NodeDisconnected(address)
	if len(mKeeper.GetNodeStat(address).ConnectionHistory) != 1 {
		t.Fatal("ConnectionHistory not updated")
	}
}

func TestMatrixKeeperContentMatrixOps(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	mKeeper := NewMatrixKeeper(initDatastore(t))
	if mKeeper.NodeMatrix == nil {
		t.Fatal("NodeMatrix keeper should not be null")
	}
	defer mKeeper.db.Close()
	hash := "Hash"
	address := "discoveredNodeOne"
	tag := "Tag"
	var size int64 = 256
	mKeeper.NewContent(hash)
	mKeeper.ContentDownloadStarted(tag, hash, size)
	cs := mKeeper.GetContentStat(hash)
	if cs.Tag != tag && cs.DownloadStartedAt == 0 {
		t.Fatal("ContentDownloadStart failed")
	}
	mKeeper.ContentDownloadFinished(hash)
	cs = mKeeper.GetContentStat(hash)
	if cs.AvgSpeed == 0 && cs.DownloadFinishedAt == 0 {
		t.Fatal("ContentDownloadFinished failed")
	}
	mKeeper.ContentAddProvider(hash, peer.ID(address))
	if len(mKeeper.GetContentStat(hash).ProvidedBy) != 1 {
		t.Fatal("ContentAddProvider not updated")
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
	ctx, cancle := context.WithCancel(context.Background())
	mKeeper.StartTicker(ctx)
	<-time.After(time.Second * 31)
	if mKeeper.GetTotalUptime() != 30 {
		t.Fatal("TotalUptime should be 30")
	}
	defer func() {
		cancle()
		mKeeper.Close()
	}()
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
