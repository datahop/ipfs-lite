//package datahop is a mobile client for running a minimalistic ipfs node.
package datahop

//go:generate gomobile bind -o datahop.aar -target=android github.com/datahop/ipfs-lite/mobile

import (
	"context"
	"errors"
	"strings"
	"time"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/datahop/ipfs-lite/version"
	logger "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	log = logger.Logger("datahop")
	hop *datahop
)

// Hook is used by clients
type ConnectionHook interface {
	PeerConnected(string)
	PeerDisconnected(string)
}

// BleHook is used by clients
type BleHook interface {
	StartAdvertising()
	StopAdvertising()
	StartScanning()
	StopScanning()
	StartGATTServer()
	StopGATTServer()
}

// Hook is used by clients
type WifiHook interface {
	StartHotspot() (string, string)
	StopHotspot()
	Connect(string, string)
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
	identity        *ipfslite.Identity
	hook            ConnectionHook
	networkNotifier network.Notifiee
}

func init() {
	logger.SetLogLevel("datahop", "Debug")
	logger.SetLogLevel("ipfslite", "Debug")
}

// Initialises the .datahop repo, if required at the given location with the given swarm port as config.
// Default swarm port is 4501
func Init(root string, h ConnectionHook) error {
	identity, err := ipfslite.Init(root, "0")
	if err != nil {
		return err
	}
	n := &Notifier{}
	hop = &datahop{
		root:            root,
		identity:        identity,
		hook:            h,
		networkNotifier: n,
	}
	return nil
}

// Starts an ipfs node in a go routine
func Start() error {
	if hop == nil {
		return errors.New("start failed. datahop not initialised")
	}

	r, err := ipfslite.Open(hop.root)
	if err != nil {
		log.Error("Repo Open Failed : ", err.Error())
		return err
	}
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		hop.ctx = ctx
		hop.cancel = cancel
		p, err := ipfslite.New(hop.ctx, r)
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

// Connects to a given peer address
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

// Connects to a given peerInfo string
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

// Returns string of the peer.AddrInfo []byte of the node
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

// Returns peerId of the node
func GetID() string {
	return hop.identity.PeerID
}

// Returns a comma(,) separated string of all the possible addresses of a node
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

// Checks if the node is running
func IsNodeOnline() bool {
	if hop.peer != nil {
		return hop.peer.IsOnline()
	}
	return false
}

// Returns a comma(,) separated string of all the connected peers of a node
func Peers() string {
	if hop != nil && hop.peer != nil {
		return strings.Join(hop.peer.Peers(), ",")
	}
	return "No Peers connected"
}

// App version
func Version() string {
	return version.Version
}

// Stops the node
func Stop() {
	hop.cancel()
}
