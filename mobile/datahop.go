// Package datahop is a mobile client for running a minimalistic ipfs node.
package datahop

//go:generate gomobile bind -o datahop.aar -target=android github.com/datahop/ipfs-lite/mobile

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
func Init(root string, connManager ConnectionManager, bleManager BleManager) error {
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
		hook:            connManager,
		networkNotifier: n,
		ble:             bleManager,
		ctx:             ctx,
		cancel:          cancel,
		repo:            r,
	}
	return nil
}

func DiskUsage() (int64, error) {
	if hop == nil {
		return 0, errors.New("datahop not initialised")
	}
	du, err := datastore.DiskUsage(hop.repo.Datastore())
	if err != nil {
		return 0, err
	}
	return int64(du), nil
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

// PeerInfo Returns string of the peer.AddrInfo []byte of the node
func PeerInfo() string {
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

// ID Returns peerId of the node
func ID() string {
	return hop.identity.PeerID
}

// Addrs Returns a comma(,) separated string of all the possible addresses of a node
func Addrs() string {
	for i := 0; i < 5; i++ {
		addrs := []string{}
		if hop.peer != nil {
			for _, v := range hop.peer.Host.Addrs() {
				if !strings.HasPrefix(v.String(), "127") {
					addrs = append(addrs, v.String()+"/p2p/"+hop.peer.Host.ID().String())
				}
			}
			return strings.Join(addrs, ",")
		}
		<-time.After(time.Millisecond * 200)
	}
	return "Could not get peer address"
}

// InterfaceAddrs returns a list of addresses at which this network
// listens. It expands "any interface" addresses (/ip4/0.0.0.0, /ip6/::) to
// use the known local interfaces.
func InterfaceAddrs() string {
	for i := 0; i < 5; i++ {
		addrs := []string{}
		if hop.peer != nil {
			interfaceAddrs, err := hop.peer.Host.Network().InterfaceListenAddresses()
			if err == nil {
				for _, v := range interfaceAddrs {
					if !strings.HasPrefix(v.String(), "127") {
						addrs = append(addrs, v.String()+"/p2p/"+hop.peer.Host.ID().String())
					}
				}
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
		return nil
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
