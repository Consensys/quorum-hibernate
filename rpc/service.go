package rpc

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/config"

	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json"
	"github.com/rs/cors"
)

const (
	ReadTimeout  = 10 * time.Second
	WriteTimeout = 10 * time.Second
	IdleTimeout  = 60 * time.Second
)

type RPCService struct {
	qn          *node.NodeControl
	cors        []string
	httpAddress string
	httpServer  *http.Server
	errCh       chan error
	shutdownWg  sync.WaitGroup
}

func NewRPCService(qn *node.NodeControl, config *config.RPCServer, backendErrorChan chan error) *RPCService {
	return &RPCService{
		qn:          qn,
		cors:        config.RPCCorsList,
		httpAddress: config.RpcAddr,
		errCh:       backendErrorChan,
	}
}

func (r *RPCService) Start() error {
	log.Info("Starting node manager JSON-RPC server")

	jsonrpcServer := rpc.NewServer()
	jsonrpcServer.RegisterCodec(json.NewCodec(), "application/json")
	if err := jsonrpcServer.RegisterService(node.NewNodeRPCAPIs(r.qn, r.qn.GetNodeConfig()), "node"); err != nil {
		return err
	}

	serverWithCors := cors.New(cors.Options{AllowedOrigins: r.cors}).Handler(jsonrpcServer)
	r.httpServer = &http.Server{
		Addr:    r.httpAddress,
		Handler: serverWithCors,

		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}

	tlsCfg := r.qn.GetNodeConfig().BasicConfig.Server.TLSConfig
	if tlsCfg != nil {
		var err error
		r.httpServer.TLSConfig, err = tlsCfg.TLSConfig()
		if err != nil {
			return err
		}
	}

	r.shutdownWg.Add(1)
	go func() {
		defer r.shutdownWg.Done()
		log.Info("Started node manager JSON-RPC server", "Addr", r.httpAddress)
		if tlsCfg != nil {
			if err := r.httpServer.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
				log.Error("Unable to start node manager JSON-RPC server", "err", err)
				r.errCh <- err
			}
		} else {
			if err := r.httpServer.ListenAndServe(); err != http.ErrServerClosed {
				log.Error("Unable to start node manager JSON-RPC server", "err", err)
				r.errCh <- err
			}
		}
	}()

	log.Info("JSON-RPC HTTP endpoint opened", "url", fmt.Sprintf("http://%s", r.httpServer.Addr))
	return nil
}

func (r *RPCService) Stop() {
	log.Info("Stopping node manager JSON-RPC server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if r.httpServer != nil {
		if err := r.httpServer.Shutdown(ctx); err != nil {
			log.Error("node manager JSON-RPC server shutdown failed", "err", err)
		}
		r.shutdownWg.Wait()

		log.Info("RPC HTTP endpoint closed", "url", fmt.Sprintf("http://%s", r.httpServer.Addr))
	}

	log.Info("RPC service stopped")
}
