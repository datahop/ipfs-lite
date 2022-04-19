package replication

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	ic "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	mh "github.com/multiformats/go-multihash"
)

type GroupMetadata struct {
	Name           string
	OwnerID        peer.ID
	OwnerPublicKey []byte
	GroupID        peer.ID
	IsOpen         bool
}

func (m *Manager) CreateGroup(name string, ownerID peer.ID, ownerPrivateKey ic.PrivKey) (*GroupMetadata, error) {
	gID, err := createGroupId(name)
	if err != nil {
		return nil, err
	}
	key, err := ownerPrivateKey.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	gMeta := &GroupMetadata{
		Name:           name,
		OwnerID:        ownerID,
		OwnerPublicKey: key,
		GroupID:        gID,
		IsOpen:         false,
	}
	// In GroupMetadata is created using the "/groups/{groupID}" namespace
	err = m.tagGroup(fmt.Sprintf("%s/%s", groupPrefix, gID.String()), gMeta)
	if err != nil {
		return nil, err
	}
	//Also "/group-member/{peerID}/{groupID}" is created with public key as value
	err = m.storePublicKey(fmt.Sprintf("%s/%s/%s", groupMemberPrefix, ownerID.String(), gID.String()), ownerPrivateKey.GetPublic())
	if err != nil {
		return nil, err
	}

	return gMeta, nil
}

func (m *Manager) CreateOpenGroup(name string, ownerID peer.ID, ownerPrivateKey ic.PrivKey) (*GroupMetadata, error) {
	gID, err := createGroupId(name)
	if err != nil {
		return nil, err
	}
	key, err := ownerPrivateKey.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	gMeta := &GroupMetadata{
		Name:           name,
		OwnerID:        ownerID,
		OwnerPublicKey: key,
		GroupID:        gID,
		IsOpen:         true,
	}

	// In GroupMetadata is created using the "/groups/{groupID}" namespace
	err = m.tagGroup(fmt.Sprintf("%s/%s", groupPrefix, gID.String()), gMeta)
	if err != nil {
		return nil, err
	}

	//Also "/group-member/{peerID}/{groupID}" is created with public key as value
	err = m.storePublicKey(fmt.Sprintf("%s/%s/%s", groupMemberPrefix, ownerID.String(), gID.String()), ownerPrivateKey.GetPublic())
	if err != nil {
		return nil, err
	}

	sk := m.repo.StateKeeper()
	_, err = sk.AddOrUpdateState(gID.String(), true, nil)
	if err != nil {
		return nil, err
	}
	return gMeta, nil
}

func (m *Manager) GroupAddMember(memberPeerId, newMemberPeerId, groupID peer.ID, memberPrivateKey ic.PrivKey, newMemberPublicKey ic.PubKey) error {
	// Check if the group is open
	groupTag := fmt.Sprintf("%s/%s", groupPrefix, groupID.String())
	groupMetaBytes, err := m.crdt.Get(datastore.NewKey(groupTag))
	if err != nil {
		return err
	}
	gm := &GroupMetadata{}
	err = json.Unmarshal(groupMetaBytes, gm)
	if err != nil {
		return err
	}

	// if not open check if member is owner
	if !gm.IsOpen {
		if gm.OwnerID != memberPeerId {
			return fmt.Errorf("user does not have permission to add member in this group")
		}
	}

	// Check If the current user is a member of the group
	key, err := memberPrivateKey.GetPublic().Raw()
	if err != nil {
		return err
	}
	memberTag := fmt.Sprintf("%s/%s/%s", groupMemberPrefix, memberPeerId.String(), groupID.String())
	memberPublicKey, err := m.crdt.Get(datastore.NewKey(memberTag))
	if err != nil {
		return err
	}

	if base64.StdEncoding.EncodeToString(key) != string(memberPublicKey) {
		return fmt.Errorf("user is not a member of this group")
	}

	//Add newMemberPublicKey "/group-member/{peerID}/{groupID}" is created with public key as value
	err = m.storePublicKey(fmt.Sprintf("%s/%s/%s", groupMemberPrefix, newMemberPeerId.String(), groupID.String()), newMemberPublicKey)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) GroupGetInfo(memberPeerId, groupID peer.ID, memberPrivateKey ic.PrivKey) (*GroupMetadata, error) {
	// Check if the group is open
	groupTag := fmt.Sprintf("%s/%s", groupPrefix, groupID.String())
	groupMetaBytes, err := m.crdt.Get(datastore.NewKey(groupTag))
	if err != nil {
		return nil, err
	}
	gm := &GroupMetadata{}
	err = json.Unmarshal(groupMetaBytes, gm)
	if err != nil {
		return nil, err
	}
	// if not open check if member is owner
	if !gm.IsOpen {
		if gm.OwnerID != memberPeerId {
			return nil, fmt.Errorf("user does not have permission to get group info")
		}
	}

	// Check If the current user is a member of the group
	key, err := memberPrivateKey.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	memberTag := fmt.Sprintf("%s/%s/%s", groupMemberPrefix, memberPeerId.String(), groupID.String())
	memberPublicKey, err := m.crdt.Get(datastore.NewKey(memberTag))
	if err != nil {
		return nil, err
	}
	if base64.StdEncoding.EncodeToString(key) != string(memberPublicKey) {
		return nil, fmt.Errorf("user is not a member of this group")
	}

	return gm, nil
}

