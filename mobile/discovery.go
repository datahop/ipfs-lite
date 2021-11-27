package datahop

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const DiscoveryServiceTag = "_datahop-discovery._ble"

type ServiceType string

const (
	ScanAndAdvertise ServiceType = "Both"
	OnlyScan         ServiceType = "OnlyScan"
	OnlyAdv          ServiceType = "OnlyAdv"
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
	id              string
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
	service         ServiceType

	isHost        bool
	stateInformer *stateInformer
}

func NewDiscoveryService(
	id string,
	discDriver DiscoveryDriver,
	advDriver AdvertisingDriver,
	scanTime int,
	interval int,
	hs WifiHotspot,
	con WifiConnection,
	serviceTag string,
) (Service, error) {
	if serviceTag == "" {
		serviceTag = DiscoveryServiceTag
	}
	discovery := &discoveryService{
		id:              id,
		discovery:       discDriver,
		advertiser:      advDriver,
		tag:             serviceTag,
		wifiHS:          hs,
		wifiCon:         con,
		scan:            scanTime,
		interval:        interval,
		connected:       false,
		stopSignal:      make(chan struct{}),
		advertisingInfo: make(map[string]string),
	}
	return discovery, nil
}

func (b *discoveryService) Start() {
	log.Debug("discoveryService Start")
	b.discovery.Start(b.tag, b.id, b.scan, b.interval)
	b.advertiser.Start(b.tag, b.id)
	stepsLog.Debug("discovery & advertiser started")
	b.service = ScanAndAdvertise
}

func (b *discoveryService) StartOnlyAdvertising() {
	log.Debug("discoveryService Start advertising only")
	b.advertiser.Start(b.tag, b.id)
	b.service = OnlyAdv
}

func (b *discoveryService) StartOnlyScanning() {
	log.Debug("discoveryService Start scanning only")
	b.discovery.Start(b.tag, b.id, b.scan, b.interval)
	b.service = OnlyScan
}

func (b *discoveryService) AddAdvertisingInfo(topic string, info string) {
	log.Debug("discoveryService AddAdvertisingInfo :", topic, info)

	if b.advertisingInfo[topic] != info {
		b.discovery.AddAdvertisingInfo(topic, info)
		b.advertiser.AddAdvertisingInfo(topic, info)
		b.advertisingInfo[topic] = info
	}
}

func (b *discoveryService) Close() error {
	mtx.Lock()
	defer mtx.Unlock()
	log.Debug("discoveryService Close")
	b.discovery.Stop()
	b.advertiser.Stop()
	stepsLog.Debug("discovery & advertiser stopped")
	hop.wifiCon.Disconnect()
	hop.wifiHS.Stop()
	stepsLog.Debug("wifi conn & hotspot stopped")
	b.isHost = false
	return nil
}

func (b *discoveryService) close() error {
	log.Debug("discoveryService Close")
	b.discovery.Stop()
	b.advertiser.Stop()
	stepsLog.Debug("discovery & advertiser stopped")
	hop.wifiCon.Disconnect()
	hop.wifiHS.Stop()
	stepsLog.Debug("wifi conn & hotspot stopped")
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
	stepsLog.Debug("discovery new peer device same status", device, topic)
	//	hop.discoveryDriver.Stop()
}

func (b *discoveryService) DiscoveryPeerDifferentStatus(device string, topic string, network string, pass string, peerId string) {
	log.Debug("discovery new peer device different status", device, topic, network, pass, peerId)
	stepsLog.Debug("discovery new peer device different status", device, topic, network, pass, peerId)
	if b.connected {
		return
	}
	mtx.Lock()
	defer mtx.Unlock()
	hop.comm.Repo.Matrix().BLEDiscovered(peerId)
	hop.wifiCon.Connect(network, pass, "", peerId)
}

func (b *discoveryService) AdvertiserPeerSameStatus() {
	log.Debug("advertising new peer device same status")
	stepsLog.Debug("advertising new peer device same status")
	b.advertiser.NotifyEmptyValue()
}

func (b *discoveryService) AdvertiserPeerDifferentStatus(topic string, value []byte, id string) {
	log.Debugf("advertising new peer device different status : %s", string(value))
	stepsLog.Debugf("advertising new peer device different status : %s", string(value))
	log.Debugf("peerinfo: %s", id)
	mtx.Lock()
	defer mtx.Unlock()
	if hop.comm.Node.IsPeerConnected(id) {
		log.Debug("Peer already connected")
		return
	}
	hop.comm.Repo.Matrix().BLEDiscovered(id)
	if !b.isHost {
		b.wifiHS.Start()
		stepsLog.Debug("wifi hotspot started")
		b.isHost = true
	}
}

func (b *discoveryService) OnConnectionSuccess(started int64, completed int64, rssi int, speed int, freq int) {
	mtx.Lock()
	defer mtx.Unlock()
	log.Debug("Connection success")
	hop.comm.Repo.Matrix().WifiConnected(hop.wifiCon.Host(), rssi, speed, freq)
	b.connected = true
}

func (b *discoveryService) OnConnectionFailure(code int, started int64, failed int64) {
	mtx.Lock()
	defer mtx.Unlock()
	log.Debug("Connection failure ", code)
	hop.wifiCon.Disconnect()
	b.connected = false
}

func (b *discoveryService) OnDisconnect() {
	log.Debug("OnDisconnect")
	b.connected = false
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
	//if b.numConnected > 0 && num == 0 && b.service != "onlyAdv" {
	//	stepsLog.Debug("discovery started")
	//	b.discovery.Start(b.tag, b.id, b.scan, b.interval)
	//}
	//b.numConnected = num
}
