package datahop

//go:generate gomobile bind -o datahop.aar -target=android github.com/datahop/ipfs-lite/mobile

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/datahop/ipfs-lite/internal/config"
	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/datahop/ipfs-lite/internal/security"
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
	log      = logger.Logger("datahop")
	stepsLog = logger.Logger("step")
	hop      *datahop

	// ErrNoPeersConnected is returned if there is no peer connected
	ErrNoPeersConnected = errors.New("no Peers connected")

	// ErrNoPeerAddress re returned if peer address is not available
	ErrNoPeerAddress = errors.New("could not get peer address")
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
		return
	}
	err = hop.peer.HandlePeerFoundWithError(peerInfo)
	if err != nil {
		log.Error("HandlePeerFoundWithError failed : ", err.Error())
		hop.peer.Repo.Matrix().NodeConnectionFailed(peerInfo.ID.String())
		return
	}
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
	logger.SetLogLevel("step", "Debug")
	logger.SetLogLevel("replication", "Debug")
	logger.SetLogLevel("matrix", "Debug")
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
	service, err := NewDiscoveryService(cfg.Identity.PeerID, hop.discDriver, hop.advDriver, 1000, 20000, hop.wifiHS, hop.wifiCon, ipfslite.ServiceTag)
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
func Start(shouldBootstrap bool) error {
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
		if shouldBootstrap {
			hop.peer.Bootstrap(ipfslite.DefaultBootstrapPeers())
		}
		wg.Done()
		select {
		case <-hop.peer.Ctx.Done():
			log.Debug("Context Closed ")
		}
	}()
	wg.Wait()
	log.Debug("Node Started")
	stepsLog.Debug("ipfs started")
	return nil
}

const (
	FloodSubID = protocol.ID("/hopfloodsub/1.0.0")
	TopicCRDT  = "CRDTStateLine"
)

