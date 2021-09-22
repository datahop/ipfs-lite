package ipfslite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-ipns"
	dag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-verifcid"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dualdht "github.com/libp2p/go-libp2p-kad-dht/dual"
	record "github.com/libp2p/go-libp2p-record"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	"github.com/libp2p/go-tcp-transport"
	"github.com/multiformats/go-multiaddr"
)

var (
	defaultBootstrapAddresses = []string{
		"/ip4/52.66.216.67/tcp/4501/p2p/QmcWEJqQD3bPMT5Mr7ijdwVCmVjUh5Z7CysiTQPgr2VZBC",
	}

	// libp2pOptionsExtra provides some useful libp2p options
	// to create a fully featured libp2p host. It can be used with
	// SetupLibp2p.
	libp2pOptionsExtra = []libp2p.Option{
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.DisableRelay(),
		libp2p.NATPortMap(),
		libp2p.ConnectionManager(connmgr.NewConnManager(100, 600, time.Minute)),
		libp2p.EnableNATService(),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
	}
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
