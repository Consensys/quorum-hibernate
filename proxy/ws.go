package proxy

import (
	"fmt"
	"github.com/ConsenSysQuorum/node-manager/node"
	"io"
	"log"
	"net"
	"net/http"
)

func StartWsProxyServer(qn *node.QuorumNode) {
	go func() {
		proxyUrl, port := qn.GetProxyInfo("WS")
		http.Handle("/ws", WebsocketProxy(proxyUrl))
		log.Printf("ListenAndServe for %s node %s started at http://localhost:%d%s", "WS", proxyUrl, port, "/ws")
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			log.Fatalf("ListenAndServe for WS node %s failed: %v", proxyUrl, err)
		}
	}()
}

func WebsocketProxy(target string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("WS request recieved %s", r.RequestURI)
		d, err := net.Dial("tcp", target)
		if err != nil {
			http.Error(w, "Error contacting backend server.", 500)
			log.Printf("Error dialing websocket backend %s: %v", target, err)
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Not a hijacker?", 500)
			return
		}
		nc, _, err := hj.Hijack()
		if err != nil {
			log.Printf("Hijack error: %v", err)
			return
		}
		defer nc.Close()
		defer d.Close()

		err = r.Write(d)
		if err != nil {
			log.Printf("Error copying request to target: %v", err)
			return
		}

		errc := make(chan error, 2)
		cp := func(dst io.Writer, src io.Reader) {
			log.Printf("copy data")
			_, err := io.Copy(dst, src)
			errc <- err
		}
		go cp(d, nc)
		go cp(nc, d)
		<-errc
	})
}
