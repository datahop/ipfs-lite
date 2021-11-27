package cmd

import (
	"fmt"

	"github.com/datahop/ipfs-lite/cli/out"
	ipfslite "github.com/datahop/ipfs-lite/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// InitializeDocCommand creates the doc command
func InitializeDocCommand(comm *ipfslite.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "doc",
		Short: "Use to generate documentation",
		Long: ` 
This command is used to generate documentation
for the CLI.
         `,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			var err error
			err = doc.GenMarkdownTree(cmd.Root(), dir)
			if err != nil {
				return err
			}
			err = out.Print(cmd, fmt.Sprintf("Documentation generated at %s", dir), parseFormat(cmd))
			if err != nil {
				return err
			}
			return nil
		},
	}
}
