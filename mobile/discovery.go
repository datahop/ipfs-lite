package datahop

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	p2pnet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const DiscoveryServiceTag = "_datahop-discovery._ble"

type ServiceType string

const (
	ScanAndAdvertise ServiceType = "Both"
	OnlyScan                     = "OnlyScan"
	OnlyAdv                      = "OnlyAdv"
)

type Service interface {
	io.Closer
	RegisterNotifee(Notifee)
	UnregisterNotifee(Notifee)
}

type Notifee interface {
	HandlePeerFound(string)
}

type stateInformer struct {
	ctx    context.Context
	cancel context.CancelFunc
	*pubsub.Topic
}

type p2pHostConnectivity interface {
	Connectedness(peer.ID) p2pnet.Connectedness
}

type d2dConnectivityMatrix interface {
	BLEDiscovered(string)
	WifiConnected(string, int, int, int)
}

type identity interface {
	ID() string
	PeerInfo() string
}

type Message struct {
	Id        string
	CRDTState string
}

func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *Message) Unmarshal(val []byte) error {
	return json.Unmarshal(val, m)
}

type discoveryService struct {
	discovery       DiscoveryDriver
	advertiser      AdvertisingDriver
	tag             string
	lk              sync.Mutex
	notifees        []Notifee
	wifiHS          WifiHotspot
	wifiCon         WifiConnection
	scan            int
	interval        int
	stopSignal      chan struct{}
	advertisingInfo map[string]string
	connected       bool //wifi connection status connected/disconnected
	numConnected    int
	service         ServiceType

	isHost bool

	p2pHostConnectivity   p2pHostConnectivity
	d2dConnectivityMatrix d2dConnectivityMatrix

	identity identity
	// handleConnectionRequest will take care of the incoming connection request.
	// but it is not safe to use this approach, as in case of multiple back to
	// back connection requests we might loose some connection request as
	// handleConnectionRequest gets overwritten every time. we can actually rely
	// on mdns at this point of time as peers are already connected to the group owner
	handleConnectionRequest func()
}

func NewDiscoveryService(
	discDriver DiscoveryDriver,
	advDriver AdvertisingDriver,
	scanTime int,
	interval int,
	hs WifiHotspot,
	con WifiConnection,
	serviceTag string,
	p2pHostConnectivity p2pHostConnectivity,
	d2dConnectivityMatrix d2dConnectivityMatrix,
	informer identity,
) Service {
	if serviceTag == "" {
		serviceTag = DiscoveryServiceTag
	}
	discovery := &discoveryService{
		discovery:             discDriver,
		advertiser:            advDriver,
		tag:                   serviceTag,
		wifiHS:                hs,
		wifiCon:               con,
		scan:                  scanTime,
		interval:              interval,
		connected:             false,
		stopSignal:            make(chan struct{}),
		advertisingInfo:       make(map[string]string),
		p2pHostConnectivity:   p2pHostConnectivity,
		d2dConnectivityMatrix: d2dConnectivityMatrix,
		identity:              informer,
	}
	return discovery
}

func (b *discoveryService) Start() {
	log.Debug("discoveryService Start")
	b.discovery.Start(b.tag, b.identity.ID(), b.scan, b.interval)
	b.advertiser.Start(b.tag, b.identity.PeerInfo())
	b.service = "Both"
}

func (b *discoveryService) StartOnlyAdvertising() {
	log.Debug("discoveryService Start advertising only")
	//b.discovery.Start(b.tag, b.scan, b.interval)
	b.advertiser.Start(b.tag, b.identity.PeerInfo())
	b.service = "OnlyAdv"
}

func (b *discoveryService) StartOnlyScanning() {
	log.Debug("discoveryService Start scanning only")
	b.discovery.Start(b.tag, b.identity.ID(), b.scan, b.interval)
	b.service = "OnlyScan"
	//b.advertiser.Start(b.tag)
}

func (b *discoveryService) AddAdvertisingInfo(topic string, info string) {
	log.Debug("discoveryService AddAdvertisingInfo :", topic, info)
	if b.advertisingInfo[topic] != info {
		b.discovery.AddAdvertisingInfo(topic, info)
		b.advertiser.AddAdvertisingInfo(topic, info)
		b.advertisingInfo[topic] = info
	}
}

func (b *discoveryService) handleEntry(peerInfoByteString string) {
	log.Debug("discoveryService handleEntry")
	b.lk.Lock()
	for _, n := range b.notifees {
		go n.HandlePeerFound(peerInfoByteString)
	}
	b.lk.Unlock()
}

