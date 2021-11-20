package cmd

import (
	ipfslite "github.com/datahop/ipfs-lite/pkg"
	"github.com/datahop/ipfs-lite/version"
	"github.com/spf13/cobra"
)

// InitVersionCmd creates the version command
func InitVersionCmd(comm *ipfslite.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Datahop cli version",
		Long: `
This command is used to get cli version
		`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version.CliVersion)
		},
	}
}
