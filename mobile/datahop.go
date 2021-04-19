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
	lpeer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	log = logger.Logger("datahop")
	hop *datahop
)

type datahop struct {
	ctx    context.Context
	cancel context.CancelFunc
	root   string
	peer   *ipfslite.Peer
}

func init() {
	logger.SetLogLevel("*", "Debug")
}

// Initialises the .datahop repo, if required at the given location with the given swarm port as config.
// Default swarm port is 4501
func Init(root string) error {
	if err := ipfslite.Init(root, "0"); err != nil {
		return err
	}
	hop = &datahop{
		root: root,
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
		peer, err := ipfslite.New(hop.ctx, r)
		if err != nil {
			log.Error("Node setup failed : ", err.Error())
			return
		}
		hop.peer = peer
		select {
		case <-hop.ctx.Done():
			log.Debug("Context Closed")
		}
	}()
	log.Debug("Node Started")
	return nil
}

// Connects to a given peer address
func Connect(address string) error {
	addr, _ := ma.NewMultiaddr(address)
	peerInfo, _ := lpeer.AddrInfosFromP2pAddrs(addr)

	for _, v := range peerInfo {
		err := hop.peer.Connect(context.Background(), v)
		if err != nil {
			return err
		}
	}
	return nil
}

// Returns peerId of the node
func GetID() string {
	for i := 0; i < 5; i++ {
		if hop.peer != nil {
			return hop.peer.Host.ID().String()
		}
		<-time.After(time.Millisecond * 200)
	}
	return "Could not get peer ID"
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
