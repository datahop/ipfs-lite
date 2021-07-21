package cmd

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/datahop/ipfs-lite/internal/config"
	"github.com/ipfs/go-datastore"
	"github.com/spf13/cobra"
)

var InfoCmd *cobra.Command

type Info struct {
	IsDaemonRunning bool
	Config          config.Config
	Peers           []string
	CRDTStatus      *bloom.BloomFilter
	DiskUsage       int64
	Addresses       []string
}

func InitInfoCmd(comm *common.Common) {
	InfoCmd = &cobra.Command{
		Use:   "info",
		Short: "Get datahop node information",
		Long:  `Add Long Description`,
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
			// config
			// id
			// address
			// public key
			// port
			cfg, err := comm.Repo.Config()
			if err != nil {
				log.Error("Unable to get config ", err)
				return err
			}
			info.Config = *cfg
			info.Config.Identity.PrivKey = ""

			// crdt status
			info.CRDTStatus = comm.Repo.State()
			b, err := json.Marshal(info)
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
			cmd.Printf(string(b))
			return nil
		},
	}
}
