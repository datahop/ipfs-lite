package datahop

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
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

type bleDiscoveryService struct {
	discovery       BleDiscoveryDriver
	advertiser      BleAdvertisingDriver
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

func NewBleDiscoveryService(peerhost host.Host, discDriver BleDiscoveryDriver, advDriver BleAdvertisingDriver, scanTime int, interval int, hs WifiHotspot, con WifiConnection, serviceTag string) (Service, error) {
	adv := make(map[string][]byte)

	if serviceTag == "" {
		serviceTag = ServiceTag
	}
	fmt.Println("NewBleDiscoveryService")
	bleDiscovery := &bleDiscoveryService{
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

	return bleDiscovery, nil
}

func (b *bleDiscoveryService) Start() {
	b.discovery.Start(b.tag, b.scan, b.interval)
	b.advertiser.Start(b.tag)
}

func (b *bleDiscoveryService) AddAdvertisingInfo(topic string, info []byte) {
	b.discovery.AddAdvertisingInfo(topic, info)
	b.advertiser.AddAdvertisingInfo(topic, info)
	b.advertiser.Stop()
	b.advertiser.Start(b.tag)
}

func (b *bleDiscoveryService) handleEntry(peerInfoByteString string) {
	b.lk.Lock()
	for _, n := range b.notifees {
		go n.HandlePeerFound(peerInfoByteString)
	}
	b.lk.Unlock()
}

func (b *bleDiscoveryService) Close() error {
	fmt.Println(b.discovery)
	b.discovery.Stop()
	b.advertiser.Stop()
	return nil
}
func (b *bleDiscoveryService) RegisterNotifee(n Notifee) {
	b.lk.Lock()
	b.notifees = append(b.notifees, n)
	b.lk.Unlock()
}

func (b *bleDiscoveryService) UnregisterNotifee(n Notifee) {
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

func (b *bleDiscoveryService) PeerDiscovered(device string) {
	log.Debug("BLE discovery new peer device ", device)
}

func (b *bleDiscoveryService) PeerSameStatusDiscovered(device string, topic string) {
	log.Debug("BLE discovery new peer device same status", device, topic)
	//	hop.discoveryDriver.Stop()
}

func (b *bleDiscoveryService) PeerDifferentStatusDiscovered(device string, topic string, network string, pass string, peerinfo string) {
	log.Debug("BLE discovery new peer device different status", device, topic, network, pass, peerinfo)
	b.Close()
	hop.wifiCon.Connect(network, pass, "192.168.49.2")
}

func (b *bleDiscoveryService) SameStatusDiscovered() {
	log.Debug("BLE advertising new peer device same status")
	b.advertiser.NotifyEmptyValue()
}

func (b *bleDiscoveryService) DifferentStatusDiscovered(topic string, value []byte) {
	log.Debug("BLE advertising new peer device different status")
	//hop.advertisingDriver.NotifyNetworkInformation("topic1",GetPeerInfo())
	b.advertisingInfo[topic] = value
	b.discovery.Stop()
	b.wifiHS.Start()
	//hop.wifiHS.Start()
}

func (b *bleDiscoveryService) OnConnectionSuccess() {
	log.Debug("Connection success")
	time.Sleep(10 * time.Second) // pauses execution for 2 seconds
	hop.wifiCon.Disconnect()
	//b.handleEntry()
}

func (b *bleDiscoveryService) OnConnectionFailure(code int) {
	log.Debug("Connection failure ", code)
	hop.wifiCon.Disconnect()

}

func (b *bleDiscoveryService) OnDisconnect() {
	log.Debug("OnDisconnect")
}

func (b *bleDiscoveryService) OnSuccess() {
	log.Debug("Network up")
}

func (b *bleDiscoveryService) OnFailure(code int) {
	log.Debug("hotspot failure ", code)
}

func (b *bleDiscoveryService) StopOnSuccess() {
	log.Debug("StopOnSuccess")
}

func (b *bleDiscoveryService) StopOnFailure(code int) {
	log.Debug("StopOnFailure ", code)
}

func (b *bleDiscoveryService) NetworkInfo(topic string, network string, password string) {
	log.Debug("hotspot info ", network, password)
	b.advertiser.NotifyNetworkInformation(network, password, PeerInfo())
}

func (b *bleDiscoveryService) ClientsConnected(num int) {
	log.Debug("hotspot clients connected ", num)
}
