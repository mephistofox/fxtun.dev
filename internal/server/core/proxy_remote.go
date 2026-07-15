package core

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

// remoteProxyPool manages reverse proxies to other nodes.
type remoteProxyPool struct {
	proxies map[string]*httputil.ReverseProxy
	mu      sync.RWMutex
}

func newRemoteProxyPool() *remoteProxyPool {
	return &remoteProxyPool{
		proxies: make(map[string]*httputil.ReverseProxy),
	}
}

func (p *remoteProxyPool) getOrCreate(nodeAddr string) *httputil.ReverseProxy {
	p.mu.RLock()
	proxy, ok := p.proxies[nodeAddr]
	p.mu.RUnlock()
	if ok {
		return proxy
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if proxy, ok = p.proxies[nodeAddr]; ok {
		return proxy
	}

	target, err := url.Parse("http://" + nodeAddr)
	if err != nil {
		// Invalid node address — don't cache, return a one-off error proxy
		return &httputil.ReverseProxy{
			Director: func(r *http.Request) {},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, e error) {
				http.Error(w, "Bad gateway: invalid node address", http.StatusBadGateway)
			},
		}
	}
	proxy = httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}
	p.proxies[nodeAddr] = proxy
	return proxy
}

// proxyToRemoteNode reverse-proxies an HTTP request to the node that owns the tunnel.
func (r *HTTPRouter) proxyToRemoteNode(w http.ResponseWriter, req *http.Request, entry *store.TunnelEntry) {
	// Prevent proxy loops: if already proxied once, return error
	if req.Header.Get("X-FxTunnel-Hop") != "" {
		r.serveErrorPage(w, http.StatusBadGateway, "Tunnel routing loop detected")
		return
	}
	req.Header.Set("X-FxTunnel-Hop", "1")
	req.Header.Set("X-Forwarded-Server", r.server.LocalNodeID())

	proxy := r.server.proxyPool.getOrCreate(entry.ServerID)
	proxy.ServeHTTP(w, req)
}
