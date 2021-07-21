package common

import (
	"context"

	ipfslite "github.com/datahop/ipfs-lite"
	"github.com/datahop/ipfs-lite/internal/repo"
)

type Common struct {
	Root     string
	Repo     repo.Repo
	LitePeer *ipfslite.Peer
	Context  context.Context
	Cancel   context.CancelFunc
}
