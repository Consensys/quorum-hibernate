package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
	"github.com/ConsenSysQuorum/node-manager/proxy"
	"github.com/ConsenSysQuorum/node-manager/rpc"
	"github.com/sirupsen/logrus"
)

type QNMApp struct {
	qrmNode      *node.NodeControl
	proxyServers []proxy.Proxy
	rpcService   *rpc.RPCService
}

var qnmApp = QNMApp{}

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

	log.Debug("main - node config", "cfg", nodeConfig)
	rpcBackendErrCh := make(chan error)
	proxyBackendErrCh := make(chan error)
	if !Start(nodeConfig, err, proxyBackendErrCh, rpcBackendErrCh) {
		return
	}
	waitForShutdown(rpcBackendErrCh, proxyBackendErrCh)
}

func Start(nodeConfig types.NodeConfig, err error, proxyBackendErrCh chan error, rpcBackendErrCh chan error) bool {
	qnmApp.qrmNode = node.NewQuorumNodeControl(&nodeConfig)
	if qnmApp.proxyServers, err = proxy.MakeProxyServices(qnmApp.qrmNode, proxyBackendErrCh); err != nil {
		log.Error("Start - creating proxies failed", "err", err)
		return false
	}
	qnmApp.rpcService = rpc.NewRPCService(qnmApp.qrmNode, qnmApp.qrmNode.GetRPCConfig(), rpcBackendErrCh)

	// start quorum node service
	qnmApp.qrmNode.Start()

	// start proxies
	for _, p := range qnmApp.proxyServers {
		p.Start()
	}

	// start rpc server
	if err := qnmApp.rpcService.Start(); err != nil {
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

func readNodeConfigFromFile(configFile string) (types.NodeConfig, error) {
	var nodeConfig types.NodeConfig
	var err error
	if nodeConfig, err = types.ReadNodeConfig(configFile); err != nil {
		log.Error("readNodeConfigFromFile - loading node config file failed", "configfile", configFile, "err", err)
		return types.NodeConfig{}, err
	}
	log.Info("readNodeConfigFromFile - node config file read successfully")
	if nodeConfig.NodeManagers, err = types.ReadNodeManagerConfig(nodeConfig.BasicConfig.NodeManagerConfigFile); err != nil {
		log.Error("readNodeConfigFromFile - loading node manager config failed", "err", err)
		return types.NodeConfig{}, err
	}
	log.Info("readNodeConfigFromFile - node manager config file read successfully")
	return nodeConfig, nil
}

func Shutdown() {
	for _, p := range qnmApp.proxyServers {
		p.Stop()
	}
	qnmApp.rpcService.Stop()
	qnmApp.qrmNode.Stop()
}
