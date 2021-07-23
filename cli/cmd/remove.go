package cmd

import (
	"errors"

	"github.com/ipfs/go-datastore"

	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/spf13/cobra"
)

var RemoveCmd *cobra.Command

func InitRemoveCmd(comm *common.Common) {
	RemoveCmd = &cobra.Command{
		Use:   "remove",
		Short: "Remove content from datahop network",
		Long:  `Add Long Description`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.LitePeer == nil || !comm.LitePeer.IsOnline() {
				return errors.New("daemon not running")
			}
			tag := args[0]
			// Find cid for the chosen key
			id, err := comm.LitePeer.Manager.FindTag(tag)
			if err != nil {
				log.Error("Unable to find tag ", err)
				return err
			}

			err = comm.LitePeer.DeleteFile(comm.Context, id)
			if err != nil {
				log.Error("Content removal failed ", err)
				return err
			}
			err = comm.LitePeer.Manager.Delete(datastore.NewKey(tag))
			if err != nil {
				log.Error("Replication manager delete failed")
				return err
			}
			cmd.Printf("Content with tag \"%s\" removed", tag)
			return nil
		},
	}
}
