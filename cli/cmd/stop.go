package cmd

import (
	ipfslite "github.com/datahop/ipfs-lite/pkg"
	"github.com/spf13/cobra"
)

// InitStopCmd creates the stop command
func InitStopCmd(comm *ipfslite.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop datahop daemon",
		Long: `
This command is used to stop datahop daemon
		`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Daemon Stopped")
			comm.Stop()
		},
	}
}
