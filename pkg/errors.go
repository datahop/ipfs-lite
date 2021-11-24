package pkg

import "errors"

var (
	// ErrNoPeersConnected is returned if there is no peer connected
	ErrNoPeersConnected = errors.New("no Peers connected")

	// ErrNoPeerAddress re returned if peer address is not available
	ErrNoPeerAddress = errors.New("could not get peer address")
)
