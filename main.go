package main

import (
	"flag"
	"github.com/ConsenSysQuorum/node-manager/config"
	"os"
	"os/signal"
	"syscall"

	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
	"github.com/ConsenSysQuorum/node-manager/proxy"
	"github.com/ConsenSysQuorum/node-manager/rpc"
	"github.com/sirupsen/logrus"
)

type NodeManagerApp struct {
	node         *node.NodeControl
	proxyServers []proxy.Proxy
	rpcService   *rpc.RPCService
}

var nmApp = NodeManagerApp{}

func main() {
	var verbosity int
	flag.IntVar(&verbosity, "verbosity", log.InfoLevel, "logging verbosity")
	// Read config file path
	var configFile string
	flag.StringVar(&configFile, "config", "config.toml", "config file")
	flag.Parse()
	logrus.SetLevel(logrus.Level(verbosity + 2))
	log.Debug("main - config file", "path", configFile)
	nodeConfig, err := readNodeConfigFromFile(configFile)
	if err != nil {
		log.Error("main - loading config file failed", "err", err)
		return
	}
	log.Debug("main - node config", "basic", nodeConfig.BasicConfig, "nms", nodeConfig.NodeManagers)
	rpcBackendErrCh := make(chan error)
	proxyBackendErrCh := make(chan error)
	if !Start(nodeConfig, err, proxyBackendErrCh, rpcBackendErrCh) {
		return
	}
	waitForShutdown(rpcBackendErrCh, proxyBackendErrCh)
}

func Start(nodeConfig config.Node, err error, proxyBackendErrCh chan error, rpcBackendErrCh chan error) bool {
	nmApp.node = node.NewNodeControl(&nodeConfig)
	if nmApp.proxyServers, err = proxy.MakeProxyServices(nmApp.node, proxyBackendErrCh); err != nil {
		log.Error("Start - creating proxies failed", "err", err)
		return false
	}
	nmApp.rpcService = rpc.NewRPCService(nmApp.node, nmApp.node.GetRPCConfig(), rpcBackendErrCh)

	// start node service
	nmApp.node.Start()

	// start proxies
	for _, p := range nmApp.proxyServers {
		p.Start()
	}

	// start rpc server
	if err := nmApp.rpcService.Start(); err != nil {
		log.Info("Start - rpc server failed", "err", err)
		return false
	}
	return true
}

func waitForShutdown(rpcBackendErrCh chan error, proxyBackendErrCh chan error) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)
	for {
		select {
		case err := <-sigc:
			log.Error("waitForShutdown - Received interrupt signal, shutting down...", "err", err)
			Shutdown()
			return
		case err := <-rpcBackendErrCh:
			log.Error("waitForShutdown - RPC backend failed, shutting down...", "err", err)
			Shutdown()
			return
		case err := <-proxyBackendErrCh:
			log.Error("waitForShutdown - Proxy backend failed, shutting down...", "err", err)
			Shutdown()
			return
		}
	}
}

func readNodeConfigFromFile(configFile string) (config.Node, error) {
	var nodeConfig config.Node
	var err error
	if nodeConfig, err = config.ReadNodeConfig(configFile); err != nil {
		log.Error("readNodeConfigFromFile - loading node config file failed", "configfile", configFile, "err", err)
		return config.Node{}, err
	}
	log.Info("readNodeConfigFromFile - node config file read successfully")
	if nodeConfig.NodeManagers, err = config.ReadNodeManagerConfig(nodeConfig.BasicConfig.NodeManagerConfigFile); err != nil {
		log.Error("readNodeConfigFromFile - loading node manager config failed", "err", err)
		return config.Node{}, err
	}
	log.Info("readNodeConfigFromFile - node manager config file read successfully")
	return nodeConfig, nil
}

func Shutdown() {
	for _, p := range nmApp.proxyServers {
		p.Stop()
	}
	nmApp.rpcService.Stop()
	nmApp.node.Stop()
}
