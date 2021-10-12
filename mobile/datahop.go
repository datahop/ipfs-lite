package datahop

//go:generate gomobile bind -o datahop.aar -target=android github.com/datahop/ipfs-lite/mobile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/datahop/ipfs-lite/internal/config"
	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/datahop/ipfs-lite/internal/repo"
	types "github.com/datahop/ipfs-lite/pb"
	"github.com/datahop/ipfs-lite/version"
	"github.com/golang/protobuf/proto"
	"github.com/h2non/filetype"
	"github.com/ipfs/go-datastore"
	logger "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	log = logger.Logger("datahop")
	hop *datahop

	// ErrNoPeersConnected is returned if there is no peer connected
	ErrNoPeersConnected = errors.New("no Peers connected")

	// ErrNoPeerAddress re returned if peer address is not available
	ErrNoPeerAddress = errors.New("could not get peer address")

	ErrClientNotInit = errors.New("datahop client is not initialised, call \"Init\" first")
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

type notifier struct{}

func (n *notifier) Listen(network.Network, ma.Multiaddr)      {}
func (n *notifier) ListenClose(network.Network, ma.Multiaddr) {}
func (n *notifier) Connected(net network.Network, c network.Conn) {
	// NodeMatrix management
	hop.peer.Repo.Matrix().NodeConnected(c.RemotePeer().String())
	hop.peer.Manager.StartUnfinishedDownload(c.RemotePeer())

	if hop.hook != nil {
		hop.hook.PeerConnected(c.RemotePeer().String())
	}
}
func (n *notifier) Disconnected(net network.Network, c network.Conn) {
	// NodeMatrix management
	hop.peer.Repo.Matrix().NodeDisconnected(c.RemotePeer().String())
	if hop.hook != nil {
		hop.hook.PeerDisconnected(c.RemotePeer().String())
	}
}
func (n *notifier) OpenedStream(net network.Network, s network.Stream) {
	//log.Debug("Opened stream")
}
func (n *notifier) ClosedStream(network.Network, network.Stream) {
	//log.Debug("Closed stream")
}

type discNotifee struct{}

func (n *discNotifee) HandlePeerFound(peerInfoByteString string) {
	var peerInfo peer.AddrInfo
	err := peerInfo.UnmarshalJSON([]byte(peerInfoByteString))
	if err != nil {
		log.Error("peerInfo.UnmarshalJSON failed ", err.Error())
		return
	}
	err = hop.peer.HandlePeerFoundWithError(peerInfo)
	if err == nil {
		return
	}
	log.Error("HandlePeerFoundWithError failed ", err.Error())
	hop.peer.Repo.Matrix().NodeConnectionFailed(peerInfo.ID.String())
	return
}

type connectednessChecker struct{}

func (c *connectednessChecker) Connectedness(id peer.ID) network.Connectedness {
	if hop != nil || hop.peer != nil {
		return network.NotConnected
	}
	return hop.peer.Host.Network().Connectedness(id)
}

type identityInformer struct{}

func (i *identityInformer) ID() string {
	if hop != nil {
		return hop.identity.PeerID
	}
	return ""
}

func (i *identityInformer) PeerInfo() string {
	if hop != nil {
		if hop.peer != nil && hop.peer.IsOnline() {
			pi, err := interfacePeerInfo()
			if err != nil {
				return err.Error()
			}
			return pi
		}
		pr := peer.AddrInfo{
			ID: peer.ID(i.ID()),
		}
		prb, _ := pr.MarshalJSON()
		return string(prb)
	}
	return ""
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

	stateInformer *stateInformer
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
	n := &notifier{}
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
	service := NewDiscoveryService(
		hop.discDriver,
		hop.advDriver,
		1000,
		20000,
		hop.wifiHS,
		hop.wifiCon,
		DiscoveryServiceTag,
		&connectednessChecker{},
		hop.repo.Matrix(),
		&identityInformer{},
	)
	res, ok := service.(*discoveryService)
	if !ok {
		return errors.New("discovery service is not initialised")
	}
	hop.discService = res
	hop.discService.RegisterNotifee(hop.notifier)
	return nil
}

// State returns number of keys in crdt store
func State() ([]byte, error) {
	return hop.repo.State().MarshalJSON()
}

