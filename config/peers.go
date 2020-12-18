package config

import (
	"errors"
	"fmt"
)

type PeerArr []*Peer

func (a *PeerArr) IsValid() error {
	for _, c := range *a {
		if err := c.IsValid(); err != nil {
			return err
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
		return errors.New("name is empty")
	}
	if c.RpcUrl == "" {
		return namedValidationError{name: c.Name, errMsg: "rpcUrl is empty"}
	}
	if err := isValidUrl(c.RpcUrl); err != nil {
		return namedValidationError{name: c.Name, errMsg: fmt.Sprintf("invalid rpcUrl: %v", err)}
	}
	if c.TLSConfig != nil {
		if err := c.TLSConfig.IsValid(); err != nil {
			return err
		}
	}
	return nil
}
