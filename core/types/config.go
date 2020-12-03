package types

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/naoina/toml"
)

// namedValidationError provides additional context to an error, useful for providing context when there is a validation error with an element in an array
type namedValidationError struct {
	name, errMsg string
}

func (e namedValidationError) Error() string {
	return fmt.Sprintf("name = %v: %v", e.name, e.errMsg)
}

type ProxyConfig struct {
	Name         string   `toml:"name"`         // name of node manager process
	Type         string   `toml:"type"`         // proxy scheme - http or ws
	ProxyAddr    string   `toml:"proxyAddr"`    // proxy address
	UpstreamAddr string   `toml:"upstreamAddr"` // upstream address of the proxy address
	ProxyPaths   []string `toml:"proxyPaths"`   // httpRequestURI paths of the upstream address
	// httpRequestURI paths of the upstream address that should be ignored for activity
	IgnorePathsForActivity []string `toml:"ignorePathsForActivity"`
	ReadTimeout            int      `toml:"readTimeout"`  // readTimeout of the proxy server
	WriteTimeout           int      `toml:"writeTimeout"` // writeTimeout of the proxy server
}

func (c ProxyConfig) IsHttp() bool {
	return strings.ToLower(c.Type) == "http"
}

func (c ProxyConfig) IsWS() bool {
	return strings.ToLower(c.Type) == "ws"
}

// IsValid returns nil if the ProxyConfig is valid else returns error
func (c ProxyConfig) IsValid() error {
	if c.Name == "" {
		return errors.New("name is empty")
	}
	if !c.IsWS() && !c.IsHttp() {
		return namedValidationError{name: c.Name, errMsg: "invalid type. supports only http or ws"}
	}
	if c.ProxyAddr == "" {
		return namedValidationError{name: c.Name, errMsg: "proxyAddr is empty"}
	}
	if c.UpstreamAddr == "" {
		return namedValidationError{name: c.Name, errMsg: "upstreamAddr is empty"}
	}
	if err := isValidUrl(c.ProxyAddr); err != nil {
		return namedValidationError{name: c.Name, errMsg: fmt.Sprintf("invalid proxyAddr: %v", err)}
	}
	if err := isValidUrl(c.UpstreamAddr); err != nil {
		return namedValidationError{name: c.Name, errMsg: fmt.Sprintf("invalid upstreamAddr: %v", err)}
	}
	if len(c.ProxyPaths) == 0 {
		return namedValidationError{name: c.Name, errMsg: "proxyPaths is empty"}
	}
	if c.ReadTimeout == 0 {
		return namedValidationError{name: c.Name, errMsg: "readTimeout is zero"}
	}
	if c.WriteTimeout == 0 {
		return namedValidationError{name: c.Name, errMsg: "writeTimeout is zero"}
	}
	return nil
}

type NodeManagerConfig struct {
	Name       string `toml:"name"`       // Name of the other node manager
	PrivManKey string `toml:"privManKey"` // PrivManKey managed by the other node manager
	RpcUrl     string `toml:"rpcUrl"`     // RPC url of the other node manager
}

// IsValid returns nil if the NodeManagerConfig is valid else returns error
func (c NodeManagerConfig) IsValid() error {
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
	return nil
}

type ProcessConfig struct {
	Name         string   `toml:"name"`         // name of process. should be bcclnt or privman
	ControlType  string   `toml:"controlType"`  // control type supported. shell or docker
	ContainerId  string   `toml:"containerId"`  // docker container id. required if controlType is docker
	StopCommand  []string `toml:"stopCommand"`  // stop command. required if controlType is shell
	StartCommand []string `toml:"startCommand"` // start command. required if controlType is shell
}

func (c ProcessConfig) IsShell() bool {
	return strings.ToLower(c.ControlType) == "shell"
}

func (c ProcessConfig) IsDocker() bool {
	return strings.ToLower(c.ControlType) == "docker"
}

func (c ProcessConfig) IsBcClient() bool {
	return strings.ToLower(c.Name) == "bcclnt"
}

func (c ProcessConfig) IsPrivacyManager() bool {
	return strings.ToLower(c.Name) == "privman"
}

func (c ProcessConfig) IsValid() error {
	if !c.IsDocker() && !c.IsShell() {
		return errors.New("invalid controlType. supports only shell or docker")
	}
	if !c.IsBcClient() && !c.IsPrivacyManager() {
		return errors.New("invalid name. supports only bcclnt or privman")
	}
	if c.IsDocker() && c.ContainerId == "" {
		return errors.New("containerId is empty for docker controlType")
	}
	if c.IsShell() && (len(c.StartCommand) == 0 || len(c.StopCommand) == 0) {
		return errors.New("startCommand or stopCommand is empty for shell controlType")
	}
	return nil
}

