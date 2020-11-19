package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/log"
)

func makeHttpHandler(ps *ProxyServer) (http.HandlerFunc, error) {

	return func(res http.ResponseWriter, req *http.Request) {

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Error("reading body failed", "name", ps.proxyCfg.Name, "path", req.RequestURI, "err", err)
			http.Error(res, "reading request body failed", http.StatusInternalServerError)
			return
		}

		// you can reassign the body if you need to parse it as multipart
		req.Body = ioutil.NopCloser(bytes.NewReader(body))

		if req.RequestURI != "/partyinfo" {
			logRequestPayload(req, ps.proxyCfg.Name, ps.proxyCfg.UpstreamAddr, string(body))
		}

		if req.RequestURI != "/partyinfo" && req.RequestURI != "/upcheck" {
			ps.qrmNode.ResetInactiveTime()
		}

		log.Info("request", "path", req.RequestURI)

		if err := ps.qrmNode.IsNodeBusy(); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		if req.RequestURI != "/partyinfo" {
			if ps.qrmNode.PrepareNode() {
				log.Info("node prepared to accept request")
			} else {
				http.Error(res, "node prepare failed", http.StatusInternalServerError)
				return
			}
		}

		if err := HandlePrivateTx(body, ps); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		// Forward request to original request
		log.Info("forwarding request to proxy")
		ps.rp.ServeHTTP(res, req)
	}, nil
}