package proxy

import "github.com/ConsenSysQuorum/node-manager/node"

func MakeProxyServices(qn *node.QuorumNodeControl, errc chan error) ([]Proxy, error) {
	var proxies []Proxy
	for _, c := range qn.GetProxyConfig() {
		if p, err := NewProxyServer(qn, c, errc); err != nil {
			return nil, err
		} else {
			proxies = append(proxies, p)
		}
	}
	return proxies, nil
}
