package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/datahop/ipfs-lite/cli/out"
	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/datahop/ipfs-lite/internal/security"
	"github.com/h2non/filetype"
	format "github.com/ipfs/go-ipld-format"
	"github.com/spf13/cobra"
)

// InitAddCmd creates the add command
func InitAddCmd(comm *common.Common) *cobra.Command {
	addCommand := &cobra.Command{
		Use:   "add",
		Short: "Add content into datahop network",
		Long: `
This command is used to add a file/content in the 
datahop network addressable by a given tag.

Example:

	// To tag the content with filename after adding

	$ datahop add '/home/sabyasachi/Downloads/go1.17.linux-amd64.tar.gz' -p -j
	
	// The file will be added the in network with the filename in the format below
	"/go1.17.linux-amd64.tar.gz": {
		"Size": 134787877,
		"Type": "application/gzip",
		"Name": "go1.17.linux-amd64.tar.gz",
		"Hash": {
			"/": "bafybeia4ssmbshzjwcuhq6xl3b7pjmfapy6buaaheh75hf7qzjzvs4rogq"
		},
		"Timestamp": 1632207586,
		"Owner": "QmXpiaCz3M7bRz47ZRUP3uq1WUfquqTNrfzi3j24eNXpe5"
	}

	$ datahop add '/home/sabyasachi/Downloads/go1.17.linux-amd64.tar.gz' -p -j -t golang_latest
	
	The file will be added the in network with provided tag in the format below

	"/golang_latest": {
		"Size": 134787877,
		"Type": "application/gzip",
		"Name": "go1.17.linux-amd64.tar.gz",
		"Hash": {
			"/": "bafybeia4ssmbshzjwcuhq6xl3b7pjmfapy6buaaheh75hf7qzjzvs4rogq"
		},
		"Timestamp": 1632207767,
		"Owner": "QmXpiaCz3M7bRz47ZRUP3uq1WUfquqTNrfzi3j24eNXpe5"
	},
		`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.LitePeer == nil || !comm.LitePeer.IsOnline() {
				return errors.New("daemon not running")
			}
			filePath := args[0]
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
			shouldEncrypt := true
			passphrase, _ := cmd.Flags().GetString("passphrase")
			if passphrase == "" {
				shouldEncrypt = false
			}
			var n format.Node
			if shouldEncrypt {
				buf := bytes.NewBuffer(nil)
				_, err = io.Copy(buf, f)
				if err != nil {
					log.Errorf("Failed io.Copy file encryption:%s", err.Error())
					return err
				}
				byteContent, err := security.Encrypt(buf.Bytes(), passphrase)
				if err != nil {
					log.Errorf("Encryption failed :%s", err.Error())
					return err
				}
				n, err = comm.LitePeer.AddFile(comm.Context, bytes.NewReader(byteContent), nil)
				if err != nil {
					return err
				}
			} else {
				n, err = comm.LitePeer.AddFile(comm.Context, f, nil)
				if err != nil {
					return err
				}
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
			tag, _ := cmd.Flags().GetString("tag")
			if tag == "" {
				tag = filepath.Base(f.Name())
			}
			meta := &replication.Metatag{
				Size:        fileinfo.Size(),
				Type:        kind.MIME.Value,
				Name:        filepath.Base(f.Name()),
				Hash:        n.Cid(),
				Timestamp:   time.Now().Unix(),
				Owner:       comm.LitePeer.Host.ID(),
				Tag:         tag,
				IsEncrypted: shouldEncrypt,
			}
			err = comm.LitePeer.Manager.Tag(tag, meta)
			if err != nil {
				return err
			}
			err = out.Print(cmd, meta, parseFormat(cmd))
			if err != nil {
				return err
			}
			return nil
		},
	}
	addCommand.Flags().StringP("tag", "t", "",
		"Tag for the file/content")
	addCommand.Flags().String("passphrase", "",
		"Passphrase to encrypt content")
	return addCommand
}
