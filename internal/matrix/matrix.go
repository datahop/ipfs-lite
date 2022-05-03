package matrix

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	log = logging.Logger("matrix")

	nodeMatrixKey    = datastore.NewKey("/node-matrix")
	contentMatrixKey = datastore.NewKey("/content-matrix")
)

// ContentMatrix keeps record for each hash
type ContentMatrix struct {
	Tag                string
	Size               int64
	AvgSpeed           float32
	DownloadStartedAt  int64
	DownloadFinishedAt int64
	ProvidedBy         []peer.ID
}

// NodeMatrix keeps track of uptime and connectivity matrix of a node
type NodeMatrix struct {
	TotalUptime     int64                            // total time node has been inline (seconds)
	NodesDiscovered map[string]*DiscoveredNodeMatrix // Nodes discovered this session
}

// DiscoveredNodeMatrix keeps record for each discovered node
type DiscoveredNodeMatrix struct {
	ConnectionAlive                  bool
	ConnectionSuccessCount           int
	ConnectionFailureCount           int
	LastSuccessfulConnectionDuration int64
	BLEDiscoveredAt                  int64
	WifiConnectedAt                  int64
	RSSI                             int
	Speed                            int
	Frequency                        int
	IPFSConnectedAt                  int64
	DiscoveryDelays                  []int64 // from BLE Discovery to ipfs Connection
	ConnectionHistory                []ConnectionInfo
}

// ConnectionInfo keeps d2d connection info
type ConnectionInfo struct {
	BLEDiscoveredAt int64
	WifiConnectedAt int64
	RSSI            int
	Speed           int
	Frequency       int
	IPFSConnectedAt int64
	DisconnectedAt  int64
}

// MatrixKeeper takes care of entire node matrix
type MatrixKeeper struct {
	mtx           sync.Mutex
	db            datastore.Datastore
	NodeMatrix    *NodeMatrix
	ContentMatrix map[string]*ContentMatrix
}

// NewMatrixKeeper creates a new matrix keeper
func NewMatrixKeeper(ds datastore.Datastore) *MatrixKeeper {
	mKeeper := &MatrixKeeper{
		db: ds,
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

// StartTicker starts a ticker to flush matrix information into datastore after certain interval
func (mKeeper *MatrixKeeper) StartTicker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 30):
				mKeeper.mtx.Lock()
				mKeeper.NodeMatrix.TotalUptime += 30
				mKeeper.flush()
				mKeeper.mtx.Unlock()
			}
		}
	}()
}

// Flush matrix info into datastore
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

// Close stops the ticker job and flushes matrix
func (mKeeper *MatrixKeeper) Close() error {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()
	err := mKeeper.flush()
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debug("matrix closing")
	return nil
}

// GetNodeStat returns node connectivity info of a given node
func (mKeeper *MatrixKeeper) GetNodeStat(address string) DiscoveredNodeMatrix {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	return *mKeeper.NodeMatrix.NodesDiscovered[address]
}

// GetContentStat returns content info of a given hash
func (mKeeper *MatrixKeeper) GetContentStat(hash string) ContentMatrix {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()
	log.Debug(mKeeper.ContentMatrix)
	return *mKeeper.ContentMatrix[hash]
}

// GetTotalUptime returns the total time the node has been running
func (mKeeper *MatrixKeeper) GetTotalUptime() int64 {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	return mKeeper.NodeMatrix.TotalUptime
}

// BLEDiscovered updates BLE connectivity info with another node
func (mKeeper *MatrixKeeper) BLEDiscovered(address string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.NodeMatrix.NodesDiscovered[address] == nil {
		mKeeper.NodeMatrix.NodesDiscovered[address] = &DiscoveredNodeMatrix{
			DiscoveryDelays:   []int64{},
			ConnectionHistory: []ConnectionInfo{},
		}
	}
	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	if nodeMatrix.ConnectionAlive {
		return
	}
	nodeMatrix.BLEDiscoveredAt = time.Now().Unix()
}

// WifiConnected updates wifi connectivity info with another node
func (mKeeper *MatrixKeeper) WifiConnected(address string, rssi, speed, freq int) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.NodeMatrix.NodesDiscovered[address] == nil {
		mKeeper.NodeMatrix.NodesDiscovered[address] = &DiscoveredNodeMatrix{
			DiscoveryDelays:   []int64{},
			ConnectionHistory: []ConnectionInfo{},
		}
	}
	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	if nodeMatrix.ConnectionAlive {
		return
	}
	nodeMatrix.WifiConnectedAt = time.Now().Unix()
	nodeMatrix.RSSI = rssi
	nodeMatrix.Speed = speed
	nodeMatrix.Frequency = freq
}

// NodeConnected updates connectivity time info with another node
func (mKeeper *MatrixKeeper) NodeConnected(address string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.NodeMatrix.NodesDiscovered[address] == nil {
		mKeeper.NodeMatrix.NodesDiscovered[address] = &DiscoveredNodeMatrix{
			DiscoveryDelays:   []int64{},
			ConnectionHistory: []ConnectionInfo{},
		}
	}
	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	nodeMatrix.ConnectionSuccessCount++
	nodeMatrix.IPFSConnectedAt = time.Now().Unix()
	nodeMatrix.ConnectionAlive = true
	if nodeMatrix.BLEDiscoveredAt != 0 {
		delay := nodeMatrix.IPFSConnectedAt - nodeMatrix.BLEDiscoveredAt
		if delay == 0 {
			delay = 1
		}
		nodeMatrix.DiscoveryDelays = append(nodeMatrix.DiscoveryDelays, delay)
	}
}

