package ipfslite

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"time"

	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	// DefaultConfigFile is the filename of the configuration file
	DefaultConfigFile = "config"

	SwarmPort = "4501"
)

var (
	defaultReprovideInterval = 12 * time.Hour
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

	// ReprovideInterval sets how often to reprovide records to the DHT
	ReprovideInterval time.Duration
}

// ConfigFilename returns the configuration file path given a configuration root
// directory. If the configuration root directory is empty, use the default one
func ConfigFilename(configroot string) (string, error) {
	return filepath.Join(configroot, DefaultConfigFile), nil
}

func ConfigInit(nbits int, swarmPort string) (*Config, error) {
	identity, err := identityConfig(nbits)
	if err != nil {
		return nil, err
	}
	conf := &Config{
		Addresses:         addressesConfig(swarmPort),
		Bootstrap:         nil,
		Identity:          identity,
		ReprovideInterval: defaultReprovideInterval,
	}

	return conf, nil
}

func identityConfig(nbits int) (Identity, error) {
	ident := Identity{}

	sk, pk, err := ci.GenerateKeyPair(ci.Ed25519, nbits)
	if err != nil {
		return ident, err
	}

	skbytes, err := sk.Bytes()
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
	if swarmPort == "0" {
		swarmPort = SwarmPort
	}
	return Addresses{
		Swarm: []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", swarmPort),
			fmt.Sprintf("/ip6/::/tcp/%s", swarmPort),
		},
	}
}
