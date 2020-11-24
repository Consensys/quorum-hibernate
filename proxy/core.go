package proxy

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
)

// Proxy represents a proxy server
// It allows the server to be started and stopped.
type Proxy interface {
	Start()
	Stop()
}

var (
	ErrParticipantsDown = errors.New("Some participant nodes are down")
	ErrNodeNotReady     = errors.New("node is not ready to accept request")
)

func MakeProxyServices(qn *node.NodeControl, errc chan error) ([]Proxy, error) {
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

// HandlePrivateTx helps with processing private transactions.
// if the body is a private transaction request it will get participants of the transaction and
// wake them up via qnm2qnm rpc call.
// if body is not a private transaction it will return nil.
func HandlePrivateTx(body []byte, ps *ProxyServer) error {
	// TODO If privacy manager proxy works as expected, can this be removed?
	if participants, err := ps.qrmNode.GetTxHandler().IsPrivateTx(body); err != nil {
		return err
	} else if participants != nil {
		log.Info("HandlePrivateTx - participants", "keys", participants)
		if status, err := ps.qrmNode.PrepareNodeManagerForPrivateTx(participants); err != nil {
			return fmt.Errorf("HandlePrivateTx - preparePrivateTx failed err=%v", err)
		} else if !status {
			return fmt.Errorf("HandlePrivateTx - preparePrivateTx failed some participants are down err=%v", err)
		} else {
			log.Info("private tx prep completed successfully.")
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
