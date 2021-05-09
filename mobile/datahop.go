// Package datahop is a mobile client for running a minimalistic ipfs node.
package datahop

// add a separate GOMODCACHE fro faster builds

//go:generate ./generate.sh

import (
	"context"
	"errors"
	"strings"
	"time"

	ipfslite "github.com/datahop/ipfs-lite"
	types "github.com/datahop/ipfs-lite/pb"
	"github.com/datahop/ipfs-lite/version"
	"github.com/golang/protobuf/proto"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	logger "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	log = logger.Logger("datahop")
	hop *datahop
)

// ConnectionManager is used by clients to get notified client connection
type ConnectionManager interface {
	PeerConnected(string)
	PeerDisconnected(string)
}

// BleManager is used by clients to interact with Bluetooth Low Energy
type BleManager interface {
	StartAdvertising()
	StopAdvertising()
	StartScanning()
	StopScanning()
	StartGATTServer()
	StopGATTServer()
}

// WifiP2PManager is used by clients to mange wifi direct
type WifiP2PManager interface {
	StartHotspot() (string, error) // Returns "ssid:password"
	StopHotspot()
}

// WifiManager is used by clients to mange wifi connection
type WifiManager interface {
	Connect(string, string) // takes in ssid and password
	Disconnect()
}

type Notifier struct{}

func (n *Notifier) Listen(network.Network, ma.Multiaddr)      {}
func (n *Notifier) ListenClose(network.Network, ma.Multiaddr) {}
func (n *Notifier) Connected(net network.Network, c network.Conn) {
	hop.hook.PeerConnected(c.RemotePeer().String())
}
func (n *Notifier) Disconnected(net network.Network, c network.Conn) {
	hop.hook.PeerDisconnected(c.RemotePeer().String())
}
func (n *Notifier) OpenedStream(net network.Network, s network.Stream) {}
func (n *Notifier) ClosedStream(network.Network, network.Stream)       {}

type datahop struct {
	ctx             context.Context
	cancel          context.CancelFunc
	root            string
	peer            *ipfslite.Peer
	identity        ipfslite.Identity
	hook            ConnectionManager
	networkNotifier network.Notifiee
	ble             BleManager
	repo            ipfslite.Repo
}

func init() {
	logger.SetLogLevel("datahop", "Debug")
	logger.SetLogLevel("ipfslite", "Debug")
}

