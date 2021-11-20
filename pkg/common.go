package ipfslite

import (
	"context"
	"os"
	"path/filepath"

	"github.com/datahop/ipfs-lite/internal/matrix"
	"github.com/libp2p/go-libp2p-core/network"
	ma "github.com/multiformats/go-multiaddr"

	ipfslite "github.com/datahop/ipfs-lite/internal/ipfs"
	"github.com/datahop/ipfs-lite/internal/repo"
)

// Common features for cli commands
type Common struct {
	Root     string
	Port     string
	Repo     repo.Repo
	LitePeer *ipfslite.Peer
	Context  context.Context
	Cancel   context.CancelFunc
}

type Notifier struct {
	matrix *matrix.MatrixKeeper
}

func (n *Notifier) Listen(network.Network, ma.Multiaddr)      {}
func (n *Notifier) ListenClose(network.Network, ma.Multiaddr) {}
func (n *Notifier) Connected(net network.Network, c network.Conn) {
	// NodeMatrix management
	n.matrix.NodeConnected(c.RemotePeer().String())
}
func (n *Notifier) Disconnected(net network.Network, c network.Conn) {
	// NodeMatrix management
	n.matrix.NodeDisconnected(c.RemotePeer().String())
}
func (n *Notifier) OpenedStream(net network.Network, s network.Stream) {}
func (n *Notifier) ClosedStream(network.Network, network.Stream)       {}

func New(path, port string) (*Common, error) {
	ctx, cancel := context.WithCancel(context.Background())
	home, err := os.UserHomeDir()
	if err != nil {
		os.Exit(1)
	}
	root := filepath.Join(home, path)
	return &Common{
		Root:    root,
		Port:    port,
		Context: ctx,
		Cancel:  cancel,
	}, nil
}

func NewNotifier(matrix *matrix.MatrixKeeper) *Notifier {
	return &Notifier{matrix: matrix}
}