func (m *Manager) GroupGetAllGroups(ownerID peer.ID, ownerPrivateKey ic.PrivKey) ([]GroupMetadata, error) {
	groups := []GroupMetadata{}
	key, err := ownerPrivateKey.GetPublic().Raw()
	if err != nil {
		return groups, err
	}
	prefix := fmt.Sprintf("%s/%s", groupMemberPrefix, ownerID.String())
	q := query.Query{Prefix: prefix}
	r, err := m.crdt.Query(q)
	if err != nil {
		return groups, err
	}
	defer r.Close()
	for j := range r.Next() {
		if base64.StdEncoding.EncodeToString(key) == string(j.Value) {
			groupIDString := strings.Split(j.Key, "/")[3]
			groupTag := fmt.Sprintf("%s/%s", groupPrefix, groupIDString)
			v, err := m.crdt.Get(datastore.NewKey(groupTag))
			if err != nil {
				continue
			}
			gm := &GroupMetadata{}
			err = json.Unmarshal(v, gm)
			if err != nil {
				continue
			}
			groups = append(groups, *gm)
		}
	}
	return groups, nil
}

func (m *Manager) GroupAddContent(peerId, groupID peer.ID, privateKey ic.PrivKey, meta *ContentMetatag) error {
	// Check if the group is open
	groupTag := fmt.Sprintf("%s/%s", groupPrefix, groupID.String())
	groupMetaBytes, err := m.crdt.Get(datastore.NewKey(groupTag))
	if err != nil {
		return err
	}
	gm := &GroupMetadata{}
	err = json.Unmarshal(groupMetaBytes, gm)
	if err != nil {
		return err
	}

	// if not open check if member is owner
	if !gm.IsOpen {
		if gm.OwnerID != peerId {
			return fmt.Errorf("user does not have permission to add content in this group")
		}
	}

	// Check If the current user is a member of the group
	key, err := privateKey.GetPublic().Raw()
	if err != nil {
		return err
	}
	memberTag := fmt.Sprintf("%s/%s/%s", groupMemberPrefix, peerId.String(), groupID.String())
	memberPublicKey, err := m.crdt.Get(datastore.NewKey(memberTag))
	if err != nil {
		return err
	}
	if base64.StdEncoding.EncodeToString(key) != string(memberPublicKey) {
		return fmt.Errorf("user is not a member of this group")
	}
	meta.Group = groupID.String()
	err = m.Tag(fmt.Sprintf("%s/%s/%s", groupIndexPrefix, groupID.String(), meta.Tag), meta)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) GroupGetAllContent(peerId, groupID peer.ID, privateKey ic.PrivKey) ([]*ContentMetatag, error) {
	// Check If the current user is a member of the group
	key, err := privateKey.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	memberTag := fmt.Sprintf("%s/%s/%s", groupMemberPrefix, peerId.String(), groupID.String())
	memberPublicKey, err := m.crdt.Get(datastore.NewKey(memberTag))
	if err != nil {
		return nil, err
	}
	if base64.StdEncoding.EncodeToString(key) != string(memberPublicKey) {
		return nil, fmt.Errorf("user is not a member of this group")
	}

	q := query.Query{Prefix: fmt.Sprintf("%s/%s", groupIndexPrefix, groupID.String())}
	r, err := m.crdt.Query(q)
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

func createGroupId(name string) (peer.ID, error) {
	unique := fmt.Sprintf("%s+%d", name, time.Now().Unix())
	h, err := mh.Sum([]byte(unique), mh.SHA2_256, -1)
	if err != nil {
		return "", err
	}
	return peer.ID(h), nil
}

func (m *Manager) tagGroup(tag string, meta *GroupMetadata) error {
	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return m.Put(datastore.NewKey(tag), bMeta)
}

func (m *Manager) storePublicKey(tag string, publicKey ic.PubKey) error {
	b, err := publicKey.Raw()
	if err != nil {
		return err
	}
	return m.Put(datastore.NewKey(tag), []byte(base64.StdEncoding.EncodeToString(b)))
}
