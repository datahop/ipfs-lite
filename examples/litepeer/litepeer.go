package main

// This example launches an IPFS-Lite peer and fetches a hello-world
// hash from the IPFS network.repo

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	ipfslite "github.com/datahop/ipfs-lite/internal/ipfs"

	"github.com/datahop/ipfs-lite/internal/repo"
)

func main() {
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

	lite, err := ipfslite.New(ctx, cancel, r, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(lite.Host.ID())
	addrs := []string{}
	for _, v := range lite.Host.Addrs() {
		if !strings.HasPrefix(v.String(), "127") {
			addrs = append(addrs, v.String()+"/p2p/"+lite.Host.ID().String())
		}
	}
	fmt.Println(addrs)
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
}
