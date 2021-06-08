package replication

import (
	"context"
	"time"

	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	crdt "github.com/ipfs/go-ds-crdt"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var (
	log = logging.Logger("replication")
)

type Manager struct {
	ctx         context.Context
	crdt        *crdt.Datastore
	contentChan chan cid.Cid
}

func New(
	ctx context.Context,
	repo repo.Repo,
	h host.Host,
	dagSyncer crdt.DAGSyncer,
	st datastore.Batching,
	prefix string,
	topic string,
	broadcastInterval time.Duration,
) (*Manager, error) {
	log.Debug("NEW MANAGER")
	psub, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}
	// TODO Add RegisterTopicValidator
	pubsubBC, err := crdt.NewPubSubBroadcaster(ctx, psub, topic)
	if err != nil {
		return nil, err
	}
	contentChan := make(chan cid.Cid)
	crdtOpts := crdt.DefaultOptions()
	crdtOpts.Logger = log
	crdtOpts.RebroadcastInterval = broadcastInterval
	crdtOpts.PutHook = func(k datastore.Key, v []byte) {
		log.Debugf("Added: [%s] -> %s\n", k, string(v))
		id, err := cid.Cast(v)
		if err != nil {
			return
		}
		log.Debugf("Added: [%s] -> %s\n", k, id.String())
		contentChan <- id
		state := repo.State()
		state++
		log.Debugf("New State: %d\n", state)
		err = repo.SetState(state)
		if err != nil {
			log.Errorf("SetState failed %s\n", err.Error())
		}
	}
	crdtOpts.DeleteHook = func(k datastore.Key) {
		// TODO reduce count
		log.Debugf("Removed: [%s]\n", k)
	}
	crdtStore, err := crdt.New(st, datastore.NewKey(prefix), dagSyncer, pubsubBC, crdtOpts)
	if err != nil {
		return nil, err
	}

	return &Manager{
		ctx:         ctx,
		crdt:        crdtStore,
		contentChan: contentChan,
	}, nil
}

func (m *Manager) Close() error {
	close(m.contentChan)
	return m.crdt.Close()
}

func (m *Manager) Put(key datastore.Key, v []byte) error {
	return m.crdt.Put(key, v)
}

func (m *Manager) Get(key datastore.Key) ([]byte, error) {
	return m.crdt.Get(key)
}

func (m *Manager) Has(key datastore.Key) (bool, error) {
	return m.crdt.Has(key)
}

func (m *Manager) StartContentWatcher() {
	go func() {
		for {
			select {
			case <-m.ctx.Done():
				return
			case id := <-m.contentChan:
				log.Debugf("got %s\n", id.String())
				// do something
			}
		}
	}()
}
