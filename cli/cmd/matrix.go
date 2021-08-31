package cmd

import (
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/datahop/ipfs-lite/cli/out"
	"github.com/spf13/cobra"
)

func InitMatrixCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "matrix",
		Short: "Get node connectivity and content matirx",
		Long:  `Add Long Description`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.LitePeer != nil {
				matrixSnapshot := comm.LitePeer.Repo.Matrix().SnapshotStruct()
				// output
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

				err := out.Print(cmd, matrixSnapshot, f)
				if err != nil {
					log.Error("Unable to get config ", err)
					return err
				}
			}
			return nil
		},
	}
}
