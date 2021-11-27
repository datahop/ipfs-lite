package cmd

import (
	"errors"

	"github.com/datahop/ipfs-lite/cli/out"
	ipfslite "github.com/datahop/ipfs-lite/pkg"
	"github.com/spf13/cobra"
)

// InitIndexCmd creates the index command
func InitIndexCmd(comm *ipfslite.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "index",
		Short: "Index datahop node content",
		Long: `
This command is used to get the index of tag-content

Example:

	To pretty print the index in json format

	$ datahop index -j -p

	{
		"/go1.17.linux-amd64.tar.gz": {
			"Size": 134787877,
			"Type": "application/gzip",
			"Name": "go1.17.linux-amd64.tar.gz",
			"Hash": {
				"/": "bafybeia4ssmbshzjwcuhq6xl3b7pjmfapy6buaaheh75hf7qzjzvs4rogq"
			},
			"Timestamp": 1632207586,
			"Owner": "QmXpiaCz3M7bRz47ZRUP3uq1WUfquqTNrfzi3j24eNXpe5"
		},
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
	}
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.Node == nil || !comm.Node.IsOnline() {
				return errors.New("daemon not running")
			}
			tags, err := comm.Node.ReplManager().Index()
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
