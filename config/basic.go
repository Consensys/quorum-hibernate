package config

import (
	"errors"
	"fmt"
)

type Basic struct {
	Name                 string          `toml:"name" json:"name"`                                     // name of this node manager
	DisableStrictMode    bool            `toml:"disableStrictMode" json:"disableStrictMode"`           // strict mode keeps consensus nodes alive always
	UpchkPollingInterval int             `toml:"upcheckPollingInterval" json:"upcheckPollingInterval"` // up check polling interval in seconds for the quorumClient and privacyManager
	PeersConfigFile      string          `toml:"peersConfigFile" json:"peersConfigFile"`               // node manager config file path
	InactivityTime       int             `toml:"inactivityTime" json:"inactivityTime"`                 // inactivity time for blockchain client and privacy manager
	ResyncTime           int             `toml:"resyncTime" json:"resyncTime"`                         // time after which client should be started to sync up with network
	QuorumClient         *QuorumClient   `toml:"quorumClient" json:"quorumClient"`
	PrivacyManager       *PrivacyManager `toml:"privacyManager" json:"privacyManager"`
	Server               *RPCServer      `toml:"server" json:"server"`   // RPC server config of this node manager
	Proxies              []*Proxy        `toml:"proxies" json:"proxies"` // proxies managed by this node manager
}

func (c Basic) IsResyncTimerSet() bool {
	return c.ResyncTime != 0
}

func (c Basic) IsRaft() bool {
	return c.QuorumClient.IsRaft()
}

func (c Basic) IsIstanbul() bool {
	return c.QuorumClient.IsIstanbul()
}

func (c Basic) IsClique() bool {
	return c.QuorumClient.IsClique()
}

func (c Basic) IsQuorumClient() bool {
	return c.QuorumClient.IsQuorumClient()
}

func (c Basic) IsBesuClient() bool {
	return c.QuorumClient.IsBesuClient()
}

func (c Basic) IsValid() error {
	if c.Name == "" {
		return errors.New("name is empty")
	}

	if c.PeersConfigFile == "" {
		return errors.New("peersConfigFile is empty")
	}

	if c.UpchkPollingInterval <= 0 {
		return errors.New("upcheckPollingInterval must be greater than zero")
	}

	if c.InactivityTime < 60 {
		return errors.New("inactivityTime must be greater than or equal to 60 (seconds)")
	}

	if c.IsResyncTimerSet() && c.ResyncTime < c.InactivityTime {
		return errors.New("resyncTime must be reasonably greater than the inactivityTime")
	}

	if c.Server == nil {
		return errors.New("server is empty")
	}

	if c.QuorumClient == nil {
		return errors.New("quorumClient is empty")
	}

	if err := c.QuorumClient.IsValid(); err != nil {
		return fmt.Errorf("invalid quorumClient: %v", err)
	}

	if c.PrivacyManager != nil {
		if err := c.PrivacyManager.IsValid(); err != nil {
			return fmt.Errorf("invalid privManProcess: %v", err)
		}
	}

	if err := c.Server.IsValid(); err != nil {
		return err
	}

	if len(c.Proxies) == 0 {
		return errors.New("proxies is empty")
	}

	for _, n := range c.Proxies {
		if err := n.IsValid(); err != nil {
			return fmt.Errorf("invalid proxies config: %v", err)
		}
	}

	return nil
}
