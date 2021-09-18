package datahop

type AdvertisingDriver interface {
	// Start the native driver
	Start(localPID, peerinfo string)
	AddAdvertisingInfo(topic string, info string)
	// Stop the native driver
	Stop()
	NotifyNetworkInformation(network string, pass string)
	NotifyEmptyValue()
}
type DiscoveryDriver interface {
	// Start the native driver
	Start(localPID, peerinfo string, scanTime int, interval int)
	AddAdvertisingInfo(topic string, info string)
	// Stop the native driver
	Stop()
}

type DiscoveryNotifier interface {
	DiscoveryPeerSameStatus(device string, topic string)
	DiscoveryPeerDifferentStatus(device string, topic string, network string, pass string, info string)
}

type AdvertisementNotifier interface {
	AdvertiserPeerSameStatus()
	AdvertiserPeerDifferentStatus(topic string, value []byte, peerinfo string)
}
