package cmd

import (
	"strings"

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
	CRDTStatus      string
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
			// is daemon running
			if comm.LitePeer != nil {
				info.IsDaemonRunning = comm.LitePeer.IsOnline()
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
			// peers
			info.Peers = comm.LitePeer.Peers()

			// crdt status
			st, err := comm.Repo.State().MarshalJSON()
			if err != nil {
				log.Error("Unable to get crdt state ", err)
				return err
			}
			info.CRDTStatus = string(st)
			// disk usage
			du, err := datastore.DiskUsage(comm.LitePeer.Repo.Datastore())
			if err != nil {
				log.Error("Unable to get datastore usage ", err)
				return err
			}
			info.DiskUsage = int64(du)

			addrs := []string{}
			if comm.LitePeer != nil {
				for _, v := range comm.LitePeer.Host.Addrs() {
					if !strings.HasPrefix(v.String(), "127") {
						addrs = append(addrs, v.String()+"/p2p/"+comm.LitePeer.Host.ID().String())
					}
				}
				info.Addresses = addrs
			}
			cmd.Printf("%+v\n", *info)
			return nil
		},
	}
}
