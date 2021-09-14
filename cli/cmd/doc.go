package cmd

import (
	"fmt"

	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/datahop/ipfs-lite/cli/out"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func InitializeDocCommand(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "doc",
		Short: "Use to generate Document",
		Long: ` 
This command is used to generate documentation for the CLI.
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
				log.Error("Unable to get config ", err)
				return err
			}
			return nil
		},
	}
}
