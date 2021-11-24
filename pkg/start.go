package pkg

import (
	"context"

	ipfslite "github.com/datahop/ipfs-lite/internal/ipfs"
	"github.com/datahop/ipfs-lite/pkg/store/ipfs"
)

func (comm *Common) Start() (<-chan struct{}, error) {
	// TODO: take swarm.key location as parameter
	// TODO: check if repo exists
	ctx2, cancel := context.WithCancel(comm.Context)
	litePeer, err := ipfslite.New(ctx2, cancel, comm.Repo, nil)
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
