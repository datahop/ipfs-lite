package ipfs

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/datahop/ipfs-lite/internal/gateway"
	"github.com/datahop/ipfs-lite/internal/replication"
	"github.com/datahop/ipfs-lite/internal/repo"
	"github.com/grandcat/zeroconf"
	"github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	chunker "github.com/ipfs/go-ipfs-chunker"
	files "github.com/ipfs/go-ipfs-files"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs/importer/balanced"
	"github.com/ipfs/go-unixfs/importer/helpers"
	"github.com/ipfs/go-unixfs/importer/trickle"
	ufsio "github.com/ipfs/go-unixfs/io"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	inet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/libp2p/go-libp2p-core/routing"
	swarm "github.com/libp2p/go-libp2p-swarm"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	"github.com/libp2p/go-tcp-transport"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/multiformats/go-multihash"
)

func init() {
	ipld.Register(cid.DagProtobuf, merkledag.DecodeProtobufBlock)
	ipld.Register(cid.Raw, merkledag.DecodeRawBlock)
	ipld.Register(cid.DagCBOR, cbor.DecodeBlock) // need to decode CBOR
}

const (
	// ServiceTag is used for mDNS
	ServiceTag = "_datahop-discovery._tcp"

	defaultCrdtNamespace           = "/crdt"
	defaultCrdtRebroadcastInterval = time.Second * 3
	defaultMDNSInterval            = time.Second * 5
	defaultTopic                   = "datahop-crdt"
	defaultAutoDownload            = true
)

var (
	log = logging.Logger("ipfslite")
)

// Peer is an IPFS-Lite peer. It provides a DAG service that can fetch and put
// blocks from/to the IPFS network.
type Peer struct {
	Ctx             context.Context
	Cancel          context.CancelFunc
	Host            host.Host
	Store           datastore.Batching
	DHT             routing.Routing
	Repo            repo.Repo
	ipld.DAGService // become a DAG service
	bstore          blockstore.Blockstore
	bserv           blockservice.BlockService
	online          bool
	mtx             sync.Mutex
	Manager         *replication.Manager
	Stopped         chan bool
	CrdtTopic       string
}

// Option is interface for setting up the datahop ipfslite node
type Option func(*Options)

// WithmDNSInterval changes default mDNS rebroadcast interval
func WithmDNSInterval(interval time.Duration) Option {
	return func(h *Options) {
		h.mDNSInterval = interval
	}
}

// WithRebroadcastInterval changes default crdt rebroadcast interval
func WithRebroadcastInterval(interval time.Duration) Option {
	return func(h *Options) {
		h.crdtRebroadcastInterval = interval
	}
}

// WithmDNS decides if the ipfs node will start with mDNS or not
func WithmDNS(withmDNS bool) Option {
	return func(h *Options) {
		h.withmDNS = withmDNS
	}
}

// WithAutoDownload decides if content will be downloaded automatically
func WithAutoDownload(autoDownload bool) Option {
	return func(h *Options) {
		h.autoDownload = autoDownload
	}
}

// WithCrdtTopic sets the replication crdt listen topic
func WithCrdtTopic(topic string) Option {
	return func(h *Options) {
		h.crdtTopic = topic
	}
}

// WithCrdtNamespace sets the replication crdt namespace
func WithCrdtNamespace(ns string) Option {
	return func(h *Options) {
		h.crdtPrefix = ns
	}
}

// Options for setting up the datahop ipfslite node
type Options struct {
	mDNSInterval            time.Duration
	crdtRebroadcastInterval time.Duration
	crdtPrefix              string
	withmDNS                bool
	autoDownload            bool
	crdtTopic               string
}

func defaultOptions() *Options {
	return &Options{
		mDNSInterval:            defaultMDNSInterval,
		crdtRebroadcastInterval: defaultCrdtRebroadcastInterval,
		withmDNS:                true,
		crdtTopic:               defaultTopic,
		crdtPrefix:              defaultCrdtNamespace,
		autoDownload:            defaultAutoDownload,
	}
}

