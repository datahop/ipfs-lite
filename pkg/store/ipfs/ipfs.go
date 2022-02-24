package ipfs

import (
	"context"
	"io"
	"time"

	"github.com/datahop/ipfs-lite/internal/ipfs"
	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/datahop/ipfs-lite/pkg/store"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type IPFSNode struct {
	peer *ipfs.Peer
}

func New(litepeer *ipfs.Peer) *IPFSNode {
	return &IPFSNode{peer: litepeer}
}

func (I *IPFSNode) Add(ctx context.Context, reader io.Reader, info *store.Info) (string, error) {
	n, err := I.peer.AddFile(ctx, reader, nil)
	if err != nil {
		return "", err
	}
	meta := &replication.ContentMetatag{
		Size:        info.Size,
		Type:        info.Type,
		Name:        info.Name,
		Hash:        n.Cid(),
		Timestamp:   time.Now().Unix(),
		Owner:       I.peer.Host.ID(),
		Tag:         info.Tag,
		IsEncrypted: info.IsEncrypted,
	}
	err = I.peer.Manager.Tag(info.Tag, meta)
	if err != nil {
		return "", err
	}
	return n.Cid().String(), nil
}

func (I *IPFSNode) AddDir(ctx context.Context, dir string, info *store.Info) (string, error) {
	n, err := I.peer.AddDir(ctx, dir, nil)
	if err != nil {
		return "", err
	}
	meta := &replication.ContentMetatag{
		Size:        info.Size,
		Type:        "directory",
		Name:        info.Name,
		Hash:        n.Cid(),
		Timestamp:   time.Now().Unix(),
		Owner:       I.peer.Host.ID(),
		Tag:         info.Tag,
		IsEncrypted: info.IsEncrypted,
		Links:       n.Links(),
	}
	err = I.peer.Manager.Tag(info.Tag, meta)
	if err != nil {
		return "", err
	}
	return n.Cid().String(), nil
}

func (I *IPFSNode) Get(ctx context.Context, tag string) (io.ReadSeekCloser, *store.Info, error) {
	meta, err := I.peer.Manager.FindTag(tag)
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
	r, err := I.peer.GetFile(ctx, meta.Hash)
	if err != nil {
		return nil, nil, err
	}
	return r, info, nil
}

func (I *IPFSNode) Delete(ctx context.Context, tag string) error {
	meta, err := I.peer.Manager.FindTag(tag)
	if err != nil {
		return err
	}

	err = I.peer.DeleteFile(ctx, meta.Hash)
	if err != nil {
		return err
	}
	err = I.peer.Manager.Delete(datastore.NewKey(tag))
	if err != nil {
		return err
	}
	return nil
}

func (I *IPFSNode) NetworkNotifiee(n network.Notifiee) {
	I.peer.Host.Network().Notify(n)
}

func (I *IPFSNode) Bootstrap(peers []peer.AddrInfo) {
	I.peer.Bootstrap(peers)
}

func (I *IPFSNode) NewFloodsubWithProtocols(ps []protocol.ID, opts ...pubsub.Option) (*pubsub.PubSub, error) {
	return pubsub.NewFloodsubWithProtocols(I.peer.Ctx, I.peer.Host,
		ps, opts...)
}

func (I *IPFSNode) Peers() []string {
	return I.peer.Peers()
}

func (I *IPFSNode) Connect(info peer.AddrInfo) error {
	if I.IsPeerConnected(info.ID.String()) {
		return nil
	}
	return I.peer.Connect(I.peer.Ctx, info)
}

func (I *IPFSNode) AddrInfo() *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    I.peer.Host.ID(),
		Addrs: I.peer.Host.Addrs(),
	}
}

func (I *IPFSNode) IsOnline() bool {
	return I.peer.IsOnline()
}

func (I *IPFSNode) ReplManager() *replication.Manager {
	return I.peer.Manager
}

func (I *IPFSNode) IsPeerConnected(id string) bool {
	conn := I.peer.Host.Network().Connectedness(peer.ID(id))
	return conn == network.Connected
}

func (I *IPFSNode) GetPrivKey() crypto.PrivKey {
	return I.peer.Host.Peerstore().PrivKey(I.peer.Host.ID())
}

func (I *IPFSNode) GetPubKey(id peer.ID) crypto.PubKey {
	return I.peer.Host.Peerstore().PubKey(id)
}

func (I *IPFSNode) GroupAdd(ctx context.Context, reader io.Reader, info *store.Info, groupIDString string) (string, error) {
	groupID, err := peer.Decode(groupIDString)
	if err != nil {
		return "", err
	}
	n, err := I.peer.AddFile(ctx, reader, nil)
	if err != nil {
		return "", err
	}
	meta := &replication.ContentMetatag{
		Size:        info.Size,
		Type:        info.Type,
		Name:        info.Name,
		Hash:        n.Cid(),
		Timestamp:   time.Now().Unix(),
		Owner:       I.peer.Host.ID(),
		Tag:         info.Tag,
		IsEncrypted: info.IsEncrypted,
		Group:       groupIDString,
	}

	err = I.peer.Manager.GroupAddContent(I.peer.Host.ID(), groupID, I.GetPrivKey(), meta)
	if err != nil {
		return "", err
	}
	return n.Cid().String(), nil
}

func (I *IPFSNode) GroupAddDir(ctx context.Context, dir string, info *store.Info, groupIDString string) (string, error) {
	groupID, err := peer.Decode(groupIDString)
	if err != nil {
		return "", err
	}
	n, err := I.peer.AddDir(ctx, dir, nil)
	if err != nil {
		return "", err
	}
	meta := &replication.ContentMetatag{
		Size:        info.Size,
		Type:        "directory",
		Name:        info.Name,
		Hash:        n.Cid(),
		Timestamp:   time.Now().Unix(),
		Owner:       I.peer.Host.ID(),
		Tag:         info.Tag,
		IsEncrypted: info.IsEncrypted,
		Group:       groupIDString,
		Links:       n.Links(),
	}
	err = I.peer.Manager.GroupAddContent(I.peer.Host.ID(), groupID, I.GetPrivKey(), meta)
	if err != nil {
		return "", err
	}
	return n.Cid().String(), nil
}
