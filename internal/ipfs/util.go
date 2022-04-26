package ipfs

import (
	"context"
	"fmt"
	"strings"

	relay "github.com/libp2p/go-libp2p-circuit"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-ipns"
	dag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-verifcid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dualdht "github.com/libp2p/go-libp2p-kad-dht/dual"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/multiformats/go-multiaddr"
)

var (
	defaultBootstrapAddresses = []string{
		"/ip4/3.7.249.218/tcp/4501/p2p/QmVi5g82WvrDd8dTi1LvhWPhmNKGkC9R1rzth4nicTz6Wo",
	}

	relayAddr = "/ip4/3.7.249.218/tcp/4501/p2p/QmVi5g82WvrDd8dTi1LvhWPhmNKGkC9R1rzth4nicTz6Wo"
)

// DefaultBootstrapPeers returns the default datahop bootstrap peers (for use
// with NewLibp2pHost.
func DefaultBootstrapPeers() []peer.AddrInfo {
	maddrs := make([]multiaddr.Multiaddr, len(defaultBootstrapAddresses))
	for i, addr := range defaultBootstrapAddresses {
		maddrs[i], _ = multiaddr.NewMultiaddr(addr)
	}
	defaults, _ := peer.AddrInfosFromP2pAddrs(maddrs...)
	return defaults
}

// SetupLibp2p returns a routed host and DHT instances that can be used to
// easily create an ipfslite Peer. You may consider to use Peer.Bootstrap()
// or Peer.Connect() after creating the IPFS-Lite Peer to connect to other peers.
//
// Additional libp2p options can be passed.
// Interesting options to pass: NATPortMap() EnableAutoRelay(),
// libp2p.EnableNATService(), DisableRelay(), ConnectionManager(...)... see
// https://godoc.org/github.com/libp2p/go-libp2p#Option for more info.
func SetupLibp2p(
	ctx context.Context,
	hostKey crypto.PrivKey,
	listenAddrs []multiaddr.Multiaddr,
	ds datastore.Batching,
	opts ...libp2p.Option,
) (host.Host, *dualdht.DHT, error) {
	var ddht *dualdht.DHT
	var err error
	finalOpts := []libp2p.Option{
		libp2p.Identity(hostKey),
		libp2p.ListenAddrs(listenAddrs...),
		libp2p.EnableRelay(relay.OptHop),
		libp2p.EnableAutoRelay(),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			ddht, err = newDHT(ctx, h, ds)
			return ddht, err
		}),
	}
	finalOpts = append(finalOpts, opts...)
	h, err := libp2p.New(
		ctx,
		finalOpts...,
	)
	if err != nil {
		return nil, nil, err
	}
	return h, ddht, nil
}

func newDHT(ctx context.Context, h host.Host, ds datastore.Batching) (*dualdht.DHT, error) {
	dhtOpts := []dualdht.Option{
		dualdht.DHTOption(dht.NamespacedValidator("pk", record.PublicKeyValidator{})),
		dualdht.DHTOption(dht.NamespacedValidator("ipns", ipns.Validator{KeyBook: h.Peerstore()})),
		dualdht.DHTOption(dht.Concurrency(10)),
		dualdht.DHTOption(dht.Mode(dht.ModeAuto)),
	}
	if ds != nil {
		dhtOpts = append(dhtOpts, dualdht.DHTOption(dht.Datastore(ds)))
	}
	return dualdht.New(ctx, h, dhtOpts...)
}

// descendants recursively finds all the descendants of the given roots and
// adds them to the given cid.Set, using the provided dag.GetLinks function
// to walk the tree.
func descendants(ctx context.Context, getLinks dag.GetLinks, set *cid.Set, roots []cid.Cid) error {
	verifyGetLinks := func(ctx context.Context, c cid.Cid) ([]*ipld.Link, error) {
		err := verifcid.ValidateCid(c)
		if err != nil {
			return nil, err
		}

		return getLinks(ctx, c)
	}
	verboseCidError := func(err error) error {
		if strings.Contains(err.Error(), verifcid.ErrBelowMinimumHashLength.Error()) ||
			strings.Contains(err.Error(), verifcid.ErrPossiblyInsecureHashFunction.Error()) {
			err = fmt.Errorf("\"%s\"\nPlease run 'ipfs pin verify'"+
				" to list insecure hashes. If you want to read them,"+
				" please downgrade your go-ipfs to 0.4.13\n", err)
			log.Error(err)
		}
		return err
	}
	for _, c := range roots {
		// Walk recursively walks the dag and adds the keys to the given set
		err := dag.Walk(ctx, verifyGetLinks, c, set.Visit, dag.Concurrent())

		if err != nil {
			err = verboseCidError(err)
			return err
		}
	}
	return nil
}
