package datahop

type WifiAwareClientDriver interface {
	Connect(peerId string)
	Disconnect()
	Host() string
}

type WifiAwareServerDriver interface {
	Start(peerId string, port int)
	Stop()
}


type WifiAwareNotifier interface {
	OnConnectionFailure(message string)
	OnConnectionSuccess(ip string, port int, peerId string)
	OnDisconnect()
}