// New creates an IPFS-Lite Peer. It uses the given datastore, libp2p Host and
// Routing (usuall the DHT). Peer implements the ipld.DAGService interface.
func New(
	ctx context.Context,
	cancelFunc context.CancelFunc,
	r repo.Repo,
	swarmKey []byte,
	opts ...Option,
) (*Peer, error) {
	options := defaultOptions()
	for _, option := range opts {
		option(options)
	}
	cfg, err := r.Config()
	if err != nil {
		return nil, err
	}
	privb, _ := base64.StdEncoding.DecodeString(cfg.Identity.PrivKey)
	privKey, _ := crypto.UnmarshalPrivateKey(privb)

	listenAddrs := []multiaddr.Multiaddr{}
	confAddrs := cfg.Addresses.Swarm
	for _, v := range confAddrs {
		listen, _ := multiaddr.NewMultiaddr(v)
		listenAddrs = append(listenAddrs, listen)
	}

	// libp2pOptionsExtra provides some useful libp2p options
	// to create a fully featured libp2p host. It can be used with
	// SetupLibp2p.
	libp2pOptionsExtra := []libp2p.Option{
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.NATPortMap(),
		libp2p.ConnectionManager(connmgr.NewConnManager(100, 600, time.Minute)),
		libp2p.EnableNATService(),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
	}
	if swarmKey != nil {
		log.Info("got swarm key. starting private network")
		psk, err := pnet.DecodeV1PSK(bytes.NewReader(swarmKey))
		if err != nil {
			log.Error("swarm key decode failed")
			return nil, err
		}
		libp2pOptionsExtra = append(libp2pOptionsExtra, libp2p.PrivateNetwork(psk))
	}
	h, dht, err := SetupLibp2p(
		ctx,
		privKey,
		listenAddrs,
		r.Datastore(),
		libp2pOptionsExtra...,
	)
	if err != nil {
		return nil, err
	}
	p := &Peer{
		Ctx:     ctx,
		Cancel:  cancelFunc,
		Host:    h,
		DHT:     dht,
		Store:   r.Datastore(),
		Repo:    r,
		Stopped: make(chan bool, 1),
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
	err = p.setupCrdtStore(options)
	if err != nil {
		p.bserv.Close()
		return nil, err
	}
	p.Manager.StartContentWatcher()

	p.Repo.Matrix().StartTicker(p.Ctx)

	p.mtx.Lock()
	p.online = true
	p.mtx.Unlock()

	go p.runGateway(ctx)
	go p.autoclose()
	p.ZeroConfScan()

	return p, nil
}

func (p *Peer) runGateway(ctx context.Context) {
	listeningMultiAddr := "/ip4/0.0.0.0/tcp/8080"
	addr, err := multiaddr.NewMultiaddr(listeningMultiAddr)
	if err != nil {
		log.Error("http newMultiaddr:", err.Error())
		return
	}

	list, err := manet.Listen(addr)
	if err != nil {
		log.Error("http manet Listen:", err.Error())
		return
	}
	defer list.Close()

	// we might have listened to /tcp/0 - let's see what we are listing on
	addr = list.Multiaddr()
	log.Infof("API server listening on %s", addr)
	topMux := http.NewServeMux()
	gway := gateway.NewGatewayHandler(p, merkledag.NewReadOnlyDagService(merkledag.NewSession(ctx, p)), p.bserv)

	topMux.Handle(gateway.ContentPathPrefix, gway)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		topMux.ServeHTTP(w, r)
	})
	server := &http.Server{
		Handler: handler,
	}
	defer server.Close()
	err = server.Serve(manet.NetListener(list))
	if err != nil {
		log.Error("serve :", err.Error())
		return
	}
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

func (p *Peer) setupCrdtStore(opts *Options) error {
	ctx, cancel := context.WithCancel(p.Ctx)
	manager, err := replication.New(ctx, cancel, p.Repo, p.Host, p, p.Store, opts.crdtPrefix, opts.crdtTopic, opts.crdtRebroadcastInterval, p, p.Host.Peerstore(), opts.autoDownload)
	if err != nil {
		return err
	}
	p.Manager = manager
	p.CrdtTopic = opts.crdtTopic
	return nil
}

