package cmd

import (
	"strings"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/datahop/ipfs-lite/cli/out"
	"github.com/datahop/ipfs-lite/internal/config"
	ipfslite "github.com/datahop/ipfs-lite/pkg"
	"github.com/ipfs/go-datastore"
	"github.com/spf13/cobra"
)

type info struct {
	IsDaemonRunning bool
	Config          config.Config
	Peers           []string
	CRDTStatus      *bloom.BloomFilter
	DiskUsage       int64
	Addresses       []string
}

// InitInfoCmd creates the info command
func InitInfoCmd(comm *ipfslite.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Get datahop node information",
		Long: `
This command is used to get the local node information

Example:

	To pretty print the node info in json format

	$ datahop info -j -p

	{
		"IsDaemonRunning": true,
		"Config": {
			"Identity": {
				"PeerID": "QmXpiaCz3M7bRz47ZRUP3uq1WUfquqTNrfzi3j24eNXpe5"
			},
			"Addresses": {
				"Swarm": [
					"/ip4/0.0.0.0/tcp/4501"
				]
			},
			"Bootstrap": [],
			"SwarmPort": "4501"
		},
		"Peers": [],
		"CRDTStatus": {
			"m": 2000,
			"k": 5,
			"b": "AAAAAAAAB9AAAAAAAAAAIAAABAAAAAEAAAAAAAAABAAAAAABAAIAAAAAAAAAAAAQAAAAAIAAAAAAAAAAAAAAAAAQAAAAAAAAAAgAAAAAAAAAEAAAAAAAAAAAAAAAAAEgAAAAAAAAAAABAAAAAAAIAAAAAAAAAAAAAAAAAAAAEBAEAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAAAAgAEAAAAAAAAAAAAEAAgAAAAAAAAAAAAAAABDAAAAAAAAAAAAAQAhAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		},
		"DiskUsage": 135071757,
		"Addresses": [
			"/ip4/192.168.29.24/tcp/4501/p2p/QmXpiaCz3M7bRz47ZRUP3uq1WUfquqTNrfzi3j24eNXpe5",
			"/ip4/127.0.0.1/tcp/4501/p2p/QmXpiaCz3M7bRz47ZRUP3uq1WUfquqTNrfzi3j24eNXpe5"
		]
	}
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			inf := &info{}
			if comm.Node != nil {
				// is daemon running
				inf.IsDaemonRunning = comm.Node.IsOnline()
				// peers
				inf.Peers = comm.Node.Peers()
				// addresses
				addrs := []string{}
				if comm.Node != nil {
					pr := comm.Node.AddrInfo()
					for _, v := range pr.Addrs {
						if !strings.HasPrefix(v.String(), "/ip4/127") {
							addrs = append(addrs, v.String()+"/p2p/"+pr.ID.String())
						}
					}
					inf.Addresses = addrs
				}
				// disk usage
				du, err := datastore.DiskUsage(comm.Repo.Datastore())
				if err != nil {
					log.Error("Unable to get datastore usage ", err)
					return err
				}
				inf.DiskUsage = int64(du)
			} else {
				inf.IsDaemonRunning = false
			}

			cfg, err := comm.Repo.Config()
			if err != nil {
				log.Error("Unable to get config ", err)
				return err
			}
			inf.Config = *cfg
			inf.Config.Identity.PrivKey = ""

			// crdt status
			inf.CRDTStatus = comm.Repo.State()

			// output
			err = out.Print(cmd, inf, parseFormat(cmd))
			if err != nil {
				log.Error("Unable to get config ", err)
				return err
			}
			return nil
		},
	}
}

func parseFormat(cmd *cobra.Command) out.Format {
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
	return f
}
