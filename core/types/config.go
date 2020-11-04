package types

import (
	"errors"
	"os"

	"github.com/naoina/toml"
)

type ProxyConfig struct {
	Name    string `toml:"name"`
	Type    string `toml:"type"` // http or ws
	DestUrl string `toml:"destUrl"`
}

func (c ProxyConfig) IsHttp() bool {
	return c.Type == "http"
}

func (c ProxyConfig) IsWS() bool {
	return c.Type == "ws"
}

type NodeManagerConfig struct {
	Name       string `toml:"name"`
	TesseraKey string `toml:"tesseraKey"`
	RpcUrl     string `toml:"rpcUrl"`
}

type GethProcessConfig struct {
	ControlType  string   `toml:"controlType"` // SHELL or Docker
	StopCommand  []string `toml:"stopCommand"`
	StartCommand []string `toml:"startCommand"`
}

type RPCServerConfig struct {
	RpcAddr     string   `toml:"rpcAddr"` // SHELL or Docker
	RPCCorsList []string `toml:"rpcCorsList"`
	RPCVHosts   []string `toml:"rpcvHosts"`
}

type NodeConfig struct {
	Name               string               `toml:"name"`
	GethRpcUrl         string               `toml:"gethRpcUrl"`
	ProxyAddr          string               `toml:"proxyAddr"`
	GethInactivityTime int                  `toml:"gethInactivityTime"`
	Server             *RPCServerConfig     `toml:"server"`
	GethProcess        *GethProcessConfig   `toml:"gethProcess"`
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
	if err = input.Validate(); err != nil {
		return NodeConfig{}, err
	}

	return input, nil
}

func (nc NodeConfig) Validate() error {
	if len(nc.Proxies) == 0 {
		return errors.New("proxies config is empty")
	}
	if nc.GethProcess == nil {
		return errors.New("geth process config is empty")
	}

	if nc.GethRpcUrl == "" {
		return errors.New("geth rpc url is empty")
	}
	return nil
}
