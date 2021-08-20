package cmd

import (
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/spf13/cobra"
)

func InitGetCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get content by tag",
		Long:  `Add Long Description`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Get Placeholder")
		},
	}
}
