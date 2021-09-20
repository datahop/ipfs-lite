package datahop

type WifiConnection interface {
	Connect(network, pass, ip, host string)
	Disconnect()
	Host() string
}

type WifiConnectionNotifier interface {
	OnConnectionSuccess(started, completed int64, rssi, speed, freq int)
	OnConnectionFailure(code int, started, failed int64)
	OnDisconnect()
}