// Init Initialises the .datahop repo, if required at the given location with the given swarm port as config.
// Default swarm port is 4501
func Init(root string, h ConnectionManager, ble BleManager) error {
	err := ipfslite.Init(root, "0")
	if err != nil {
		return err
	}
	n := &Notifier{}
	r, err := ipfslite.Open(root)
	if err != nil {
		log.Error("Repo Open Failed : ", err.Error())
		return err
	}
	cfg, err := r.Config()
	if err != nil {
		log.Error("Config Failed : ", err.Error())
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	hop = &datahop{
		root:            root,
		identity:        cfg.Identity,
		hook:            h,
		networkNotifier: n,
		ble:             ble,
		ctx:             ctx,
		cancel:          cancel,
		repo:            r,
	}
	return nil
}

func DiskUsage() (uint64, error) {
	if hop == nil {
		return 0, errors.New("datahop not initialised")
	}
	return datastore.DiskUsage(hop.repo.Datastore())
}

// StartDiscovery initialises ble services
//
// It starts three services in three different go routines. Advertising scanning and gatt server
// Advertising happens every one minute interval for 30 seconds. Same for scanning. Although
// scanning starts 30 seconds after initials start, i.e. After advertising stops.
func StartDiscovery() error {
	if hop == nil {
		return errors.New("start failed. datahop not initialised")
	}
	go func() {
		advertiseTicker := time.NewTicker(time.Minute * 1)
		defer advertiseTicker.Stop()
		for {
			hop.ble.StartAdvertising()
			<-time.After(time.Second * 30)
			hop.ble.StopAdvertising()
			select {
			case <-hop.ctx.Done():
				return
			case <-advertiseTicker.C:
			}
		}
	}()
	go func() {
		hop.ble.StartGATTServer()
		<-hop.ctx.Done()
		hop.ble.StopGATTServer()
	}()
	go func() {
		<-time.After(time.Second * 30)
		scannerTicker := time.NewTicker(time.Minute * 1)
		defer scannerTicker.Stop()
		for {
			hop.ble.StartScanning()
			<-time.After(time.Second * 30)
			hop.ble.StopScanning()
			select {
			case <-hop.ctx.Done():
				return
			case <-scannerTicker.C:
			}
		}
	}()
	return nil
}

// Start an ipfslite node in a go routine
func Start() error {
	if hop == nil {
		return errors.New("start failed. datahop not initialised")
	}

	go func() {
		ctx, cancel := context.WithCancel(hop.ctx)
		p, err := ipfslite.New(ctx, cancel, hop.repo)
		if err != nil {
			log.Error("Node setup failed : ", err.Error())
			return
		}
		hop.peer = p
		hop.peer.Host.Network().Notify(hop.networkNotifier)
		select {
		case <-hop.ctx.Done():
			log.Debug("Context Closed")
		}
	}()
	log.Debug("Node Started")
	return nil
}

// ConnectWithAddress Connects to a given peer address
func ConnectWithAddress(address string) error {
	addr, _ := ma.NewMultiaddr(address)
	peerInfo, _ := peer.AddrInfosFromP2pAddrs(addr)

	for _, v := range peerInfo {
		err := hop.peer.Connect(context.Background(), v)
		if err != nil {
			return err
		}
	}
	return nil
}

// ConnectWithPeerInfo Connects to a given peerInfo string
func ConnectWithPeerInfo(peerInfoByteString string) error {
	var peerInfo peer.AddrInfo
	err := peerInfo.UnmarshalJSON([]byte(peerInfoByteString))
	if err != nil {
		return err
	}
	err = hop.peer.Connect(context.Background(), peerInfo)
	if err != nil {
		return err
	}
	return nil
}

// Bootstrap Connects to a given peerInfo string
func Bootstrap(peerInfoByteString string) error {
	var peerInfo peer.AddrInfo
	err := peerInfo.UnmarshalJSON([]byte(peerInfoByteString))
	if err != nil {
		return err
	}
	hop.peer.Bootstrap([]peer.AddrInfo{peerInfo})
	return nil
}

// GetPeerInfo Returns string of the peer.AddrInfo []byte of the node
func GetPeerInfo() string {
	for i := 0; i < 5; i++ {
		if hop.peer != nil {
			pr := peer.AddrInfo{
				ID:    hop.peer.Host.ID(),
				Addrs: hop.peer.Host.Addrs(),
			}
			prb, err := pr.MarshalJSON()
			if err != nil {
				return "Could not get peerInfo"
			}
			return string(prb)
		}
		<-time.After(time.Millisecond * 200)
	}
	return "Could not get peerInfo"
}

// GetID Returns peerId of the node
func GetID() string {
	return hop.identity.PeerID
}

// GetAddress Returns a comma(,) separated string of all the possible addresses of a node
func GetAddress() string {
	for i := 0; i < 5; i++ {
		addrs := []string{}
		if hop.peer != nil {
			for _, v := range hop.peer.Host.Addrs() {
				addrs = append(addrs, v.String()+"/p2p/"+hop.peer.Host.ID().String())
			}
			return strings.Join(addrs, ",")
		}
		<-time.After(time.Millisecond * 200)
	}
	return "Could not get peer address"
}

// IsNodeOnline Checks if the node is running
func IsNodeOnline() bool {
	if hop.peer != nil {
		return hop.peer.IsOnline()
	}
	return false
}

// Peers Returns a comma(,) separated string of all the connected peers of a node
func Peers() string {
	if hop != nil && hop.peer != nil {
		return strings.Join(hop.peer.Peers(), ",")
	}
	return "No Peers connected"
}

// Replicate adds a record in the crdt store
func Replicate(replica []byte) error {
	if hop != nil && hop.peer != nil {
		if hop.peer.CrdtStore == nil {
			return errors.New("replication module not running")
		}
		r := types.Replica{}
		err := proto.Unmarshal(replica, &r)
		if err != nil {
			return err
		}

		dsKey := datastore.NewKey(r.GetKey())
		err = hop.peer.CrdtStore.Put(dsKey, r.GetValue())
		if err != nil {
			return err
		}
	}
	return errors.New("datahop ipfs-lite node is not running")
}

// GetReplicatedValue retrieves a record from the crdt store
func GetReplicatedValue(key string) ([]byte, error) {
	if hop != nil && hop.peer != nil {
		if hop.peer.CrdtStore == nil {
			return []byte{}, errors.New("replication module not running")
		}
		dsKey := datastore.NewKey(key)
		value, err := hop.peer.CrdtStore.Get(dsKey)
		if err != nil {
			return []byte{}, err
		}
		r := types.Replica{
			Key:   key,
			Value: value,
		}
		data, err := proto.Marshal(&r)
		if err != nil {
			return []byte{}, err
		}
		return data, nil
	}
	return []byte{}, errors.New("datahop ipfs-lite node is not running")
}

// GetReplicatedContent retrieves all records from the crdt store
func GetReplicatedContent() ([]byte, error) {
	if hop != nil && hop.peer != nil {
		if hop.peer.CrdtStore == nil {
			return []byte{}, errors.New("replication module not running")
		}
		records := types.Content{}
		results, err := hop.peer.CrdtStore.Query(query.Query{})
		if err != nil {
			return []byte{}, err
		}
		for v := range results.Next() {
			records.Replicas = append(records.Replicas, &types.Replica{
				Key:   v.Key,
				Value: v.Value,
			})
		}
		content, err := proto.Marshal(&records)
		if err != nil {
			return []byte{}, err
		}
		return content, nil
	}
	return []byte{}, errors.New("datahop ipfs-lite node is not running")
}

// RemoveReplication deletes record from the crdt store
func RemoveReplication(key string) error {
	if hop != nil && hop.peer != nil {
		if hop.peer.CrdtStore == nil {
			return errors.New("replication module not running")
		}
		dsKey := datastore.NewKey(key)
		err := hop.peer.CrdtStore.Delete(dsKey)
		if err != nil {
			return err
		}
	}
	return errors.New("datahop ipfs-lite node is not running")
}

// Version of ipfs-lite
func Version() string {
	return version.Version
}

// Stop the node
func Stop() {
	hop.peer.Cancel()
}

// Close the repo and all
func Close() {
	hop.repo.Close()
	hop.cancel()
}
