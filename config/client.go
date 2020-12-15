package config

import (
	"errors"
	"fmt"
	"strings"
)

type BlockchainClient struct {
	ClientType      string     `toml:"type" json:"type"`           // client used by this node manager. it should be quorum or besu
	Consensus       string     `toml:"consensus" json:"consensus"` // consensus used by blockchain client. ex: raft / istanbul / clique
	BcClntRpcUrl    string     `toml:"rpcUrl" json:"rpcUrl"`       // RPC url of blockchain client managed by this node manager
	BcClntTLSConfig *ClientTLS `toml:"tlsConfig" json:"tlsConfig"` // blockchain client TLS config
	BcClntProcess   *Process   `toml:"process" json:"process"`     // blockchain client process managed by this node manager
}

type PrivacyManager struct {
	PrivManKey       string     `toml:"publicKey" json:"publicKey"` // public key of privacy manager managed by this node manager
	PrivManTLSConfig *ClientTLS `toml:"tlsConfig" json:"tlsConfig"` // Privacy manager TLS config
	PrivManProcess   *Process   `toml:"process" json:"process"`     // privacy manager process managed by this node manager
}

func (c *BlockchainClient) IsRaft() bool {
	return strings.ToLower(c.Consensus) == "raft"
}

func (c *BlockchainClient) IsIstanbul() bool {
	return strings.ToLower(c.Consensus) == "istanbul"
}

func (c *BlockchainClient) IsClique() bool {
	return strings.ToLower(c.Consensus) == "clique"
}

func (c *BlockchainClient) IsGoQuorumClient() bool {
	return strings.ToLower(c.ClientType) == "goquorum"
}

func (c *BlockchainClient) IsBesuClient() bool {
	return strings.ToLower(c.ClientType) == "besu"
}

func (c *BlockchainClient) IsValid() error {
	if c.Consensus == "" {
		return errors.New("consensus is empty")
	}

	if !c.IsRaft() && !c.IsClique() && !c.IsIstanbul() {
		return errors.New("invalid consensus name. supports only raft or istanbul or clique")
	}

	if c.ClientType == "" {
		return errors.New("clientType is empty")
	}
	if !c.IsGoQuorumClient() && !c.IsBesuClient() {
		return errors.New("invalid clientType. supports only goquorum or besu")
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
