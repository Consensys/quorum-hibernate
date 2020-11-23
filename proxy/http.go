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
			const errMsg = "httpHandler - reading request body failed"
			log.Error(errMsg, "name", ps.proxyCfg.Name, "path", req.RequestURI, "err", err)
			http.Error(res, errMsg, http.StatusInternalServerError)
			return
		}

		// you can reassign the body if you need to parse it as multipart
		req.Body = ioutil.NopCloser(bytes.NewReader(body))

		if req.RequestURI != "/partyinfo" {
			logRequestPayload(req, ps.proxyCfg.Name, ps.proxyCfg.UpstreamAddr, string(body))
		}

		// TODO move request URIs to be excluded to config
		if req.RequestURI != "/partyinfo" && req.RequestURI != "/upcheck" {
			ps.qrmNode.ResetInactiveTime()
		}

		log.Info("httpHandler - request", "path", req.RequestURI)

		if err := ps.qrmNode.IsNodeBusy(); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		if req.RequestURI != "/partyinfo" {
			if ps.qrmNode.PrepareNode() {
				log.Debug("httpHandler - prepared to accept request")
			} else {
				const errMsg = "httpHandler - node prepare failed"
				log.Error(errMsg)
				http.Error(res, errMsg, http.StatusInternalServerError)
				return
			}
		}

		if err := HandlePrivateTx(body, ps); err != nil {
			log.Error("httpHandler - handling pvt tx failed", "err", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		// Forward request to original request
		log.Debug("httpHandler - forwarding request to proxy")
		ps.rp.ServeHTTP(res, req)
	}, nil
}
