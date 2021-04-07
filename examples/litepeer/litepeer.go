package main

// This example launches an IPFS-Lite peer and fetches a hello-world
// hash from the IPFS network.repo

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/ipfs/go-cid"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	root := "/tmp" + string(os.PathSeparator) + ipfslite.Root
	conf, err := ipfslite.ConfigInit(2048, "0")
	if err != nil {
		return
	}
	err = ipfslite.Init(root, conf)
	if err != nil {
		return
	}

	r, err := ipfslite.Open(root)
	if err != nil {
		return
	}

	lite, err := ipfslite.New(ctx, r)
	if err != nil {
		panic(err)
	}

	lite.Bootstrap(ipfslite.DefaultBootstrapPeers())

	c, _ := cid.Decode("QmWATWQ7fVPP2EFGu71UkfnqhYXDYH566qy47CnJDgvs8u")
	rsc, err := lite.GetFile(ctx, c)
	if err != nil {
		panic(err)
	}
	defer rsc.Close()
	content, err := ioutil.ReadAll(rsc)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(content))
}
