package proxy

import (
	"github.com/ConsenSysQuorum/node-manager/node"
)

type ProxyServer struct {
	qrmNode   *node.QuorumNode
	proxyAddr string
	errCh     chan error
}

type Proxy interface {
	Start()
	Stop()
}
