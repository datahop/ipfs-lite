package ipfslite

import (
	ipfslite "github.com/datahop/ipfs-lite/internal/ipfs"
)

func Start(comm *Common) error {
	// TODO: take swarm.key location as parameter
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
