package datahop

import (
	"context"
//	"errors"
	"io"
//	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

//	logging "github.com/ipfs/go-log/v2"
//	ma "github.com/multiformats/go-multiaddr"
//	manet "github.com/multiformats/go-multiaddr/net"
	//"github.com/whyrusleeping/mdns"
)

var bleDiscovery *bleDiscoveryService

const ServiceTag = "_ipfs-discovery._udp"

type Service interface {
	io.Closer
	RegisterNotifee(Notifee)
	UnregisterNotifee(Notifee)
}

type Notifee interface {
	HandlePeerFound(peer.AddrInfo)
}


type bleDiscoveryService struct {
	discovery BleDiscoveryDriver
	advertiser BleAdvertisingDriver
	host    host.Host
	tag     string
	lk       sync.Mutex
	notifees []Notifee

	advertisingInfo	map[string][]byte

}


/*func getDialableListenAddrs(ph host.Host) ([]*net.TCPAddr, error) {
	var out []*net.TCPAddr
	addrs, err := ph.Network().InterfaceListenAddresses()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		na, err := manet.ToNetAddr(addr)
		if err != nil {
			continue
		}
		tcp, ok := na.(*net.TCPAddr)
		if ok {
			out = append(out, tcp)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("failed to find good external addr from peerhost")
	}
	return out, nil
}*/

func NewBleDiscoveryService(ctx context.Context, peerhost host.Host, discDriver BleDiscoveryDriver, advDriver BleAdvertisingDriver, scanTime time.Duration, interval time.Duration, hs WifiHotspot, con WifiConnection, serviceTag string) (Service, error) {

	/*var ipaddrs []net.IP
	port := 4001

	addrs, err := getDialableListenAddrs(peerhost)
	if err != nil {
		log.Warn(err)
	} else {
		port = addrs[0].Port
		for _, a := range addrs {
			ipaddrs = append(ipaddrs, a.IP)
		}
	}*/

	//myid := peerhost.ID().Pretty()

	adv := make(map[string][]byte)

	//info := []string{myid}

	if serviceTag == "" {
		serviceTag = ServiceTag
	}

	bleDiscovery := &bleDiscoveryService{
		discovery:   discDriver,
		advertiser:  advDriver,
		host:     peerhost,
		tag:      serviceTag,
		advertisingInfo: adv,
	}

	/*bleDiscovery.advertisingInfo["topic1"] = []byte("hola")

	for topic, info := range bleDiscovery.advertisingInfo {
		//fmt.Println(dish, price)
		bleDiscovery.discovery.AddAdvertisingInfo(topic,info)
		bleDiscovery.advertiser.AddAdvertisingInfo(topic,info)

	}

	bleDiscovery.discovery.Start(serviceTag,int(scanTime.Milliseconds()),int(interval.Milliseconds()))

	bleDiscovery.advertiser.Start(serviceTag)*/

	//go s.pollForEntries(ctx)

	return bleDiscovery, nil
}

func (b *bleDiscoveryService) AddAdvertisingInfo (topic string, info []byte) {
	b.discovery.AddAdvertisingInfo(topic,info)
	b.advertiser.AddAdvertisingInfo(topic,info)
	b.advertiser.Stop()
	b.advertiser.Start(b.tag)
}


/*func (b *bleDiscoveryService) pollForEntries(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		//execute mdns query right away at method call and then with every tick
		entriesCh := make(chan *mdns.ServiceEntry, 16)
		go func() {
			for entry := range entriesCh {
				m.handleEntry(entry)
			}
		}()

		log.Debug("starting mdns query")
		qp := &mdns.QueryParam{
			Domain:  "local",
			Entries: entriesCh,
			Service: m.tag,
			Timeout: time.Second * 5,
		}

		err := mdns.Query(qp)
		if err != nil {
			log.Warnw("mdns lookup error", "error", err)
		}
		close(entriesCh)
		log.Debug("mdns query complete")

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			log.Debug("mdns service halting")
			return
		}
	}
}*/

