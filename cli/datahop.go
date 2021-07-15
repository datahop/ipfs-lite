package main

import (
	"fmt"
	"os"

	"github.com/datahop/ipfs-lite/cli/cmd"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "datahop",
	Short: "This is datahop cli client",
	Long:  `Add Long Description`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("datahop")
	},
}

func init() {
	rootCmd.AddCommand(cmd.DaemonCmd)
}
func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
