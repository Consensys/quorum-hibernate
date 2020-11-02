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
)

func main() {
	var verbosity int
	flag.IntVar(&verbosity, "verbosity", log.InfoLevel, "logging verbosity")
	// Read config file path
	var configFile string
	flag.StringVar(&configFile, "config", "config.toml", "config file")
	var nodeConfig types.NodeConfig
	var err error
	if nodeConfig, err = types.ReadConfig(configFile); err != nil {
		log.Error("loading config file failed", "configfile", configFile, "err", err)
		return
	}
	log.Info("config file read", "config", nodeConfig)
	qn := node.NewQuorumNode(&nodeConfig)
	qn.Start()
	proxy.StartProxyServerServices(qn)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)
	select {
	case <-sigc:
	}
	log.Info("Received interrupt signal, shutting down...")
}
