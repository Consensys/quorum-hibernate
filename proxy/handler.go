package proxy

import (
	"bytes"
	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func makeHttpHandler(qn *node.QuorumNode, destUrl string) (http.HandlerFunc, error) {
	url, err := url.Parse(destUrl)
	if err != nil {
		return nil, err
	}
	rp := httputil.NewSingleHostReverseProxy(url)
	rp.ModifyResponse = func(res *http.Response) error {
		respStatus := res.Status
		log.Info("response status", "status", respStatus, "code", res.StatusCode)
		qn.ResetInactiveTime()
		return nil
	}

	return func(res http.ResponseWriter, req *http.Request) {
		log.Info("request", "dest", destUrl, "url", req.RequestURI)
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Error("reading body failed", "err", err)
			http.Error(res, "reading request body failed", http.StatusInternalServerError)
			return
		}
		// you can reassign the body if you need to parse it as multipart
		req.Body = ioutil.NopCloser(bytes.NewReader(body))

		logRequestPayload(req, destUrl, string(body))

		qn.ResetInactiveTime()

		if qn.PrepareNode() {
			log.Info("node prepared to accept request")
		} else {
			http.Error(res, "node prepare failed", http.StatusInternalServerError)
			return
		}

		// TODO If tessera proxy works as expected this can be removed
		if core.IsPrivateTransaction(string(body)) {
			if tx, err := core.GetPrivateTx(body); err != nil {
				// TODO handle error - return error to client?
				log.Error("failed to unmarshal private tx from request", "err", err)
				http.Error(res, "failed to unmarshal private tx", http.StatusInternalServerError)
				return
			} else {
				if tx.Method == "eth_sendTransaction" {
					log.Info("private transaction request")
					if status, err := qn.RequestNodeManagerForPrivateTxPrep(tx.Params[0].PrivateFor); err != nil {
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
		log.Info("forwarding request to proxy")
		rp.ServeHTTP(res, req)
	}, nil
}

func logRequestPayload(req *http.Request, destUrl string, body string) {
	log.Info("Request received", "query", req.URL.RawQuery, "remoteAddr", req.RemoteAddr, "destUrl", destUrl, "body", body)
}
