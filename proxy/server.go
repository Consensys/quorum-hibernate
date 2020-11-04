package proxy

import (
	"fmt"
	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
	"net/http"
)

func NewProxyServer(qn *node.QuorumNode, errc chan error) Proxy {
	return &ProxyServer{qn, qn.GetProxyAddr(), errc}
}

func (np ProxyServer) Start() {
	go func() {
		for _, p := range np.qrmNode.GetProxyConfig() {
			var path string
			if p.Path == "/" {
				path = p.Name
			} else {
				path = fmt.Sprintf("/%s", p.Path)
			}

			if p.IsHttp() {
				handler, err := makeHttpHandler(np.qrmNode, p.DestUrl)
				if err != nil {
					np.errCh <- err
					return
				}
				http.HandleFunc(path, handler)
			} else if p.IsWS() {
				http.HandleFunc(path, makeWSHandler(np.qrmNode, p.DestUrl))
			}
			log.Info("added handler for proxy", "name", p.Name, "path", p.Path, "destUrl", p.DestUrl)
		}

		log.Info("ListenAndServe started", "proxyAddr", np.proxyAddr)
		err := http.ListenAndServe(np.proxyAddr, nil)
		if err != nil {
			log.Error("ListenAndServe failed", "proxyAddr", np.proxyAddr, "err", err)
			np.errCh <- err
		}
	}()
}

func (np ProxyServer) Stop() {
	panic("not implemented")
}
