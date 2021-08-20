package cmd

import (
	"errors"
	"io"
	"os"
	"time"

	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/h2non/filetype"
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
			log.Debug(filePath)
			fileinfo, err := os.Lstat(filePath)
			if err != nil {
				log.Errorf("Failed executing share command Err:%s", err.Error())
				return err
			}
			f, err := os.Open(filePath)
			if err != nil {
				log.Errorf("Failed executing share command Err:%s", err.Error())
				return err
			}
			defer f.Close()
			n, err := comm.LitePeer.AddFile(comm.Context, f, nil)
			if err != nil {
				return err
			}
			_, err = f.Seek(0, io.SeekStart)
			if err != nil {
				return err
			}
			head := make([]byte, 261)
			_, err = f.Read(head)
			if err != nil {
				return err
			}
			kind, _ := filetype.Match(head)
			meta := &replication.Metatag{
				Size:      fileinfo.Size(),
				Type:      kind.MIME.Value,
				Name:      f.Name(),
				Hash:      n.Cid(),
				Timestamp: time.Now().Unix(),
			}
			err = comm.LitePeer.Manager.Tag(f.Name(), meta)
			if err != nil {
				return err
			}
			cmd.Printf("%s added with cid : %s", filePath, n.Cid().String())
			return nil
		},
	}
}