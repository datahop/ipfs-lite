package cmd

import (
	"github.com/datahop/ipfs-lite/cli/out"
	ipfslite "github.com/datahop/ipfs-lite/pkg"
	"github.com/spf13/cobra"
)

// InitMatrixCmd creates the matrix command
func InitMatrixCmd(comm *ipfslite.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "matrix",
		Short: "Get node connectivity and content matrix",
		Long: `
This command is used to get connectivity and 
content matrix

Example:

	To pretty print the node matrix in json format

	$ datahop matrix -j -p

	{
		"ContentMatrix": {
			"bafybeicqmfjhbvuy75aluslgpjx57q7acpxbhibscwy5vv7hka42as5i5i": {
				"Size": 26,
				"AvgSpeed": 0.000024795532,
				"DownloadStartedAt": 1631086702,
				"DownloadFinishedAt": 1631086703,
				"ProvidedBy": [
					"QmXzT3KAv27w7MMdnHQ8bDqPxP2wrqoNNUcb9U14aC9wWJ"
				]
			},
		},
		"NodeMatrix": {
			"QmXzT3KAv27w7MMdnHQ8bDqPxP2wrqoNNUcb9U14aC9wWJ": {
				"ConnectionAlive": false,
				"ConnectionSuccessCount": 1,
				"ConnectionFailureCount": 0,
				"LastSuccessfulConnectionDuration": 69,
				"BLEDiscoveredAt": 0,
				"WifiConnectedAt": 0,
				"RSSI": 0,
				"Speed": 0,
				"Frequency": 0,
				"IPFSConnectedAt": 0,
				"DiscoveryDelays": [],
				"ConnectionHistory": null
			},
			"QmcWEJqQD3bPMT5Mr7ijdwVCmVjUh5Z7CysiTQPgr2VZBC": {
				"ConnectionAlive": true,
				"ConnectionSuccessCount": 1,
				"ConnectionFailureCount": 0,
				"LastSuccessfulConnectionDuration": 0,
				"BLEDiscoveredAt": 0,
				"WifiConnectedAt": 0,
				"RSSI": 0,
				"Speed": 0,
				"Frequency": 0,
				"IPFSConnectedAt": 0,
				"DiscoveryDelays": [],
				"ConnectionHistory": null
			}
		},
		"TotalUptime": 40860
	}
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if comm.Node != nil {
				mKeeper := comm.Repo.Matrix()
				nodeMatrixSnapshot := mKeeper.NodeMatrixSnapshot()
				contentMatrixSnapshot := mKeeper.ContentMatrixSnapshot()
				uptime := mKeeper.GetTotalUptime()
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
