package matrix

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
)

var (
	nodeMatrixKey    = datastore.NewKey("/node-matrix")
	contentMatrixKey = datastore.NewKey("/content-matrix")
)

type ContentMatrix struct {
	AvgSpeed       float32
	Replicators    []string
	Completed      int32
	DownloadedAt   int64
	LastProvidedAt int64
}

type NodeMatrix struct {
	TotalUptime     int64                            // total time node has been inline (seconds)
	NodesDiscovered map[string]*DiscoveredNodeMatrix // Nodes discovered this session
}

type DiscoveredNodeMatrix struct {
	ConnectionSuccessCount           int
	ConnectionFailureCount           int
	LastSuccessfulConnectionDuration int64
	LastConnected                    int64
}

type MatrixKeeper struct {
	mtx           sync.Mutex
	ctx           context.Context
	db            datastore.Datastore
	NodeMatrix    *NodeMatrix
	ContentMatrix map[string]ContentMatrix
}

func NewMatrixKeeper(ctx context.Context, ds datastore.Datastore) *MatrixKeeper {
	mKeeper := &MatrixKeeper{
		ctx: ctx,
		db:  ds,
		NodeMatrix: &NodeMatrix{
			TotalUptime:     0,
			NodesDiscovered: map[string]*DiscoveredNodeMatrix{},
		},
		ContentMatrix: map[string]ContentMatrix{},
	}
	n, err := mKeeper.db.Get(nodeMatrixKey)
	if err != nil {
		return mKeeper
	}
	c, err := mKeeper.db.Get(contentMatrixKey)
	if err != nil {
		return mKeeper
	}
	err = json.Unmarshal(n, mKeeper.NodeMatrix)
	if err != nil {
		return mKeeper
	}
	err = json.Unmarshal(c, &mKeeper.ContentMatrix)
	if err != nil {
		return mKeeper
	}
	return mKeeper
}

func (mKeeper *MatrixKeeper) StartTicker() {
	go func() {
		for {
			select {
			case <-mKeeper.ctx.Done():
				return
			case <-time.After(time.Second * 10):
				mKeeper.mtx.Lock()
				mKeeper.NodeMatrix.TotalUptime += 10
				mKeeper.mtx.Unlock()
			}
		}
	}()
}

func (mKeeper *MatrixKeeper) Flush() error {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	nm, err := json.Marshal(mKeeper.NodeMatrix)
	if err != nil {
		return err
	}
	cm, err := json.Marshal(mKeeper.ContentMatrix)
	if err != nil {
		return err
	}
	err = mKeeper.db.Put(nodeMatrixKey, nm)
	if err != nil {
		return err
	}
	err = mKeeper.db.Put(contentMatrixKey, cm)
	if err != nil {
		return err
	}
	return nil
}
