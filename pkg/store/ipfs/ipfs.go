package ipfs

import (
	"context"
	"io"
	"time"

	"github.com/datahop/ipfs-lite/internal/ipfs"
	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/datahop/ipfs-lite/pkg/store"
	"github.com/ipfs/go-datastore"
)

type IPFSStore struct {
	litepeer *ipfs.Peer
}

func New(litepeer *ipfs.Peer) *IPFSStore {
	return &IPFSStore{litepeer: litepeer}
}

func (I *IPFSStore) Add(ctx context.Context, reader io.Reader, info *store.Info) (string, error) {
	n, err := I.litepeer.AddFile(ctx, reader, nil)
	if err != nil {
		return "", err
	}
	meta := &replication.Metatag{
		Size:        info.Size,
		Type:        info.Type,
		Name:        info.Name,
		Hash:        n.Cid(),
		Timestamp:   time.Now().Unix(),
		Owner:       I.litepeer.Host.ID(),
		Tag:         info.Tag,
		IsEncrypted: info.IsEncrypted,
	}
	err = I.litepeer.Manager.Tag(info.Tag, meta)
	if err != nil {
		return "", err
	}
	return n.Cid().String(), nil
}

func (I *IPFSStore) Get(ctx context.Context, tag string) (io.Reader, *store.Info, error) {
	meta, err := I.litepeer.Manager.FindTag(tag)
	if err != nil {
		return nil, nil, err
	}
	info := &store.Info{
		Tag:         meta.Tag,
		Type:        meta.Type,
		Name:        meta.Name,
		IsEncrypted: meta.IsEncrypted,
		Size:        meta.Size,
	}
	r, err := I.litepeer.GetFile(ctx, meta.Hash)
	if err != nil {
		return nil, nil, err
	}
	return r, info, nil
}

func (I *IPFSStore) Delete(ctx context.Context, tag string) error {
	meta, err := I.litepeer.Manager.FindTag(tag)
	if err != nil {
		return err
	}

	err = I.litepeer.DeleteFile(ctx, meta.Hash)
	if err != nil {
		return err
	}
	err = I.litepeer.Manager.Delete(datastore.NewKey(tag))
	if err != nil {
		return err
	}
	return nil
}
