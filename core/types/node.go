package types

import (
	"errors"
	"fmt"
	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/naoina/toml"
	"net/http"
	"os"
)

type NodeConfig struct {
	BasicConfig  *BasicConfig         `toml:"basicConfig"` // basic config of this node manager
	NodeManagers NodeManagerConfigArr // node manager config of other node manager
}

func ReadNodeConfig(configFile string) (NodeConfig, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return NodeConfig{}, err
	}
	defer f.Close()
	var input NodeConfig
	if err = toml.NewDecoder(f).Decode(&input); err != nil {
		return NodeConfig{}, err
	}

	// check if the config is valid
	if input.BasicConfig == nil {
		return NodeConfig{}, errors.New("invalid configuration passed")
	}

	// validate config rules
	if err = input.BasicConfig.IsValid(); err != nil {
		return NodeConfig{}, err
	}

	// default populate the run mode to strict
	if input.BasicConfig.RunMode == "" {
		input.BasicConfig.RunMode = STRICT_MODE
	}

	return input, nil
}

func (c NodeConfig) IsConsensusValid(client *http.Client) error {
	const (
		adminInfoReq = `{"jsonrpc":"2.0", "method":"admin_nodeInfo", "params":[], "id":67}`
		protocolKey  = "protocols"
		ethKey       = "eth"
		consensusKey = "consensus"
		istanbulKey  = "istanbul"
	)
	log.Debug("IsConsensusValid - validating consensus info")

	if c.BasicConfig.IsBesuClient() {
		return nil
	}

	var resp map[string]interface{}
	if err := core.CallRPC(client, c.BasicConfig.BcClntRpcUrl, []byte(adminInfoReq), &resp); err == nil {
		resMap := resp["result"].(map[string]interface{})
		log.Debug("IsConsensusValid - response", "map", resMap)

		if resMap[protocolKey] == nil {
			return errors.New("IsConsensusValid - no consensus info found")
		}
		protocols, ok := resMap[protocolKey].(map[string]interface{})
		if !ok {
			return errors.New("IsConsensusValid - invalid consensus info found")
		}
		if protocols[istanbulKey] != nil {
			if c.BasicConfig.IsIstanbul() {
				return nil
			}
			return errors.New("IsConsensusValid - invalid consensus. it should be istanbul")
		}
		eth := protocols[ethKey].(map[string]interface{})
		if _, ok := eth[consensusKey]; !ok {
			return fmt.Errorf("IsConsensusValid - consensus key missing in node info api output")
		} else {
			expected := eth[consensusKey].(string)
			log.Debug("IsConsensusValid - consensus name", "name", expected)
			if expected == c.BasicConfig.Consensus {
				return nil
			}
			return fmt.Errorf("IsConsensusValid - consensus mismatch. expected:%s, have:%s", expected, c.BasicConfig.Consensus)
		}
	}
	return nil
}

func ReadNodeManagerConfig(configFile string) ([]*NodeManagerConfig, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var input NodeManagerListConfig
	if err = toml.NewDecoder(f).Decode(&input); err != nil {
		return nil, err
	}
	if err := input.NodeManagers.IsValid(); err != nil {
		return nil, err
	}

	return input.NodeManagers, nil
}

func (c NodeConfig) IsValid() error {
	if c.BasicConfig == nil {
		return errors.New("basicConfig is nil")
	}
	if err := c.BasicConfig.IsValid(); err != nil {
		return fmt.Errorf("invalid basicConfig: %v", err)
	}
	if err := c.NodeManagers.IsValid(); err != nil {
		return fmt.Errorf("invalid nodeManagers config: %v", err)
	}
	return nil
}
