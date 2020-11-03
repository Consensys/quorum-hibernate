package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/log"
	"github.com/ConsenSysQuorum/node-manager/node"
)

func StartWsProxyServer(qn *node.QuorumNode, name string, destUrl string, proxyUrl string) {
	go func() {
		http.Handle(fmt.Sprintf("/%s", name), WebsocketProxy(destUrl, qn))
		log.Info("ListenAndServe started", "name", name, "destUrl", destUrl, "proxy", proxyUrl, "path", name)
		err := http.ListenAndServe(proxyUrl, nil)
		if err != nil {
			log.Error("ListenAndServe failed", "destUrl", destUrl, "err", err)
		}
	}()
}

// TODO handle reading request, handling private tx and checking response message status
// TODO try gorilla websocket
func WebsocketProxy(destUrl string, qn *node.QuorumNode) http.Handler {
	return http.HandlerFunc(func(repsonse http.ResponseWriter, request *http.Request) {
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
	})
}
