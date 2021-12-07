package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	gopath "path"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ipld/go-ipld-prime/datamodel"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	bsfetcher "github.com/ipfs/go-fetcher/impl/blockservice"
	files "github.com/ipfs/go-ipfs-files"
	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log/v2"
	ipfspath "github.com/ipfs/go-path"
	"github.com/ipfs/go-path/resolver"
	unixfile "github.com/ipfs/go-unixfs/file"
	"github.com/ipfs/interface-go-ipfs-core/path"
	dagpb "github.com/ipld/go-codec-dagpb"
	pipld "github.com/ipld/go-ipld-prime"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/libp2p/go-libp2p-core/routing"
)

const (
	ContentPathPrefix = "/ipfs/"
)

var log = logging.Logger("gateway")

type Peer interface {
	Get(context.Context, cid.Cid) (ipld.Node, error)
}

type GatewayHandler struct {
	peer Peer
	dag  ipld.DAGService
	bs   blockservice.BlockService
}

func NewGatewayHandler(peer Peer, dag ipld.DAGService, bserv blockservice.BlockService) *GatewayHandler {
	return &GatewayHandler{
		peer: peer,
		dag:  dag,
		bs:   bserv,
	}
}

func parseIpfsPath(p string) (cid.Cid, string, error) {
	rootPath, err := ipfspath.ParsePath(p)
	if err != nil {
		return cid.Cid{}, "", err
	}

	// Check the path.
	rsegs := rootPath.Segments()
	if rsegs[0] != "ipfs" {
		return cid.Cid{}, "", fmt.Errorf("WritableGateway: only ipfs paths supported")
	}

	rootCid, err := cid.Decode(rsegs[1])
	if err != nil {
		return cid.Cid{}, "", err
	}

	return rootCid, ipfspath.Join(rsegs[2:]), nil
}

func (i *GatewayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Debug("ServeHTTP : ", r.URL.String())

	// the hour is a hard fallback, we don't expect it to happen, but just in case
	ctx, cancel := context.WithTimeout(r.Context(), time.Hour)
	defer cancel()
	r = r.WithContext(ctx)

	defer func() {
		if r := recover(); r != nil {
			log.Error("A panic occurred in the gateway handler!")
			log.Error(r)
			debug.PrintStack()
		}
	}()
	i.getOrHeadHandler(w, r)

	errmsg := "Method " + r.Method + " not allowed: "
	var status int
	status = http.StatusBadRequest
	errmsg = errmsg + "bad request for " + r.URL.Path
	http.Error(w, errmsg, status)
}

func (i *GatewayHandler) getOrHeadHandler(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	escapedURLPath := r.URL.EscapedPath()

	// HostnameOption might have constructed an IPNS/IPFS path using the Host header.
	// In this case, we need the original path for constructing redirects
	// and links that match the requested URL.
	// For example, http://example.net would become /ipns/example.net, and
	// the redirects and links would end up as http://example.net/ipns/example.net
	requestURI, err := url.ParseRequestURI(r.RequestURI)
	if err != nil {
		webError(w, "failed to parse request path", err, http.StatusInternalServerError)
		return
	}
	originalUrlPath := requestURI.Path
	log.Debug("originalUrlPath ", originalUrlPath)
	// Service Worker registration request
	if r.Header.Get("Service-Worker") == "script" {
		// Disallow Service Worker registration on namespace roots
		// https://github.com/ipfs/go-ipfs/issues/4025
		matched, _ := regexp.MatchString(`^/ip[fn]s/[^/]+$`, r.URL.Path)
		if matched {
			err := fmt.Errorf("registration is not allowed for this scope")
			webError(w, "navigator.serviceWorker", err, http.StatusBadRequest)
			return
		}
	}

	parsedPath := path.New(urlPath)
	log.Debug("parsedPath ", parsedPath)
	if err := parsedPath.IsValid(); err != nil {
		webError(w, "invalid ipfs path", err, http.StatusBadRequest)
		return
	}
	resolvedPath, err := i.ResolvePath(r.Context(), parsedPath)
	if err != nil {
		webError(w, "unable to resolve path", err, http.StatusBadRequest)
		return
	}

	node, err := i.dag.Get(r.Context(), resolvedPath.Cid())
	if err != nil {
		return
	}
	p := strings.Split(urlPath, "/")
	c, err := cid.Decode(p[2])
	if err != nil {
		webError(w, "unable to decode cid", err, http.StatusBadRequest)
		return
	}
	log.Debug("cid ", c.String())
	dr, err := unixfile.NewUnixfsFile(r.Context(), i.dag, node)
	if err != nil {
		webError(w, "failed to get cid content "+escapedURLPath, err, http.StatusNotFound)
		return
	}
	defer dr.Close()

	// we need to figure out whether this is a directory before doing most of the heavy lifting below
	_, ok := dr.(files.Directory)
	log.Debug("Directory : ", ok)

	modtime := time.Unix(1, 0)
	if f, ok := dr.(files.File); ok {
		urlFilename := r.URL.Query().Get("filename")
		var name string
		if urlFilename != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename*=UTF-8''%s", url.PathEscape(urlFilename)))
			name = urlFilename
		} else {
			name = getFilename(urlPath)
		}
		i.serveFile(w, r, name, modtime, f)
		return
	}

	// A HTML directory index will be presented, be sure to set the correct
	// type instead of relying on autodetection (which may fail).
	w.Header().Set("Content-Type", "text/html")
	if r.Method == http.MethodHead {
		return
	}

	// construct the correct back link
	// https://github.com/ipfs/go-ipfs/issues/1365
	var backLink string = originalUrlPath

	// don't go further up than /ipfs/$hash/
	pathSplit := ipfspath.SplitList(urlPath)
	switch {
	// keep backlink
	case len(pathSplit) == 3: // url: /ipfs/$hash

	// keep backlink
	case len(pathSplit) == 4 && pathSplit[3] == "": // url: /ipfs/$hash/

	// add the correct link depending on whether the path ends with a slash
	default:
		if strings.HasSuffix(backLink, "/") {
			backLink += "./.."
		} else {
			backLink += "/.."
		}
	}

	hash := c.String()
	log.Debug(hash)
}