// StartDiscovery starts BLE discovery
func StartDiscovery(advertising bool, scanning bool, autoDisconnect bool) error {
	if hop != nil {
		if hop.discService != nil {
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
			log.Debug("autoDisconnect : ", autoDisconnect)
			if autoDisconnect {
				err := startCRDTStateWatcher()
				if err != nil {
					log.Error("StartDiscovery: autoDisconnect: startCRDTStateWatcher failed ", err.Error())
					return err
				}
			}
			if advertising && scanning {
				hop.discService.Start()
				log.Debug("Started discovery")
				stepsLog.Debug("discoveryService started")
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
		return errors.New("discovery service is not initialised")
	}
	return errors.New("datahop service is not initialised")
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
	ctx, cancle := context.WithCancel(hop.ctx)
	hop.discService.stateInformer = &stateInformer{
		ctx:    ctx,
		cancel: cancle,
		Topic:  topic,
	}
	sub, err := hop.discService.stateInformer.Subscribe()
	if err != nil {
		return err
	}
	go func() {
		for {
			state, err := FilterFromState()
			if err != nil {
				return
			}
			msg := &Message{
				Id:        ID(),
				CRDTState: state,
			}
			msgBytes, err := msg.Marshal()
			if err != nil {
				return
			}
			select {
			case <-hop.discService.stateInformer.ctx.Done():
				log.Debug("Stop stateInformer Publisher")
				return
			case <-time.After(time.Second * 10):
				err := hop.discService.stateInformer.Publish(hop.ctx, msgBytes)
				if err != nil {
					log.Error("startCRDTStateWatcher : message publish error ", err.Error())
					continue
				}
			}
		}
	}()
	go func() {
		var mtx sync.Mutex
		for {
			got, err := sub.Next(hop.ctx)
			if err != nil {
				return
			}
			msg := Message{}
			err = msg.Unmarshal(got.GetData())
			if err != nil {
				return
			}
			if msg.Id != ID() {
				state, err := FilterFromState()
				if err != nil {
					continue
				}
				if msg.CRDTState == state {
					mtx.Lock()
					log.Debug("state watcher : same state")
					// Publish state again
					newMsg := &Message{
						Id:        ID(),
						CRDTState: state,
					}
					msgBytes, err := newMsg.Marshal()
					err = hop.discService.stateInformer.Publish(hop.ctx, msgBytes)
					if err != nil {
						log.Error("startCRDTStateWatcher : message publish error ", err.Error())
					}
					if !hop.discService.isHost && hop.discService.connected {
						log.Debug("state watcher : same state : not host")
						hop.wifiCon.Disconnect()
						stepsLog.Debug("wifi conn stopped")
						hop.discService.connected = false
					}
					log.Debugf("state watcher : same state : host : %v : peers : %d\n", hop.discService.isHost, len(hop.peer.Peers()))
					if hop.discService.isHost && len(hop.peer.Peers()) < 2 {
						log.Debug("state watcher : same state : host")
						hop.discService.isHost = false
						hop.discService.wifiHS.Stop()
						stepsLog.Debug("wifi hotspot stopped")
					}
					mtx.Unlock()
				}
			}
		}
	}()
	return nil
}

// StopDiscovery stops BLE discovery
func StopDiscovery() error {
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
		log.Debugf("trying peerInfo %d", i)
		if hop.peer != nil {
			pr := peer.AddrInfo{
				ID:    hop.peer.Host.ID(),
				Addrs: hop.peer.Host.Addrs(),
			}
			log.Debugf("Peer %s : %v", hop.peer.Host.ID(), pr)
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
	if hop != nil && hop.peer != nil && len(hop.peer.Peers()) > 0 {
		peers := &types.StringSlice{
			Output: hop.peer.Peers(),
		}
		return proto.Marshal(peers)
	}
	return nil, ErrNoPeersConnected
}

// Add adds a record in the store
func Add(tag string, content []byte, passphrase string) error {
	if hop != nil && hop.peer != nil {
		buf := bytes.NewReader(nil)
		shouldEncrypt := true
		if passphrase == "" {
			shouldEncrypt = false
		}
		if shouldEncrypt {
			byteContent, err := security.Encrypt(content, passphrase)
			if err != nil {
				log.Errorf("Encryption failed :%s", err.Error())
				return err
			}
			buf = bytes.NewReader(byteContent)
		} else {
			buf = bytes.NewReader(content)
		}
		n, err := hop.peer.AddFile(context.Background(), buf, nil)
		if err != nil {
			return err
		}
		kind, _ := filetype.Match(content)
		meta := &replication.Metatag{
			Size:        int64(len(content)),
			Type:        kind.MIME.Value,
			Name:        tag,
			Hash:        n.Cid(),
			Timestamp:   time.Now().Unix(),
			Owner:       hop.peer.Host.ID(),
			Tag:         tag,
			IsEncrypted: shouldEncrypt,
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
func Get(tag string, passphrase string) ([]byte, error) {
	if hop != nil && hop.peer != nil {
		meta, err := hop.peer.Manager.FindTag(tag)
		if err != nil {
			return nil, err
		}
		log.Debugf("%s => %+v", tag, meta)
		if meta.IsEncrypted {
			if passphrase == "" {
				log.Error("passphrase is empty")
				return nil, errors.New("passphrase is empty")
			}
		}
		r, err := hop.peer.GetFile(context.Background(), meta.Hash)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		if meta.IsEncrypted {
			buf := bytes.NewBuffer(nil)
			_, err = io.Copy(buf, r)
			if err != nil {
				log.Errorf("Failed io.Copy file encryption:%s", err.Error())
				return nil, err
			}
			byteContent, err := security.Decrypt(buf.Bytes(), passphrase)
			if err != nil {
				log.Errorf("decryption failed :%s", err.Error())
				return nil, err
			}
			return byteContent, nil
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
	return version.MobileVersion
}

// Stop the node
func Stop() {
	hop.peer.Repo.Matrix().Flush()
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
}

// UpdateTopicStatus adds BLE advertising info
func UpdateTopicStatus(topic string, value string) {
	hop.discService.AddAdvertisingInfo(topic, value)
}

// Matrix returns matrix measurements
func Matrix() (string, error) {
	if hop != nil && hop.peer != nil {
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
	return len(hop.peer.Manager.DownloadManagerStatus())
}

// GetDiscoveryNotifier returns discovery notifier
func GetDiscoveryNotifier() DiscoveryNotifier {
	return hop.discService
}

// GetAdvertisementNotifier returns advertisement notifier
func GetAdvertisementNotifier() AdvertisementNotifier {
	return hop.discService
}

// GetWifiHotspotNotifier returns wifi hotspot notifier
func GetWifiHotspotNotifier() WifiHotspotNotifier {
	return hop.discService
}

// GetWifiConnectionNotifier returns wifi connection notifier
func GetWifiConnectionNotifier() WifiConnectionNotifier {
	return hop.discService
}

func StartMeasurements(length, delay int) {
	go func() {
		stepsLog.Debug("starting measurement loop")
		contentLength := length
		for {
			content := make([]byte, contentLength)
			_, err := rand.Read(content)
			if err != nil {
				stepsLog.Error("content creation failed :", err)
				continue
			}
			err = Add(time.Now().String(), content, "")
			if err != nil {
				stepsLog.Error("Measurement content addition failed : ", err.Error())
				continue
			}
			stepsLog.Debug("content added in measurement loop")
			select {
			case <-hop.ctx.Done():
				return
			case <-time.After(time.Second * time.Duration(delay)):
				StopDiscovery()
				Stop()
				<-time.After(time.Second * 5)
				Start(false)
				StartDiscovery(true, true, true)
			}
		}
	}()
}

func AddAGB() {
	go func() {
		stepsLog.Debug("AddAGB: starting adding a gb content")
		contentLength := 1000000000
		content := make([]byte, contentLength)
		_, err := rand.Read(content)
		if err != nil {
			stepsLog.Error("AddAGB: content creation failed :", err)
			return
		}
		err = Add(time.Now().String(), content, "")
		if err != nil {
			stepsLog.Error("AddAGB: content addition failed : ", err.Error())
			return
		}
		stepsLog.Debug("AddAGB: added a gb content")
	}()
}