func (b *discoveryService) Close() error {
	log.Debug("discoveryService Close")
	b.discovery.Stop()
	b.advertiser.Stop()
	b.wifiCon.Disconnect()
	b.wifiHS.Stop()
	b.isHost = false
	return nil
}
func (b *discoveryService) RegisterNotifee(n Notifee) {
	log.Debug("discoveryService RegisterNotifee")
	b.lk.Lock()
	b.notifees = append(b.notifees, n)
	b.lk.Unlock()
}

func (b *discoveryService) UnregisterNotifee(n Notifee) {
	log.Debug("discoveryService UnregisterNotifee")
	b.lk.Lock()
	found := -1
	for i, notif := range b.notifees {
		if notif == n {
			found = i
			break
		}
	}
	if found != -1 {
		b.notifees = append(b.notifees[:found], b.notifees[found+1:]...)
	}
	b.lk.Unlock()
}

func (b *discoveryService) DiscoveryPeerSameStatus(device string, topic string) {
	log.Debug("discovery new peer device same status", device, topic)
	//	hop.discoveryDriver.Stop()
}

func (b *discoveryService) DiscoveryPeerDifferentStatus(device string, topic string, network string, pass string, peerinfo string) {
	log.Debug("discovery new peer device different status", device, topic, network, pass, peerinfo)
	var peerInfo peer.AddrInfo
	err := peerInfo.UnmarshalJSON([]byte(peerinfo))
	if err != nil {
		log.Errorf("failed parsing peerinfo %s", err.Error())
		return
	}
	conn := b.p2pHostConnectivity.Connectedness(peerInfo.ID)
	if conn == p2pnet.Connected {
		log.Debug("Peer already connected")
		return
	}
	b.handleConnectionRequest = func() {
		b.handleEntry(peerinfo)
	}
	if b.connected {
		b.handleConnectionRequest()
	}
	b.d2dConnectivityMatrix.BLEDiscovered(peerInfo.ID.String())
	b.wifiCon.Connect(network, pass, "192.168.49.2", peerInfo.ID.String())
}

func (b *discoveryService) AdvertiserPeerSameStatus() {
	log.Debug("advertising new peer device same status")
	b.advertiser.NotifyEmptyValue()
}

func (b *discoveryService) AdvertiserPeerDifferentStatus(topic string, value []byte, id string) {
	log.Debugf("advertising new peer device different status : %s", string(value))
	log.Debugf("peerinfo: %s", id)
	log.Debugf("discoveryService: %+v", b)

	conn := b.p2pHostConnectivity.Connectedness(peer.ID(id))
	if conn == p2pnet.Connected {
		log.Debug("Peer already connected")
		return
	}

	b.d2dConnectivityMatrix.BLEDiscovered(id)
	b.discovery.Stop()
	b.wifiHS.Start()
	b.isHost = true
}

func (b *discoveryService) OnConnectionSuccess(started int64, completed int64, rssi int, speed int, freq int) {
	log.Debug("Connection success")
	b.d2dConnectivityMatrix.WifiConnected(b.wifiCon.Host(), rssi, speed, freq)
	b.connected = true
	<-time.After(time.Second * 2)
	b.handleConnectionRequest()
}

func (b *discoveryService) OnConnectionFailure(code int, started int64, failed int64) {
	log.Debug("Connection failure ", code)
	b.wifiCon.Disconnect()
	b.connected = false
}

func (b *discoveryService) OnDisconnect() {
	log.Debug("OnDisconnect")
	b.connected = false
	if b.service != "OnlyAdv" {
		b.discovery.Start(b.tag, b.identity.ID(), b.scan, b.interval)
	}
}

func (b *discoveryService) OnSuccess() {
	log.Debug("Network up")
}

func (b *discoveryService) OnFailure(code int) {
	log.Debug("hotspot failure ", code)
}

func (b *discoveryService) StopOnSuccess() {
	log.Debug("StopOnSuccess")
}

func (b *discoveryService) StopOnFailure(code int) {
	log.Debug("StopOnFailure ", code)
}

func (b *discoveryService) NetworkInfo(network string, password string) {
	log.Debug("hotspot info ", network, password)
	b.advertiser.NotifyNetworkInformation(network, password)
}

func (b *discoveryService) ClientsConnected(num int) {
	log.Debug("hotspot clients connected ", num)
	if b.numConnected > 0 && num == 0 && b.service != "onlyAdv" {
		b.discovery.Start(b.tag, b.identity.ID(), b.scan, b.interval)
	}
	b.numConnected = num
}
