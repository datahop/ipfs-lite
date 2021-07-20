package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/asabya/go-ipc-uds"
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
	cmd.InitStatCmd(comm)
	cmd.InitStopCmd(comm)
	rootCmd.AddCommand(cmd.DaemonCmd)
	rootCmd.AddCommand(cmd.StatCmd)
	rootCmd.AddCommand(cmd.StopCmd)

	socketPath := filepath.Join("/tmp", SockPath)
	if len(os.Args) > 1 {
		if os.Args[1] != "daemon" && uds.IsIPCListening(socketPath) {
			opts := uds.Options{
				Size:       512,
				SocketPath: filepath.Join("/tmp", SockPath),
			}
			r, w, c, err := uds.Dialer(opts)
			if err != nil {
				log.Error(err)
				goto Execute
			}
			defer c()
			err = w(os.Args[1])
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			v, err := r()
			if err != nil {
				log.Error(err)
				os.Exit(1)

			}
			fmt.Println(v)
			return
		}
		if os.Args[1] == "daemon" {
			if uds.IsIPCListening(socketPath) {
				fmt.Println("Datahop daemon is already running")
				return
			}
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
			in, err := uds.Listener(context.Background(), opts)
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			go func() {
				for {
					client := <-in
					go func() {
						for {
							ip, err := client.Read()
							if err != nil {
								break
							}
							commandStr := string(ip)
							log.Debug("run command :", commandStr)
							var (
								childCmd *cobra.Command
							)
							if rootCmd.TraverseChildren {
								childCmd, _, err = rootCmd.Traverse([]string{commandStr})
							} else {
								childCmd, _, err = rootCmd.Find([]string{commandStr})
							}
							outBuf := new(bytes.Buffer)
							childCmd.SetOut(outBuf)
							if childCmd.Args != nil {
								if err := childCmd.Args(childCmd, []string{commandStr}); err != nil {
									return
								}
							}
							if childCmd.PreRunE != nil {
								if err := childCmd.PreRunE(childCmd, []string{commandStr}); err != nil {
									return
								}
							} else if childCmd.PreRun != nil {
								childCmd.PreRun(childCmd, []string{commandStr})
							}

							if childCmd.RunE != nil {
								if err := childCmd.RunE(childCmd, []string{commandStr}); err != nil {
									return
								}
							} else if childCmd.Run != nil {
								childCmd.Run(childCmd, []string{commandStr})
							}
							out := outBuf.Next(outBuf.Len())
							outBuf.Reset()
							err = client.Write(out)
							if err != nil {
								log.Error("Write error", err)
								client.Close()
								break
							}
						}
					}()
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
