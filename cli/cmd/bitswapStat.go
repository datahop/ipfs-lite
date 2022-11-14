package cmd

import (
	"github.com/datahop/ipfs-lite/cli/out"
	ipfslite "github.com/datahop/ipfs-lite/pkg"
	"github.com/spf13/cobra"
)

// InitBitswapStatCmd creates the bitswapStat command
func InitBitswapStatCmd(comm *ipfslite.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "bs",
		Short: "Get datahop node bitswap stat",
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.Node != nil {
				stat, err := comm.Node.BitswapStat()
				if err != nil {
					log.Error("Unable to get bitswap stat ", err)
					return err
				}
				err = out.Print(cmd, stat, parseFormat(cmd))
				if err != nil {
					log.Error("Unable to print bitswap stat ", err)
					return err
				}
				return nil
			}
			return out.Print(cmd, "Node not running", parseFormat(cmd))
		},
	}
}
