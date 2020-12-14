package config

import (
	"errors"
	"fmt"
	"strings"
)

type QuorumClient struct {
	ClientType      string     `toml:"clientType" json:"clientType"`                       // client used by this node manager. it should be quorum or besu
	Consensus       string     `toml:"consensus" json:"consensus"`                         // consensus used by blockchain client. ex: raft / istanbul / clique
	BcClntRpcUrl    string     `toml:"quorumClientRpcUrl" json:"quorumClientRpcUrl"`       // RPC url of blockchain client managed by this node manager
	BcClntTLSConfig *ClientTLS `toml:"quorumClientTlsConfig" json:"quorumClientTlsConfig"` // blockchain client TLS config
	BcClntProcess   *Process   `toml:"quorumClientProcess" json:"quorumClientProcess"`     // blockchain client process managed by this node manager
}

type PrivacyManager struct {
	PrivManKey       string     `toml:"privacyManagerKey" json:"privacyManagerKey"`             // public key of privacy manager managed by this node manager
	PrivManTLSConfig *ClientTLS `toml:"privacyManagerTlsConfig" json:"privacyManagerTlsConfig"` // Privacy manager TLS config
	PrivManProcess   *Process   `toml:"privacyManagerProcess" json:"privacyManagerProcess"`     // privacy manager process managed by this node manager
}

func (c *QuorumClient) IsRaft() bool {
	return strings.ToLower(c.Consensus) == "raft"
}

func (c *QuorumClient) IsIstanbul() bool {
	return strings.ToLower(c.Consensus) == "istanbul"
}

func (c *QuorumClient) IsClique() bool {
	return strings.ToLower(c.Consensus) == "clique"
}

func (c *QuorumClient) IsQuorumClient() bool {
	return strings.ToLower(c.ClientType) == "quorum"
}

func (c *QuorumClient) IsBesuClient() bool {
	return strings.ToLower(c.ClientType) == "besu"
}

func (c *QuorumClient) IsValid() error {
	if c.Consensus == "" {
		return errors.New("consensus is empty")
	}

	if !c.IsRaft() && !c.IsClique() && !c.IsIstanbul() {
		return errors.New("invalid consensus name. supports only raft or istanbul or clique")
	}

	if c.ClientType == "" {
		return errors.New("clientType is empty")
	}
	if !c.IsQuorumClient() && !c.IsBesuClient() {
		return errors.New("invalid clientType. supports only quorum or besu")
	}

	if c.BcClntProcess == nil {
		return errors.New("bcClntProcess is empty")
	}

	if c.BcClntRpcUrl == "" {
		return errors.New("bcClntRpcUrl is empty")
	}

	if err := c.BcClntProcess.IsValid(); err != nil {
		return fmt.Errorf("invalid bcClntProcess: %v", err)
	}

	if c.BcClntTLSConfig != nil {
		if err := c.BcClntTLSConfig.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

func (c *PrivacyManager) IsValid() error {
	if c.PrivManProcess != nil {

		if c.PrivManKey == "" {
			return errors.New("privManKey is empty")
		}

		if err := c.PrivManProcess.IsValid(); err != nil {
			return fmt.Errorf("invalid privManProcess: %v", err)
		}
	}

	if c.PrivManTLSConfig != nil {
		if err := c.PrivManTLSConfig.IsValid(); err != nil {
			return err
		}
	}

	return nil
}
