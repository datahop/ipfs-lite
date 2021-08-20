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
		Long:  `Add Long Description`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.LitePeer == nil || !comm.LitePeer.IsOnline() {
				return errors.New("daemon not running")
			}
			tags, err := comm.LitePeer.Manager.Index()
			if err != nil {
				return err
			}

			// output
			pFlag, _ := cmd.Flags().GetBool("pretty")
			jFlag, _ := cmd.Flags().GetBool("json")
			log.Debug(pFlag, jFlag)
			var f out.Format
			if jFlag {
				f = out.Json
			}
			if pFlag {
				f = out.PrettyJson
			}
			if !pFlag && !jFlag {
				f = out.NoStyle
			}
			err = out.Print(cmd, tags, f)
			if err != nil {
				log.Error("Unable to get config ", err)
				return err
			}
			return nil
		},
	}
}
