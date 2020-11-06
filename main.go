package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/ConsenSysQuorum/node-manager/rpc"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
	"github.com/ConsenSysQuorum/node-manager/proxy"
)

type QNMApp struct {
	qrmNode      *node.QuorumNode
	proxyServers []proxy.Proxy
	rpcService   *rpc.RPCService
}

var qnmApp = QNMApp{}

func main() {

	var verbosity int
	var nodeConfig types.NodeConfig
	var err error

	flag.IntVar(&verbosity, "verbosity", log.InfoLevel, "logging verbosity")
	// Read config file path
	var configFile string
	flag.StringVar(&configFile, "config", "config.toml", "config file")
	flag.Parse()
	log.Info("config file", "path", configFile)

	if nodeConfig, err = types.ReadConfig(configFile); err != nil {
		log.Error("loading config file failed", "configfile", configFile, "err", err)
		return
	}
	log.Info("config file read")

	rpcBackendErrCh := make(chan error)
	proxyBackendErrCh := make(chan error)
	qnmApp.qrmNode = node.NewQuorumNode(&nodeConfig)
	if qnmApp.proxyServers, err = proxy.MakeProxyServices(qnmApp.qrmNode, proxyBackendErrCh); err != nil {
		log.Error("creating proxies failed", "err", err)
		return
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
		log.Info("rpc server failed", "err", err)
		return
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)
	for {
		select {
		case err := <-sigc:
			log.Error("Received interrupt signal, shutting down...", "err", err)
			Shutdown()
			return
		case err := <-rpcBackendErrCh:
			log.Error("RPC backend failed, shutting down...", "err", err)
			Shutdown()
			return
		case err := <-proxyBackendErrCh:
			log.Error("Proxy backend failed, shutting down...", "err", err)
			Shutdown()
			return
		}
	}

}

func Shutdown() {
	for _, p := range qnmApp.proxyServers {
		p.Stop()
	}
	qnmApp.rpcService.Stop()
	qnmApp.qrmNode.Stop()
}
