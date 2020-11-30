package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/log"
)

// makeHttpHandler returns a function to serve HTTP requests from clients
func makeHttpHandler(ps *ProxyServer) (http.HandlerFunc, error) {

	return func(res http.ResponseWriter, req *http.Request) {
		if err := ps.qrmNode.IsNodeBusy(); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			const errMsg = "Reading request failed"
			log.Error(errMsg, "name", ps.proxyCfg.Name, "path", req.RequestURI, "err", err)
			http.Error(res, errMsg, http.StatusInternalServerError)
			return
		}

		// you can reassign the body if you need to parse it as multipart
		req.Body = ioutil.NopCloser(bytes.NewReader(body))

		if req.RequestURI != "/partyinfo" && req.RequestURI != "/upcheck" && req.RequestURI != "/partyinfo/validate" {
			logRequestPayload(req, ps.proxyCfg.Name, ps.proxyCfg.UpstreamAddr, string(body))
			ps.qrmNode.ResetInactiveTime()

			log.Info("httpHandler - request", "path", req.RequestURI)

			if ps.qrmNode.PrepareNode() {
				log.Debug("httpHandler - prepared to accept request")
			} else {
				log.Error("httpHandler - prepare node failed")
				http.Error(res, ErrNodeNotReady.Error(), http.StatusInternalServerError)
				return
			}

			if err := HandlePrivateTx(body, ps); err != nil {
				log.Error("httpHandler - handling pvt tx failed", "err", err)
				http.Error(res, ErrParticipantsDown.Error(), http.StatusInternalServerError)
				return
			}

		}

		// Forward request to original request
		log.Debug("httpHandler - forwarding request to proxy")
		ps.rp.ServeHTTP(res, req)
	}, nil
}
