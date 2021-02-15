package config

import "net/url"

type PeerArr []*Peer

func (a *PeerArr) IsValid() error {
	nameList := make(map[string]bool, len(*a))
	for i, c := range *a {
		// check if the name is duplicate
		if _, ok := nameList[c.Name]; ok {
			return newArrFieldErr("peers", i, newFieldErr("name", isNotUniqueErr))
		}

		// validate peer entry
		if err := c.IsValid(); err != nil {
			return newArrFieldErr("peers", i, err)
		}
		nameList[c.Name] = true
	}
	return nil
}

type NodeHibernatorList struct {
	Peers PeerArr `toml:"peers" json:"peers"` // node hibernator config list of other node hibernator
}

type Peer struct {
	Name       string     `toml:"name" json:"name"`                           // Name of the other node hibernator
	PrivManKey string     `toml:"privacyManagerKey" json:"privacyManagerKey"` // PrivManKey managed by the other node hibernator
	RpcUrl     string     `toml:"rpcUrl" json:"rpcUrl"`                       // RPC url of the other node hibernator
	TLSConfig  *ClientTLS `toml:"tlsConfig" json:"tlsConfig"`                 // tls config
}

// IsValid returns nil if the Peer is valid else returns error
func (c Peer) IsValid() error {
	if c.Name == "" {
		return newFieldErr("name", isEmptyErr)
	}
	if c.RpcUrl == "" {
		return newFieldErr("rpcUrl", isEmptyErr)
	}
	if _, err := url.Parse(c.RpcUrl); err != nil {
		return newFieldErr("rpcUrl", err)
	}
	if c.TLSConfig != nil {
		if err := c.TLSConfig.IsValid(); err != nil {
			return newFieldErr("tlsConfig", err)
		}
	}
	return nil
}
