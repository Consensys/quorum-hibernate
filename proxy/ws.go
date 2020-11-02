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

func WebsocketProxy(target string, qn *node.QuorumNode) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("WS request recieved", "reqUri", r.RequestURI)
		d, err := net.Dial("tcp", target)
		if err != nil {
			http.Error(w, "Error contacting backend server.", 500)
			log.Error("Error dialing websocket", "backend", target, "err", err)
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Not a hijacker?", 500)
			return
		}
		nc, _, err := hj.Hijack()
		if err != nil {
			log.Error("Hijack error", "err", err)
			return
		}
		defer nc.Close()
		defer d.Close()

		err = r.Write(d)
		if err != nil {
			log.Error("Error copying request to target", "err", err)
			return
		}

		errc := make(chan error, 2)
		cp := func(dst io.Writer, src io.Reader) {
			log.Info("copy data")
			qn.ResetInactiveTime()
			_, err := io.Copy(dst, src)
			errc <- err
		}
		go cp(d, nc)
		go cp(nc, d)
		<-errc
	})
}
