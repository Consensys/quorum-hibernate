package config

import (
	"errors"
	"fmt"
	"strings"
)

type Basic struct {
	Name                 string     `toml:"name" json:"name"`                                       // name of this node manager
	DisableStrictMode    bool       `toml:"disableStrictMode" json:"disableStrictMode"`             // strict mode keeps consensus nodes alive always
	BcClntRpcUrl         string     `toml:"quorumClientRpcUrl" json:"quorumClientRpcUrl"`           // RPC url of blockchain client managed by this node manager
	BcClntTLSConfig      *ClientTLS `toml:"quorumClientTlsConfig" json:"quorumClientTlsConfig"`     // blockchain client TLS config
	PrivManTLSConfig     *ClientTLS `toml:"privacyManagerTlsConfig" json:"privacyManagerTlsConfig"` // Privacy manager TLS config
	PrivManKey           string     `toml:"privacyManagerKey" json:"privacyManagerKey"`             // public key of privacy manager managed by this node manager
	Consensus            string     `toml:"consensus" json:"consensus"`                             // consensus used by blockchain client. ex: raft / istanbul / clique
	ClientType           string     `toml:"clientType" json:"clientType"`                           // client used by this node manager. it should be quorum or besu
	UpchkPollingInterval int        `toml:"upcheckPollingInterval" json:"upcheckPollingInterval"`   // up check polling interval in seconds for the node
	PeersConfigFile      string     `toml:"peersConfigFile" json:"peersConfigFile"`                 // node manager config file path
	InactivityTime       int        `toml:"inactivityTime" json:"inactivityTime"`                   // inactivity time for blockchain client and privacy manager
	ResyncTime           int        `toml:"resyncTime" json:"resyncTime"`                           // time after which client should be started to sync up with network
	Server               *RPCServer `toml:"server" json:"server"`                                   // RPC server config of this node manager
	BcClntProcess        *Process   `toml:"quorumClientProcess" json:"quorumClientProcess"`         // blockchain client process managed by this node manager
	PrivManProcess       *Process   `toml:"privacyManagerProcess" json:"privacyManagerProcess"`     // privacy manager process managed by this node manager
	Proxies              []*Proxy   `toml:"proxies" json:"proxies"`                                 // proxies managed by this node manager
}

func (c Basic) IsRaft() bool {
	return strings.ToLower(c.Consensus) == "raft"
}

func (c Basic) IsResyncTimerSet() bool {
	return c.ResyncTime != 0
}

func (c Basic) IsIstanbul() bool {
	return strings.ToLower(c.Consensus) == "istanbul"
}

func (c Basic) IsClique() bool {
	return strings.ToLower(c.Consensus) == "clique"
}

func (c Basic) IsQuorumClient() bool {
	return strings.ToLower(c.ClientType) == "quorum"
}

func (c Basic) IsBesuClient() bool {
	return strings.ToLower(c.ClientType) == "besu"
}

func (c Basic) IsValid() error {
	if c.Name == "" {
		return errors.New("name is empty")
	}

	if c.PeersConfigFile == "" {
		return errors.New("peersConfigFile is empty")
	}

	err := c.isConsensusValid()
	if err != nil {
		return err
	}

	err = c.IsClientTypeValid()
	if err != nil {
		return err
	}

	if c.UpchkPollingInterval <= 0 {
		return errors.New("upcheckPollingInterval must be greater than zero")
	}

	if c.BcClntProcess == nil {
		return errors.New("bcClntProcess is empty")
	}

	if c.BcClntRpcUrl == "" {
		return errors.New("bcClntRpcUrl is empty")
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

	if err := c.BcClntProcess.IsValid(); err != nil {
		return fmt.Errorf("invalid bcClntProcess: %v", err)
	}

	if c.PrivManProcess != nil {

		if c.PrivManKey == "" {
			return errors.New("privManKey is empty")
		}

		if err := c.PrivManProcess.IsValid(); err != nil {
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

	if c.BcClntTLSConfig != nil {
		if err := c.BcClntTLSConfig.IsValid(); err != nil {
			return err
		}
	}

	if c.PrivManTLSConfig != nil {
		if err := c.BcClntTLSConfig.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

func (c Basic) isConsensusValid() error {
	if c.Consensus == "" {
		return errors.New("consensus is empty")
	}

	if !c.IsRaft() && !c.IsClique() && !c.IsIstanbul() {
		return errors.New("invalid consensus name. supports only raft or istanbul or clique")
	}
	return nil
}

func (c Basic) IsClientTypeValid() error {
	if c.ClientType == "" {
		return errors.New("clientType is empty")
	}
	if !c.IsQuorumClient() && !c.IsBesuClient() {
		return errors.New("invalid clientType. supports only quorum or besu")
	}
	return nil
}
