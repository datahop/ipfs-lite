package matrix

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
)

var (
	nodeMatrixKey    = datastore.NewKey("/node-matrix")
	contentMatrixKey = datastore.NewKey("/content-matrix")
	log              = logging.Logger("matrix")
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
	mtx             sync.Mutex
	stop            chan struct{}
	isTickerRunning bool
	db              datastore.Datastore
	NodeMatrix      *NodeMatrix
	ContentMatrix   map[string]ContentMatrix
}

func NewMatrixKeeper(ds datastore.Datastore) *MatrixKeeper {
	mKeeper := &MatrixKeeper{
		stop: make(chan struct{}),
		db:   ds,
		NodeMatrix: &NodeMatrix{
			TotalUptime:     0,
			NodesDiscovered: map[string]*DiscoveredNodeMatrix{},
		},
		ContentMatrix: map[string]ContentMatrix{},
	}
	n, err := mKeeper.db.Get(nodeMatrixKey)
	if err != nil {
		log.Error("Unable to get nodeMatrixKey : ", err)
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
	if mKeeper.isTickerRunning {
		return
	}
	mKeeper.mtx.Lock()
	mKeeper.isTickerRunning = true
	mKeeper.mtx.Unlock()
	go func() {
		for {
			select {
			case <-mKeeper.stop:
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

	return mKeeper.flush()
}

func (mKeeper *MatrixKeeper) flush() error {
	nm, err := json.Marshal(mKeeper.NodeMatrix)
	if err != nil {
		log.Error(err)
		return err
	}
	cm, err := json.Marshal(mKeeper.ContentMatrix)
	if err != nil {
		log.Error(err)
		return err
	}
	err = mKeeper.db.Put(nodeMatrixKey, nm)
	if err != nil {
		log.Error(err)
		return err
	}
	err = mKeeper.db.Put(contentMatrixKey, cm)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debug("Flushed")
	return nil
}

func (mKeeper *MatrixKeeper) Close() error {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()
	err := mKeeper.flush()
	if err != nil {
		log.Error(err)
		return err
	}
	if mKeeper.isTickerRunning {
		mKeeper.stop <- struct{}{}
	}
	return nil
}

func (mKeeper *MatrixKeeper) GetNodeStat(address string) DiscoveredNodeMatrix {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	return *mKeeper.NodeMatrix.NodesDiscovered[address]
}

func (mKeeper *MatrixKeeper) GetTotalUptime() int64 {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	return mKeeper.NodeMatrix.TotalUptime
}

func (mKeeper *MatrixKeeper) NodeConnected(address string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.NodeMatrix.NodesDiscovered[address] == nil {
		mKeeper.NodeMatrix.NodesDiscovered[address] = &DiscoveredNodeMatrix{}
	}
	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	nodeMatrix.ConnectionSuccessCount++
	nodeMatrix.LastConnected = time.Now().Unix()
}

func (mKeeper *MatrixKeeper) NodeDisconnected(address string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	nodeMatrix.LastSuccessfulConnectionDuration = time.Now().Unix() - nodeMatrix.LastConnected
}

func (mKeeper *MatrixKeeper) Snapshot() ([]byte, error) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	return json.Marshal(mKeeper)
}

func (mKeeper *MatrixKeeper) SnapshotStruct() MatrixKeeper {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()
	log.Debug(mKeeper.NodeMatrix.TotalUptime)
	return *mKeeper
}
