package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/datahop/ipfs-lite/cli/out"
	ipfslite "github.com/datahop/ipfs-lite/pkg"
	"github.com/spf13/cobra"
)

// InitGetCmd creates the get command
func InitGetCmd(comm *ipfslite.Common) *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: "Get content by tag",
		Long: `
This command is used to get file/content from the
datahop network by a simple tag

Example:

	// To save at default "download" location

	$ datahop get /test.txt

	// To save at a given location

	$ datahop get /test.txt --location /home/sabyasachi/Documents
	$ datahop get /test.txt -l /home/sabyasachi/Documents
		`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// get command will have an --l or -location flag to save
			// the file at a specific location , else it will stream
			// the content into user default Downloads folder
			tag := args[0]
			destination, _ := cmd.Flags().GetString("location")
			if destination == "" {
				usr, err := user.Current()
				if err != nil {
					log.Errorf("Unable to get user home directory :%s", err.Error())
					return err
				}
				destination = usr.HomeDir + string(os.PathSeparator) + "Downloads"
			}
			r, info, err := comm.Node.Get(comm.Context, tag)
			if err != nil {
				return err
			}
			defer r.Close()

			output := destination + string(os.PathSeparator) + info.Name
			extension := filepath.Ext(output)
			// Rename if file already exist
			count := 0
			tmpout := output
			var filename string
			for {
				_, err := os.Stat(tmpout)
				if err == nil {
					count++
					if extension != "" {
						tmpout = fmt.Sprintf("%s_%d%s", strings.TrimSuffix(output, extension), count, extension)
					} else {
						tmpout = fmt.Sprintf("%s_%d", output, count)
					}
				} else {
					filename = filepath.Base(tmpout)
					break
				}
			}

			if info.IsEncrypted {
				passphrase, _ := cmd.Flags().GetString("passphrase")
				if passphrase == "" {
					log.Error("passphrase is empty")
					return errors.New("passphrase is empty")
				}
				buf := bytes.NewBuffer(nil)
				_, err = io.Copy(buf, r)
				if err != nil {
					log.Errorf("Failed io.Copy file encryption:%s", err.Error())
					return err
				}
				byteContent, err := comm.Encryption.DecryptContent(buf.Bytes(), passphrase)
				if err != nil {
					log.Errorf("decryption failed :%s", err.Error())
					return err
				}
				output = destination + string(os.PathSeparator) + filename
				file, err := os.Create(output)
				if err != nil {
					return err
				}
				defer file.Close()
				_, err = file.Write(byteContent)
				if err != nil {
					return err
				}
				if err != nil {
					log.Errorf("Unable to save file :%s", err.Error())
					return err
				}
			} else {
				output = destination + string(os.PathSeparator) + filename
				file, err := os.Create(output)
				if err != nil {
					return err
				}
				defer file.Close()
				_, err = io.Copy(file, r)
				if err != nil {
					log.Errorf("Unable to save file :%s", err.Error())
					return err
				}
			}
			err = out.Print(cmd, info, parseFormat(cmd))
			if err != nil {
				return err
			}
			return nil
		},
	}
	command.Flags().StringP("location", "l", "", "File destination")
	command.Flags().String("passphrase", "",
		"Passphrase to decrypt content")
	return command
}
