package rpc

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/core/types"
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
	qn          *node.QuorumNodeControl
	cors        []string
	httpAddress string
	httpServer  *http.Server
	errCh       chan error
	shutdownWg  sync.WaitGroup
}

func NewRPCService(qn *node.QuorumNodeControl, config *types.RPCServerConfig, backendErrorChan chan error) *RPCService {
	return &RPCService{
		qn:          qn,
		cors:        config.RPCCorsList,
		httpAddress: config.RpcAddr,
		errCh:       backendErrorChan,
	}
}

func (r *RPCService) Start() error {
	log.Info("Starting Node JSON-RPC server")

	jsonrpcServer := rpc.NewServer()
	jsonrpcServer.RegisterCodec(json.NewCodec(), "application/json")
	if err := jsonrpcServer.RegisterService(node.NewNodeRPCAPIs(r.qn), "node"); err != nil {
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

	r.shutdownWg.Add(1)
	go func() {
		defer r.shutdownWg.Done()
		log.Info("Started JSON-RPC server", "Addr", r.httpAddress)
		if err := r.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Error("Unable to start JSON-RPC server", "err", err)
			r.errCh <- err
		}
	}()

	log.Info("JSON-RPC HTTP endpoint opened", "url", fmt.Sprintf("http://%s", r.httpServer.Addr))
	return nil
}

func (r *RPCService) Stop() {
	log.Info("Stopping JSON-RPC server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if r.httpServer != nil {
		if err := r.httpServer.Shutdown(ctx); err != nil {
			log.Error("JSON-RPC server shutdown failed", "err", err)
		}
		r.shutdownWg.Wait()

		log.Info("RPC HTTP endpoint closed", "url", fmt.Sprintf("http://%s", r.httpServer.Addr))
	}

	log.Info("RPC service stopped")
}
