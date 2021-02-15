package proxy

import (
	"context"
	"crypto/tls"
	golog "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/ConsenSys/quorum-hibernate/config"
	"github.com/ConsenSys/quorum-hibernate/log"
	"github.com/ConsenSys/quorum-hibernate/node"
)

// ProxyServer represents a proxy server
type ProxyServer struct {
	nodeCtrl      *node.NodeControl // node controller
	proxyCfg      *config.Proxy     // proxy config
	ignorePathMap map[string]bool
	mux           *http.ServeMux
	srv           *http.Server           // http server for the proxy
	rp            *httputil.ReverseProxy // handler for http reverse proxy
	wp            *WebsocketProxy        // handler for websocket
	errCh         chan error             // error channel
	shutdownWg    sync.WaitGroup
}

// CanIgnoreRequest implements Proxy.CanIgnoreRequest
func (ps ProxyServer) CanIgnoreRequest(req string) bool {
	_, ok := ps.ignorePathMap[req]
	return ok
}

func NewProxyServer(qn *node.NodeControl, pc *config.Proxy, errc chan error) (Proxy, error) {
	ps := &ProxyServer{qn, pc, make(map[string]bool), nil, nil, nil, nil, errc, sync.WaitGroup{}}
	url, err := url.Parse(ps.proxyCfg.UpstreamAddr)
	if err != nil {
		return nil, err
	}

	ps.mux = http.NewServeMux()

	for _, p := range ps.proxyCfg.IgnorePathsForActivity {
		ps.ignorePathMap[p] = true
	}

	if ps.proxyCfg.IsHttp() {
		err = initHttpHandler(ps, url)
		if err != nil {
			return nil, err
		}
	} else if ps.proxyCfg.IsWS() {
		err = initWSHandler(ps)
		if err != nil {
			return nil, err
		}
	}
	ps.srv = &http.Server{
		Handler:      ps.mux,
		Addr:         ps.proxyCfg.ProxyAddr,
		WriteTimeout: time.Duration(ps.proxyCfg.WriteTimeout) * time.Second,
		ReadTimeout:  time.Duration(ps.proxyCfg.ReadTimeout) * time.Second,
		ErrorLog:     golog.New(log.ErrWriter, "", 0),
	}
	if pc.ProxyServerTLSConfig != nil {
		ps.srv.TLSConfig = pc.ProxyServerTLSConfig.TlsCfg
		ps.srv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler)) // disable HTTP/2 to prevent need to use ciphers with 128 bit keys
	}
	log.Info("ProxyServer - created proxy server for config", "cfg", *pc)
	return ps, nil
}

func initHttpHandler(ps *ProxyServer, url *url.URL) error {
	ps.rp = httputil.NewSingleHostReverseProxy(url)

	if ps.proxyCfg.ClientTLSConfig != nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = ps.proxyCfg.ClientTLSConfig.TlsCfg //TODO(cjh) may not work, may have to set rpTransport.DialTLSContext
		ps.rp.Transport = transport
	}
	ps.rp.ModifyResponse = func(res *http.Response) error {
		respStatus := res.Status
		log.Debug("initHttpHandler - response status", "status", respStatus, "code", res.StatusCode)
		return nil
	}
	h, err := makeHttpHandler(ps)
	if err != nil {
		return err
	}
	for _, p := range ps.proxyCfg.ProxyPaths {
		ps.mux.Handle(p, h)
		log.Info("initHttpHandler - registering http handler", "proxyAddr", ps.proxyCfg.ProxyAddr, "upstrAddr", ps.proxyCfg.UpstreamAddr, "name", ps.proxyCfg.Name, "type", ps.proxyCfg.Type, "path", p)
	}
	return nil
}

func initWSHandler(ps *ProxyServer) error {
	//TODO(cjh) support TLS
	var err error
	if ps.wp, err = WSProxyHandler(ps, ps.proxyCfg.UpstreamAddr); err != nil {
		return err
	}
	for _, p := range ps.proxyCfg.ProxyPaths {
		ps.mux.Handle(p, ps.wp)
		log.Info("initWSHandler - registering WS handler", "proxyAddr", ps.proxyCfg.ProxyAddr, "upstrAddr", ps.proxyCfg.UpstreamAddr, "name", ps.proxyCfg.Name, "type", ps.proxyCfg.Type, "path", p)
	}
	return nil
}

// Start starts the proxy server
func (ps ProxyServer) Start() {
	ps.shutdownWg.Add(1)
	go func() {
		defer ps.shutdownWg.Done()
		log.Info("Start - ListenAndServe started", "proxyAddr", ps.proxyCfg.ProxyAddr, "upstream", ps.proxyCfg.UpstreamAddr)

		var err error

		if ps.proxyCfg.ProxyServerTLSConfig != nil {
			log.Debug("Starting TLS-enabled proxy server", "proxyAddr", ps.proxyCfg.ProxyAddr, "upstream", ps.proxyCfg.UpstreamAddr)
			err = ps.srv.ListenAndServeTLS("", "")
		} else {
			log.Debug("Starting proxy server", "proxyAddr", ps.proxyCfg.ProxyAddr, "upstream", ps.proxyCfg.UpstreamAddr)
			err = ps.srv.ListenAndServe()
		}

		if err != nil {
			log.Error("Start - ListenAndServe failed", "proxyAddr", ps.proxyCfg.ProxyAddr, "upstream", ps.proxyCfg.UpstreamAddr, "err", err)
			ps.errCh <- err
		}
	}()
}

// Stop stops the proxy server
func (ps ProxyServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if ps.srv != nil {
		if err := ps.srv.Shutdown(ctx); err != nil {
			log.Error("Stop - failed to shutdown", "name", ps.proxyCfg.Name, "err", err)
		}
		ps.shutdownWg.Wait()
		log.Info("Stop - server shutdown completed", "name", ps.proxyCfg.Name)
	}
}
