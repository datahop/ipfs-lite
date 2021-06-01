package datahop

type AdvertisingDriver interface {
	// Start the native driver
	Start(localPID string)
	AddAdvertisingInfo(topic string, info []byte)
	// Stop the native driver
	Stop()
	NotifyNetworkInformation(network string, pass string, info string)
	NotifyEmptyValue()
}
type DiscoveryDriver interface {
	// Start the native driver
	Start(localPID string, scanTime int, interval int)
	AddAdvertisingInfo(topic string, info []byte)
	// Stop the native driver
	Stop()
}

type DiscoveryNotifier interface {
	PeerDiscovered(device string)
	PeerSameStatusDiscovered(device string, topic string)
	PeerDifferentStatusDiscovered(device string, topic string, network string, pass string, info string)
}

type AdvertisementNotifier interface {
	SameStatusDiscovered()
	DifferentStatusDiscovered(topic string, value []byte)
}
