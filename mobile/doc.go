/*
Package datahop is a mobile client for running a minimalistic datahop ipfslite node.

A datahop ipfslite node uses a persistent repo and a config file same as go-ipfs
to have necessary config options cashed. It uses a global "datahop" object to run
and maintain the ipfslite node.

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

	cm := ConnManager{}
	err := Init(root, cm)
	if err != nil {
		panic(err)
	}

Start the datahop ipfslite node by calling "Start"

	err := Start()
	if err != nil {
		panic(err)
	}

To check if the ipfslite node is running or not

	isOnline := IsNodeOnline()
	if isOnline {
		// node is online
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

*/
package datahop
