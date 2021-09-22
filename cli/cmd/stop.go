package cmd

import (
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/spf13/cobra"
)

// InitStopCmd creates the stop command
func InitStopCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop datahop daemon",
		Long: `
This command is used to stop datahop daemon
		`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Daemon Stopped")
			comm.Cancel()
		},
	}
}