// NodeConnectionFailed updates connectivity fail time info with another node
func (mKeeper *MatrixKeeper) NodeConnectionFailed(address string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.NodeMatrix.NodesDiscovered[address] == nil {
		mKeeper.NodeMatrix.NodesDiscovered[address] = &DiscoveredNodeMatrix{
			DiscoveryDelays:   []int64{},
			ConnectionHistory: []ConnectionInfo{},
		}
	}

	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	nodeMatrix.ConnectionFailureCount++

	nodeMatrix.BLEDiscoveredAt = 0
	nodeMatrix.WifiConnectedAt = 0
	nodeMatrix.RSSI = 0
	nodeMatrix.Speed = 0
	nodeMatrix.Frequency = 0
}

// NodeDisconnected updates connectivity disconnection time info with another node
func (mKeeper *MatrixKeeper) NodeDisconnected(address string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	nodeMatrix := mKeeper.NodeMatrix.NodesDiscovered[address]
	nodeMatrix.LastSuccessfulConnectionDuration = time.Now().Unix() - nodeMatrix.IPFSConnectedAt

	history := ConnectionInfo{
		BLEDiscoveredAt: nodeMatrix.BLEDiscoveredAt,
		WifiConnectedAt: nodeMatrix.WifiConnectedAt,
		RSSI:            nodeMatrix.RSSI,
		Speed:           nodeMatrix.Speed,
		Frequency:       nodeMatrix.Frequency,
		IPFSConnectedAt: nodeMatrix.IPFSConnectedAt,
		DisconnectedAt:  time.Now().Unix(),
	}
	if nodeMatrix.ConnectionHistory == nil {
		nodeMatrix.ConnectionHistory = []ConnectionInfo{}
	}
	nodeMatrix.ConnectionHistory = append(nodeMatrix.ConnectionHistory, history)
	nodeMatrix.ConnectionAlive = false
	nodeMatrix.BLEDiscoveredAt = 0
	nodeMatrix.WifiConnectedAt = 0
	nodeMatrix.IPFSConnectedAt = 0
	nodeMatrix.RSSI = 0
	nodeMatrix.Speed = 0
	nodeMatrix.Frequency = 0
}

func (mKeeper *MatrixKeeper) NewContent(hash string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	if mKeeper.ContentMatrix[hash] == nil {
		mKeeper.ContentMatrix[hash] = &ContentMatrix{
			ProvidedBy: []peer.ID{},
		}
	}
}

// ContentDownloadStarted updates content download start time of a hash
func (mKeeper *MatrixKeeper) ContentDownloadStarted(tag, hash string, size int64) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()
	contentMatrix := mKeeper.ContentMatrix[hash]
	contentMatrix.Tag = tag
	contentMatrix.DownloadStartedAt = time.Now().Unix()
	contentMatrix.Size = size
	log.Debug("ContentDownloadStarted : ", contentMatrix)
}

// ContentDownloadFinished updates content download finished time of a hash
func (mKeeper *MatrixKeeper) ContentDownloadFinished(hash string) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	contentMatrix := mKeeper.ContentMatrix[hash]
	contentMatrix.DownloadFinishedAt = time.Now().Unix()
	timeConsumed := contentMatrix.DownloadFinishedAt - contentMatrix.DownloadStartedAt
	if timeConsumed == 0 {
		timeConsumed = 1
	}
	contentMatrix.AvgSpeed = sizeInMB(contentMatrix.Size) / float32(timeConsumed)
	log.Debug("ContentDownloadFinished : ", contentMatrix)
}

// ContentAddProvider updates content providers of a hash
func (mKeeper *MatrixKeeper) ContentAddProvider(hash string, provider peer.ID) {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()
	contentMatrix := mKeeper.ContentMatrix[hash]
	contentMatrix.ProvidedBy = append(contentMatrix.ProvidedBy, provider)
}

// NodeMatrixSnapshot returns a snapshot of the node connectivity matrix
func (mKeeper *MatrixKeeper) NodeMatrixSnapshot() map[string]interface{} {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	retMap := make(map[string]interface{})
	for k, v := range mKeeper.NodeMatrix.NodesDiscovered {
		retMap[k] = *v
	}
	log.Debug("NodeMatrixSnapshot ", retMap)
	return retMap
}

// ContentMatrixSnapshot returns a snapshot of the content distribution matrix
func (mKeeper *MatrixKeeper) ContentMatrixSnapshot() map[string]interface{} {
	mKeeper.mtx.Lock()
	defer mKeeper.mtx.Unlock()

	retMap := make(map[string]interface{})
	for k, v := range mKeeper.ContentMatrix {
		retMap[k] = *v
	}
	log.Debug("ContentMatrixSnapshot ", retMap)
	return retMap
}

func sizeInMB(size int64) float32 {
	return float32(size) / float32(1024*1024)
}
