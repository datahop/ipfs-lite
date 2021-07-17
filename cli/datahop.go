package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/asabya/uds"
	"github.com/datahop/ipfs-lite/cli/cmd"
	"github.com/datahop/ipfs-lite/cli/common"
	logger "github.com/ipfs/go-log/v2"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "datahop",
	Short: "This is datahop cli client",
	Long:  `Add Long Description`,
}
var SockPath = "uds.sock"
var log = logging.Logger("cmd")

func init() {
	logger.SetLogLevel("uds", "Debug")
	logger.SetLogLevel("cmd", "Debug")

}
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	comm := &common.Common{
		IsDaemon: false,
		Context:  ctx,
		Cancel:   cancel,
	}
	cmd.InitDaemonCmd(comm)
	cmd.InitStopCmd(comm)
	rootCmd.AddCommand(cmd.DaemonCmd)
	rootCmd.AddCommand(cmd.StopCmd)
	if len(os.Args) > 1 {
		if os.Args[1] != "daemon" {
			opts := uds.Options{
				Size:       512,
				SocketPath: filepath.Join("/tmp", SockPath),
			}
			in, out, err := uds.Dialer(opts)
			if err != nil {
				log.Error(err)
				goto Execute
			}
			in <- os.Args[1]
			fmt.Println(<-out)
			return
		}
		if os.Args[1] == "daemon" {
			_, err := os.Stat(filepath.Join("/tmp", SockPath))
			if !os.IsNotExist(err) {
				err := os.Remove(filepath.Join("/tmp", SockPath))
				if err != nil {
					log.Error(err)
					os.Exit(1)
				}
			}
			opts := uds.Options{
				Size:       512,
				SocketPath: filepath.Join("/tmp", SockPath),
			}
			out, ext, err := uds.Listener(context.Background(), opts)
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			go func() {
				for {
					data := <-out
					var (
						childCmd *cobra.Command
					)
					if rootCmd.TraverseChildren {
						childCmd, _, err = rootCmd.Traverse([]string{data})
					} else {
						childCmd, _, err = rootCmd.Find([]string{data})
					}
					outBuf := new(bytes.Buffer)
					childCmd.SetOut(outBuf)
					if childCmd.Args != nil {
						if err := childCmd.Args(childCmd, []string{data}); err != nil {
							return
						}
					}
					if childCmd.PreRunE != nil {
						if err := childCmd.PreRunE(childCmd, []string{data}); err != nil {
							return
						}
					} else if childCmd.PreRun != nil {
						childCmd.PreRun(childCmd, []string{data})
					}

					if childCmd.RunE != nil {
						if err := childCmd.RunE(childCmd, []string{data}); err != nil {
							return
						}
					} else if childCmd.Run != nil {
						childCmd.Run(childCmd, []string{data})
					}
					out := outBuf.Next(outBuf.Len())
					outBuf.Reset()
					if err != nil {
						log.Error(err)
						os.Exit(1)
					}
					ext <- string(out)
				}
			}()
		}
	}
Execute:
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
