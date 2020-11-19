package types

import (
	"errors"
	"github.com/ConsenSysQuorum/node-manager/log"
	"net/url"
	"os"
	"strings"

	"github.com/naoina/toml"
)

type ProxyConfig struct {
	Name         string   `toml:"name"`
	Type         string   `toml:"type"` // http or ws
	ProxyAddr    string   `toml:"proxyAddr"`
	UpstreamAddr string   `toml:"upstreamAddr"`
	ProxyPaths   []string `toml:"proxyPaths"`
	ReadTimeout  int      `toml:"readTimeout"`
	WriteTimeout int      `toml:"writeTimeout"`
}

func (c ProxyConfig) IsHttp() bool {
	return strings.ToLower(c.Type) == "http"
}

func (c ProxyConfig) IsWS() bool {
	return strings.ToLower(c.Type) == "ws"
}

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
	Name       string `toml:"name"`
	TesseraKey string `toml:"tesseraKey"`
	EnodeId    string `toml:"enodeId"`
	RpcUrl     string `toml:"rpcUrl"`
}

func (c NodeManagerConfig) IsValid() error {
	if c.Name == "" {
		return errors.New("node manager name is empty.")
	}
	if c.TesseraKey == "" {
		return errors.New("node manager tesseraKey is empty.")
	}
	if c.EnodeId == "" {
		return errors.New("node manager enodeId is empty.")
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
	Name         string   `toml:"name"`
	ControlType  string   `toml:"controlType"` // SHELL or Docker
	ContainerId  string   `toml:"containerId"`
	StopCommand  []string `toml:"stopCommand"`
	StartCommand []string `toml:"startCommand"`
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

type NodeConfig struct {
	Name               string               `toml:"name"`
	GethRpcUrl         string               `toml:"gethRpcUrl"`
	TesseraUpcheckUrl  string               `toml:"tesseraUpcheckUrl"`
	EnodeId            string               `toml:"enodeId"`
	Consensus          string               `toml:"consensus"`
	GethInactivityTime int                  `toml:"gethInactivityTime"`
	Server             *RPCServerConfig     `toml:"server"`
	GethProcess        *ProcessConfig       `toml:"gethProcess"`
	TesseraProcess     *ProcessConfig       `toml:"tesseraProcess"`
	Proxies            []*ProxyConfig       `toml:"proxies"`
	NodeManagers       []*NodeManagerConfig `toml:"nodeManagers"`
}

func ReadConfig(configFile string) (NodeConfig, error) {
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
	if err = input.IsValid(); err != nil {
		return NodeConfig{}, err
	}

	return input, nil
}

func (c NodeConfig) IsRaft() bool {
	return strings.ToLower(c.Consensus) == "raft"
}

func (c NodeConfig) IsIstanbul() bool {
	return strings.ToLower(c.Consensus) == "istanbul"
}

func (c NodeConfig) IsClique() bool {
	return strings.ToLower(c.Consensus) == "clique"
}

func (c NodeConfig) IsValid() error {
	if c.Name == "" {
		return errors.New("Name is empty")
	}

	if c.Consensus == "" {
		return errors.New("consensus is empty")
	}

	if !c.IsRaft() && !c.IsClique() && !c.IsIstanbul() {
		return errors.New("invalid consensus name. supports only raft or istanbul or clique")
	}

	if len(c.Proxies) == 0 {
		return errors.New("proxies config is empty")
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

	if c.EnodeId == "" {
		return errors.New("enodeId is empty")
	}

	if c.GethInactivityTime < 60 {
		return errors.New("GethInactivityTime should be greater than or equal to 60seconds")
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

	for _, n := range c.NodeManagers {
		if err := n.IsValid(); err != nil {
			return err
		}
	}

	for _, n := range c.Proxies {
		if err := n.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

func isValidUrl(addr string) bool {
	u, err := url.Parse(addr)
	log.Debug("parse", "url", u, "err", err)
	return err == nil
}
