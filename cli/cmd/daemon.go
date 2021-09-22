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

type notifier struct {
	matrix *matrix.MatrixKeeper
}

func (n *notifier) Listen(network.Network, ma.Multiaddr)      {}
func (n *notifier) ListenClose(network.Network, ma.Multiaddr) {}
func (n *notifier) Connected(net network.Network, c network.Conn) {
	// NodeMatrix management
	n.matrix.NodeConnected(c.RemotePeer().String())
}
func (n *notifier) Disconnected(net network.Network, c network.Conn) {
	// NodeMatrix management
	n.matrix.NodeDisconnected(c.RemotePeer().String())
}
func (n *notifier) OpenedStream(net network.Network, s network.Stream) {}
func (n *notifier) ClosedStream(network.Network, network.Stream)       {}

// InitDaemonCmd creates the daemon command
func InitDaemonCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "daemon",
		Short: "Start datahop daemon",
		Long: `
This command is used to start the Datahop Daemon.
		`,
		Run: func(cmd *cobra.Command, args []string) {
			litePeer, err := ipfslite.New(comm.Context, comm.Cancel, comm.Repo)
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			comm.LitePeer = litePeer
			networkNotifier := notifier{litePeer.Repo.Matrix()}
			litePeer.Host.Network().Notify(&networkNotifier)
			cfg, err := comm.Repo.Config()
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			datahopCli := `
       __              __                __                                              __  __ 
      /  |            /  |              /  |                                            /  |/  |
  ____$$ |  ______   _$$ |_     ______  $$ |____    ______    ______            _______ $$ |$$/ 
 /    $$ | /      \ / $$   |   /      \ $$      \  /      \  /      \  ______  /       |$$ |/  |
/$$$$$$$ | $$$$$$  |$$$$$$/    $$$$$$  |$$$$$$$  |/$$$$$$  |/$$$$$$  |/      |/$$$$$$$/ $$ |$$ |
$$ |  $$ | /    $$ |  $$ | __  /    $$ |$$ |  $$ |$$ |  $$ |$$ |  $$ |$$$$$$/ $$ |      $$ |$$ |
$$ \__$$ |/$$$$$$$ |  $$ |/  |/$$$$$$$ |$$ |  $$ |$$ \__$$ |$$ |__$$ |        $$ \_____ $$ |$$ |
$$    $$ |$$    $$ |  $$  $$/ $$    $$ |$$ |  $$ |$$    $$/ $$    $$/         $$       |$$ |$$ |
 $$$$$$$/  $$$$$$$/    $$$$/   $$$$$$$/ $$/   $$/  $$$$$$/  $$$$$$$/           $$$$$$$/ $$/ $$/ 
                                                            $$ |                                
                                                            $$ |                                
                                                            $$/                                 
`
			fmt.Println(datahopCli)
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
