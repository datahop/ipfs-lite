package replication

import (
	ic "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

type GroupMetadata struct {
	Name           string
	OwnerID        peer.ID
	OwnerPublicKey string
	GroupID        peer.ID
	IsOpen         bool
}

func (m *Manager) CreateGroup(name string, ownerID peer.ID, ownerPrivateKey ic.PrivKey) error {
	return nil
}

func (m *Manager) GroupAddMember(peerId, groupID peer.ID, memberPublicKey ic.PubKey) error {
	return nil
}

func (m *Manager) GroupGetAllGroups(ownerID peer.ID, privateKey ic.PrivKey) ([]GroupMetadata, error) {
	return []GroupMetadata{}, nil
}

func (m *Manager) GroupAddContent(peerId, groupID peer.ID, privateKey ic.PrivKey, meta *ContentMetatag) error {
	return nil
}

func (m *Manager) GroupGetAllContent(ownerID peer.ID, privateKey ic.PrivKey) (map[string]*ContentMetatag, error) {
	indexes := map[string]*ContentMetatag{}
	return indexes, nil
}