// FilterFromState returns the bloom filter from state
func FilterFromState() (string, error) {
	byt, err := State()
	var dat map[string]interface{}
	if err := json.Unmarshal(byt, &dat); err != nil {
		return "", err
	}
	filter := dat["b"].(string)
	return filter, err
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
func Start(shouldBootstrap, autoDisconnectFromHost bool) error {
	if hop == nil {
		return ErrClientNotInit
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
		if shouldBootstrap {
			hop.peer.Bootstrap(ipfslite.DefaultBootstrapPeers())
		}
		log.Debug("autoDisconnect : ", autoDisconnectFromHost)
		if autoDisconnectFromHost {
			err := startCRDTStateWatcher()
			if err != nil {
				log.Error("Start: autoDisconnectFromHost: startCRDTStateWatcher failed ", err.Error())
				wg.Done()
				return
			}
		}
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

const (
	FloodSubID = protocol.ID("/hopfloodsub/1.0.0")
	TopicCRDT  = "CRDTStateLine"
)

// StartDiscovery starts BLE discovery
func StartDiscovery(advertising bool, scanning bool) error {
	if hop != nil {
		if hop.discService == nil {
			return errors.New("discovery service is not initialised")
		}
		go func() {
			for {
				bf, err := FilterFromState()
				if err != nil {
					log.Error("Unable to filter state")
					return
				}
				log.Debug("Filter state ", bf)
				hop.discService.AddAdvertisingInfo(CRDTStatus, bf)
				select {
				case <-hop.discService.stopSignal:
					log.Error("Stop AddAdvertisingInfo Routine")
					return
				case <-time.After(time.Second * 10):
				}
			}
		}()
		if advertising && scanning {
			hop.discService.Start()
			log.Debug("Started discovery")
			return nil
		} else if advertising {
			hop.discService.StartOnlyAdvertising()
			log.Debug("Started discovery only advertising")
			return nil
		} else if scanning {
			hop.discService.StartOnlyScanning()
			log.Debug("Started discovery only scanning")
			return nil
		}
		return errors.New("no advertising and no scanning enabled")
	}
	return ErrClientNotInit
}

func startCRDTStateWatcher() error {
	log.Debug("startCRDTStateWatcher")
	var pubsubOptions []pubsub.Option
	ps, err := pubsub.NewFloodsubWithProtocols(context.Background(), hop.peer.Host,
		[]protocol.ID{FloodSubID}, pubsubOptions...)
	if err != nil {
		return err
	}
	topic, err := ps.Join(TopicCRDT)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(hop.peer.Ctx)
	hop.stateInformer = &stateInformer{
		ctx:    ctx,
		cancel: cancel,
		Topic:  topic,
	}
	sub, err := hop.stateInformer.Subscribe()
	if err != nil {
		return err
	}
	go func() {
		for {
			state, err := FilterFromState()
			if err != nil {
				return
			}
			id, err := ID()
			if err != nil {
				return
			}
			msg := &Message{
				Id:        id,
				CRDTState: state,
			}
			msgBytes, err := msg.Marshal()
			if err != nil {
				return
			}
			select {
			case <-hop.stateInformer.ctx.Done():
				log.Debug("Stop stateInformer Publisher")
				return
			case <-time.After(time.Second * 10):
				err := hop.stateInformer.Publish(hop.stateInformer.ctx, msgBytes)
				if err != nil {
					log.Error("startCRDTStateWatcher : message publish error ", err.Error())
					continue
				}
				log.Debugf("startCRDTStateWatcher : sent message %+v\n", msg)
			}
		}
	}()
	go func() {
		for {
			got, err := sub.Next(hop.stateInformer.ctx)
			if err != nil {
				return
			}
			msg := Message{}
			err = msg.Unmarshal(got.GetData())
			if err != nil {
				return
			}
			log.Debugf("startCRDTStateWatcher : got message %+v\n", msg)
			id, err := ID()
			if err != nil {
				return
			}
			if msg.Id != id {
				state, err := FilterFromState()
				if err != nil {
					continue
				}
				if msg.CRDTState == state && hop.discService != nil && !hop.discService.isHost {
					hop.wifiCon.Disconnect()
					hop.discService.connected = false
				}
			}
		}
	}()
	return nil
}

// StopDiscovery stops BLE discovery
func StopDiscovery() error {
	if hop == nil {
		return ErrClientNotInit
	}
	log.Debug("Stopping discovery")
	if hop.discService != nil {
		go func() {
			hop.discService.stopSignal <- struct{}{}
		}()
		return hop.discService.Close()
	}
	return errors.New("discovery service is not initialised")
}

// ConnectWithAddress Connects to a given peer address
func ConnectWithAddress(address string) error {
	if hop == nil {
		return ErrClientNotInit
	}
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
	if hop == nil {
		return ErrClientNotInit
	}
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
	if hop == nil {
		return ErrClientNotInit
	}
	var peerInfo peer.AddrInfo
	err := peerInfo.UnmarshalJSON([]byte(peerInfoByteString))
	if err != nil {
		return err
	}
	hop.peer.Bootstrap([]peer.AddrInfo{peerInfo})
	return nil
}

func interfacePeerInfo() (string, error) {
	if hop == nil {
		return "", ErrClientNotInit
	}
	id, err := peer.Decode(hop.identity.PeerID)
	if err != nil {
		return "", err
	}
	pr := peer.AddrInfo{
		ID: id,
	}
	if hop.peer != nil && hop.peer.IsOnline() {
		if hop.peer != nil {
			interfaceAddrs, err := hop.peer.Host.Network().InterfaceListenAddresses()
			if err != nil {
				return "", err
			}
			pr.Addrs = interfaceAddrs
			log.Debugf("Peer %s : %v", hop.peer.Host.ID(), pr)
		}
	}
	prb, _ := pr.MarshalJSON()
	return string(prb), nil
}

// PeerInfo Returns a string of the peer.AddrInfo []byte of the node
func PeerInfo() (string, error) {
	if hop == nil {
		return "", ErrClientNotInit
	}
	id, err := peer.Decode(hop.identity.PeerID)
	if err != nil {
		return "", err
	}
	pr := peer.AddrInfo{
		ID: id,
	}
	if hop.peer != nil && hop.peer.IsOnline() {
		if hop.peer != nil {
			addrs := hop.peer.Host.Addrs()
			interfaceAddrs, err := hop.peer.Host.Network().InterfaceListenAddresses()
			if err == nil {
				addrs = append(addrs, interfaceAddrs...)
			}
			pr.Addrs = unique(addrs)
			log.Debugf("Peer %s : %v", hop.peer.Host.ID(), pr)
		}
	}
	prb, _ := pr.MarshalJSON()
	return string(prb), nil
}

var hostAddress = "/ip4/192.168.49.1"
var nonHostAddress = "/ip4/192.168.49"

func unique(e []ma.Multiaddr) []ma.Multiaddr {
	r := []ma.Multiaddr{}
	hostIndex, nonHostIndex := -1, -1
	for _, s := range e {
		if !contains(r[:], s) {
			if strings.HasPrefix(s.String(), hostAddress) {
				hostIndex = len(r)
			} else if strings.HasPrefix(s.String(), nonHostAddress) {
				nonHostIndex = len(r)
			}
			r = append(r, s)
		}
	}
	if hostIndex != -1 && nonHostIndex != -1 {
		r = append(r[:hostIndex], r[hostIndex+1:]...)
	}
	return r
}

func contains(e []ma.Multiaddr, c ma.Multiaddr) bool {
	for _, s := range e {
		if s.Equal(c) {
			return true
		}
	}
	return false
}

// ID Returns peerId of the node
func ID() (string, error) {
	if hop == nil {
		return "", ErrClientNotInit
	}
	return hop.identity.PeerID, nil
}

// Addrs Returns a comma(,) separated string of all the possible addresses of a node
func Addrs() ([]byte, error) {
	if hop == nil {
		return nil, ErrClientNotInit
	}
	for i := 0; i < 5; i++ {
		addrs := []string{}
		if hop.peer != nil && hop.peer.IsOnline() {
			for _, v := range hop.peer.Host.Addrs() {
				if !strings.HasPrefix(v.String(), "127") {
					addrs = append(addrs, v.String()+"/p2p/"+hop.peer.Host.ID().String())
				}
			}
			addrsOP := &types.StringSlice{
				Output: addrs,
			}
			return proto.Marshal(addrsOP)
		}
		<-time.After(time.Millisecond * 200)
	}
	return nil, errors.New("could not get peer address")
}

// InterfaceAddrs returns a list of addresses at which this network
// listens. It expands "any interface" addresses (/ip4/0.0.0.0, /ip6/::) to
// use the known local interfaces.
func InterfaceAddrs() ([]byte, error) {
	if hop == nil {
		return nil, ErrClientNotInit
	}
	for i := 0; i < 5; i++ {
		addrs := []string{}
		if hop.peer != nil && hop.peer.IsOnline() {
			interfaceAddrs, err := hop.peer.Host.Network().InterfaceListenAddresses()
			if err == nil {
				for _, v := range interfaceAddrs {
					if !strings.HasPrefix(v.String(), "127") {
						addrs = append(addrs, v.String()+"/p2p/"+hop.peer.Host.ID().String())
					}
				}
			}
			addrsOP := &types.StringSlice{
				Output: addrs,
			}
			return proto.Marshal(addrsOP)
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
	if hop == nil {
		return nil, ErrClientNotInit
	}
	if hop.peer != nil && len(hop.peer.Peers()) > 0 {
		peers := &types.StringSlice{
			Output: hop.peer.Peers(),
		}
		return proto.Marshal(peers)
	}
	return nil, ErrNoPeersConnected
}

// Add adds a record in the store
func Add(tag string, content []byte) error {
	if hop == nil {
		return ErrClientNotInit
	}
	if hop.peer != nil && hop.peer.IsOnline() {
		buf := bytes.NewReader(content)
		n, err := hop.peer.AddFile(context.Background(), buf, nil)
		if err != nil {
			return err
		}
		kind, _ := filetype.Match(content)
		meta := &replication.Metatag{
			Size:      int64(len(content)),
			Type:      kind.MIME.Value,
			Name:      tag,
			Hash:      n.Cid(),
			Timestamp: time.Now().Unix(),
			Owner:     hop.peer.Host.ID(),
			Tag:       tag,
		}
		err = hop.peer.Manager.Tag(tag, meta)
		if err != nil {
			return err
		}

		// Update advertise info
		bf, err := FilterFromState()
		if err != nil {
			log.Error("Unable to filter state")
			return err
		}
		hop.discService.AddAdvertisingInfo(CRDTStatus, bf)
		return nil
	}
	return errors.New("datahop ipfs-lite node is not running")
}

// Get gets a record from the store by given tag
func Get(tag string) ([]byte, error) {
	if hop == nil {
		return nil, ErrClientNotInit
	}
	if hop.peer != nil && hop.peer.IsOnline() {
		meta, err := hop.peer.Manager.FindTag(tag)
		if err != nil {
			return nil, err
		}
		log.Debugf("%s => %+v", tag, meta)
		r, err := hop.peer.GetFile(context.Background(), meta.Hash)
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
	if hop == nil {
		return nil, ErrClientNotInit
	}
	if hop.peer != nil && hop.peer.IsOnline() {
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
	return version.MobileVersion
}

// Stop the node
func Stop() {
	if hop == nil {
		return
	}
	hop.peer.Repo.Matrix().Flush()
	hop.peer.Cancel()

	select {
	case <-hop.peer.Stopped:
	case <-time.After(time.Second * 5):
	}
}

// Close the repo and all
func Close() {
	if hop == nil {
		return
	}
	hop.repo.Close()
	hop.cancel()
	hop = nil
}

// UpdateTopicStatus adds BLE advertising info
func UpdateTopicStatus(topic string, value string) {
	hop.discService.AddAdvertisingInfo(topic, value)
}

// Matrix returns matrix measurements
func Matrix() (string, error) {
	if hop == nil {
		return "", ErrClientNotInit
	}
	if hop.peer != nil && hop.peer.IsOnline() {
		nodeMatrixSnapshot := hop.peer.Repo.Matrix().NodeMatrixSnapshot()
		contentMatrixSnapshot := hop.peer.Repo.Matrix().ContentMatrixSnapshot()
		uptime := hop.peer.Repo.Matrix().GetTotalUptime()
		matrix := map[string]interface{}{}
		matrix["TotalUptime"] = uptime
		matrix["NodeMatrix"] = nodeMatrixSnapshot
		matrix["ContentMatrix"] = contentMatrixSnapshot
		b, err := json.Marshal(matrix)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", errors.New("datahop ipfs-lite node is not running")
}

func DownloadsInProgress() int {
	if hop == nil {
		return 0
	}
	return len(hop.peer.Manager.DownloadManagerStatus())
}

// GetDiscoveryNotifier returns discovery notifier
func GetDiscoveryNotifier() DiscoveryNotifier {
	if hop == nil {
		return nil
	}
	return hop.discService
}

// GetAdvertisementNotifier returns advertisement notifier
func GetAdvertisementNotifier() AdvertisementNotifier {
	if hop == nil {
		return nil
	}
	return hop.discService
}

// GetWifiHotspotNotifier returns wifi hotspot notifier
func GetWifiHotspotNotifier() WifiHotspotNotifier {
	if hop == nil {
		return nil
	}
	return hop.discService
}

// GetWifiConnectionNotifier returns wifi connection notifier
func GetWifiConnectionNotifier() WifiConnectionNotifier {
	if hop == nil {
		return nil
	}
	return hop.discService
}
