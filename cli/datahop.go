package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	uds "github.com/asabya/go-ipc-uds"
	"github.com/datahop/ipfs-lite/cli/cmd"
	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/datahop/ipfs-lite/pkg"
	logger "github.com/ipfs/go-log/v2"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	argSeparator = "$^~@@*"
)

var (
	rootCmd = &cobra.Command{
		Use:   "datahop",
		Short: "This is datahop cli client",
		Long: `
The Datahop CLI client gives access to datahop
network through a CLI Interface.
		`,
	}
	sockPath = "datahop.sock"
	log      = logging.Logger("cmd")
)

func init() {
	_ = logger.SetLogLevel("uds", "Debug")
	_ = logger.SetLogLevel("cmd", "Debug")
}

func main() {
	usr, err := user.Current()
	if err != nil {
		fmt.Printf("Failed getting user home directory. Is USER set?\n")
		os.Exit(1)
	}
	absoluteRoot := usr.HomeDir + string(os.PathSeparator) + repo.Root

	err = pkg.Init(absoluteRoot, "0")
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	comm := &pkg.Common{}
	tmp := os.TempDir()
	socketPath := filepath.Join(tmp, sockPath)
	if !uds.IsIPCListening(socketPath) {
		comm, err = pkg.New(context.Background(), absoluteRoot, "0", nil)
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}

	rootCmd.PersistentFlags().BoolP("json", "j", false, "json output")
	rootCmd.PersistentFlags().BoolP("pretty", "p", false, "pretty json output")

	var allCommands []*cobra.Command
	allCommands = append(
		allCommands,
		cmd.InitDaemonCmd(comm),
		cmd.InitInfoCmd(comm),
		cmd.InitStopCmd(comm),
		cmd.InitAddCmd(comm),
		cmd.InitAddDirCmd(comm),
		cmd.InitIndexCmd(comm),
		cmd.InitRemoveCmd(comm),
		cmd.InitVersionCmd(comm),
		cmd.InitMatrixCmd(comm),
		cmd.InitializeDocCommand(comm),
		cmd.InitGetCmd(comm),
		cmd.InitCompletionCmd(comm),
	)

	for _, i := range allCommands {
		rootCmd.AddCommand(i)
	}
	// check help flag
	for _, v := range os.Args {
		if v == "-h" || v == "--help" {
			log.Debug("Executing help command")
			if err := rootCmd.Execute(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}

	if len(os.Args) > 1 {
		if os.Args[1] != "daemon" && uds.IsIPCListening(socketPath) {
			opts := uds.Options{
				SocketPath: filepath.Join(tmp, sockPath),
			}
			r, w, c, err := uds.Dialer(opts)
			if err != nil {
				log.Error(err)
				goto Execute
			}
			defer pkg.CheckError(log, c)
			err = w(strings.Join(os.Args[1:], argSeparator))
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
			_, err := os.Stat(filepath.Join(tmp, sockPath))
			if !os.IsNotExist(err) {
				err := os.Remove(filepath.Join(tmp, sockPath))
				if err != nil {
					log.Error(err)
					os.Exit(1)
				}
			}
			opts := uds.Options{
				SocketPath: filepath.Join(tmp, sockPath),
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
							if len(ip) == 0 {
								break
							}
							commandStr := string(ip)
							log.Debug("run command :", commandStr)
							var (
								childCmd *cobra.Command
								flags    []string
							)
							command := strings.Split(commandStr, argSeparator)
							if rootCmd.TraverseChildren {
								childCmd, flags, err = rootCmd.Traverse(command)
							} else {
								childCmd, flags, err = rootCmd.Find(command)
							}
							if err != nil {
								err = client.Write([]byte(err.Error()))
								if err != nil {
									log.Error("Write error", err)
									client.Close()
								}
								break
							}
							childCmd.Flags().VisitAll(func(f *pflag.Flag) {
								err := f.Value.Set(f.DefValue)
								if err != nil {
									log.Error("Unable to set flags ", childCmd.Name(), f.Name, err.Error())
								}
							})
							if err := childCmd.Flags().Parse(flags); err != nil {
								log.Error("Unable to parse flags ", err.Error())
								err = client.Write([]byte(err.Error()))
								if err != nil {
									log.Error("Write error", err)
									client.Close()
								}
								break
							}
							outBuf := new(bytes.Buffer)
							childCmd.SetOut(outBuf)
							if childCmd.Args != nil {
								if err := childCmd.Args(childCmd, flags); err != nil {
									err = client.Write([]byte(err.Error()))
									if err != nil {
										log.Error("Write error", err)
										client.Close()
									}
									break
								}
							}
							if childCmd.PreRunE != nil {
								if err := childCmd.PreRunE(childCmd, flags); err != nil {
									err = client.Write([]byte(err.Error()))
									if err != nil {
										log.Error("Write error", err)
										client.Close()
									}
									break
								}
							} else if childCmd.PreRun != nil {
								childCmd.PreRun(childCmd, command)
							}

							if childCmd.RunE != nil {
								if err := childCmd.RunE(childCmd, flags); err != nil {
									err = client.Write([]byte(err.Error()))
									if err != nil {
										log.Error("Write error", err)
										client.Close()
									}
									break
								}
							} else if childCmd.Run != nil {
								childCmd.Run(childCmd, flags)
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
