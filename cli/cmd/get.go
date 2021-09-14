package cmd

import (
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/spf13/cobra"
)

func InitGetCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get content by tag",
		Long: `
"The commend is used to get file/content from the
datahop network by a simple tag"
		`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// TODO get command
			cmd.Printf("Get Placeholder")
		},
	}
}
