package cmd

import (
	"errors"

	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/datahop/ipfs-lite/cli/out"
	"github.com/spf13/cobra"
)

func InitIndexCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "index",
		Short: "Index datahop node content",
		Long: `
"The commend is used to get the index of tag-content"
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.LitePeer == nil || !comm.LitePeer.IsOnline() {
				return errors.New("daemon not running")
			}
			tags, err := comm.LitePeer.Manager.Index()
			if err != nil {
				return err
			}

			// output
			err = out.Print(cmd, tags, parseFormat(cmd))
			if err != nil {
				log.Error("Unable to get config ", err)
				return err
			}
			return nil
		},
	}
}
