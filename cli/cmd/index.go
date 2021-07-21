package cmd

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/spf13/cobra"
)

var IndexCmd *cobra.Command

func InitIndexCmd(comm *common.Common) {
	IndexCmd = &cobra.Command{
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
			b, err := json.Marshal(tags)
			if err != nil {
				log.Error("Unable to get crdt state ", err)
				return err
			}
			pFlag, _ := cmd.Flags().GetBool("pretty")
			log.Debug(pFlag)
			if pFlag {
				var prettyJSON bytes.Buffer
				err = json.Indent(&prettyJSON, b, "", "\t")
				if err != nil {
					log.Debug("JSON parse error: ", err)
					return err
				}
				cmd.Printf("%s\n", prettyJSON.String())
				return nil
			}
			cmd.Println(string(b))
			return nil
		},
	}
}
