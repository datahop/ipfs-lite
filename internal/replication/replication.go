package replication

import (
	"context"
	"encoding/json"
	"time"

	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	crdt "github.com/ipfs/go-ds-crdt"
	logging "github.com/ipfs/go-log/v2"
	ufsio "github.com/ipfs/go-unixfs/io"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var (
	log = logging.Logger("replication")
)

// Metatag keeps meta information of a content in the crdt store
type Metatag struct {
	Size      int64
	Type      string
	Name      string
	Hash      cid.Cid
	Timestamp int64
	Owner     peer.ID
}

// Manager handles replication
type Manager struct {
	ctx         context.Context
	cancel      context.CancelFunc
	crdt        *crdt.Datastore
	contentChan chan cid.Cid
	syncer      Syncer
	repo        repo.Repo
}

// Syncer gets the file and finds file provider from the network
type Syncer interface {
	GetFile(context.Context, cid.Cid) (ufsio.ReadSeekCloser, error)
	FindProviders(context.Context, cid.Cid) []peer.ID
}

// New creates a new replication manager
func New(
	ctx context.Context,
	cancel context.CancelFunc,
	r repo.Repo,
	h host.Host,
	dagSyncer crdt.DAGSyncer,
	st datastore.Batching,
	prefix string,
	topic string,
	broadcastInterval time.Duration,
	syncer Syncer,
) (*Manager, error) {
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
		m := &Metatag{}
		err := json.Unmarshal(v, m)
		if err != nil {
			log.Error(err.Error())
			return
		}
		contentChan <- m.Hash
		state := r.State().Add([]byte(k.Name()))
		log.Debugf("New State: %d\n", state)
		err = r.SetState()
		if err != nil {
			log.Errorf("SetState failed %s\n", err.Error())
		}
		r.Matrix().ContentDownloadStarted(m.Hash.String(), m.Size)
	}
	crdtOpts.DeleteHook = func(k datastore.Key) {
		log.Debugf("Removed: [%s]\n", k)
		state := r.State().Add([]byte("removed " + k.Name()))
		log.Debugf("New State: %d\n", state)
		err = r.SetState()
		if err != nil {
			log.Errorf("SetState failed %s\n", err.Error())
		}
	}
	crdtStore, err := crdt.New(st, datastore.NewKey(prefix), dagSyncer, pubsubBC, crdtOpts)
	if err != nil {
		return nil, err
	}

	return &Manager{
		ctx:         ctx,
		crdt:        crdtStore,
		contentChan: contentChan,
		syncer:      syncer,
		cancel:      cancel,
		repo:        r,
	}, nil
}

// Close the crdt store
func (m *Manager) Close() error {
	m.cancel()
	return m.crdt.Close()
}

// Tag a given meta info in the store
func (m *Manager) Tag(tag string, meta *Metatag) error {
	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	err = m.Put(datastore.NewKey(tag), bMeta)
	if err != nil {
		return err
	}
	return nil
}

// FindTag gets the meta info of a given tag from the store
func (m *Manager) FindTag(tag string) (*Metatag, error) {
	b, err := m.Get(datastore.NewKey(tag))
	if err != nil {
		return nil, err
	}
	meta := &Metatag{}
	err = json.Unmarshal(b, meta)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

// Index returns the tag-mata info as key:value
func (m *Manager) Index() (map[string]*Metatag, error) {
	indexes := map[string]*Metatag{}
	r, err := m.crdt.Query(query.Query{})
	if err != nil {
		return indexes, err
	}
	defer r.Close()
	for j := range r.Next() {
		m := &Metatag{}
		err := json.Unmarshal(j.Entry.Value, m)
		if err != nil {
			continue
		}
		indexes[j.Key] = m
	}
	return indexes, nil
}

// GetAllTags returns all tags
func (m *Manager) GetAllTags() ([]string, error) {
	tags := []string{}
	r, err := m.crdt.Query(query.Query{})
	if err != nil {
		return tags, err
	}
	defer r.Close()
	for j := range r.Next() {
		tags = append(tags, j.Key)
	}
	return tags, nil
}

// GetAllCids returns all the cids in the crdt store
func (m *Manager) GetAllCids() ([]cid.Cid, error) {
	cids := []cid.Cid{}
	r, err := m.crdt.Query(query.Query{})
	if err != nil {
		return cids, err
	}
	defer r.Close()
	for j := range r.Next() {
		meta := &Metatag{}
		err = json.Unmarshal(j.Entry.Value, meta)
		if err != nil {
			continue
		}
		cids = append(cids, meta.Hash)
	}
	return cids, nil
}

// Put stores the object `value` named by `key`.
func (m *Manager) Put(key datastore.Key, v []byte) error {
	return m.crdt.Put(key, v)
}

// Delete removes the value for given `key`.
func (m *Manager) Delete(key datastore.Key) error {
	return m.crdt.Delete(key)
}

// Get retrieves the object `value` named by `key`.
// Get will return ErrNotFound if the key is not mapped to a value.
func (m *Manager) Get(key datastore.Key) ([]byte, error) {
	return m.crdt.Get(key)
}

// Has returns whether the `key` is mapped to a `value`.
// In some contexts, it may be much cheaper only to check for existence of
// a value, rather than retrieving the value itself. (e.g. HTTP HEAD).
// The default implementation is found in `GetBackedHas`.
func (m *Manager) Has(key datastore.Key) (bool, error) {
	return m.crdt.Has(key)
}

// StartContentWatcher watches on incoming contents and gets content in datastore
func (m *Manager) StartContentWatcher() {
	go func() {
		for {
			select {
			case <-m.ctx.Done():
				return
			case id := <-m.contentChan:
				log.Debugf("got %s\n", id.String())
				go func() {
					providers := m.syncer.FindProviders(m.ctx, id)
					for _, provider := range providers {
						m.repo.Matrix().ContentAddProvider(id.String(), provider)
					}
					_, err := m.syncer.GetFile(m.ctx, id)
					if err != nil {
						log.Errorf("replication sync failed for %s, Err : %s", id.String(), err.Error())
						return
					}
					m.repo.Matrix().ContentDownloadFinished(id.String())
				}()
			}
		}
	}()
}
