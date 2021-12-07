module github.com/datahop/ipfs-lite

go 1.15

require (
	github.com/asabya/go-ipc-uds v0.1.1
	github.com/bits-and-blooms/bloom/v3 v3.1.0
	github.com/facebookgo/atomicfile v0.0.0-20151019160806-2de1f203e7d5
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2
	github.com/grandcat/zeroconf v1.0.0
	github.com/h2non/filetype v1.1.3
	github.com/ipfs/go-bitswap v0.5.1
	github.com/ipfs/go-blockservice v0.2.1
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-datastore v0.5.1
	github.com/ipfs/go-ds-crdt v0.2.1
	github.com/ipfs/go-ds-leveldb v0.5.0
	github.com/ipfs/go-fetcher v1.6.1
	github.com/ipfs/go-fs-lock v0.0.7
	github.com/ipfs/go-ipfs-blockstore v1.1.0
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/ipfs/go-ipfs-files v0.0.9
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-ipns v0.1.2
	github.com/ipfs/go-log/v2 v2.3.0
	github.com/ipfs/go-merkledag v0.5.1
	github.com/ipfs/go-path v0.2.1
	github.com/ipfs/go-unixfs v0.3.1
	github.com/ipfs/go-unixfsnode v1.1.3 // indirect
	github.com/ipfs/go-verifcid v0.0.1
	github.com/ipfs/interface-go-ipfs-core v0.5.2
	github.com/ipld/go-codec-dagpb v1.3.0
	github.com/ipld/go-ipld-prime v0.14.1
	github.com/libp2p/go-libp2p v0.16.0
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.11.0
	github.com/libp2p/go-libp2p-kad-dht v0.15.0
	github.com/libp2p/go-libp2p-pubsub v0.6.0
	github.com/libp2p/go-libp2p-record v0.1.3
	github.com/libp2p/go-libp2p-swarm v0.8.0
	github.com/libp2p/go-libp2p-tls v0.3.1
	github.com/libp2p/go-tcp-transport v0.4.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.4.1
	github.com/multiformats/go-multihash v0.1.0
	github.com/plexsysio/taskmanager v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/warpfork/go-testmark v0.9.0 // indirect
	github.com/whyrusleeping/cbor-gen v0.0.0-20210219115102-f37d292932f2 // indirect
	go.uber.org/goleak v1.1.11 // indirect
	go.uber.org/zap v1.19.1 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/sys v0.0.0-20211025112917-711f33c9992c // indirect
	google.golang.org/protobuf v1.27.1
)

replace github.com/plexsysio/taskmanager => github.com/asabya/taskmanager v0.1.0
