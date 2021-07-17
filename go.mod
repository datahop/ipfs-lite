module github.com/datahop/ipfs-lite

go 1.15

require (
	github.com/asabya/uds v0.0.0-20200603183238-f227537d11c5
	github.com/bits-and-blooms/bloom/v3 v3.0.1
	github.com/facebookgo/atomicfile v0.0.0-20151019160806-2de1f203e7d5
	github.com/golang/protobuf v1.5.2
	github.com/ipfs/go-bitswap v0.3.4
	github.com/ipfs/go-blockservice v0.1.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-crdt v0.1.20
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-fs-lock v0.0.6
	github.com/ipfs/go-ipfs-blockstore v1.0.3
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/ipfs/go-ipfs-files v0.0.8 // indirect
	github.com/ipfs/go-ipfs-provider v0.5.1
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-ipns v0.1.0
	github.com/ipfs/go-log/v2 v2.3.0
	github.com/ipfs/go-merkledag v0.3.2
	github.com/ipfs/go-unixfs v0.2.6
	github.com/libp2p/go-libp2p v0.14.2
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/libp2p/go-libp2p-kad-dht v0.12.1
	github.com/libp2p/go-libp2p-pubsub v0.4.1
	github.com/libp2p/go-libp2p-record v0.1.3
	github.com/libp2p/go-libp2p-swarm v0.5.0
	github.com/libp2p/go-libp2p-tls v0.1.3
	github.com/libp2p/go-tcp-transport v0.2.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.2
	github.com/multiformats/go-multihash v0.0.15
	github.com/spf13/cobra v0.0.5
	golang.org/x/tools v0.1.0 // indirect
	google.golang.org/protobuf v1.26.0
)

replace github.com/asabya/uds => ../../asabya/uds
