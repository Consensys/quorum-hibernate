package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/ConsenSys/quorum-hibernate/config"
	"github.com/ConsenSys/quorum-hibernate/log"
	"github.com/ConsenSys/quorum-hibernate/node"
	"github.com/ConsenSys/quorum-hibernate/proxy"
	"github.com/ConsenSys/quorum-hibernate/rpc"
	"github.com/sirupsen/logrus"
)

type NodeHibernatorApp struct {
	node         *node.NodeControl
	proxyServers []proxy.Proxy
	rpcService   *rpc.RPCService
}

var nhApp = NodeHibernatorApp{}

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
		log.Error("unable to load config", "err", err)
		return
	}
	log.Debug("main - node config", "basic", nodeConfig.BasicConfig, "nhs", nodeConfig.Peers)
	rpcBackendErrCh := make(chan error)
	proxyBackendErrCh := make(chan error)
	if !Start(nodeConfig, err, proxyBackendErrCh, rpcBackendErrCh) {
		return
	}
	waitForShutdown(rpcBackendErrCh, proxyBackendErrCh)
}

func Start(nodeConfig *config.Node, err error, proxyBackendErrCh chan error, rpcBackendErrCh chan error) bool {
	nhApp.node = node.NewNodeControl(nodeConfig)
	if nhApp.proxyServers, err = proxy.MakeProxyServices(nhApp.node, proxyBackendErrCh); err != nil {
		log.Error("Start - creating proxies failed", "err", err)
		return false
	}
	nhApp.rpcService = rpc.NewRPCService(nhApp.node, nhApp.node.GetRPCConfig(), rpcBackendErrCh)

	// start node service
	nhApp.node.Start()

	// start proxies
	for _, p := range nhApp.proxyServers {
		p.Start()
	}

	// start rpc server
	if err := nhApp.rpcService.Start(); err != nil {
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

func readNodeConfigFromFile(configFile string) (*config.Node, error) {
	nhReader, err := config.NewNodeHibernatorReader(configFile)
	if err != nil {
		return nil, err
	}

	log.Debug("readNodeConfigFromFile - loading node hibernator config file")
	nhConfig, err := nhReader.Read()
	if err != nil {
		return nil, err
	}

	log.Debug("readNodeConfigFromFile - validating node hibernator config file")
	// validate config rules
	if err = nhConfig.IsValid(); err != nil {
		return nil, err
	}

	log.Debug("readNodeConfigFromFile - loading peers config file")
	peersReader, err := config.NewPeersReader(nhConfig.PeersConfigFile)
	if err != nil {
		return nil, err
	}

	peersConfig, err := peersReader.Read()
	if err != nil {
		return nil, err
	}
	log.Debug("readNodeConfigFromFile - validating peers config file")

	if err := peersConfig.IsValid(); err != nil {
		return nil, err
	}

	return &config.Node{
		BasicConfig: &nhConfig,
		Peers:       peersConfig,
	}, nil
}

func Shutdown() {
	for _, p := range nhApp.proxyServers {
		p.Stop()
	}
	nhApp.rpcService.Stop()
	nhApp.node.Stop()
	log.ErrWriter.Close()
}
