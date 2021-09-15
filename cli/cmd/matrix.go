package cmd

import (
	"github.com/datahop/ipfs-lite/cli/common"
	"github.com/datahop/ipfs-lite/cli/out"
	"github.com/spf13/cobra"
)

func InitMatrixCmd(comm *common.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "matrix",
		Short: "Get node connectivity and content matrix",
		Long: `
"The commend is used to get connectivity and 
content matrix"
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.LitePeer != nil {
				nodeMatrixSnapshot := comm.LitePeer.Repo.Matrix().NodeMatrixSnapshot()
				contentMatrixSnapshot := comm.LitePeer.Repo.Matrix().ContentMatrixSnapshot()
				uptime := comm.LitePeer.Repo.Matrix().GetTotalUptime()
				// output
				matrix := map[string]interface{}{}
				matrix["TotalUptime"] = uptime
				matrix["NodeMatrix"] = nodeMatrixSnapshot
				matrix["ContentMatrix"] = contentMatrixSnapshot
				err := out.Print(cmd, matrix, parseFormat(cmd))
				if err != nil {
					log.Error("Unable to get config ", err)
					return err
				}
			}
			return nil
		},
	}
}
