package datahop

type WifiHotspot interface {
	Start() //(string, string)
	Stop()
}

type WifiHotspotNotifier interface {
	OnSuccess()
	OnFailure(code int)
	StopOnSuccess()
	StopOnFailure(code int)
	NetworkInfo(network string, password string)
	ClientsConnected(num int)
}
