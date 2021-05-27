package datahop

type BleAdvertisingDriver interface {
	// Start the native driver
	Start(localPID string)
	AddAdvertisingInfo(topic string, info []byte)
	// Stop the native driver
	Stop()
	NotifyNetworkInformation(topic string, network string, pass string, info string)
	NotifyEmptyValue(topic string)
}
type BleDiscoveryDriver interface {
	// Start the native driver
	Start(localPID string, scanTime int, interval int)
	AddAdvertisingInfo(topic string, info []byte)
	// Stop the native driver
	Stop()
}

type BleDiscNotifier interface {
	PeerDiscovered(device string)
	PeerSameStatusDiscovered(device string, topic string)
	PeerDifferentStatusDiscovered(device string, topic string, network string, pass string, info string)
}

type BleAdvNotifier interface {
	SameStatusDiscovered()
	DifferentStatusDiscovered(value []byte)
}
