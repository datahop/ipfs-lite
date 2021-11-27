package cmd

import (
	"errors"

	ipfslite "github.com/datahop/ipfs-lite/pkg"
	"github.com/ipfs/go-datastore"
	"github.com/spf13/cobra"
)

// InitRemoveCmd creates the remove command
func InitRemoveCmd(comm *ipfslite.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove content from datahop network",
		Long: `
This command is used to remove file/content from the
datahop network by a simple tag
		`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.Node == nil || !comm.Node.IsOnline() {
				return errors.New("daemon not running")
			}
			tag := args[0]
			err := comm.Node.Delete(comm.Context, tag)
			if err != nil {
				log.Error("Content removal failed ", err)
				return err
			}
			err = comm.Node.ReplManager().Delete(datastore.NewKey(tag))
			if err != nil {
				log.Error("Replication manager delete failed")
				return err
			}
			cmd.Printf("Content with tag \"%s\" removed", tag)
			return nil
		},
	}
}
