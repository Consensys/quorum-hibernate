package proxy

import (
	"bytes"
	"fmt"
	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func StartProxyServerServices(qn *node.QuorumNode) {

	for _, p := range qn.GetProxyConfig() {
		if p.IsHttp() {
			pserver, err := NewProxyServer(qn, p.Name, p.DestUrl, p.ProxyUrl)
			if err != nil {
				log.Error("RPC proxy failed", "name", p.Name, "destUrl", p.DestUrl)
			}
			pserver.Start()
		} else if p.IsWS() {
			StartWsProxyServer(qn, p.Name, p.DestUrl, p.ProxyUrl)
		}
	}

}

func NewProxyServer(qn *node.QuorumNode, name string, destUrl string, proxyUrl string) (Proxy, error) {
	url, err := url.Parse(destUrl)
	if err != nil {
		return nil, err
	}
	rp := httputil.NewSingleHostReverseProxy(url)
	rp.ModifyResponse = func(res *http.Response) error {
		respStatus := res.Status
		log.Info("response status", "status", respStatus)
		qn.ResetInactiveTime()
		return nil
	}
	rpcProxy := ProxyServer{qn, name, destUrl, proxyUrl, rp}
	return rpcProxy, nil
}

func (np ProxyServer) Start() {
	go func() {
		http.HandleFunc(fmt.Sprintf("/%s", np.name), np.forwardRequest)
		log.Info("ListenAndServe started", "name", np.name, "destUrl", np.destUrl, "proxy", np.proxyUrl, "path", np.name)
		err := http.ListenAndServe(np.proxyUrl, nil)
		if err != nil {
			log.Error("ListenAndServe failed", "name", np.name, "destUrl", np.destUrl, "err", err)
		}
	}()
}

func (np ProxyServer) Stop() {
	panic("not implemented")
}

func (np ProxyServer) forwardRequest(res http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("reading body failed", "err", err)
		http.Error(res, "reading request body failed", http.StatusInternalServerError)
		return
	}
	// you can reassign the body if you need to parse it as multipart
	req.Body = ioutil.NopCloser(bytes.NewReader(body))
	// Log the request
	bodyStr := string(body)

	np.logRequestPayload(req.URL.String(), string(body))

	np.qrmNode.ResetInactiveTime()

	if np.qrmNode.PrepareNode() {
		log.Info("node prepared to accept request")
	} else {
		http.Error(res, "node prepare failed", http.StatusInternalServerError)
		return
	}

	if core.IsPrivateTransaction(bodyStr) {
		if tx, err := core.GetPrivateTx(body); err != nil {
			// TODO handle error - return error to client?
			log.Error("failed to unmarshal private tx from request", "err", err)
			http.Error(res, "failed to unmarshal private tx", http.StatusInternalServerError)
			return
		} else {
			if tx.Method == "eth_sendTransaction" {
				log.Info("private transaction request")
				if status, err := np.qrmNode.RequestNodeManagerForPrivateTxPrep(tx.Params[0].PrivateFor); err != nil {
					log.Error("preparePrivateTx failed", "err", err)
					http.Error(res, "private tx prep failed", http.StatusInternalServerError)
					return
				} else if !status {
					log.Error("preparePrivateTx failed some participants are down", "err", err)
					http.Error(res, "private tx prep failed, some participants are down", http.StatusInternalServerError)
					return
				} else {
					log.Info("private tx prep completed successfully.")
				}
			}
		}
	}

	// Forward request to original request
	log.Info("forwarding request to node", "destUrl", np.destUrl, "body", string(body))
	np.serveReverseProxy(res, req)
	log.Info("-----------------------")
}

func (np ProxyServer) serveReverseProxy(res http.ResponseWriter, req *http.Request) {
	// Note that ServeHttp is non blocking & uses a go routine under the hood
	np.rp.ServeHTTP(res, req)
}

func (np ProxyServer) logRequestPayload(reqUrl string, body string) {
	log.Info("Request received", "name", np.name, "reqUrl", reqUrl, "destUrl", np.destUrl, "body", body)
}
