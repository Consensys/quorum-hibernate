package config

type PeerArr []*Peer

func (a *PeerArr) IsValid() error {
	for i, c := range *a {
		if err := c.IsValid(); err != nil {
			return newArrFieldErr("peers", i, err)
		}
	}
	return nil
}

type NodeManagerList struct {
	Peers PeerArr `toml:"peers" json:"peers"` // node manger config list of other node manager
}

type Peer struct {
	Name       string     `toml:"name" json:"name"`                           // Name of the other node manager
	PrivManKey string     `toml:"privacyManagerKey" json:"privacyManagerKey"` // PrivManKey managed by the other node manager
	RpcUrl     string     `toml:"rpcUrl" json:"rpcUrl"`                       // RPC url of the other node manager
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
	if err := isValidUrl(c.RpcUrl); err != nil {
		return newFieldErr("rpcUrl", err)
	}
	if c.TLSConfig != nil {
		if err := c.TLSConfig.IsValid(); err != nil {
			return newFieldErr("tlsConfig", err)
		}
	}
	return nil
}
