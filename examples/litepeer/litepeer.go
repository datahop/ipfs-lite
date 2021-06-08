package main

// This example launches an IPFS-Lite peer and fetches a hello-world
// hash from the IPFS network.repo

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	root := "/tmp" + string(os.PathSeparator) + repo.Root
	err := repo.Init(root, "0")
	if err != nil {
		return
	}

	r, err := repo.Open(root)
	if err != nil {
		return
	}

	lite, err := ipfslite.New(ctx, cancel, r)
	if err != nil {
		panic(err)
	}
	fmt.Println(lite.Host.ID(), lite.Repo.State())
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
	myvalue := "myValue51299858"
	key := datastore.NewKey("myke15a12351y2")
	err = lite.CrdtStore.Put(key, []byte(myvalue))
	if err != nil {
		panic(err)
	}

	<-time.After(time.Minute * 5)
}
