package config

import (
	"errors"
	"fmt"
	"github.com/naoina/toml"
	"os"
)

func ReadPeersConfig(configFile string) ([]*Peer, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var input NodeManagerList
	if err = toml.NewDecoder(f).Decode(&input); err != nil {
		return nil, err
	}
	if err := input.Peers.IsValid(); err != nil {
		return nil, err
	}

	return input.Peers, nil
}

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
	Peers PeerArr `toml:"peers"` // node manger config list of other node manager
}

type Peer struct {
	Name       string     `toml:"name"`       // Name of the other node manager
	PrivManKey string     `toml:"privManKey"` // PrivManKey managed by the other node manager
	RpcUrl     string     `toml:"rpcUrl"`     // RPC url of the other node manager
	TLSConfig  *ClientTLS `toml:"tlsConfig"`  // tls config
}

// IsValid returns nil if the Peer is valid else returns error
func (c Peer) IsValid() error {
	if c.Name == "" {
		return errors.New("name is empty")
	}
	if c.PrivManKey == "" {
		return namedValidationError{name: c.Name, errMsg: "privManKey is empty"}
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
