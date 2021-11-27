/*
Package datahop is a mobile client for running a minimalistic datahop ipfslite node.

Datahop ipfslite node uses a persistent repo and a config file same as go-ipfs
to save necessary config options. It uses a global "datahop" object to run the
ipfslite node.

As this package is built keeping the mobile platform in mind (using gomobile),
all the functions deals with string or byte array to support gomobile datatype.

To create the global "datahop" object and a persistent repository the client has
to call "Init". It takes the location to create the repository and an "ConnectionManager".

	type ConnManager struct{}
	func (m ConnManager) PeerConnected(s string) {
		// do something
	}
	func (m ConnManager) PeerDisconnected(s string) {
		// do something
	}

	...

	type DiscoveryDriver struct{}
	func (m DiscoveryDriver) Start(localPID, peerInfo string, scanTime int, interval int) {
		// do nothing
	}

	func (m DiscoveryDriver) AddAdvertisingInfo(topic string, info string) {
		// do nothing
	}

	...

	type AdvertisingDriver struct{}
	func (m AdvertisingDriver) Start(localPID, peerInfo string) {
		// do nothing
	}

	func (m AdvertisingDriver) AddAdvertisingInfo(topic string, info string) {
		// do nothing
	}

	func (m AdvertisingDriver) Stop() {
		// do nothing
	}

	func (m AdvertisingDriver) NotifyNetworkInformation(network string, pass string) {
		// do nothing
	}

	func (m AdvertisingDriver) NotifyEmptyValue() {
		// do nothing
	}

	...

	type WifiHotspot struct{}
	func (m WifiHotspot) Start() {
		// do nothing
	}

	func (m WifiHotspot) Stop() {
		// do nothing
	}

	...

	type WifiConnection struct{}
		func (m WifiConnection) Connect(network, pass, ip, host string) {
		// do nothing
	}

	func (m WifiConnection) Disconnect() {
		// do nothing
	}

	func (m WifiConnection) Host() string {
		return ""
	}

	...

	cm := ConnManager{}
	err := Init(root, cm)
	if err != nil {
		panic(err)
	}

Start the datahop ipfslite node by calling "Start". It takes one parameter to
bootstrap the node with default datahop bootstrap node

	var shouldBootstrap bool
	err := Start(shouldBootstrap)
	if err != nil {
		panic(err)
	}

To be able to connect datahop ipfslite node to a private network node call
"StartPrivate". It takes two parameters "shouldBootstrap" & "secret_string"

	var shouldBootstrap bool
	var swarmKey []byte
	err := StartPrivate(shouldBootstrap, swarmKey)
	if err != nil {
		panic(err)
	}

To check if node is online

	is Online := IsNodeOnline()

BootstrapWithAddress will take a node id and bootstrap with it

	err := BootstrapWithAddress("/ip4/52.66.216.67/tcp/4501/p2p/QmcWEJqQD3bPMT5Mr7ijdwVCmVjUh5Z7CysiTQPgr2VZBC")
	if err != nil {
		panic(err)
	}

BootstrapWithPeerInfo will take peer info of a node in bytes, then bootstrap with it

	err := BootstrapWithPeerInfo("some peerinfo in byte array form")
	if err != nil {
		panic(err)
	}

To check how much storage the ipfslite node has taken up

	storageAllocated, err := DiskUsage()
	if err != nil {
		panic(err)
	}

All the ipfslite node id related information can be obtained by the following functions.
See the corresponding function definitions for more information

	id := ID()

	...

	addresses := Addrs()

	...

	addresses := InterfaceAddrs()

	...

	addresses := PeerInfo()

"PeerInfo" returns a string of the peer.AddrInfo []byte of the node. we can actually pass this around
to connect with other nodes

	peerInfo := PeerInfo()

	...

	// On some other node
	err := ConnectWithPeerInfo(peerInfo)
	if err != nil {
		panic(err)
	}

Clients can connect with each other using peer address aswell

	err := ConnectWithAddress(otherPeerAddress)
	if err != nil {
		panic(err)
	}

To see all the connected peers

	connectedPeers := Peers()

To stop the ipfslite node

	Stop()

To close the global "datahop"

	Close()

To check "package" version

	version := Version()

StartDiscovery starts D2D discovery. It takes three boolean values as parameter.
	- "advertising" : To Advertise BLE Scanning
	- "scanning" : To Scan BLE advertisements from other devices
	- "autoDisconnect" : Auto disconnect from a D2D discovered network

	var (
		advertising bool
		scanning bool
		autoDisconnect bool
	)
	err := StartDiscovery(advertising, scanning, autoDisconnect)
	if err != nil {
		panic(err)
	}

StopDiscovery stops D2D discovery.

	err := StopDiscovery()
	if err != nil {
		panic(err)
	}

UpdateTopicStatus will update the value of a BLE topic

	err := UpdateTopicStatus("myTopic", "My new value")
	if err != nil {
		panic(err)
	}

Add content into the node as well as network. It takes three params
	- "tag": a unique string to tag the given content in the network
	- "content": the content to add in the network in byte array form
	- "passphrase": to encrypt the content. pass "" (blank string) to add content without encryption

	err := Add("myTag", []byte("myContent", ""))
	if err != nil {
		panic(err)
	}

Get content from the network by name tag. It takes two params
	- "tag": a unique string tag for getting content from the network
	- "passphrase": to decrypt the content. pass "" (blank string) to get content without encryption

	err := Get("myTag", ""))
	if err != nil {
		panic(err)
	}

DownloadsInProgress will show how many downloads are currently in progress

GetTags will show available tags in the local node

Matrix will return node connection and content distribution matrix

State will return current bloom filter state for local content

FilterFromState will extract bloom filter string from State
*/
package datahop
