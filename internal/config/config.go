package config

import (
	"encoding/base64"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

var log = logging.Logger("config")

const (
	// SwarmPort will be used to listen on the swarm
	SwarmPort = "4501"

	DefaultPrivKeyBitSize = 2048
)

// Identity tracks the configuration of the local node's identity.
type Identity struct {
	PeerID  string
	PrivKey string `json:",omitempty"`
}

// Addresses stores the (string) multiaddr addresses for the node.
type Addresses struct {
	Swarm []string // addresses for the swarm to listen on
}

// Config wraps configuration options for the Peer.
type Config struct {
	Identity  Identity  // local node's peer identity
	Addresses Addresses // local node's addresses
	Bootstrap []string
	SwarmPort string
}

// NewConfig creates an identity and config for the node
func NewConfig(swarmPort string) (*Config, error) {
	identity, err := identityConfig(DefaultPrivKeyBitSize)
	if err != nil {
		return nil, err
	}
	log.Debug("Generated new id ", identity.PeerID)
	if swarmPort == "0" {
		swarmPort = SwarmPort
	}
	conf := &Config{
		Addresses: addressesConfig(swarmPort),
		Bootstrap: []string{},
		Identity:  identity,
		SwarmPort: swarmPort,
	}
	return conf, nil
}

func identityConfig(nbits int) (Identity, error) {
	ident := Identity{}
	sk, pk, err := ci.GenerateKeyPair(ci.RSA, nbits)
	if err != nil {
		return ident, err
	}
	skbytes, err := ci.MarshalPrivateKey(sk)
	if err != nil {
		return ident, err
	}
	ident.PrivKey = base64.StdEncoding.EncodeToString(skbytes)
	id, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return ident, err
	}
	ident.PeerID = id.Pretty()
	return ident, nil
}

func addressesConfig(swarmPort string) Addresses {
	return Addresses{
		Swarm: []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", swarmPort),
		},
	}
}
