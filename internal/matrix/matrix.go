package matrix

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
)

var (
	log = logging.Logger("matrix")

	nodeMatrixKey    = datastore.NewKey("/node-matrix")
	contentMatrixKey = datastore.NewKey("/content-matrix")
)

type Client int

const (
	MobileClient Client = iota
	CliClient
)

type ContentMatrix struct {
	Size               int64
	AvgSpeed           float32
	Replicators        []string
	DownloadStartedAt  int64
	DownloadFinishedAt int64
	LastProvidedAt     int64
}

type NodeMatrix struct {
	ClientInfo      Client
	TotalUptime     int64                            // total time node has been inline (seconds)
	NodesDiscovered map[string]*DiscoveredNodeMatrix // Nodes discovered this session
}

type DiscoveredNodeMatrix struct {
	ConnectionSuccessCount           int
	ConnectionFailureCount           int
	LastSuccessfulConnectionDuration int64
	BLEDiscoveryAt                   int64
	WifiDirectAt                     int64
	ConnectedAt                      int64
}

type MatrixKeeper struct {
	mtx             sync.Mutex
	stop            chan struct{}
	isTickerRunning bool
	db              datastore.Datastore
	NodeMatrix      *NodeMatrix
	ContentMatrix   map[string]*ContentMatrix
}

func NewMatrixKeeper(ds datastore.Datastore) *MatrixKeeper {
	mKeeper := &MatrixKeeper{
		stop: make(chan struct{}),
		db:   ds,
		NodeMatrix: &NodeMatrix{
			TotalUptime:     0,
			NodesDiscovered: map[string]*DiscoveredNodeMatrix{},
		},
		ContentMatrix: map[string]*ContentMatrix{},
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

func (mKeeper *MatrixKeeper) GetContentStat(hash string) ContentMatrix {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()
	log.Debug(mKeeper.ContentMatrix)
	return *mKeeper.ContentMatrix[hash]
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
	nodeMatrix.ConnectedAt = time.Now().Unix()
}

func (mKeeper *MatrixKeeper) BLEDiscovered(address string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.NodeMatrix.NodesDiscovered[address] == nil {
		mKeeper.NodeMatrix.NodesDiscovered[address] = &DiscoveredNodeMatrix{}
	}
	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	nodeMatrix.BLEDiscoveryAt = time.Now().Unix()
}

func (mKeeper *MatrixKeeper) WifiConnected(address string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.NodeMatrix.NodesDiscovered[address] == nil {
		mKeeper.NodeMatrix.NodesDiscovered[address] = &DiscoveredNodeMatrix{}
	}
	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	nodeMatrix.WifiDirectAt = time.Now().Unix()
}

func (mKeeper *MatrixKeeper) NodeConnectionFailed(address string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.NodeMatrix.NodesDiscovered[address] == nil {
		mKeeper.NodeMatrix.NodesDiscovered[address] = &DiscoveredNodeMatrix{}
	}

	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	nodeMatrix.ConnectionFailureCount++
}

func (mKeeper *MatrixKeeper) NodeDisconnected(address string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	nodeMatrix.LastSuccessfulConnectionDuration = time.Now().Unix() - nodeMatrix.ConnectedAt
}

func (mKeeper *MatrixKeeper) ContentDownloadStarted(hash string, size int64) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.ContentMatrix[hash] == nil {
		mKeeper.ContentMatrix[hash] = &ContentMatrix{}
	}
	contentMatrix := mKeeper.ContentMatrix[hash]
	contentMatrix.DownloadStartedAt = time.Now().Unix()
	contentMatrix.Size = size
	log.Debug("ContentDownloadStarted : ", contentMatrix)
}

func (mKeeper *MatrixKeeper) ContentDownloadFinished(hash string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.ContentMatrix[hash] == nil {
		mKeeper.ContentMatrix[hash] = &ContentMatrix{}
	}
	contentMatrix := mKeeper.ContentMatrix[hash]
	contentMatrix.DownloadFinishedAt = time.Now().Unix()
	timeConsumed := contentMatrix.DownloadFinishedAt - contentMatrix.DownloadStartedAt
	if timeConsumed == 0 {
		timeConsumed = 1
	}
	contentMatrix.AvgSpeed = sizeInMB(contentMatrix.Size) / float32(timeConsumed)
	log.Debug("ContentDownloadFinished : ", contentMatrix)
}

func (mKeeper *MatrixKeeper) NodeMatrixSnapshot() map[string]interface{} {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	retMap := make(map[string]interface{})
	for k, v := range mKeeper.NodeMatrix.NodesDiscovered {
		retMap[k] = *v
	}
	return retMap
}

func (mKeeper *MatrixKeeper) ContentMatrixSnapshot() map[string]interface{} {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	retMap := make(map[string]interface{})
	for k, v := range mKeeper.ContentMatrix {
		retMap[k] = *v
	}
	return retMap
}

func sizeInMB(size int64) float32 {
	return float32(size) / float32(1024*1024)
}
