package cmd

import (
	"fmt"
	"os"
	"os/signal"

	pkg2 "github.com/datahop/ipfs-lite/pkg"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
)

var log = logging.Logger("cmd")

// InitDaemonCmd creates the daemon command
func InitDaemonCmd(comm *pkg2.Common) *cobra.Command {
	return &cobra.Command{
		Use:   "daemon",
		Short: "Start datahop daemon",
		Long: `
This command is used to start the Datahop Daemon.
		`,
		Run: func(cmd *cobra.Command, args []string) {
			err := pkg2.Start(comm)
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
			var sigChan chan os.Signal
			sigChan = make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt)
			for {
				select {
				case <-sigChan:
					fmt.Println()
					comm.Cancel()
					return
				case <-comm.Context.Done():
					return
				}
			}
		},
	}
}
