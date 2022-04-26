package pkg

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"

	ipfslite "github.com/datahop/ipfs-lite/internal/ipfs"
	"github.com/datahop/ipfs-lite/pkg/store/ipfs"
)

const skBase = `/key/swarm/psk/1.0.0/
/base16/`

func (comm *Common) Start(key string, autoDownload bool) (<-chan struct{}, error) {
	var swarmKey []byte
	if key != "" {
		byteKey := md5.Sum([]byte(key))
		hexKey := hex.EncodeToString(byteKey[:])
		keyString := fmt.Sprintf("%s\n%s%s", skBase, hexKey, hexKey)
		swarmKey = []byte(keyString)
	}

	// TODO: check if repo exists
	ctx2, cancel := context.WithCancel(comm.Context)
	litePeer, err := ipfslite.New(ctx2, cancel, comm.Repo, swarmKey, ipfslite.WithAutoDownload(autoDownload))
	if err != nil {
		log.Errorf("pkg Start: %s", err.Error())
		return nil, err
	}
	comm.Node = ipfs.New(litePeer)
	return ctx2.Done(), nil
}

func (comm *Common) Stop() {
	comm.Repo.Matrix().Flush()
	comm.Repo.Close()
	comm.cancel()
}
