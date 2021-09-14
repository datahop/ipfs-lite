package cmd

import (
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/spf13/cobra"
)

func InitStopCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop datahop daemon",
		Long:  `Add Long Description`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Daemon Stopped")
			comm.Cancel()
		},
	}
}
