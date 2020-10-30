package main

import (
	"github.com/ConsenSysQuorum/node-manager/node"
	"github.com/ConsenSysQuorum/node-manager/proxy"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	qn := node.NewQuorumNode()
	qn.Start()
	proxy.StartProxyServerServices(qn)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)
	select {
	case <-sigc:
	}
	log.Printf("Received interrupt signal, shutting down...")
}
