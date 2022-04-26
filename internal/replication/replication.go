package replication

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	crdt "github.com/ipfs/go-ds-crdt"
	format "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log/v2"
	ic "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/plexsysio/taskmanager"
)

var (
	log                = logging.Logger("replication")
	syncMtx            sync.Mutex
	checkMembership    func(string, []byte) bool
	getContentMetadata func(groupID string) ([]*ContentMetatag, error)
)

const (
	groupPrefix       = "/groups"
	groupMemberPrefix = "/group-member"
	groupIndexPrefix  = "/group-index"
)

// ContentMetatag keeps meta information of a content in the crdt store
type ContentMetatag struct {
	Tag         string
	Size        int64
	Type        string
	Name        string
	Hash        cid.Cid
	Timestamp   int64
	Owner       peer.ID
	IsEncrypted bool
	Group       string `json:"omitempty"`
	Links       []*format.Link
}

// Manager handles replication
type Manager struct {
	ctx          context.Context
	cancel       context.CancelFunc
	crdt         *crdt.Datastore
	syncer       Syncer
	pubKeyGetter PubKeyGetter
	repo         repo.Repo

	dlManager *taskmanager.TaskManager
	download  chan ContentMetatag
}

// Syncer gets the file and finds file provider from the network
type Syncer interface {
	Download(context.Context, cid.Cid) error
	FindProviders(context.Context, cid.Cid) []peer.ID
	ConnectIfNotConnectedUsingRelay(context.Context, []peer.ID)
}

