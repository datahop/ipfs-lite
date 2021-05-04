// Package ipfslite is a lightweight IPFS peer which runs the minimal setup to
// provide an `ipld.DAGService`, "Add" and "Get" UnixFS files from IPFS.
package ipfslite

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	crdt "github.com/ipfs/go-ds-crdt"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	chunker "github.com/ipfs/go-ipfs-chunker"
	provider "github.com/ipfs/go-ipfs-provider"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs/importer/balanced"
	"github.com/ipfs/go-unixfs/importer/helpers"
	"github.com/ipfs/go-unixfs/importer/trickle"
	ufsio "github.com/ipfs/go-unixfs/io"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	inet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	swarm "github.com/libp2p/go-libp2p-swarm"
	"github.com/libp2p/go-libp2p/p2p/discovery"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

func init() {
	ipld.Register(cid.DagProtobuf, merkledag.DecodeProtobufBlock)
	ipld.Register(cid.Raw, merkledag.DecodeRawBlock)
	ipld.Register(cid.DagCBOR, cbor.DecodeBlock) // need to decode CBOR
}

const (
	ServiceTag              = "_datahop-discovery._tcp"
)

var (
	log = logging.Logger("ipfslite")
	defaultCrdtNamespace = "/crdt"
	defaultCrdtRebroadcastInterval = time.Second * 3
	defaultTopic  = "datahop-crdt"
)

// Peer is an IPFS-Lite peer. It provides a DAG service that can fetch and put
// blocks from/to the IPFS network.
type Peer struct {
	Ctx             context.Context
	Host            host.Host
	Store           datastore.Batching
	DHT             routing.Routing
	Repo            Repo
	Provider        provider.System
	ipld.DAGService // become a DAG service
	bstore          blockstore.Blockstore
	bserv           blockservice.BlockService
	online          bool
	mtx             sync.Mutex
	CrdtStore       *crdt.Datastore
}

// New creates an IPFS-Lite Peer. It uses the given datastore, libp2p Host and
// Routing (usuall the DHT). Peer implements the ipld.DAGService interface.
func New(
	ctx context.Context,
	r Repo,
) (*Peer, error) {
	cfg, err := r.Config()
	if err != nil {
		return nil, err
	}
	privb, _ := base64.StdEncoding.DecodeString(cfg.Identity.PrivKey)
	privKey, _ := crypto.UnmarshalPrivateKey(privb)

	listenAddrs := []multiaddr.Multiaddr{}

	// Listen for local interface addresses
	ifaces := listMulticastInterfaces()
	for _, iface := range ifaces {
		v4, _ := addrsForInterface(&iface)
		for _, v := range v4 {
			if !strings.HasPrefix(v.String(), "127") {
				listen, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", v.String(), cfg.SwarmPort))
				listenAddrs = append(listenAddrs, listen)
			}
		}
	}
	if len(listenAddrs) == 0 {
		confAddrs := cfg.Addresses.Swarm
		for _, v := range confAddrs {
			listen, _ := multiaddr.NewMultiaddr(v)
			listenAddrs = append(listenAddrs, listen)
		}
	}

	h, dht, err := SetupLibp2p(
		ctx,
		privKey,
		listenAddrs,
		r.Datastore(),
		Libp2pOptionsExtra...,
	)
	if err != nil {
		return nil, err
	}
	p := &Peer{
		Ctx:   ctx,
		Host:  h,
		DHT:   dht,
		Store: r.Datastore(),
		Repo:  r,
	}
	err = p.setupBlockstore()
	if err != nil {
		return nil, err
	}
	err = p.setupBlockService()
	if err != nil {
		return nil, err
	}
	err = p.setupDAGService()
	if err != nil {
		p.bserv.Close()
		return nil, err
	}
	if err != nil {
		p.bserv.Close()
		return nil, err
	}
	err = p.setupCrdtStore()

	p.mtx.Lock()
	p.online = true
	p.mtx.Unlock()

	go p.autoclose()

	// Register mDNS
	mDnsService, err := discovery.NewMdnsService(ctx, p.Host, time.Second, ServiceTag)
	if err != nil {
		log.Error("mDns service failed")
	} else {
		mDnsService.RegisterNotifee(p)
		log.Debug("mDNS service stared")
	}
	return p, nil
}

func (p *Peer) setupBlockstore() error {
	bs := blockstore.NewBlockstore(p.Store)
	bs = blockstore.NewIdStore(bs)
	cachedbs, err := blockstore.CachedBlockstore(p.Ctx, bs, blockstore.DefaultCacheOpts())
	if err != nil {
		return err
	}
	p.bstore = cachedbs
	return nil
}

