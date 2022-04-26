package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/datahop/ipfs-lite/internal/ipfs"

	"github.com/datahop/ipfs-lite/pkg"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
)

var log = logging.Logger("cmd")

// InitDaemonCmd creates the daemon command
func InitDaemonCmd(comm *pkg.Common) *cobra.Command {
	command := &cobra.Command{
		Use:   "daemon",
		Short: "Start datahop daemon",
		Long: `
This command is used to start the Datahop Daemon.
		`,
		Run: func(cmd *cobra.Command, args []string) {
			sk, _ := cmd.Flags().GetString("secret")
			autoDownload, _ := cmd.Flags().GetBool("dl")
			bootstrap, _ := cmd.Flags().GetBool("bootstrap")
			done, err := comm.Start(sk, autoDownload)
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			if bootstrap {
				comm.Node.Bootstrap(ipfs.DefaultBootstrapPeers())
			}
			cfg, err := comm.Repo.Config()
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			datahopCli := `
       __              __                __                                              __  __ 
      /  |            /  |              /  |                                            /  |/  |
  ____$$ |  ______   _$$ |_     ______  $$ |____    ______    ______            _______ $$ |$$/ 
 /    $$ | /      \ / $$   |   /      \ $$      \  /      \  /      \  ______  /       |$$ |/  |
/$$$$$$$ | $$$$$$  |$$$$$$/    $$$$$$  |$$$$$$$  |/$$$$$$  |/$$$$$$  |/      |/$$$$$$$/ $$ |$$ |
$$ |  $$ | /    $$ |  $$ | __  /    $$ |$$ |  $$ |$$ |  $$ |$$ |  $$ |$$$$$$/ $$ |      $$ |$$ |
$$ \__$$ |/$$$$$$$ |  $$ |/  |/$$$$$$$ |$$ |  $$ |$$ \__$$ |$$ |__$$ |        $$ \_____ $$ |$$ |
$$    $$ |$$    $$ |  $$  $$/ $$    $$ |$$ |  $$ |$$    $$/ $$    $$/         $$       |$$ |$$ |
 $$$$$$$/  $$$$$$$/    $$$$/   $$$$$$$/ $$/   $$/  $$$$$$/  $$$$$$$/           $$$$$$$/ $$/ $$/ 
                                                            $$ |                                
                                                            $$ |                                
                                                            $$/                                 
`
			fmt.Println(datahopCli)
			fmt.Println("Datahop daemon running on port", cfg.SwarmPort)
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt)
			for {
				select {
				case <-sigChan:
					fmt.Println()
					comm.Stop()
					return
				case <-done:
					return
				}
			}
		},
	}
	command.Flags().String("secret", "",
		"Group secret key")
	command.Flags().Bool("dl", true,
		"Auto download content")
	command.Flags().Bool("bootstrap", true,
		"Bootstrap with datahop bootstrap nodes")
	return command
}