// PubKeyGetter
type PubKeyGetter interface {
	PubKey(peer.ID) ic.PubKey
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
	pubKeyGetter PubKeyGetter,
	autoDownload bool,
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
	contentChan := make(chan ContentMetatag)
	crdtOpts := crdt.DefaultOptions()
	crdtOpts.Logger = log
	crdtOpts.RebroadcastInterval = broadcastInterval
	crdtOpts.PutHook = func(k datastore.Key, v []byte) {
		log.Debugf("CRDT Replication :: Added Key: [%s] -> Value: %s\n", k, string(v))
		if strings.HasPrefix(k.String(), groupMemberPrefix) {
			groupID := k.List()[2]
			userID := k.List()[1]
			if h.ID().String() != userID {
				return
			}
			indexes, err := getContentMetadata(groupID)
			if err != nil {
				log.Errorf("AddOrUpdateState failed %s\n", err.Error())
				return
			}
			deltas := []string{}
			for _, v := range indexes {
				deltas = append(deltas, v.Tag)
			}
			syncMtx.Lock()
			sk := r.StateKeeper()
			_, err = sk.AddOrUpdateState(groupID, true, deltas)
			if err != nil {
				log.Errorf("AddOrUpdateState failed %s\n", err.Error())
			}
			syncMtx.Unlock()
		}
		if !strings.HasPrefix(k.String(), groupMemberPrefix) && !strings.HasPrefix(k.String(), groupPrefix) {
			m := &ContentMetatag{}
			err := json.Unmarshal(v, m)
			if err != nil {
				log.Error(err.Error())
				return
			}
			if strings.HasPrefix(k.String(), groupIndexPrefix) {
				groupID := k.List()[1]
				key, err := pubKeyGetter.PubKey(h.ID()).Raw()
				if err != nil {
					return
				}
				memberTag := fmt.Sprintf("%s/%s/%s", groupMemberPrefix, h.ID().String(), groupID)

				isMember := checkMembership(memberTag, key)
				if !isMember {
					return
				}
				syncMtx.Lock()
				sk := r.StateKeeper()
				_, err = sk.AddOrUpdateState(groupID, isMember, []string{m.Tag})
				if err != nil {
					log.Errorf("AddOrUpdateState failed %s\n", err.Error())
				}
				syncMtx.Unlock()
				if !isMember {
					return
				}
			}

			if h.ID() != m.Owner && autoDownload {
				contentChan <- *m
			} else {
				syncMtx.Lock()
				state := r.State().Add([]byte(m.Tag))
				log.Debugf("New State: %d\n", state)
				err := r.SetState()
				if err != nil {
					log.Errorf("SetState failed %s\n", err.Error())
				}
				syncMtx.Unlock()
			}
		}
	}
	crdtOpts.DeleteHook = func(k datastore.Key) {
		log.Debugf("CRDT Replication :: Removed: [%s]\n", k)
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

	checkMembership = func(memberTag string, key []byte) bool {
		memberPublicKey, err := crdtStore.Get(datastore.NewKey(memberTag))
		if err != nil {
			return false
		}
		return base64.StdEncoding.EncodeToString(key) == string(memberPublicKey)
	}

	getContentMetadata = func(groupID string) ([]*ContentMetatag, error) {
		q := query.Query{Prefix: fmt.Sprintf("%s/%s", groupIndexPrefix, groupID)}
		r, err := crdtStore.Query(q)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		indexes := []*ContentMetatag{}
		for j := range r.Next() {
			meta := &ContentMetatag{}
			err = json.Unmarshal(j.Entry.Value, meta)
			if err != nil {
				continue
			}
			indexes = append(indexes, meta)
		}
		return indexes, nil
	}

	return &Manager{
		ctx:          ctx,
		crdt:         crdtStore,
		download:     contentChan,
		syncer:       syncer,
		pubKeyGetter: pubKeyGetter,
		cancel:       cancel,
		repo:         r,
		dlManager:    taskmanager.New(1, 100, time.Second*15, log),
	}, nil
}

// Close the crdt store
func (m *Manager) Close() error {
	if err := m.dlManager.Stop(m.ctx); err != nil {
		return err
	}
	m.cancel()
	return m.crdt.Close()
}

// Tag a given meta info in the store
func (m *Manager) Tag(tag string, meta *ContentMetatag) error {
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
func (m *Manager) FindTag(tag string) (*ContentMetatag, error) {
	b, err := m.Get(datastore.NewKey(tag))
	if err != nil {
		return nil, err
	}
	meta := &ContentMetatag{}
	err = json.Unmarshal(b, meta)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

// Index returns the tag-mata info as key:value
func (m *Manager) Index() (map[string]*ContentMetatag, error) {
	indexes := map[string]*ContentMetatag{}
	r, err := m.crdt.Query(query.Query{})
	if err != nil {
		return indexes, err
	}
	defer r.Close()
	for j := range r.Next() {
		m := &ContentMetatag{}
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
		meta := &ContentMetatag{}
		err = json.Unmarshal(j.Entry.Value, meta)
		if err != nil {
			continue
		}
		cids = append(cids, meta.Hash)
	}
	return cids, nil
}

// DownloadManagerStatus returns taskmanager status for handling downloads
func (m *Manager) DownloadManagerStatus() (res map[string]taskmanager.TaskStatus) {
	return m.dlManager.TaskStatus()
}

// StartUnfinishedDownload starts unfinished downloads once peer comes back
func (m *Manager) StartUnfinishedDownload(pid peer.ID) {
	cm := m.repo.Matrix().ContentMatrix
	for i, v := range cm {
		if v.DownloadFinishedAt == 0 {
			log.Debug("Got unfinished download : ", i, v.ProvidedBy)
			for _, provider := range v.ProvidedBy {
				if provider == pid {
					meta, err := m.FindTag(v.Tag)
					if err != nil {
						continue
					}
					m.download <- *meta
				}
			}
		}
	}
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
			case meta := <-m.download:
				id := meta.Hash
				log.Debugf("got %s\n", id.String())
				go func() {
					mat := m.repo.Matrix()
					mat.NewContent(id.String())
					providers := m.syncer.FindProviders(m.ctx, id)
					for _, provider := range providers {
						mat.ContentAddProvider(id.String(), provider)
					}

					ctx, cancel := context.WithCancel(m.ctx)
					var cb func()
					if meta.Group != "" {
						cb = func() {
							mat.ContentDownloadFinished(id.String())
							syncMtx.Lock()
							sk := m.repo.StateKeeper()
							_, err := sk.AddOrUpdateState(meta.Group, true, []string{meta.Tag})
							if err != nil {
								log.Errorf("AddOrUpdateState failed %s\n", err.Error())
							}
							syncMtx.Unlock()
						}
					} else {
						cb = func() {
							mat.ContentDownloadFinished(id.String())
							syncMtx.Lock()
							state := m.repo.State().Add([]byte(meta.Tag))
							log.Debugf("New State: %d\n", state)
							err := m.repo.SetState()
							if err != nil {
								log.Errorf("SetState failed %s\n", err.Error())
							}
							syncMtx.Unlock()
						}
					}
					// TODO check if we are connected with the providers. if not, try to connect using relay.
					m.syncer.ConnectIfNotConnectedUsingRelay(m.ctx, providers)

					t := newDownloaderTask(ctx, cancel, meta, m.syncer, cb)
					done, err := m.dlManager.Go(t)
					if err != nil {
						log.Errorf("content watcher: unable to start downloader task for %s : %s", meta.Name, err.Error())
						return
					}
					mat.ContentDownloadStarted(meta.Tag, id.String(), meta.Size)
					select {
					case <-ctx.Done():
						return
					case <-done:
						return
					}
				}()
			}
		}
	}()
}

type DownloaderTask struct {
	ctx      context.Context
	cancel   context.CancelFunc
	metatag  ContentMetatag
	syncer   Syncer
	callback func()
}

func newDownloaderTask(ctx context.Context, cancel context.CancelFunc, metatag ContentMetatag, syncer Syncer, cb func()) *DownloaderTask {
	return &DownloaderTask{
		ctx:      ctx,
		cancel:   cancel,
		metatag:  metatag,
		syncer:   syncer,
		callback: cb,
	}
}

func (d *DownloaderTask) Execute(ctx context.Context) error {
	<-time.After(time.Second)
	err := d.syncer.Download(d.ctx, d.metatag.Hash)
	if err != nil {
		log.Errorf("replication sync failed for %s, Err : %s", d.metatag.Hash.String(), err.Error())
		return err
	}
	d.callback()
	return nil
}

func (d *DownloaderTask) Name() string {
	return d.metatag.Name
}
