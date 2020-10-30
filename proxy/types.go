package proxy

import (
	"github.com/ConsenSysQuorum/node-manager/node"
	"net/http/httputil"
)

type ProxyServer struct {
	qrmNode   *node.QuorumNode
	name      string
	destUrl   string
	proxyPort int
	rp        *httputil.ReverseProxy
}

type Proxy interface {
	Start()
	Stop()
}
