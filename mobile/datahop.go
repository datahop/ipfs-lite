package datahop

//go:generate gomobile bind -o datahop.aar -target=android github.com/datahop/ipfs-lite/mobile

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/datahop/ipfs-lite/internal/config"
	"github.com/datahop/ipfs-lite/internal/repo"
	types "github.com/datahop/ipfs-lite/pb"
	"github.com/datahop/ipfs-lite/version"
	"github.com/golang/protobuf/proto"
	"github.com/ipfs/go-datastore"
	logger "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	log                 = logger.Logger("datahop")
	hop                 *datahop
	ErrNoPeersConnected = errors.New("no Peers connected")
	ErrNoPeerAddress    = errors.New("could not get peer address")
)

const (
	NoPeersConnected = "No Peers connected"
	CRDTStatus       = "datahop-crdt-status"
)

// ConnectionManager is used by clients to get notified client connection
type ConnectionManager interface {
	PeerConnected(string)
	PeerDisconnected(string)
}

type Notifier struct{}

func (n *Notifier) Listen(network.Network, ma.Multiaddr)      {}
func (n *Notifier) ListenClose(network.Network, ma.Multiaddr) {}
func (n *Notifier) Connected(net network.Network, c network.Conn) {
	if hop.hook != nil {
		hop.hook.PeerConnected(c.RemotePeer().String())
	}
}
func (n *Notifier) Disconnected(net network.Network, c network.Conn) {
	if hop.hook != nil {
		hop.hook.PeerDisconnected(c.RemotePeer().String())
	}
}
func (n *Notifier) OpenedStream(net network.Network, s network.Stream) {}
func (n *Notifier) ClosedStream(network.Network, network.Stream)       {}

type discNotifee struct{}

func (n *discNotifee) HandlePeerFound(peerInfoByteString string) {
	var peerInfo peer.AddrInfo
	err := peerInfo.UnmarshalJSON([]byte(peerInfoByteString))
	if err != nil {
		return
	}
	hop.peer.HandlePeerFound(peerInfo)
}

type datahop struct {
	ctx             context.Context
	cancel          context.CancelFunc
	root            string
	peer            *ipfslite.Peer
	identity        config.Identity
	hook            ConnectionManager
	wifiHS          WifiHotspot
	wifiCon         WifiConnection
	discDriver      DiscoveryDriver
	advDriver       AdvertisingDriver
	networkNotifier network.Notifiee
	repo            repo.Repo
	notifier        Notifee
	discService     *discoveryService
}

func init() {
	logger.SetLogLevel("ipfslite", "Debug")
	logger.SetLogLevel("datahop", "Debug")
}

// Init Initialises the .datahop repo, if required at the given location with the given swarm port as config.
// Default swarm port is 4501
func Init(
	root string,
	connManager ConnectionManager,
	discDriver DiscoveryDriver,
	advDriver AdvertisingDriver,
	hs WifiHotspot,
	con WifiConnection,
) error {
	err := repo.Init(root, "0")
	if err != nil {
		return err
	}
	n := &Notifier{}
	r, err := repo.Open(root)
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
	dn := &discNotifee{}
	hop = &datahop{
		root:            root,
		identity:        cfg.Identity,
		hook:            connManager,
		networkNotifier: n,
		ctx:             ctx,
		cancel:          cancel,
		repo:            r,
		notifier:        dn,
		wifiHS:          hs,
		wifiCon:         con,
		discDriver:      discDriver,
		advDriver:       advDriver,
	}
	service, err := NewDiscoveryService(hop.discDriver, hop.advDriver, 1000, 20000, hop.wifiHS, hop.wifiCon, ipfslite.ServiceTag)
	if err != nil {
		log.Error("ble discovery setup failed : ", err.Error())
		return err
	}
	if res, ok := service.(*discoveryService); ok {
		hop.discService = res
		hop.discService.RegisterNotifee(hop.notifier)
	}
	return nil
}

// State returns number of keys in crdt store
func State() ([]byte, error) {
	return hop.repo.State().MarshalJSON()
}

// DiskUsage returns number of bytes stored in the datastore
func DiskUsage() (int64, error) {
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
	ctx, cancel := context.WithCancel(hop.ctx)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		p, err := ipfslite.New(ctx, cancel, hop.repo)
		if err != nil {
			log.Error("Node setup failed : ", err.Error())
			wg.Done()
			return
		}
		hop.peer = p
		hop.peer.Host.Network().Notify(hop.networkNotifier)

		wg.Done()
		select {
		case <-hop.peer.Ctx.Done():
			log.Debug("Context Closed ")
		}
	}()
	wg.Wait()
	log.Debug("Node Started")
	return nil
}

func StartDiscovery() error {
	if hop.discService != nil {
		hop.discService.Start()
		go func() {
			for {
				st, err := State()
				if err != nil {
					log.Error("Unable to fetch state")
					return
				}
				hop.discService.AddAdvertisingInfo(CRDTStatus, st)
				select {
				case <-hop.discService.stopSignal:
					log.Error("Stop AddAdvertisingInfo Routine")
					return
				case <-time.After(time.Second * 20):
				}
			}
		}()
		log.Debug("Stated discovery")
		return nil
	} else {
		return errors.New("discovery service is not initialised")
	}
}

