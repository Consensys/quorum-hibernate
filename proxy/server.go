package proxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/ConsenSysQuorum/node-manager/node"
)

func StartProxyServerServices(qn *node.QuorumNode) {
	var proxyNames = []string{"RPC", "GRAPHQL"}
	for _, pname := range proxyNames {
		destUrl, proxyPort := qn.GetProxyInfo(pname)
		pserver, err := NewProxyServer(qn, pname, destUrl, proxyPort)
		if err != nil {
			log.Fatalf("RPC proxy failed for url %s %s", pname, destUrl)
		}
		pserver.Start()
	}

	StartWsProxyServer(qn)

}

func NewProxyServer(qn *node.QuorumNode, name string, proxyUrl string, port int) (Proxy, error) {
	url, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}
	rp := httputil.NewSingleHostReverseProxy(url)
	rp.ModifyResponse = func(res *http.Response) error {
		respStatus := res.Status
		resBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Printf("ERROR: reading response body failed err:%v", err)
		}
		// you can reassign the body if you need to parse it as multipart
		res.Body = ioutil.NopCloser(bytes.NewReader(resBody))
		log.Printf("%s response status:%s body:%s\n", name, respStatus, resBody)
		qn.ResetInactiveTime()
		return nil
	}
	rpcProxy := ProxyServer{qn, name, proxyUrl, port, rp}
	return rpcProxy, nil
}

func (np ProxyServer) Start() {
	go func() {
		path := fmt.Sprintf("/%s", strings.ToLower(np.name))
		http.HandleFunc(path, np.forwardRequest)
		log.Printf("ListenAndServe for %s node %s started at http://localhost:%d%s", np.name, np.destUrl, np.proxyPort, path)
		err := http.ListenAndServe(fmt.Sprintf(":%d", np.proxyPort), nil)
		if err != nil {
			log.Fatalf("ListenAndServe for %s node %s failed: %v", np.name, np.destUrl, err)
		}
	}()
}

func (np ProxyServer) Stop() {
	panic("not implemented")
}

func (np ProxyServer) forwardRequest(res http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("ERROR: reading body failed err:%v", err)
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
		log.Printf("waiting for node start to complete...")
		np.qrmNode.WaitStartNode()
		log.Printf("node start completed")
		np.qrmNode.GetProcessIdOfNode()
	} else if !up {
		np.qrmNode.SetNodeDown()
		np.qrmNode.StartNode(true)
		np.qrmNode.GetProcessIdOfNode()
	} else {
		np.qrmNode.SetNodeUp()
		log.Printf("node %s is UP", np.destUrl)
	}
	// Forward request to original request
	log.Printf("forwarding request to node %s %s\n", np.destUrl, body)
	np.serveReverseProxy(res, req)
	log.Printf("-----------------------\n")
}

func (np ProxyServer) serveReverseProxy(res http.ResponseWriter, req *http.Request) {
	// Note that ServeHttp is non blocking & uses a go routine under the hood
	np.rp.ServeHTTP(res, req)
}

func (np ProxyServer) logRequestPayload(reqUrl string, body string) {
	log.Printf("%s Request recieved -> reqUrl:%s destUrl:%s body:%s", np.name, reqUrl, np.destUrl, body)
}
