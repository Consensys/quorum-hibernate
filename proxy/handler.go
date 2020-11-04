package proxy

import (
	"bytes"
	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
	"io"
	"io/ioutil"
	"net"
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

// TODO handle reading request, handling private tx and checking response message status
// TODO try gorilla websocket
func makeWSHandler(qn *node.QuorumNode, destUrl string) http.HandlerFunc {
	return func(repsonse http.ResponseWriter, request *http.Request) {
		defer log.Info("WS handlerFunc finished", "remoteAddr", request.RemoteAddr, "uri", request.RequestURI)
		log.Info("WS request recieved", "reqUri", request.RequestURI, "remoteAddr", request.RemoteAddr)
		qn.ResetInactiveTime()
		if qn.PrepareNode() {
			log.Info("node prepared")
		} else {
			log.Info("node prepare failed")
			http.Error(repsonse, "node prepare failed", http.StatusInternalServerError)
			return
		}
		log.Info("WS dial tcp", "destUrl", destUrl)
		destConn, err := net.Dial("tcp", destUrl)
		if err != nil {
			log.Error("Error dialing websocket", "backend", destUrl, "err", err)
			http.Error(repsonse, "Error contacting backend server.", http.StatusInternalServerError)
			return
		}
		srcRespHijack, ok := repsonse.(http.Hijacker)
		if !ok {
			http.Error(repsonse, "ws request failed. Not a hijacker", http.StatusInternalServerError)
			return
		}
		srcRespNetConn, _, err := srcRespHijack.Hijack()
		if err != nil {
			log.Error("Hijack error", "err", err)
			http.Error(repsonse, "WS request failed. Hijack error", http.StatusInternalServerError)
			return
		}
		defer srcRespNetConn.Close()
		defer destConn.Close()

		err = request.Write(destConn)
		if err != nil {
			log.Error("Error copying request to target", "err", err)
			return
		}

		errc := make(chan error, 2)
		copyFunc := func(dst io.Writer, src io.Reader, isSrc bool) {
			defer log.Info("copy func finished")
			if isSrc {
				log.Info("WS copy data from src -> dest")
			} else {
				log.Info("WS copy data from dest -> src")
			}
			qn.ResetInactiveTime()
			n, err := io.Copy(dst, src)
			log.Info("WS copied data", "n", n, "err", err)
			errc <- err
		}
		go copyFunc(destConn, srcRespNetConn, true)
		log.Info("go src to dest started")
		go copyFunc(srcRespNetConn, destConn, false)
		log.Info("go dest to src started")
		if err := <-errc; err != nil {
			log.Info("copyFunc failed", "err", err)
		} else {
			log.Info("copy func successful")
		}
	}
}

func logRequestPayload(req *http.Request, destUrl string, body string) {
	log.Info("Request received", "query", req.URL.RawQuery, "remoteAddr", req.RemoteAddr, "destUrl", destUrl, "body", body)
}
