package cmd

import (
	"errors"
	"os"

	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/spf13/cobra"
)

var AddCmd *cobra.Command

func InitAddCmd(comm *common.Common) {
	AddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add content into datahop network",
		Long:  `Add Long Description`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.LitePeer == nil || !comm.LitePeer.IsOnline() {
				return errors.New("daemon not running")
			}
			filePath := args[0]
			_, err := os.Lstat(filePath)
			if err != nil {
				log.Errorf("Failed executing share command Err:%s", err.Error())
				return err
			}
			f, err := os.Open(filePath)
			if err != nil {
				log.Errorf("Failed executing share command Err:%s", err.Error())
				return err
			}
			n, err := comm.LitePeer.AddFile(comm.Context, f, nil)
			if err != nil {
				return err
			}
			err = comm.LitePeer.Manager.Tag(f.Name(), n.Cid())
			if err != nil {
				return err
			}
			cmd.Printf("%s added with cid : %s", filePath, n.Cid().String())
			return nil
		},
	}
}