/*func (b *bleDiscoveryService) handleEntry(e *mdns.ServiceEntry) {
	log.Debugf("Handling MDNS entry: [IPv4 %s][IPv6 %s]:%d %s", e.AddrV4, e.AddrV6, e.Port, e.Info)
	mpeer, err := peer.Decode(e.Info)
	if err != nil {
		log.Warn("Error parsing peer ID from mdns entry: ", err)
		return
	}

	if mpeer == m.host.ID() {
		log.Debug("got our own mdns entry, skipping")
		return
	}

	var addr net.IP
	if e.AddrV4 != nil {
		addr = e.AddrV4
	} else if e.AddrV6 != nil {
		addr = e.AddrV6
	} else {
		log.Warn("Error parsing multiaddr from mdns entry: no IP address found")
		return
	}

	maddr, err := manet.FromNetAddr(&net.TCPAddr{
		IP:   addr,
		Port: e.Port,
	})
	if err != nil {
		log.Warn("Error parsing multiaddr from mdns entry: ", err)
		return
	}

	pi := peer.AddrInfo{
		ID:    mpeer,
		Addrs: []ma.Multiaddr{maddr},
	}

	m.lk.Lock()
	for _, n := range m.notifees {
		go n.HandlePeerFound(pi)
	}
	m.lk.Unlock()
}*/

func (b *bleDiscoveryService) Close()  error {
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
	log.Debug("BLE discovery new peer device ",device)
}

func (b *bleDiscoveryService) PeerSameStatusDiscovered(device string, topic string) {
	log.Debug("BLE discovery new peer device same status",device,topic)
	//	hop.discoveryDriver.Stop()
}

func (b *bleDiscoveryService) PeerDifferentStatusDiscovered(device string, topic string, network string, pass string, info string) {
	log.Debug("BLE discovery new peer device different status",device,topic,network,pass,info)
	bleDiscovery.Close()
	//hop.wifiHS.Stop()
	//time.Sleep(2 * time.Second)                                     // pauses execution for 2 seconds
	hop.wifiCon.Connect(network,pass,"192.168.49.2")
}


func (b *bleDiscoveryService) SameStatusDiscovered() {
	log.Debug("BLE advertising new peer device same status")
	bleDiscovery.advertiser.NotifyEmptyValue("topic1")
}

func (b *bleDiscoveryService) DifferentStatusDiscovered(value []byte) {
	log.Debug("BLE advertising new peer device different status")
	//hop.advertisingDriver.NotifyNetworkInformation("topic1",GetPeerInfo())
	//hop.advertisingInfo["topic1"] = value
	//hop.discoveryDriver.Stop()
	//hop.wifiHS.Stop()
	//hop.wifiHS.Start()
}


func (b *bleDiscoveryService) OnConnectionSuccess(){
	log.Debug("Connection success")
	time.Sleep(10 * time.Second)                                     // pauses execution for 2 seconds
	hop.wifiCon.Disconnect()

}

func (b *bleDiscoveryService) OnConnectionFailure(code int){
	log.Debug("Connection failure ",code)
	hop.wifiCon.Disconnect()

}

func (b *bleDiscoveryService) OnDisconnect(){
	log.Debug("OnDisconnect")

}

func (b *bleDiscoveryService) OnSuccess(){
	log.Debug("Network up")
}

func (b *bleDiscoveryService) OnFailure(code int){
	log.Debug("hotspot failure ",code)
}

func (b *bleDiscoveryService) StopOnSuccess(){
	log.Debug("StopOnSuccess")
}

func (b *bleDiscoveryService) StopOnFailure(code int){
	log.Debug("StopOnFailure ",code)
}

func (b *bleDiscoveryService) NetworkInfo(network string, password string){
	log.Debug("hotspot info ",network,password)
	//hop.advertisingDriver.NotifyNetworkInformation("topic1",network,password,GetPeerInfo())
}

func (b *bleDiscoveryService) ClientsConnected(num int){
	log.Debug("hotspot clients connected ",num)
}




