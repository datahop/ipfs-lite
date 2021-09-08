package datahop

import (
	"io"
	"sync"
)

const ServiceTag = "_datahop-discovery._ble"

type Service interface {
	io.Closer
	RegisterNotifee(Notifee)
	UnregisterNotifee(Notifee)
}

type Notifee interface {
	HandlePeerFound(string)
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
	advertisingInfo map[string][]byte
	stopSignal      chan struct{}
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
) (Service, error) {
	if serviceTag == "" {
		serviceTag = ServiceTag
	}
	discovery := &discoveryService{
		discovery:       discDriver,
		advertiser:      advDriver,
		tag:             serviceTag,
		wifiHS:          hs,
		wifiCon:         con,
		scan:            scanTime,
		interval:        interval,
		stopSignal:      make(chan struct{}),
		advertisingInfo: make(map[string][]byte),
	}
	return discovery, nil
}

func (b *discoveryService) Start() {
	log.Debug("discoveryService Start")
	b.discovery.Start(b.tag, b.scan, b.interval)
	b.advertiser.Start(b.tag)
}

func (b *discoveryService) AddAdvertisingInfo(topic string, info []byte) {
	log.Debug("discoveryService AddAdvertisingInfo :", topic, string(info))
	b.discovery.AddAdvertisingInfo(topic, info)
	b.advertiser.AddAdvertisingInfo(topic, info)
	b.advertiser.Stop()
	b.advertiser.Start(b.tag)
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
	hop.wifiCon.Disconnect()
	hop.wifiHS.Stop()
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

func (b *discoveryService) PeerDiscovered(device string) {
	log.Debug("discovery new peer device ", device)
}

func (b *discoveryService) PeerSameStatusDiscovered(device string, topic string) {
	log.Debug("discovery new peer device same status", device, topic)
	//	hop.discoveryDriver.Stop()
}

func (b *discoveryService) PeerDifferentStatusDiscovered(device string, topic string, network string, pass string, peerinfo string) {
	log.Debug("discovery new peer device different status", device, topic, network, pass, peerinfo)
	hop.wifiCon.Connect(network, pass, "192.168.49.2")
	b.handleConnectionRequest = func() {
		b.handleEntry(peerinfo)
	}
}

func (b *discoveryService) SameStatusDiscovered() {
	log.Debug("advertising new peer device same status")
	b.advertiser.NotifyEmptyValue()
}

func (b *discoveryService) DifferentStatusDiscovered(topic string, value []byte) {
	log.Debug("advertising new peer device different status", string(value))
	//hop.advertisingDriver.NotifyNetworkInformation("topic1",GetPeerInfo())
	b.advertisingInfo[topic] = value
	b.discovery.Stop()
	b.wifiHS.Start()
}

func (b *discoveryService) OnConnectionSuccess(started int64, completed int64, rssi int, speed int, freq int){
	log.Debug("Connection success")
	b.handleConnectionRequest()
}

func (b *discoveryService) OnConnectionFailure(code int,started int64, failed int64) {
	log.Debug("Connection failure ", code)
	hop.wifiCon.Disconnect()
}

func (b *discoveryService) OnDisconnect() {
	log.Debug("OnDisconnect")
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
	b.advertiser.NotifyNetworkInformation(network, password, PeerInfo())
}

func (b *discoveryService) ClientsConnected(num int) {
	log.Debug("hotspot clients connected ", num)
}