func (p *Peer) ConnectIfNotConnectedUsingRelay(ctx context.Context, providers []peer.ID) {
	for _, v := range providers {
		conn := p.Host.Network().Connectedness(v)
		if conn != inet.Connected {
			pi, err := peer.AddrInfoFromString(fmt.Sprintf("%s/p2p-circuit/p2p/%s", relayAddr, v.String()))
			if err != nil {
				continue
			}
			err = p.Host.Connect(ctx, *pi)
			if err == nil {
				fmt.Println("connection established with ", fmt.Sprintf("%s/p2p-circuit/p2p/%s", relayAddr, v.String()))
			}
		}
	}
}

// FindProviders check dht to check for providers of a given cid
func (p *Peer) FindProviders(ctx context.Context, id cid.Cid) []peer.ID {
	providerAddresses := []peer.ID{}
	providers := p.DHT.FindProvidersAsync(ctx, id, 0)
FindProvider:
	for {
		select {
		case provider := <-providers:
			if provider.ID == p.Host.ID() {
				continue
			}
			if provider.ID.String() == "" {
				break FindProvider
			}
			providerAddresses = append(providerAddresses, provider.ID)
		case <-time.After(time.Second):
			break FindProvider
		case <-ctx.Done():
			break FindProvider
		}
	}
	return providerAddresses
}

