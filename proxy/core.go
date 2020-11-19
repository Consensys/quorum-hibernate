package proxy

import (
	"fmt"
	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
	"io"
	"net/http"
)

// Proxy represents a proxy server that can be started / stopped
type Proxy interface {
	Start()
	Stop()
}

func MakeProxyServices(qn *node.QuorumNodeControl, errc chan error) ([]Proxy, error) {
	var proxies []Proxy
	for _, c := range qn.GetProxyConfig() {
		if p, err := NewProxyServer(qn, c, errc); err != nil {
			return nil, err
		} else {
			proxies = append(proxies, p)
		}
	}
	return proxies, nil
}

func HandlePrivateTx(body []byte, ps *ProxyServer) error {
	// TODO If tessera proxy works as expected this can be removed
	if core.IsPrivateTransaction(string(body)) {
		if tx, err := core.GetPrivateTx(body); err != nil {
			// TODO handle error - return error to client?
			log.Error("failed to unmarshal private tx from request", "err", err)
			return fmt.Errorf("failed to unmarshal private tx from request err=%v", err)
		} else {
			if tx.Method == "eth_sendTransaction" {
				log.Info("private transaction request")
				if status, err := ps.qrmNode.RequestNodeManagerForPrivateTxPrep(tx.Params[0].PrivateFor); err != nil {
					log.Error("preparePrivateTx failed", "err", err)
					return fmt.Errorf("preparePrivateTx failed err=%v", err)
				} else if !status {
					log.Error("preparePrivateTx failed some participants are down", "err", err)
					return fmt.Errorf("preparePrivateTx failed some participants are down err=%v", err)
				} else {
					log.Info("private tx prep completed successfully.")
				}
			}
		}
	}
	return nil
}

func logRequestPayload(req *http.Request, name string, destUrl string, body string) {
	log.Info("Request received", "name", name, "path", req.RequestURI, "remoteAddr", req.RemoteAddr, "destUrl", destUrl, "body", body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func copyResponse(rw http.ResponseWriter, resp *http.Response) error {
	copyHeader(rw.Header(), resp.Header)
	rw.WriteHeader(resp.StatusCode)
	defer resp.Body.Close()

	_, err := io.Copy(rw, resp.Body)
	return err
}