func (i *GatewayHandler) serveFile(w http.ResponseWriter, req *http.Request, name string, modtime time.Time, file files.File) {
	http.ServeContent(w, req, name, modtime, file)
}

func webError(w http.ResponseWriter, message string, err error, defaultCode int) {
	if _, ok := err.(resolver.ErrNoLink); ok {
		webErrorWithCode(w, message, err, http.StatusNotFound)
	} else if err == routing.ErrNotFound {
		webErrorWithCode(w, message, err, http.StatusNotFound)
	} else if err == context.DeadlineExceeded {
		webErrorWithCode(w, message, err, http.StatusRequestTimeout)
	} else {
		webErrorWithCode(w, message, err, defaultCode)
	}
}

func webErrorWithCode(w http.ResponseWriter, message string, err error, code int) {
	http.Error(w, fmt.Sprintf("%s: %s", message, err), code)
	if code >= 500 {
		log.Warnf("server error: %s", err)
	}
}

// return a 500 error and log
func internalWebError(w http.ResponseWriter, err error) {
	webErrorWithCode(w, "internalWebError", err, http.StatusInternalServerError)
}

func (i *GatewayHandler) ResolvePath(ctx context.Context, p path.Path) (path.Resolved, error) {
	if _, ok := p.(path.Resolved); ok {
		return p.(path.Resolved), nil
	}
	if err := p.IsValid(); err != nil {
		return nil, err
	}

	ip := ipfspath.Path(p.String())
	if ip.Segments()[0] != "ipfs" && ip.Segments()[0] != "ipld" {
		return nil, fmt.Errorf("unsupported path namespace: %s", p.Namespace())
	}

	ipldFetcher := bsfetcher.NewFetcherConfig(i.bs)
	ipldFetcher.PrototypeChooser = dagpb.AddSupportToChooser(func(lnk datamodel.Link, lnkCtx pipld.LinkContext) (datamodel.NodePrototype, error) {
		if tlnkNd, ok := lnkCtx.LinkNode.(schema.TypedLinkNode); ok {
			return tlnkNd.LinkTargetNodePrototype(), nil
		}
		return basicnode.Prototype.Any, nil
	})
	r := resolver.NewBasicResolver(ipldFetcher)

	node, rest, err := r.ResolveToLastNode(ctx, ip)
	if err != nil {
		return nil, err
	}

	root, err := cid.Parse(ip.Segments()[1])
	if err != nil {
		return nil, err
	}

	return path.NewResolvedPath(ip, node, root, gopath.Join(rest...)), nil
}

func getFilename(s string) string {
	return gopath.Base(s)
}
