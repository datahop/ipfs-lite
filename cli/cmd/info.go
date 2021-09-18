package cmd

import (
	"strings"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/datahop/ipfs-lite/cli/out"
	"github.com/datahop/ipfs-lite/internal/config"
	"github.com/ipfs/go-datastore"
	"github.com/spf13/cobra"
)

type Info struct {
	IsDaemonRunning bool
	Config          config.Config
	Peers           []string
	CRDTStatus      *bloom.BloomFilter
	DiskUsage       int64
	Addresses       []string
}

func InitInfoCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Get datahop node information",
		Long: `
"The commend is used to get the local node information"
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := &Info{}
			if comm.LitePeer != nil {
				// is daemon running
				info.IsDaemonRunning = comm.LitePeer.IsOnline()
				// peers
				info.Peers = comm.LitePeer.Peers()
				// addresses
				addrs := []string{}
				if comm.LitePeer != nil {
					for _, v := range comm.LitePeer.Host.Addrs() {
						if !strings.HasPrefix(v.String(), "127") {
							addrs = append(addrs, v.String()+"/p2p/"+comm.LitePeer.Host.ID().String())
						}
					}
					info.Addresses = addrs
				}
				// disk usage
				du, err := datastore.DiskUsage(comm.LitePeer.Repo.Datastore())
				if err != nil {
					log.Error("Unable to get datastore usage ", err)
					return err
				}
				info.DiskUsage = int64(du)
			} else {
				info.IsDaemonRunning = false
			}

			cfg, err := comm.Repo.Config()
			if err != nil {
				log.Error("Unable to get config ", err)
				return err
			}
			info.Config = *cfg
			info.Config.Identity.PrivKey = ""

			// crdt status
			info.CRDTStatus = comm.Repo.State()

			// output
			err = out.Print(cmd, info, parseFormat(cmd))
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
