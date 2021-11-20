package ipfslite

import (
	ipfslite "github.com/datahop/ipfs-lite/internal/ipfs"
	"github.com/datahop/ipfs-lite/internal/repo"
)

func Start(comm *Common) error {
	// TODO: take swarm.key location as parameter
	// TODO: check if repo exists
	r, err := repo.Open(comm.Root)
	if err != nil {
		log.Error(err)
		return err
	}
	comm.Repo = r
	litePeer, err := ipfslite.New(comm.Context, comm.Cancel, comm.Repo, nil)
	if err != nil {
		log.Errorf("pkg Start: %s", err.Error())
		return err
	}
	comm.LitePeer = litePeer
	networkNotifier := NewNotifier(litePeer.Repo.Matrix())
	litePeer.Host.Network().Notify(networkNotifier)
	return nil
}
