package proxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
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
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	// you can reassign the body if you need to parse it as multipart
	req.Body = ioutil.NopCloser(bytes.NewReader(body))
	// Log the request
	np.logRequestPayload(req.URL.String(), string(body))
	np.qrmNode.ResetInactiveTime()
	if up, err := np.qrmNode.IsNodeUp(); err != nil {
		np.qrmNode.RequestStartNode()
		log.Info("waiting for node start to complete...")
		np.qrmNode.WaitStartNode()
		log.Info("node start completed")
	} else if !up {
		np.qrmNode.SetNodeDown()
		np.qrmNode.RequestStartNode()
		log.Info("waiting for node start to complete...")
		np.qrmNode.WaitStartNode()
		log.Info("node start completed")
	} else {
		np.qrmNode.SetNodeUp()
		log.Info("node UP", "destUrl", np.destUrl)
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
