package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/datahop/ipfs-lite/cli/out"
	ipfslite "github.com/datahop/ipfs-lite/pkg"
	"github.com/datahop/ipfs-lite/pkg/store"
	"github.com/h2non/filetype"
	"github.com/spf13/cobra"
)

// InitAddCmd creates the add command
func InitAddCmd(comm *ipfslite.Common) *cobra.Command {
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
			if comm.Node == nil || !comm.Node.IsOnline() {
				return errors.New("daemon not running")
			}
			filePath := args[0]
			fileinfo, err := os.Lstat(filePath)
			if err != nil {
				log.Errorf("Failed executing add command Err:%s", err.Error())
				return err
			}
			f, err := os.Open(filePath)
			if err != nil {
				log.Errorf("Failed executing add command Err:%s", err.Error())
				return err
			}
			defer f.Close()
			shouldEncrypt := true
			passphrase, _ := cmd.Flags().GetString("passphrase")
			if passphrase == "" {
				shouldEncrypt = false
			}

			head := make([]byte, 261)
			_, err = f.Read(head)
			if err != nil {
				return err
			}
			kind, _ := filetype.Match(head)

			_, err = f.Seek(0, io.SeekStart)
			if err != nil {
				return err
			}

			tag, _ := cmd.Flags().GetString("tag")
			if tag == "" {
				tag = filepath.Base(f.Name())
			}
			info := &store.Info{
				Tag:         tag,
				Type:        kind.MIME.Value,
				Name:        filepath.Base(f.Name()),
				IsEncrypted: shouldEncrypt,
				Size:        fileinfo.Size(),
			}
			var id string
			if shouldEncrypt {
				buf := bytes.NewBuffer(nil)
				_, err = io.Copy(buf, f)
				if err != nil {
					log.Errorf("Failed io.Copy file encryption:%s", err.Error())
					return err
				}
				byteContent, err := comm.Encryption.EncryptContent(buf.Bytes(), passphrase)
				if err != nil {
					log.Errorf("Encryption failed :%s", err.Error())
					return err
				}

				id, err = comm.Node.Add(comm.Context, bytes.NewReader(byteContent), info)
				if err != nil {
					return err
				}
			} else {
				id, err = comm.Node.Add(comm.Context, f, info)
				if err != nil {
					return err
				}
			}
			_ = id
			// TODO: show the id
			err = out.Print(cmd, info, parseFormat(cmd))
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

// InitAddDirCmd creates the add command
func InitAddDirCmd(comm *ipfslite.Common) *cobra.Command {
	addCommand := &cobra.Command{
		Use:   "addDir",
		Short: "Add directory into datahop network",
		Long: `
This command is used to add a directory in the 
datahop network addressable by a given tag.
		`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.Node == nil || !comm.Node.IsOnline() {
				return errors.New("daemon not running")
			}
			filePath := args[0]
			fileinfo, err := os.Lstat(filePath)
			if err != nil {
				log.Errorf("Failed executing add command Err:%s", err.Error())
				return err
			}

			tag, _ := cmd.Flags().GetString("tag")
			if tag == "" {
				tag = filepath.Base(fileinfo.Name())
			}
			info := &store.Info{
				Tag:         tag,
				Type:        "directory",
				Name:        filepath.Base(fileinfo.Name()),
				IsEncrypted: false,
				Size:        fileinfo.Size(),
			}
			id, err := comm.Node.AddDir(comm.Context, filePath, info)
			if err != nil {
				return err
			}
			_ = id
			// TODO: show the id
			err = out.Print(cmd, info, parseFormat(cmd))
			if err != nil {
				return err
			}
			return nil
		},
	}
	addCommand.Flags().StringP("tag", "t", "",
		"Tag for the content")
	return addCommand
}
