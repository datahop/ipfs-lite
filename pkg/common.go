package pkg

import (
	"context"

	_ "golang.org/x/mobile/bind"

	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/datahop/ipfs-lite/pkg/store"
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

	store.Store
}

// Common features for cli commands
type Common struct {
	root    string
	port    string
	Repo    repo.Repo
	Context context.Context
	cancel  context.CancelFunc
	Node    Node
}

func New(ctx context.Context, root, port string) (*Common, error) {
	ctx2, cancel := context.WithCancel(ctx)
	r, err := repo.Open(root)
	if err != nil {
		log.Error("Repo Open Failed : ", err.Error())
		cancel()
		return nil, err
	}
	return &Common{
		Repo:    r,
		root:    root,
		port:    port,
		Context: ctx2,
		cancel:  cancel,
	}, nil
}

func (comm *Common) GetRoot() string {
	return comm.root
}

func (comm *Common) GetPort() string {
	return comm.port
}