package matrix

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/datahop/ipfs-lite/internal/repo"
)

var root = filepath.Join("./test", "root1")

func TestNewMatrixKeeper(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	r := initRepo(t)
	defer r.Close()
	mKeeper := NewMatrixKeeper(r.Datastore())
	if mKeeper == nil {
		t.Fatal("Matrix keeper should not be null")
	}
}

func TestNodeMatrix(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	r := initRepo(t)
	defer r.Close()
	mKeeper := NewMatrixKeeper(r.Datastore())
	if mKeeper.NodeMatrix == nil {
		t.Fatal("NodeMatrix keeper should not be null")
	}

}

func TestMatrixKeeperFlush(t *testing.T) {
	<-time.After(time.Second)
	defer removeRepo(t)
	r := initRepo(t)
	defer r.Close()
	mKeeper := NewMatrixKeeper(r.Datastore())
	if mKeeper == nil {
		t.Fatal("Matrix keeper should not be null")
	}
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
	r := initRepo(t)
	defer r.Close()
	mKeeper := NewMatrixKeeper(r.Datastore())
	if mKeeper == nil {
		t.Fatal("Matrix keeper should not be null")
	}
	mKeeper.NodeMatrix.TotalUptime = 120
	discoveredNodeOne := DiscoveredNodeMatrix{
		ConnectionSuccessCount:           5,
		ConnectionFailureCount:           4,
		LastSuccessfulConnectionDuration: 120,
		LastConnected:                    time.Now().Unix(),
	}
	mKeeper.NodeMatrix.NodesDiscovered["discoveredNodeOne"] = discoveredNodeOne
	discoveredNodeTwo := DiscoveredNodeMatrix{
		ConnectionSuccessCount:           3,
		ConnectionFailureCount:           0,
		LastSuccessfulConnectionDuration: 1200,
		LastConnected:                    time.Now().Unix(),
	}
	mKeeper.NodeMatrix.NodesDiscovered["discoveredNodeTwo"] = discoveredNodeTwo
	err := mKeeper.Flush()
	if err != nil {
		t.Fatal(err)
	}
	mKeeper2 := NewMatrixKeeper(mKeeper.db)
	if mKeeper.NodeMatrix.NodesDiscovered["discoveredNodeTwo"].LastConnected != mKeeper2.NodeMatrix.NodesDiscovered["discoveredNodeTwo"].LastConnected {
		t.Fatal("discoveredNodeTwo LastConnected mismatch")
	}
}

func removeRepo(t *testing.T) {
	err := os.RemoveAll(root)
	if err != nil {
		t.Fatal(err)
	}
}

func initRepo(t *testing.T) repo.Repo {
	err := repo.Init(root, "0")
	if err != nil {
		t.Fatal(err)
	}
	r, err := repo.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	ds := r.Datastore()
	if ds == nil {
		t.Fatal(err)
	}
	return r
}
