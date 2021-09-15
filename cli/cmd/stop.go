package cmd

import (
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/spf13/cobra"
)

func InitStopCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop datahop daemon",
		Long: `
"The commend is used to stop datahop daemon"
		`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Daemon Stopped")
			comm.Cancel()
		},
	}
}
