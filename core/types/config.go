package types

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/ConsenSysQuorum/node-manager/log"

	"github.com/naoina/toml"
)

type ProxyConfig struct {
	Name         string   `toml:"name"`         // Name of qnm process
	Type         string   `toml:"type"`         // proxy scheme - http or ws
	ProxyAddr    string   `toml:"proxyAddr"`    // proxy address
	UpstreamAddr string   `toml:"upstreamAddr"` // upstream address of the proxy address
	ProxyPaths   []string `toml:"proxyPaths"`   // httpRequestURI paths of the upstream address
	ReadTimeout  int      `toml:"readTimeout"`  // readTimeout of the proxy server
	WriteTimeout int      `toml:"writeTimeout"` // writeTimeout of the proxy server
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
		return errors.New("proxy config name is empty.")
	}
	if !c.IsWS() && !c.IsHttp() {
		return errors.New("proxy config - unsupported proxy type. supports http or ws only.")
	}
	if c.ProxyAddr == "" {
		return errors.New("proxy config - proxyAddr is empty.")
	}
	if c.UpstreamAddr == "" {
		return errors.New("proxy config -  upstreamAddr is empty.")
	}
	if !isValidUrl(c.ProxyAddr) {
		return errors.New("proxy addr is invalid")
	}
	if !isValidUrl(c.UpstreamAddr) {
		return errors.New("proxy upstreamAddr is invalid")
	}
	if len(c.ProxyPaths) == 0 {
		return errors.New("proxy paths is empty")
	}

	if c.ReadTimeout == 0 {
		return errors.New("proxy readTimeout is zero")
	}

	if c.WriteTimeout == 0 {
		return errors.New("proxy readTimeout is zero")
	}
	return nil
}

type NodeManagerConfig struct {
	Name       string `toml:"name"`       // Name of the other qnm
	TesseraKey string `toml:"tesseraKey"` // TesseraKey managed by the other qnm
	RpcUrl     string `toml:"rpcUrl"`     // RPC url of the other qnm
}

// IsValid returns nil if the NodeManagerConfig is valid else returns error
func (c NodeManagerConfig) IsValid() error {
	if c.Name == "" {
		return errors.New("node manager name is empty.")
	}
	if c.TesseraKey == "" {
		return errors.New("node manager tesseraKey is empty.")
	}
	if c.RpcUrl == "" {
		return errors.New("node manager rpcUrl is empty.")
	}
	if !isValidUrl(c.RpcUrl) {
		return errors.New("node manager invalid rpc url")
	}
	return nil
}

type ProcessConfig struct {
	Name         string   `toml:"name"`         // name of process. ex: geth / tessera
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

func (c ProcessConfig) IsValid() error {
	if !c.IsDocker() && !c.IsShell() {
		return errors.New("unsupported controlType. processConfig supports only shell or docker")
	}
	if c.IsDocker() && c.ContainerId == "" {
		return errors.New("containerId is empty for docker controlType.")
	}
	if c.IsShell() && (len(c.StartCommand) == 0 || len(c.StopCommand) == 0) {
		return errors.New("startCommand or stopCommand is empty for shell controlType.")
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
		return errors.New("RPC server config - empty rpcAddr")
	}
	if !isValidUrl(c.RpcAddr) {
		return errors.New("RPC server config - invalid rpcAddr")
	}
	return nil
}

type BasicConfig struct {
	Name                  string           `toml:"name"`                  // name of this qnm
	GethRpcUrl            string           `toml:"gethRpcUrl"`            // RPC url of geth managed by this qnm
	TesseraUpcheckUrl     string           `toml:"tesseraUpcheckUrl"`     // Upcheck url of tessera managed by this qnm
	TesseraKey            string           `toml:"tesseraKey"`            // Tessera key of tessera managed by this qnm
	Consensus             string           `toml:"consensus"`             // consensus used by geth. ex: raft / istanbul / clique
	NodeManagerConfigFile string           `toml:"nodeManagerConfigFile"` // node manager config file path
	InactivityTime        int              `toml:"inactivityTime"`        // inactivity time for geth and tessera
	Server                *RPCServerConfig `toml:"server"`                // RPC server config of this qnm
	GethProcess           *ProcessConfig   `toml:"gethProcess"`           // geth process managed by this qnm
	TesseraProcess        *ProcessConfig   `toml:"tesseraProcess"`        // tessera process managed by this qnm
	Proxies               []*ProxyConfig   `toml:"proxies"`               // proxies managed by this qnm
}

type NodeConfig struct {
	BasicConfig  *BasicConfig         `toml:"basicConfig"`  // basic config of this qnm
	NodeManagers []*NodeManagerConfig `toml:"nodeManagers"` // node manager config of other qnms
}

type NodeManagerListConfig struct {
	NodeManagers []*NodeManagerConfig `toml:"nodeManagers"` // node manger config list of other qnms
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
	// validate config rules
	if err = input.BasicConfig.IsValid(); err != nil {
		return NodeConfig{}, err
	}
	return input, nil
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
	// validate config rules
	for _, n := range input.NodeManagers {
		if err = n.IsValid(); err != nil {
			return nil, err
		}
	}

	return input.NodeManagers, nil
}

func (c NodeConfig) IsValid() error {
	if c.BasicConfig == nil {
		return errors.New("basic config is nil")
	}
	if err := c.BasicConfig.IsValid(); err != nil {
		return err
	}
	for _, n := range c.NodeManagers {
		if err := n.IsValid(); err != nil {
			return err
		}
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

func (c BasicConfig) IsValid() error {
	if c.Name == "" {
		return errors.New("Name is empty")
	}

	if c.NodeManagerConfigFile == "" {
		return errors.New("NodeManagerConfigFile is empty")
	}

	err := c.IsConsensusValid()
	if err != nil {
		return err
	}

	if c.GethProcess == nil {
		return errors.New("geth process config is empty")
	}

	if c.TesseraProcess == nil {
		return errors.New("tessera process config is empty")
	}

	if c.GethRpcUrl == "" {
		return errors.New("geth rpc url is empty")
	}

	if c.TesseraUpcheckUrl == "" {
		return errors.New("tessera upcheck url is empty")
	}

	if c.TesseraKey == "" {
		return errors.New("enodeId is empty")
	}

	if c.InactivityTime < 60 {
		return errors.New("InactivityTime should be greater than or equal to 60seconds")
	}

	if c.Server == nil {
		return errors.New("RPC server config is nil")
	}

	if err := c.GethProcess.IsValid(); err != nil {
		return err
	}

	if err := c.TesseraProcess.IsValid(); err != nil {
		return err
	}

	if err := c.Server.IsValid(); err != nil {
		return err
	}

	if len(c.Proxies) == 0 {
		return errors.New("proxies config is empty")
	}

	for _, n := range c.Proxies {
		if err := n.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

func (c BasicConfig) IsConsensusValid() error {
	if c.Consensus == "" {
		return errors.New("consensus is empty")
	}

	if !c.IsRaft() && !c.IsClique() && !c.IsIstanbul() {
		return errors.New("invalid consensus name. supports only raft or istanbul or clique")
	}
	return nil
}

func isValidUrl(addr string) bool {
	u, err := url.Parse(addr)
	log.Debug("isValidUrl", "url", u, "err", err)
	return err == nil
}
