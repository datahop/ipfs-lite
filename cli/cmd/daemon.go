package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/datahop/ipfs-lite/internal/matrix"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/datahop/ipfs-lite/cli/common"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/network"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
)

var log = logging.Logger("cmd")

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

func InitDaemonCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "daemon",
		Short: "Start datahop daemon",
		Long:  `Add Long Description`,
		PreRun: func(cmd *cobra.Command, args []string) {

		},
		Run: func(cmd *cobra.Command, args []string) {
			litePeer, err := ipfslite.New(comm.Context, comm.Cancel, comm.Repo)
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			comm.LitePeer = litePeer
			networkNotifier := Notifier{litePeer.Repo.Matrix()}
			litePeer.Host.Network().Notify(&networkNotifier)
			cfg, err := comm.Repo.Config()
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			fmt.Println("Datahop daemon running on port", cfg.SwarmPort)
			var sigChan chan os.Signal
			sigChan = make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt)
			for {
				select {
				case <-sigChan:
					fmt.Println()
					comm.Cancel()
					return
				case <-comm.Context.Done():
					return
				}
			}
		},
	}
}
