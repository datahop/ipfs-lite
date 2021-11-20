package ipfslite

import (
	"github.com/datahop/ipfs-lite/internal/repo"
	logging "github.com/ipfs/go-log/v2"
)

var (
	log = logging.Logger("pkg")
)

func Init(comm *Common) error {
	err := repo.Init(comm.Root, comm.Port)
	if err != nil {
		log.Errorf("pkg Init: %s", err.Error())
		return err
	}
	return nil
}