func (p *Peer) setupBlockService() error {
	bswapnet := network.NewFromIpfsHost(p.Host, p.DHT)
	bswap := bitswap.New(p.Ctx, bswapnet, p.bstore)
	p.bserv = blockservice.New(p.bstore, bswap)
	return nil
}

func (p *Peer) setupDAGService() error {
	p.DAGService = merkledag.NewDAGService(p.bserv)
	return nil
}

func (p *Peer) setupCrdtStore() error {
	psub, err := pubsub.NewGossipSub(p.Ctx, p.Host)
	if err != nil {
		return err
	}
	// TODO Add RegisterTopicValidator
	pubsubBC, err := crdt.NewPubSubBroadcaster(p.Ctx, psub, defaultTopic)
	if err != nil {
		return err
	}

	opts := crdt.DefaultOptions()
	opts.Logger = log
	opts.RebroadcastInterval = defaultCrdtRebroadcastInterval
	opts.PutHook = func(k datastore.Key, v []byte) {
		log.Debugf("Added: [%s] -> %s\n", k, string(v))
	}
	opts.DeleteHook = func(k datastore.Key) {
		log.Debugf("Removed: [%s]\n", k)
	}

	crdtStore, err := crdt.New(p.Store, datastore.NewKey(defaultCrdtNamespace), p, pubsubBC, opts)
	if err != nil {
		return err
	}
	p.CrdtStore = crdtStore
	return nil
}

func (p *Peer) autoclose() {
	<-p.Ctx.Done()
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.online = false
	p.CrdtStore.Close()
	p.Repo.Datastore().Close()
	p.Host.Close()
	p.bserv.Close()
}

func addrsForInterface(iface *net.Interface) ([]net.IP, []net.IP) {
	var v4, v6, v6local []net.IP
	addrs, _ := iface.Addrs()
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				v4 = append(v4, ipnet.IP)
			} else {
				switch ip := ipnet.IP.To16(); ip != nil {
				case ip.IsGlobalUnicast():
					v6 = append(v6, ipnet.IP)
				case ip.IsLinkLocalUnicast():
					v6local = append(v6local, ipnet.IP)
				}
			}
		}
	}
	if len(v6) == 0 {
		v6 = v6local
	}
	return v4, v6
}

func listMulticastInterfaces() []net.Interface {
	var interfaces []net.Interface
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, ifi := range ifaces {
		if (ifi.Flags & net.FlagUp) == 0 {
			continue
		}
		if (ifi.Flags & net.FlagMulticast) > 0 {
			interfaces = append(interfaces, ifi)
		}
	}

	return interfaces
}

// Bootstrap is an optional helper to connect to the given peers and bootstrap
// the Peer DHT (and Bitswap). This is a best-effort function. Errors are only
// logged and a warning is printed when less than half of the given peers
// could be contacted. It is fine to pass a list where some peers will not be
// reachable.
func (p *Peer) Bootstrap(peers []peer.AddrInfo) {
	connected := make(chan struct{})

	var wg sync.WaitGroup
	for _, pinfo := range peers {
		//h.Peerstore().AddAddrs(pinfo.ID, pinfo.Addrs, peerstore.PermanentAddrTTL)
		wg.Add(1)
		go func(pinfo peer.AddrInfo) {
			defer wg.Done()
			err := p.Host.Connect(p.Ctx, pinfo)
			if err != nil {
				log.Warn(err)
				return
			}
			log.Info("Connected to", pinfo.ID)
			connected <- struct{}{}
		}(pinfo)
	}

	go func() {
		wg.Wait()
		close(connected)
	}()

	i := 0
	for range connected {
		i++
	}
	if nPeers := len(peers); i < nPeers/2 {
		log.Warnf("only connected to %d bootstrap peers out of %d", i, nPeers)
	}

	err := p.DHT.Bootstrap(p.Ctx)
	if err != nil {
		log.Error(err)
		return
	}
}

// Session returns a session-based NodeGetter.
func (p *Peer) Session(ctx context.Context) ipld.NodeGetter {
	ng := merkledag.NewSession(ctx, p.DAGService)
	if ng == p.DAGService {
		log.Warn("DAGService does not support sessions")
	}
	return ng
}

