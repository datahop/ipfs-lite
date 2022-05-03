package datahop

//go:generate gomobile bind -o datahop.aar -target=android github.com/datahop/ipfs-lite/mobile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/datahop/ipfs-lite/internal/ipfs"
	types "github.com/datahop/ipfs-lite/pb"
	"github.com/datahop/ipfs-lite/pkg"
	"github.com/datahop/ipfs-lite/pkg/store"
	"github.com/datahop/ipfs-lite/version"
	"github.com/h2non/filetype"
	"github.com/ipfs/go-datastore"
	logger "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/protobuf/proto"
)

var (
	log      = logger.Logger("datahop")
	stepsLog = logger.Logger("step")
	hop      *datahop

	mtx sync.Mutex

	ErrNodeNotRunning       = fmt.Errorf("datahop is not initialised yet")
	ErrBlankGroupKey        = fmt.Errorf("private network secret cannot be blank")
	ErrAdvScan              = fmt.Errorf("no advertising and no scanning enabled")
	ErrDisc                 = fmt.Errorf("discovery service is not initialised")
	ErrEncryptionPassphrase = fmt.Errorf("encryption passphrase cannot be blank")
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
	hop.comm.Repo.Matrix().NodeConnected(c.RemotePeer().String())
	hop.comm.Node.ReplManager().StartUnfinishedDownload(c.RemotePeer())

	if hop.hook != nil {
		hop.hook.PeerConnected(c.RemotePeer().String())
	}
}
func (n *notifier) Disconnected(net network.Network, c network.Conn) {
	// NodeMatrix management
	r := hop.comm.Repo
	r.Matrix().NodeDisconnected(c.RemotePeer().String())
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
	log.Warnf("discNotifee HandlePeerFound with %s", peerInfoByteString)
}

type datahop struct {
	ctx             context.Context
	cancel          context.CancelFunc
	root            string
	hook            ConnectionManager
	wifiHS          WifiHotspot
	wifiCon         WifiConnection
	discDriver      DiscoveryDriver
	advDriver       AdvertisingDriver
	networkNotifier network.Notifiee
	notifier        Notifee
	discService     *discoveryService

	comm *pkg.Common
}

