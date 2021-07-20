package cmd

import (
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/spf13/cobra"
)

var StatCmd *cobra.Command

func InitStatCmd(comm *common.Common) {
	StatCmd = &cobra.Command{
		Use:   "stat",
		Short: "Get datahop node stats",
		Long:  `Add Long Description`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Stats")
		},
	}
}