// AddParams contains all of the configurable parameters needed to specify the
// importing process of a file.
type AddParams struct {
	Layout    string
	Chunker   string
	RawLeaves bool
	Hidden    bool
	Shard     bool
	NoCopy    bool
	HashFun   string
}

// AddFile chunks and adds content to the DAGService from a reader. The content
// is stored as a UnixFS DAG (default for IPFS). It returns the root
// ipld.Node.
func (p *Peer) AddFile(ctx context.Context, r io.Reader, params *AddParams) (ipld.Node, error) {
	if params == nil {
		params = &AddParams{}
	}
	if params.HashFun == "" {
		params.HashFun = "sha2-256"
	}

	prefix, err := merkledag.PrefixForCidVersion(1)
	if err != nil {
		return nil, fmt.Errorf("bad CID Version: %s", err)
	}

	hashFunCode, ok := multihash.Names[strings.ToLower(params.HashFun)]
	if !ok {
		return nil, fmt.Errorf("unrecognized hash function: %s", params.HashFun)
	}
	prefix.MhType = hashFunCode
	prefix.MhLength = -1

	dbp := helpers.DagBuilderParams{
		Dagserv:    p,
		RawLeaves:  params.RawLeaves,
		Maxlinks:   helpers.DefaultLinksPerBlock,
		NoCopy:     params.NoCopy,
		CidBuilder: &prefix,
	}

	chnk, err := chunker.FromString(r, params.Chunker)
	if err != nil {
		return nil, err
	}
	dbh, err := dbp.New(chnk)
	if err != nil {
		return nil, err
	}

	var n ipld.Node
	switch params.Layout {
	case "trickle":
		n, err = trickle.Layout(dbh)
	case "balanced", "":
		n, err = balanced.Layout(dbh)
	default:
		return nil, errors.New("invalid Layout")
	}
	return n, err
}

// GetFile returns a reader to a file as identified by its root CID. The file
// must have been added as a UnixFS DAG (default for IPFS).
func (p *Peer) GetFile(ctx context.Context, c cid.Cid) (ufsio.ReadSeekCloser, error) {
	n, err := p.Get(ctx, c)
	if err != nil {
		return nil, err
	}
	return ufsio.NewDagReader(ctx, n, p)
}

// BlockStore offers access to the blockstore underlying the Peer's DAGService.
func (p *Peer) BlockStore() blockstore.Blockstore {
	return p.bstore
}

// HasBlock returns whether a given block is available locally. It is
// a shorthand for .Blockstore().Has().
func (p *Peer) HasBlock(c cid.Cid) (bool, error) {
	return p.BlockStore().Has(c)
}

const connectionManagerTag = "user-connect"
const connectionManagerWeight = 100

// Connect connects host to a given peer
func (p *Peer) Connect(ctx context.Context, pi peer.AddrInfo) error {
	if p.Host == nil {
		return errors.New("peer is offline")
	}

	if swrm, ok := p.Host.Network().(*swarm.Swarm); ok {
		swrm.Backoff().Clear(pi.ID)
	}
	if p.Host.Network().Connectedness(pi.ID) != inet.Connected {
		if err := p.Host.Connect(ctx, pi); err != nil {
			return err
		}
		p.Host.ConnManager().TagPeer(pi.ID, connectionManagerTag, connectionManagerWeight)
	}
	return nil
}

// Peers returns a list of connected peers
func (p *Peer) Peers() []string {
	pIDs := p.Host.Network().Peers()
	peerList := []string{}
	for _, pID := range pIDs {
		peerList = append(peerList, pID.String())
	}
	return peerList
}

// Disconnect host from a given peer
func (p *Peer) Disconnect(pi peer.AddrInfo) error {
	if p.Host == nil {
		return errors.New("peer is offline")
	}
	if pi.ID.String() == "" {
		return peer.ErrInvalidAddr
	}
	net := p.Host.Network()
	if net.Connectedness(pi.ID) != inet.Connected {
		return errors.New("not connected")
	}
	if err := net.ClosePeer(pi.ID); err != nil {
		return err
	}

	for _, conn := range net.ConnsToPeer(pi.ID) {
		return conn.Close()
	}
	return nil
}

func (p *Peer) HandlePeerFound(pi peer.AddrInfo) {
	log.Debug("mDNS Found Peer : ", pi.ID)
	if err := p.Host.Connect(context.Background(), pi); err != nil {
		log.Error("Failed to connect to peer : ", pi.ID.String())
	}
}

func (p *Peer) IsOnline() bool {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	return p.online
}
