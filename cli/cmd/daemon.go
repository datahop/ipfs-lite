package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/datahop/ipfs-lite/internal/repo"

	"github.com/spf13/cobra"
)

var DaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start datahop daemon",
	Long:  `Add Long Description`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		root := filepath.Join(home, repo.Root)
		err = repo.Init(root, "0")
		if err != nil {
			return
		}

		r, err := repo.Open(root)
		if err != nil {
			return
		}
		defer r.Close()
		_, err = ipfslite.New(ctx, cancel, r)
		if err != nil {
			panic(err)
		}

		cfg, err := r.Config()
		if err != nil {
			panic(err)
		}
		fmt.Println("Datahop daemon running on port", cfg.SwarmPort)
		endWaiter := sync.WaitGroup{}
		endWaiter.Add(1)
		var sigChan chan os.Signal
		sigChan = make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		go func() {
			<-sigChan
			fmt.Println()
			endWaiter.Done()
		}()
		endWaiter.Wait()
	},
}
