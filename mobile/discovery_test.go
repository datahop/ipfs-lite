package datahop

import (
	"testing"
	"time"

	p2pnet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

type mockP2pHostConnectivity struct{}

func (m mockP2pHostConnectivity) Connectedness(id peer.ID) p2pnet.Connectedness {
	return p2pnet.NotConnected
}

type mockD2dConnectivityMatrix struct{}

func (m mockD2dConnectivityMatrix) BLEDiscovered(s string) {
	return
}

func (m mockD2dConnectivityMatrix) WifiConnected(s string, i int, i2 int, i3 int) {
	return
}

type mockIdentity struct {
	id string
}

func (m mockIdentity) ID() string {
	return m.id
}

func (m mockIdentity) PeerInfo() string {
	return m.id
}

func TestNewDiscoveryService(t *testing.T) {
	dd := MockDisDriver{}
	ad := MockAdvDriver{}
	whs := MockWifiHotspot{}
	wc := MockWifiConn{}
	p2p := mockP2pHostConnectivity{}
	d2d := mockD2dConnectivityMatrix{}
	i := mockIdentity{
		id: "one_peer",
	}
	service := NewDiscoveryService(
		dd,
		ad,
		1000,
		20000,
		whs,
		wc,
		DiscoveryServiceTag,
		p2p,
		d2d,
		i,
	)
	var discService *discoveryService
	if res, ok := service.(*discoveryService); ok {
		discService = res
	}
	go func() {
		for {
			discService.AddAdvertisingInfo(CRDTStatus, "info_one")
			select {
			case <-discService.stopSignal:
				log.Error("Stop AddAdvertisingInfo Routine")
				return
			case <-time.After(time.Second * 10):
			}
		}
	}()
	<-time.After(time.Second * 5)
	discService.stopSignal <- struct{}{}
}
