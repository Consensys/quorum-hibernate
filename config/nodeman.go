package config

import (
	"errors"
	"fmt"
)

type NodeManagerArr []*NodeManager

func (a *NodeManagerArr) IsValid() error {
	for _, c := range *a {
		if err := c.IsValid(); err != nil {
			return err
		}
	}
	return nil
}

type NodeManagerList struct {
	NodeManagers NodeManagerArr `toml:"nodeManagers"` // node manger config list of other node manager
}

type NodeManager struct {
	Name       string     `toml:"name"`       // Name of the other node manager
	PrivManKey string     `toml:"privManKey"` // PrivManKey managed by the other node manager
	RpcUrl     string     `toml:"rpcUrl"`     // RPC url of the other node manager
	TLSConfig  *ClientTLS `toml:"tlsConfig"`  // tls config
}

// IsValid returns nil if the NodeManager is valid else returns error
func (c NodeManager) IsValid() error {
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
