package datahop

type WifiConnection interface {
	Connect(network string, pass string, ip string)
	Disconnect()
}

type WifiConnectionNotifier interface {
	OnConnectionSuccess(started int64, completed int64, rssi int, speed int, freq int)
	OnConnectionFailure(code int,started int64, failed int64)
	OnDisconnect()
}