func init() {
	_ = logger.SetLogLevel("ipfslite", "Debug")
	_ = logger.SetLogLevel("datahop", "Debug")
	_ = logger.SetLogLevel("step", "Debug")
	_ = logger.SetLogLevel("replication", "Debug")
	_ = logger.SetLogLevel("matrix", "Debug")
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
	err := pkg.Init(root, "0")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	comm, err := pkg.New(ctx, root, "0", nil)
	if err != nil {
		cancel()
		return err
	}
	cfg, err := comm.Repo.Config()
	if err != nil {
		log.Error("Config Failed : ", err.Error())
		cancel()
		return err
	}
	n := &notifier{}
	dn := &discNotifee{}
	mtx.Lock()
	defer mtx.Unlock()
	hop = &datahop{
		root:            root,
		hook:            connManager,
		networkNotifier: n,
		ctx:             ctx,
		cancel:          cancel,
		notifier:        dn,
		wifiHS:          hs,
		wifiCon:         con,
		discDriver:      discDriver,
		advDriver:       advDriver,

		comm: comm,
	}

	service, err := NewDiscoveryService(cfg.Identity.PeerID, hop.discDriver, hop.advDriver, 1000, 20000, hop.wifiHS, hop.wifiCon, ipfs.ServiceTag)
	if err != nil {
		log.Error("ble discovery setup failed : ", err.Error())
		cancel()
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
	mtx.Lock()
	defer mtx.Unlock()
	return hop.comm.Repo.State().MarshalJSON()
}

func state() ([]byte, error) {
	return hop.comm.Repo.State().MarshalJSON()
}

// FilterFromState returns the bloom filter from state
func FilterFromState() (string, error) {
	byt, _ := state()
	var dat map[string]interface{}
	if err := json.Unmarshal(byt, &dat); err != nil {
		return "", err
	}
	filter := dat["b"].(string)
	return filter, nil
}

// DiskUsage returns number of bytes stored in the datastore
func DiskUsage() (int64, error) {
	mtx.Lock()
	defer mtx.Unlock()
	du, err := datastore.DiskUsage(hop.comm.Repo.Datastore())
	if err != nil {
		return 0, err
	}
	return int64(du), nil
}

type StartsOpts struct {
	ShouldBootstrap bool
}

// Start an ipfslite node in a go routine
func Start(bootstrap, autoDownload bool) error {
	if hop == nil {
		return ErrNodeNotRunning
	}
	mtx.Lock()
	defer mtx.Unlock()
	wg := sync.WaitGroup{}
	wg.Add(1)
	var startErr error
	go func() {
		done, err := hop.comm.Start("", autoDownload)
		if err != nil {
			log.Error("Node setup failed : ", err.Error())
			startErr = err
			wg.Done()
			return
		}
		hop.comm.Node.NetworkNotifiee(hop.networkNotifier)
		if bootstrap {
			hop.comm.Node.Bootstrap(ipfs.DefaultBootstrapPeers())
		}
		wg.Done()
		<-done
		log.Debug("Context Closed")
	}()
	wg.Wait()
	if startErr != nil {
		return startErr
	}
	log.Debug("Node Started")
	stepsLog.Debug("ipfs started")
	return nil
}

// StartPrivate starts an ipfslite node in a private network with provided swarmkey
func StartPrivate(swarmKey string, autoDownload bool) error {
	if swarmKey == "" {
		return ErrBlankGroupKey
	}

	if hop == nil {
		return ErrNodeNotRunning
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	var startErr error
	go func() {
		done, err := hop.comm.Start(swarmKey, autoDownload)
		if err != nil {
			log.Error("Node setup failed : ", err.Error())
			startErr = err
			wg.Done()
			return
		}
		hop.comm.Node.NetworkNotifiee(hop.networkNotifier)
		wg.Done()
		<-done
		log.Debug("Context Closed ")
	}()
	wg.Wait()
	if startErr != nil {
		return startErr
	}
	log.Debug("Node Started")
	stepsLog.Debug("ipfs started")
	return nil
}

const (
	FloodSubID = protocol.ID("/hopfloodsub/1.0.0")
	TopicCRDT  = "CRDTStateLine"
)

type DiscoveryOpts struct {
	Advertise      bool
	Scan           bool
	AutoDisconnect bool
}

// StartDiscovery starts BLE discovery
func StartDiscovery(advertise, scan, autoDisconnect bool) error {
	mtx.Lock()
	defer mtx.Unlock()

	if hop != nil {
		if hop.discService != nil {
			discService := hop.discService
			go func() {
				for {
					bf, err := FilterFromState()
					if err != nil {
						log.Error("Unable to filter state")
						return
					}
					log.Debug("Filter state ", bf)
					discService.AddAdvertisingInfo(CRDTStatus, bf)
					select {
					case <-discService.stopSignal:
						log.Error("Stop AddAdvertisingInfo Routine")
						return
					case <-time.After(time.Second * 10):
					}
				}
			}()
			if autoDisconnect {
				err := startCRDTStateWatcher()
				if err != nil {
					log.Error("StartDiscovery: autoDisconnect: startCRDTStateWatcher failed ", err.Error())
					return err
				}
			}
			if advertise && scan {
				discService.Start()
				log.Debug("Started discovery")
				stepsLog.Debug("discoveryService started")
				return nil
			} else if advertise {
				discService.StartOnlyAdvertising()
				log.Debug("Started discovery only advertising")
				return nil
			} else if scan {
				discService.StartOnlyScanning()
				log.Debug("Started discovery only scanning")
				return nil
			}
			return ErrAdvScan
		}
		return ErrDisc
	}
	return ErrNodeNotRunning
}

func startCRDTStateWatcher() error {
	log.Debug("startCRDTStateWatcher")
	var pubsubOptions []pubsub.Option
	ps, err := hop.comm.Node.NewFloodsubWithProtocols([]protocol.ID{FloodSubID}, pubsubOptions...)
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
			id, err := ID()
			if err != nil {
				return
			}
			if msg.Id != id {
				state, err := FilterFromState()
				if err != nil {
					continue
				}
				if msg.CRDTState == state {
					mtx.Lock()
					log.Debug("state watcher : same state")
					// Publish state again
					newMsg := &Message{
						Id:        id,
						CRDTState: state,
					}
					msgBytes, err := newMsg.Marshal()
					if err != nil {
						log.Debugf("startCRDTStateWatcher : marshaling new message failed : %s", err)
						log.Error("startCRDTStateWatcher : marshaling new message failed")
						continue
					}
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
					if hop.discService.isHost && len(hop.comm.Node.Peers()) < 2 {
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
	mtx.Lock()
	defer mtx.Unlock()
	log.Debug("Stopping discovery")
	if hop.discService != nil {
		go func() {
			hop.discService.stopSignal <- struct{}{}
		}()
		return hop.discService.close()
	}
	return ErrDisc
}

// ConnectWithAddress Connects to a given peer address
func ConnectWithAddress(address string) error {
	addr, _ := ma.NewMultiaddr(address)
	peerInfo, _ := peer.AddrInfosFromP2pAddrs(addr)
	mtx.Lock()
	defer mtx.Unlock()
	for _, v := range peerInfo {
		err := hop.comm.Node.Connect(v)
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
	mtx.Lock()
	defer mtx.Unlock()
	err = hop.comm.Node.Connect(peerInfo)
	if err != nil {
		return err
	}
	return nil
}

// BootstrapWithPeerInfo bootstraps to a given peerInfo string of a node
func BootstrapWithPeerInfo(peerInfoByteString string) error {
	var peerInfo peer.AddrInfo
	err := peerInfo.UnmarshalJSON([]byte(peerInfoByteString))
	if err != nil {
		return err
	}
	mtx.Lock()
	defer mtx.Unlock()
	hop.comm.Node.Bootstrap([]peer.AddrInfo{peerInfo})
	return nil
}

// BootstrapWithAddress bootstraps to a given address of a node
func BootstrapWithAddress(bootstrapAddress string) error {
	maddrs := make([]ma.Multiaddr, 1)
	var err error
	maddrs[0], err = ma.NewMultiaddr(bootstrapAddress)
	if err != nil {
		return err
	}
	infos, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	if err != nil {
		return err
	}
	hop.comm.Node.Bootstrap(infos)
	return nil
}

// PeerInfo Returns a string of the peer.AddrInfo []byte of the node
func PeerInfo() string {
	mtx.Lock()
	defer mtx.Unlock()
	for i := 0; i < 5; i++ {
		log.Debugf("trying peerInfo %d", i)
		if hop.comm != nil {
			pr := hop.comm.Node.AddrInfo()
			log.Debugf("Peer %v", pr)
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
func ID() (string, error) {
	mtx.Lock()
	defer mtx.Unlock()
	cfg, err := hop.comm.Repo.Config()
	if err != nil {
		return "", err
	}
	return cfg.Identity.PeerID, nil
}

// Addrs Returns a comma(,) separated string of all the possible addresses of a node
func Addrs() ([]byte, error) {
	mtx.Lock()
	defer mtx.Unlock()
	for i := 0; i < 5; i++ {
		addrs := []string{}
		if hop.comm.Node != nil {
			pr := hop.comm.Node.AddrInfo()
			for _, v := range pr.Addrs {
				if !strings.HasPrefix(v.String(), "/ip4/127") {
					addrs = append(addrs, v.String()+"/p2p/"+pr.ID.String())
				}
			}
			addrsOP := &types.StringSlice{
				Output: addrs,
			}
			return proto.Marshal(addrsOP)
		}
		<-time.After(time.Millisecond * 200)
	}
	return nil, pkg.ErrNoPeerAddress
}

// IsNodeOnline Checks if the node is running
func IsNodeOnline() bool {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		return hop.comm.Node.IsOnline()
	}
	return false
}

// Peers Returns a comma(,) separated string of all the connected peers of a node
func Peers() ([]byte, error) {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil && len(hop.comm.Node.Peers()) > 0 {
		peers := &types.StringSlice{
			Output: hop.comm.Node.Peers(),
		}
		return proto.Marshal(peers)
	}
	return nil, pkg.ErrNoPeersConnected
}

// Add adds a record in the store
func Add(tag string, content []byte, passphrase string) error {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		var buf *bytes.Reader
		shouldEncrypt := true
		if passphrase == "" {
			shouldEncrypt = false
		}
		if shouldEncrypt {
			byteContent, err := hop.comm.Encryption.EncryptContent(content, passphrase)
			if err != nil {
				log.Errorf("Encryption failed :%s", err.Error())
				return err
			}
			buf = bytes.NewReader(byteContent)
		} else {
			buf = bytes.NewReader(content)
		}
		kind, _ := filetype.Match(content)
		info := &store.Info{
			Tag:         tag,
			Type:        kind.MIME.Value,
			Name:        tag,
			IsEncrypted: shouldEncrypt,
			Size:        int64(len(content)),
		}
		id, err := hop.comm.Node.Add(hop.ctx, buf, info)
		if err != nil {
			return err
		}
		log.Infof("tag %s : hash %s", tag, id)
		// Update advertise info
		bf, err := FilterFromState()
		if err != nil {
			log.Error("Unable to filter state")
			return err
		}
		hop.discService.AddAdvertisingInfo(CRDTStatus, bf)
		return nil
	}
	return ErrNodeNotRunning
}

// AddDir adds a directory in the store
func AddDir(tag string, path string) error {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		filePath := path
		fileinfo, err := os.Lstat(filePath)
		if err != nil {
			log.Errorf("Failed executing find file path Err:%s", err.Error())
			return err
		}
		info := &store.Info{
			Tag:         tag,
			Type:        "directory",
			Name:        tag,
			IsEncrypted: false,
			Size:        fileinfo.Size(),
		}
		id, err := hop.comm.Node.AddDir(hop.ctx, filePath, info)
		if err != nil {
			return err
		}
		log.Infof("tag %s : hash %s", tag, id)
		// Update advertise info
		bf, err := FilterFromState()
		if err != nil {
			log.Error("Unable to filter state")
			return err
		}
		hop.discService.AddAdvertisingInfo(CRDTStatus, bf)
		return nil
	}
	return ErrNodeNotRunning
}

// Get gets a record from the store by given tag
func Get(tag string, passphrase string) ([]byte, error) {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		r, info, err := hop.comm.Node.Get(hop.ctx, tag)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		if info.IsEncrypted {
			if passphrase == "" {
				log.Error("passphrase is empty")
				return nil, ErrEncryptionPassphrase
			}
			buf := bytes.NewBuffer(nil)
			_, err = io.Copy(buf, r)
			if err != nil {
				log.Errorf("Failed io.Copy file encryption:%s", err.Error())
				return nil, err
			}
			byteContent, err := hop.comm.Encryption.DecryptContent(buf.Bytes(), passphrase)
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
	return nil, ErrNodeNotRunning
}

// GetTags gets all the tags from the store
func GetTags() ([]byte, error) {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		tags, err := hop.comm.Node.ReplManager().GetAllTags()
		if err != nil {
			return nil, err
		}
		allTags := &types.StringSlice{
			Output: tags,
		}
		return proto.Marshal(allTags)
	}
	return nil, ErrNodeNotRunning
}

// Index gets all content metatags from the store
func Index() (string, error) {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		tags, err := hop.comm.Node.ReplManager().Index()
		if err != nil {
			return "", err
		}
		tagsBytes, err := json.Marshal(tags)
		if err != nil {
			return "", err
		}
		return string(tagsBytes), nil
	}
	return "", ErrNodeNotRunning
}

// Version of ipfs-lite
func Version() string {
	return version.MobileVersion
}

// Stop the node
func Stop() {
	hop.comm.Node.Stop()
}

// Close the repo and all
func Close() {
	hop.comm.Stop()
}

// UpdateTopicStatus adds BLE advertising info
func UpdateTopicStatus(topic string, value string) {
	hop.discService.AddAdvertisingInfo(topic, value)
}

// Matrix returns matrix measurements
func Matrix() (string, error) {
	if hop != nil && hop.comm != nil {
		r := hop.comm.Repo
		nodeMatrixSnapshot := r.Matrix().NodeMatrixSnapshot()
		contentMatrixSnapshot := r.Matrix().ContentMatrixSnapshot()
		uptime := r.Matrix().GetTotalUptime()
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
	return "", ErrNodeNotRunning
}

func DownloadsInProgress() int {
	return len(hop.comm.Node.ReplManager().DownloadManagerStatus())
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

// Groups

// CreateGroup
func CreateGroup(name string) (string, error) {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		m := hop.comm.Node.ReplManager()
		gm, err := m.CreateGroup(name, hop.comm.Node.AddrInfo().ID, hop.comm.Node.GetPrivKey())
		if err != nil {
			return "", err
		}
		return gm.GroupID.String(), nil
	}
	return "", ErrNodeNotRunning
}

// CreateOpenGroup
func CreateOpenGroup(name string) (string, error) {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		m := hop.comm.Node.ReplManager()
		gm, err := m.CreateOpenGroup(name, hop.comm.Node.AddrInfo().ID, hop.comm.Node.GetPrivKey())
		if err != nil {
			return "", err
		}
		return gm.GroupID.String(), nil
	}
	return "", ErrNodeNotRunning
}

// AddMember
func AddMember(peerID, groupIDString string) error {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		newPeerId, err := peer.Decode(peerID)
		if err != nil {
			return err
		}
		groupID, err := peer.Decode(groupIDString)
		if err != nil {
			return err
		}
		nemMemberPubKey := hop.comm.Node.GetPubKey(newPeerId)

		m := hop.comm.Node.ReplManager()
		return m.GroupAddMember(hop.comm.Node.AddrInfo().ID, newPeerId, groupID, hop.comm.Node.GetPrivKey(), nemMemberPubKey)
	}
	return ErrNodeNotRunning
}

// GetAllGroups
func GetAllGroups() ([]byte, error) {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		addrs := []string{}
		m := hop.comm.Node.ReplManager()
		groups, err := m.GroupGetAllGroups(hop.comm.Node.AddrInfo().ID, hop.comm.Node.GetPrivKey())
		if err != nil {
			return nil, err
		}
		for _, v := range groups {
			addrs = append(addrs, v.GroupID.String())
		}
		addrsOP := &types.StringSlice{
			Output: addrs,
		}
		return proto.Marshal(addrsOP)
	}
	return nil, ErrNodeNotRunning
}

// GetGroupName
func GetGroupName(peerID, groupIDString string) (string, error) {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		groupID, err := peer.Decode(groupIDString)
		if err != nil {
			return "", err
		}
		m := hop.comm.Node.ReplManager()
		info, err := m.GroupGetInfo(hop.comm.Node.AddrInfo().ID, groupID, hop.comm.Node.GetPrivKey())
		if err != nil {
			return "", err
		}
		return info.Name, nil
	}
	return "", ErrNodeNotRunning
}

// GetAllContent
// AddContent

// GroupAdd adds a record in a group
func GroupAdd(tag string, content []byte, passphrase, groupID string) error {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		var buf *bytes.Reader
		shouldEncrypt := true
		if passphrase == "" {
			shouldEncrypt = false
		}
		if shouldEncrypt {
			byteContent, err := hop.comm.Encryption.EncryptContent(content, passphrase)
			if err != nil {
				log.Errorf("Encryption failed :%s", err.Error())
				return err
			}
			buf = bytes.NewReader(byteContent)
		} else {
			buf = bytes.NewReader(content)
		}
		kind, _ := filetype.Match(content)
		info := &store.Info{
			Tag:         tag,
			Type:        kind.MIME.Value,
			Name:        tag,
			IsEncrypted: shouldEncrypt,
			Size:        int64(len(content)),
		}
		id, err := hop.comm.Node.GroupAdd(hop.ctx, buf, info, groupID)
		if err != nil {
			return err
		}
		log.Infof("tag %s : hash %s", tag, id)
		// Update advertise info
		sk := hop.comm.Repo.StateKeeper()
		st, err := sk.GetState(groupID)
		if err != nil {
			return err
		}
		if st != nil {
			f, _ := st.Filter.MarshalJSON()
			var data map[string]interface{}
			if err := json.Unmarshal(f, &data); err != nil {
				return err
			}
			hop.discService.AddAdvertisingInfo(CRDTStatus, data["b"].(string))
		}
		return nil
	}
	return ErrNodeNotRunning
}

// GroupAddDir adds a directory in a group
func GroupAddDir(tag, path, groupID string) error {
	mtx.Lock()
	defer mtx.Unlock()
	if hop != nil && hop.comm != nil {
		filePath := path
		fileinfo, err := os.Lstat(filePath)
		if err != nil {
			log.Errorf("Failed executing find file path Err:%s", err.Error())
			return err
		}
		info := &store.Info{
			Tag:         tag,
			Type:        "directory",
			Name:        tag,
			IsEncrypted: false,
			Size:        fileinfo.Size(),
		}
		id, err := hop.comm.Node.GroupAddDir(hop.ctx, filePath, info, groupID)
		if err != nil {
			return err
		}
		log.Infof("tag %s : hash %s", tag, id)
		// Update advertise info
		sk := hop.comm.Repo.StateKeeper()
		st, err := sk.GetState(groupID)
		if err != nil {
			return err
		}
		f, _ := st.Filter.MarshalJSON()
		var data map[string]interface{}
		if err := json.Unmarshal(f, &data); err != nil {
			return err
		}
		hop.discService.AddAdvertisingInfo(CRDTStatus, data["b"].(string))
		return nil
	}
	return ErrNodeNotRunning
}
