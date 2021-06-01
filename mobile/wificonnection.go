package datahop

type WifiConnection interface {
	Connect(network string, pass string,ip string)
	Disconnect()
}

type WifiConnectionNotifier interface {
	OnConnectionSuccess()
	OnConnectionFailure(code int)
	OnDisconnect()
}


