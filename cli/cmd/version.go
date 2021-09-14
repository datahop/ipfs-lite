package cmd

import (
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/datahop/ipfs-lite/version"
	"github.com/spf13/cobra"
)

func InitVersionCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Datahop cli version",
		Long:  `Add Long Description`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version.CliVersion)
		},
	}
}
