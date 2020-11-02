package proxy

import (
	"net/http/httputil"

	"github.com/ConsenSysQuorum/node-manager/node"
)

type ProxyServer struct {
	qrmNode  *node.QuorumNode
	name     string
	destUrl  string
	proxyUrl string
	rp       *httputil.ReverseProxy
}

type Proxy interface {
	Start()
	Stop()
}