func StopDiscovery() error {
	if hop.discService != nil {
		hop.discService.stopSignal <- struct{}{}
		return hop.discService.Close()
	} else {
		return errors.New("discovery service is not initialised")
	}
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

// PeerInfo Returns a string of the peer.AddrInfo []byte of the node
func PeerInfo() string {
	for i := 0; i < 5; i++ {
		if hop.peer != nil {
			addrs := hop.peer.Host.Addrs()
			interfaceAddrs, err := hop.peer.Host.Network().InterfaceListenAddresses()
			if err == nil {
				addrs = append(addrs, interfaceAddrs...)
			}
			pr := peer.AddrInfo{
				ID:    hop.peer.Host.ID(),
				Addrs: interfaceAddrs,
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
func Addrs() ([]byte, error) {
	for i := 0; i < 5; i++ {
		addrs := []string{}
		if hop.peer != nil {
			for _, v := range hop.peer.Host.Addrs() {
				if !strings.HasPrefix(v.String(), "127") {
					addrs = append(addrs, v.String()+"/p2p/"+hop.peer.Host.ID().String())
				}
			}
			addrs := &types.StringSlice{
				Output: addrs,
			}
			return proto.Marshal(addrs)
		}
		<-time.After(time.Millisecond * 200)
	}
	return nil, errors.New("could not get peer address")
}

// InterfaceAddrs returns a list of addresses at which this network
// listens. It expands "any interface" addresses (/ip4/0.0.0.0, /ip6/::) to
// use the known local interfaces.
func InterfaceAddrs() ([]byte, error) {
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
			addrs := &types.StringSlice{
				Output: addrs,
			}
			return proto.Marshal(addrs)
		}
		<-time.After(time.Millisecond * 200)
	}
	return nil, errors.New("could not get peer address")
}

// IsNodeOnline Checks if the node is running
func IsNodeOnline() bool {
	if hop != nil && hop.peer != nil {
		return hop.peer.IsOnline()
	}
	return false
}

// Peers Returns a comma(,) separated string of all the connected peers of a node
func Peers() ([]byte, error) {
	if hop != nil && hop.peer != nil && len(hop.peer.Peers()) > 0 {
		peers := &types.StringSlice{
			Output: hop.peer.Peers(),
		}
		return proto.Marshal(peers)
	}
	return nil, ErrNoPeersConnected
}

// Add adds a record in the store
func Add(tag string, content []byte) error {
	if hop != nil && hop.peer != nil {
		buf := bytes.NewReader(content)
		n, err := hop.peer.AddFile(context.Background(), buf, nil)
		if err != nil {
			return err
		}
		err = hop.peer.Manager.Tag(tag, n.Cid())
		if err != nil {
			return err
		}

		// Update advertise info
		st, err := State()
		if err != nil {
			return err
		}
		hop.discService.AddAdvertisingInfo(CRDTStatus, st)

		return nil
	}
	return errors.New("datahop ipfs-lite node is not running")
}

// Get gets a record from the store by given tag
func Get(tag string) ([]byte, error) {
	if hop != nil && hop.peer != nil {
		id, err := hop.peer.Manager.FindTag(tag)
		if err != nil {
			return nil, err
		}
		r, err := hop.peer.GetFile(context.Background(), id)
		if err != nil {
			return nil, err
		}
		content, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return content, nil
	}
	return nil, errors.New("datahop ipfs-lite node is not running")
}

// GetTags gets all the tags from the store
func GetTags() ([]byte, error) {
	if hop != nil && hop.peer != nil {
		tags, err := hop.peer.Manager.GetAllTags()
		if err != nil {
			return nil, err
		}
		allTags := &types.StringSlice{
			Output: tags,
		}
		return proto.Marshal(allTags)
	}
	return nil, errors.New("datahop ipfs-lite node is not running")
}

// Version of ipfs-lite
func Version() string {
	return version.Version
}

// Stop the node
func Stop() {
	hop.peer.Cancel()
	select {
	case <-hop.peer.Stopped:
	case <-time.After(time.Second * 5):
	}
}

// Close the repo and all
func Close() {
	hop.repo.Close()
	hop.cancel()
	hop.wifiCon.Disconnect()
	hop.wifiHS.Stop()
}

func UpdateTopicStatus(topic string, value []byte) {
	hop.discService.AddAdvertisingInfo(topic, value)
}

func GetDiscoveryNotifier() DiscoveryNotifier {
	return hop.discService
}

func GetAdvertisementNotifier() AdvertisementNotifier {
	return hop.discService
}

func GetWifiHotspotNotifier() WifiHotspotNotifier {
	return hop.discService
}

func GetWifiConnectionNotifier() WifiConnectionNotifier {
	return hop.discService
}
