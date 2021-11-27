package pkg

import (
	"github.com/datahop/ipfs-lite/internal/repo"
	logging "github.com/ipfs/go-log/v2"
)

var (
	log = logging.Logger("pkg")
)

func Init(root, port string) error {
	err := repo.Init(root, port)
	if err != nil {
		log.Errorf("pkg Init: %s", err.Error())
		return err
	}
	return nil
}