type RPCServerConfig struct {
	RpcAddr     string   `toml:"rpcAddr"`
	RPCCorsList []string `toml:"rpcCorsList"`
	RPCVHosts   []string `toml:"rpcvHosts"`
}

func (c RPCServerConfig) IsValid() error {
	if c.RpcAddr == "" {
		return errors.New("rpcAddr is empty")
	}
	if err := isValidUrl(c.RpcAddr); err != nil {
		return fmt.Errorf("invalid rpcAddr: %v", err)
	}
	return nil
}

type BasicConfig struct {
	Name                  string           `toml:"name"`                   // name of this node manager
	BcClntRpcUrl          string           `toml:"bcClntRpcUrl"`           // RPC url of blockchain client managed by this node manager
	PrivManUpcheckUrl     string           `toml:"privManUpcheckUrl"`      // Upcheck url of privacy manager managed by this node manager
	PrivManKey            string           `toml:"privManKey"`             // public key of privacy manager managed by this node manager
	Consensus             string           `toml:"consensus"`              // consensus used by blockchain client. ex: raft / istanbul / clique
	ClientType            string           `toml:"clientType"`             // client used by this node manager. it should be quorum or besu
	UpchkPollingInterval  int              `toml:"upcheckPollingInterval"` // up check polling interval in seconds for the node
	NodeManagerConfigFile string           `toml:"nodeManagerConfigFile"`  // node manager config file path
	InactivityTime        int              `toml:"inactivityTime"`         // inactivity time for blockchain client and privacy manager
	Server                *RPCServerConfig `toml:"server"`                 // RPC server config of this node manager
	BcClntProcess         *ProcessConfig   `toml:"bcClntProcess"`          // blockchain client process managed by this node manager
	PrivManProcess        *ProcessConfig   `toml:"privManProcess"`         // privacy manager process managed by this node manager
	Proxies               []*ProxyConfig   `toml:"proxies"`                // proxies managed by this node manager
}

type NodeConfig struct {
	BasicConfig  *BasicConfig         `toml:"basicConfig"` // basic config of this node manager
	NodeManagers NodeManagerConfigArr // node manager config of other node manager
}

type NodeManagerConfigArr []*NodeManagerConfig

func (a *NodeManagerConfigArr) IsValid() error {
	for _, c := range *a {
		if err := c.IsValid(); err != nil {
			return err
		}
	}
	return nil
}

type NodeManagerListConfig struct {
	NodeManagers NodeManagerConfigArr `toml:"nodeManagers"` // node manger config list of other node manager
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

	return input, nil
}

func (c NodeConfig) IsConsensusValid() error {
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
	if err := core.CallRPC(c.BasicConfig.BcClntRpcUrl, []byte(adminInfoReq), &resp); err == nil {
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

func (c BasicConfig) IsRaft() bool {
	return strings.ToLower(c.Consensus) == "raft"
}

func (c BasicConfig) IsIstanbul() bool {
	return strings.ToLower(c.Consensus) == "istanbul"
}

func (c BasicConfig) IsClique() bool {
	return strings.ToLower(c.Consensus) == "clique"
}

func (c BasicConfig) IsQuorumClient() bool {
	return strings.ToLower(c.ClientType) == "quorum"
}

func (c BasicConfig) IsBesuClient() bool {
	return strings.ToLower(c.ClientType) == "besu"
}

func (c BasicConfig) IsValid() error {
	if c.Name == "" {
		return errors.New("name is empty")
	}

	if c.NodeManagerConfigFile == "" {
		return errors.New("nodeManagerConfigFile is empty")
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
		return errors.New("up check polling interval must be greater than zero")
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

	if c.Server == nil {
		return errors.New("server is empty")
	}

	if err := c.BcClntProcess.IsValid(); err != nil {
		return fmt.Errorf("invalid bcClntProcess: %v", err)
	}

	if c.PrivManProcess != nil {
		if c.PrivManUpcheckUrl == "" {
			return errors.New("privManUpcheckUrl is empty")
		}

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

	return nil
}

func (c BasicConfig) isConsensusValid() error {
	if c.Consensus == "" {
		return errors.New("consensus is empty")
	}

	if !c.IsRaft() && !c.IsClique() && !c.IsIstanbul() {
		return errors.New("invalid consensus name. supports only raft or istanbul or clique")
	}
	return nil
}

func (c BasicConfig) IsClientTypeValid() error {
	if c.ClientType == "" {
		return errors.New("client type is empty")
	}
	if !c.IsQuorumClient() && !c.IsBesuClient() {
		return errors.New("invalid client type. supports only quorum or besu")
	}
	return nil
}

func isValidUrl(addr string) error {
	_, err := url.Parse(addr)
	return err
}
