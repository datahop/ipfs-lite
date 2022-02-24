package pkg

import (
	"context"

	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/datahop/ipfs-lite/internal/security"
	"github.com/datahop/ipfs-lite/pkg/store"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type Node interface {
	Stop()
	NetworkNotifiee(network.Notifiee)
	Bootstrap([]peer.AddrInfo)
	NewFloodsubWithProtocols([]protocol.ID, ...pubsub.Option) (*pubsub.PubSub, error)
	Peers() []string
	Connect(peer.AddrInfo) error
	AddrInfo() *peer.AddrInfo
	IsOnline() bool
	ReplManager() *replication.Manager
	IsPeerConnected(string) bool
	GetPrivKey() crypto.PrivKey
	GetPubKey(id peer.ID) crypto.PubKey

	store.Store
}

type Encryption interface {
	EncryptContent([]byte, string) ([]byte, error)
	DecryptContent([]byte, string) ([]byte, error)
}

// Common features for cli commands
type Common struct {
	root       string
	port       string
	Repo       repo.Repo
	Context    context.Context
	cancel     context.CancelFunc
	Node       Node
	Encryption Encryption
}

func New(ctx context.Context, root, port string, en Encryption) (*Common, error) {
	ctx2, cancel := context.WithCancel(ctx)
	r, err := repo.Open(root)
	if err != nil {
		log.Error("Repo Open Failed : ", err.Error())
		cancel()
		return nil, err
	}
	if en == nil {
		en = &security.DefaultEncryption{}
	}
	return &Common{
		Repo:       r,
		root:       root,
		port:       port,
		Context:    ctx2,
		cancel:     cancel,
		Encryption: en,
	}, nil
}

func (comm *Common) GetRoot() string {
	return comm.root
}

func (comm *Common) GetPort() string {
	return comm.port
}