func (p *Peer) autoclose() {
	<-p.Ctx.Done()
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.Repo.Matrix().Close()
	p.online = false
	p.Manager.Close()
	p.Host.Close()
	p.bserv.Close()
	go func() {
		p.Stopped <- true
	}()
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

func (p *Peer) AddDir(ctx context.Context, dir string, params *AddParams) (ipld.Node, error) {
	stat, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if params == nil {
		params = &AddParams{}
	}
	if params.HashFun == "" {
		params.HashFun = "sha2-256"
	}

	sf, err := files.NewSerialFile(dir, false, stat)
	if err != nil {
		return nil, err
	}
	fAddr, err := NewAdder(ctx, p)
	if err != nil {
		return nil, err
	}
	fAddr.Chunker = params.Chunker
	fAddr.CidBuilder, err = merkledag.PrefixForCidVersion(1)
	if err != nil {
		return nil, err
	}
	fAddr.RawLeaves = params.RawLeaves
	fAddr.NoCopy = params.NoCopy
	nd, err := fAddr.AddAll(sf)
	if err != nil {
		return nil, err
	}
	return nd, nil
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

// Download will get the content by its root CID. Read it till the end.
func (p *Peer) Download(ctx context.Context, c cid.Cid) error {
	node, err := p.Get(ctx, c)
	if err != nil {
		return err
	}
	r, err := ufsio.NewDagReader(ctx, node, p)
	if err != nil {
		return err
	}
	defer r.Close()
	nr := int64(0)
	buf := make([]byte, 0, 4*1024)
	for {
		n, err := r.Read(buf[:cap(buf)])
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			return err
		}
		nr += int64(len(buf))
		if err != nil && err != io.EOF {
			return err
		}
	}
	log.Debugf("%s downloaded. size read : %d\n", c.String(), nr)
	buf = nil
	return nil
}

// DeleteFile removes content from blockstore by its root CID. The file
// must have been added as a UnixFS DAG (default for IPFS).
func (p *Peer) DeleteFile(ctx context.Context, c cid.Cid) error {
	found, err := p.BlockStore().Has(c)
	if err != nil {
		log.Error("Unable to find block ", err)
		return err
	}
	if !found {
		log.Warn("Content not found in datastore")
		return errors.New("content not found in datastore")
	}

	getLinks := func(ctx context.Context, cid cid.Cid) ([]*ipld.Link, error) {
		links, err := ipld.GetLinks(ctx, p, c)
		if err != nil {
			return nil, err
		}
		return links, nil
	}
	gcs := cid.NewSet()
	err = descendants(ctx, getLinks, gcs, []cid.Cid{c})
	if err != nil {
		log.Error("Descendants failed ", err)
		return err
	}
	err = gcs.ForEach(func(c cid.Cid) error {
		log.Debug(c)
		err = p.BlockStore().DeleteBlock(c)
		if err != nil {
			log.Error("Unable to remove block ", err)
			return err
		}
		return nil
	})
	if err != nil {
		log.Error("Block removal failed ", err)
		return err
	}
	return nil
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

// HandlePeerFound tries to connect to a given peerinfo
func (p *Peer) HandlePeerFound(pi peer.AddrInfo) {
	log.Debug("Discovered Peer : ", pi)
	log.Debug("Own address : ", p.Host.Addrs())
	<-time.After(time.Second)
	err := p.Connect(p.Ctx, pi)
	if err != nil {
		log.Errorf("Connect failed with peer %s for %s", pi.ID, err.Error())
	}
}

// HandlePeerFoundWithError tries to connect to a given peerinfo, returns error if failed
func (p *Peer) HandlePeerFoundWithError(pi peer.AddrInfo) error {
	log.Debug("Discovered Peer : ", pi)
	<-time.After(time.Second)
	err := p.Connect(p.Ctx, pi)
	if err != nil {
		log.Errorf("Connect failed with peer %s for %s", pi.ID, err.Error())
		return err
	}
	return nil
}

// IsOnline returns if the node ios online
func (p *Peer) IsOnline() bool {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	return p.online
}

func (p *Peer) ZeroConfScan() {
	log.Debug("zconf : ZeroConfScan")
	connections := p.Host.Network().Conns()
	for _, conn := range connections {
		log.Debug("zconf : Closing connection :", conn.RemotePeer().Pretty())
		conn.Close()
	}

	action := func(entry *zeroconf.ServiceEntry) {
		log.Debugf("zconf : Handle action : %+v\n", entry.Instance)
		idr := strings.Split(entry.Instance, ":")
		if len(idr) < 2 {
			log.Error("zconf : wrong instance ", entry.Instance)
			return
		}
		if idr[0] == p.Host.ID().String() {
			log.Debug("zconf : got own address")
			return
		}
		if len(entry.AddrIPv4) == 0 {
			log.Warn("Discovered peer with no ipv4")
			return
		}
		address := fmt.Sprintf("/ip4/%s/tcp/%s/p2p/%s", entry.AddrIPv4[0], idr[1], idr[0])
		maddr, err := multiaddr.NewMultiaddr(address)
		if err != nil {
			return
		}
		pi, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return
		}
		cNess := p.Host.Network().Connectedness(pi.ID)
		if cNess != inet.Connected {
			log.Debug("zconf : Connect from zconf : ", pi)
			err = p.Connect(p.Ctx, *pi)
			if err != nil {
				log.Error("zconf : Connecting failed : ", err.Error(), address)
				return
			}
		}
	}
	cfg, err := p.Repo.Config()
	if err != nil {
		return
	}

	err = lookupAndConnect(p.Ctx, action)
	if err != nil {
		return
	}
	go func() {
		err := p.registerZeroConf(fmt.Sprintf("%s:%s", p.Host.ID().String(), cfg.SwarmPort))
		if err != nil {
			log.Debugf("registerZeroConf failed : %s", err)
			log.Error("registerZeroConf failed")
		}
	}()
}

func (p *Peer) registerZeroConf(instance string) error {
	log.Debug("zconf : register service zconf ", instance)
	for {
		select {
		case <-p.Ctx.Done():
			return nil
		case <-time.After(time.Second * 5):
			log.Debug("zconf : register")
			server, err := zeroconf.Register(instance, "_zconf-discovery._tcp", "local.", 42424, []string{"txtv=0", "lo=1", "la=2"}, nil)
			if err != nil {
				log.Error("zconf : RegisterZeroConf failed : ", err.Error())
				continue
			}
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				<-time.After(time.Second * 3)
				log.Debug("zconf : Shutting down zeroconf server")
				server.Shutdown()
				wg.Done()
			}()
			wg.Wait()
		}
	}
}

func lookupAndConnect(ctx context.Context, action func(*zeroconf.ServiceEntry)) error {
	log.Debug("zconf : lookupAndConnect service zconf")
	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		for entry := range entries {
			log.Debugf("zconf : Got Entry %+v\n", entry)
			action(entry)
		}
	}()
	go func() {
		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			log.Error("zconf : NewResolver Failed ", err.Error())
			return
		}
		err = resolver.Browse(ctx, "_zconf-discovery._tcp", "local.", entries)
		if err != nil {
			log.Error("zconf : Unable to browse services ", err.Error())
			return
		}
	}()
	return nil
}
