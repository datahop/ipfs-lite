package cmd

import (
	"fmt"
	"os"
	"os/signal"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/datahop/ipfs-lite/cli/common"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
)

var log = logging.Logger("cmd")

var DaemonCmd *cobra.Command

func InitDaemonCmd(comm *common.Common) {
	DaemonCmd = &cobra.Command{
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
