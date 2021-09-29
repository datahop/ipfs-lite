package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/spf13/cobra"
)

// InitGetCmd creates the get command
func InitGetCmd(comm *common.Common) *cobra.Command {
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
			meta, err := comm.LitePeer.Manager.FindTag(tag)
			if err != nil {
				return err
			}
			output := destination + string(os.PathSeparator) + meta.Name
			extension := filepath.Ext(output)
			// Rename if file already exist
			count := 0
			tmpout := output
			filename := meta.Name
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
			ctx, _ := context.WithCancel(comm.Context)
			r, err := comm.LitePeer.GetFile(ctx, meta.Hash)
			if err != nil {
				log.Errorf("Unable to get file reader :%s", err.Error())
				return err
			}

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
			return nil
		},
	}
	command.Flags().StringP("location", "l", "", "File destination")
	return command
}
