package datahop

import (
	"github.com/libp2p/go-libp2p-core/host"
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
	host            host.Host
	tag             string
	lk              sync.Mutex
	notifees        []Notifee
	wifiHS          WifiHotspot
	wifiCon         WifiConnection
	scan            int
	interval        int
	advertisingInfo map[string][]byte
}

func NewDiscoveryService(peerhost host.Host, discDriver DiscoveryDriver, advDriver AdvertisingDriver, scanTime int, interval int, hs WifiHotspot, con WifiConnection, serviceTag string) (Service, error) {
	adv := make(map[string][]byte)

	if serviceTag == "" {
		serviceTag = ServiceTag
	}
	discovery := &discoveryService{
		discovery:       discDriver,
		advertiser:      advDriver,
		host:            peerhost,
		tag:             serviceTag,
		wifiHS:          hs,
		wifiCon:         con,
		scan:            scanTime,
		interval:        interval,
		advertisingInfo: adv,
	}

	return discovery, nil
}

func (b *discoveryService) Start() {
	b.discovery.Start(b.tag, b.scan, b.interval)
	b.advertiser.Start(b.tag)
}

func (b *discoveryService) AddAdvertisingInfo(topic string, info []byte) {
	b.discovery.AddAdvertisingInfo(topic, info)
	b.advertiser.AddAdvertisingInfo(topic, info)
	b.advertiser.Stop()
	b.advertiser.Start(b.tag)
}

func (b *discoveryService) handleEntry(peerInfoByteString string) {
	b.lk.Lock()
	for _, n := range b.notifees {
		go n.HandlePeerFound(peerInfoByteString)
	}
	b.lk.Unlock()
}

func (b *discoveryService) Close() error {
	b.discovery.Stop()
	b.advertiser.Stop()
	return nil
}
func (b *discoveryService) RegisterNotifee(n Notifee) {
	b.lk.Lock()
	b.notifees = append(b.notifees, n)
	b.lk.Unlock()
}

func (b *discoveryService) UnregisterNotifee(n Notifee) {
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
	b.Close()
	hop.wifiCon.Connect(network, pass, "192.168.49.2")
}

func (b *discoveryService) SameStatusDiscovered() {
	log.Debug("advertising new peer device same status")
	b.advertiser.NotifyEmptyValue()
}

func (b *discoveryService) DifferentStatusDiscovered(topic string, value []byte) {
	log.Debug("advertising new peer device different status")
	//hop.advertisingDriver.NotifyNetworkInformation("topic1",GetPeerInfo())
	b.advertisingInfo[topic] = value
	b.discovery.Stop()
	b.wifiHS.Start()
	//hop.wifiHS.Start()
}

func (b *discoveryService) OnConnectionSuccess() {
	log.Debug("Connection success")
	//time.Sleep(10 * time.Second) // pauses execution for 2 seconds
	//hop.wifiCon.Disconnect()
	//b.handleEntry()
}

func (b *discoveryService) OnConnectionFailure(code int) {
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
