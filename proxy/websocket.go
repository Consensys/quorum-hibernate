package proxy

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/ConsenSysQuorum/node-manager/log"

	"github.com/gorilla/websocket"
)

var (
	// DefaultUpgrader specifies the parameters for upgrading an HTTP
	// connection to a WebSocket connection.
	DefaultUpgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// DefaultDialer is a dialer with all fields set to the default zero values.
	DefaultDialer = websocket.DefaultDialer
)

// WebsocketProxy is an HTTP Handler that takes an incoming WebSocket
// connection and proxies it to another server.
type WebsocketProxy struct {
	ps *ProxyServer
	// Director, if non-nil, is a function that may copy additional request
	// headers from the incoming WebSocket connection into the output headers
	// which will be forwarded to another server.
	Director func(incoming *http.Request, out http.Header)

	// Backend returns the backend URL which the proxy uses to reverse proxy
	// the incoming WebSocket connection. Request is the initial incoming and
	// unmodified request.
	Backend func(*http.Request) *url.URL

	// Upgrader specifies the parameters for upgrading a incoming HTTP
	// connection to a WebSocket connection. If nil, DefaultUpgrader is used.
	Upgrader *websocket.Upgrader

	//  Dialer contains options for connecting to the backend WebSocket server.
	//  If nil, DefaultDialer is used.
	Dialer *websocket.Dialer
}

// ProxyHandler returns a new http.Handler interface that reverse proxies the
// request to the given target.
func WSProxyHandler(ps *ProxyServer, destUrl string) (*WebsocketProxy, error) {
	url, err := url.Parse(destUrl)
	if err != nil {
		return nil, err
	}
	return NewWSProxy(ps, url), nil
}

// NewProxy returns a new Websocket reverse proxy that rewrites the
// URL's to the scheme, host and base path provider in target.
func NewWSProxy(ps *ProxyServer, target *url.URL) *WebsocketProxy {
	backend := func(r *http.Request) *url.URL {
		// Shallow copy
		u := *target
		u.Fragment = r.URL.Fragment
		u.Path = r.URL.Path
		u.RawQuery = r.URL.RawQuery
		return &u
	}
	return &WebsocketProxy{ps: ps, Backend: backend}
}

// ServeHTTP implements the http.Handler that proxies WebSocket connections.
func (w *WebsocketProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer log.Info("exit serveHTTP websocket", "req", req.RequestURI, "remoteAddr", req.RemoteAddr)
	if w.Backend == nil {
		log.Error("websocketproxy: backend function is not defined")
		http.Error(rw, "backend missing", http.StatusInternalServerError)
		return
	}

	backendURL := w.Backend(req)
	if backendURL == nil {
		log.Error("websocketproxy: backend URL is nil")
		http.Error(rw, "backend url is nil", http.StatusInternalServerError)
		return
	}

	if w.ps.qrmNode.PrepareNode() {
		log.Info("node prepared to accept request")
	} else {
		err := errors.New("node prepare failed")
		log.Error("websocket: failed", "err", err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	dialer := w.Dialer
	if w.Dialer == nil {
		dialer = DefaultDialer
	}

	// Pass headers from the incoming request to the dialer to forward them to
	// the final destinations.
	requestHeader := http.Header{}
	if origin := req.Header.Get("Origin"); origin != "" {
		log.Info("req set origin", "origin", origin)
		requestHeader.Add("Origin", origin)
	}
	for _, prot := range req.Header[http.CanonicalHeaderKey("Sec-WebSocket-Protocol")] {
		log.Info("request header", "prot", prot, "key", "Sec-WebSocket-Protocol")
		requestHeader.Add("Sec-WebSocket-Protocol", prot)
	}
	for _, cookie := range req.Header[http.CanonicalHeaderKey("Cookie")] {
		log.Info("request header", "cookie", cookie, "key", "Cookie")
		requestHeader.Add("Cookie", cookie)
	}

	if req.Host != "" {
		log.Info("req set host", "host", req.Host)
		requestHeader.Set("Host", req.Host)
	}

	// Pass X-Forwarded-For headers too, code below is a part of
	// httputil.ReverseProxy.
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := req.Header["X-Forwarded-For"]; ok {
			log.Info("get X-Forwarded-For prior", "prior", prior)
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		log.Info("get X-Forwarded-For clientip", "clientip", clientIP)
		requestHeader.Set("X-Forwarded-For", clientIP)
	}

	// Set the originating protocol of the incoming HTTP request. The SSL might
	// be terminated on our site and because we doing proxy adding this would
	// be helpful for applications on the backend.
	requestHeader.Set("X-Forwarded-Proto", "http")
	if req.TLS != nil {
		log.Info("set X-Forwarded-Proto https")
		requestHeader.Set("X-Forwarded-Proto", "https")
	}

	// Enable the director to copy any additional headers it desires for
	// forwarding to the remote server.
	if w.Director != nil {
		w.Director(req, requestHeader)
	}

	// Connect to the backend URL, also pass the headers we get from the request
	// together with the Forwarded headers we prepared above.
	connBackend, resp, err := dialer.Dial(backendURL.String(), requestHeader)
	if err != nil {
		log.Error("websocketproxy: couldn't dial to remote backend url", "err", err)
		if resp != nil {
			// If the WebSocket handshake fails, ErrBadHandshake is returned
			// along with a non-nil *http.Response so that callers can handle
			// redirects, authentication, etcetera.
			if err := copyResponse(rw, resp); err != nil {
				log.Error("websocketproxy: couldn't write response after failed remote backend handshake:", "err", err)
			}
		} else {
			http.Error(rw, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
		return
	}
	log.Info("WS handler connected to backend", "name", w.ps.proxyCfg.Name, "dest", w.ps.proxyCfg.UpstreamAddr)
	defer func() {
		connBackend.Close()
		log.Info("WS handler disconnected from backend", "name", w.ps.proxyCfg.Name, "dest", w.ps.proxyCfg.UpstreamAddr)
	}()

	upgrader := w.Upgrader
	if w.Upgrader == nil {
		upgrader = DefaultUpgrader
	}

	// Only pass those headers to the upgrader.
	upgradeHeader := http.Header{}
	if hdr := resp.Header.Get("Sec-Websocket-Protocol"); hdr != "" {
		log.Info("set in upgraded header Sec-Websocket-Protocol", "hdr", hdr)
		upgradeHeader.Set("Sec-Websocket-Protocol", hdr)
	}
	if hdr := resp.Header.Get("Set-Cookie"); hdr != "" {
		upgradeHeader.Set("Set-Cookie", hdr)
		log.Info("set in upgraded header Set-Cookie", "hdr", hdr)
	}

	// Now upgrade the existing incoming request to a WebSocket connection.
	// Also pass the header that we gathered from the Dial handshake.
	connSrc, err := upgrader.Upgrade(rw, req, upgradeHeader)
	if err != nil {
		log.Error("websocketproxy: couldn't upgrade", "err", err)
		return
	}
	defer connSrc.Close()

	errClient := make(chan error, 1)
	errBackend := make(chan error, 1)
	replicateWebsocketConn := func(dst, src *websocket.Conn, errc chan error, isSrc bool) {
		defer log.Info("exit replicateWebsocketConn", "src", isSrc)
		for {
			msgType, msg, err := src.ReadMessage()
			if err != nil {
				w.closeConnWithError(dst, err)
				errc <- err
				break
			}

			w.ps.qrmNode.ResetInactiveTime()

			if w.ps.qrmNode.PrepareNode() {
				log.Info("node prepared to accept request")
			} else {
				err = errors.New("node prepare failed")
				w.closeConnWithError(dst, err)
				errc <- err
				break
			}

			if isSrc {
				log.Info("received request from source", "msgType", msgType, "msg", string(msg))
			} else {
				log.Info("sending response to destination", "msgType", msgType, "msg", string(msg))
			}

			if isSrc {
				if err := HandlePrivateTx(msg, w.ps); err != nil {
					w.closeConnWithError(dst, err)
					errc <- err
					break
				}
			}

			err = dst.WriteMessage(msgType, msg)
			if err != nil {
				errc <- err
				break
			}
		}
	}

	go replicateWebsocketConn(connSrc, connBackend, errClient, false)
	go replicateWebsocketConn(connBackend, connSrc, errBackend, true)

	var message string
	select {
	case err = <-errClient:
		message = fmt.Sprintf("websocketproxy: Error when copying from backend to client: %v", err)
	case err = <-errBackend:
		message = fmt.Sprintf("websocketproxy: Error when copying from client to backend: %v", err)
	}
	if e, ok := err.(*websocket.CloseError); !ok || e.Code == websocket.CloseAbnormalClosure {
		log.Error(message, "err", err)
	}
}

func (w *WebsocketProxy) closeConnWithError(dst *websocket.Conn, err error) {
	m := websocket.FormatCloseMessage(websocket.CloseNormalClosure, fmt.Sprintf("%v", err))
	if e, ok := err.(*websocket.CloseError); ok {
		if e.Code != websocket.CloseNoStatusReceived {
			m = websocket.FormatCloseMessage(e.Code, e.Text)
		}
	}
	dst.WriteMessage(websocket.CloseMessage, m)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func copyResponse(rw http.ResponseWriter, resp *http.Response) error {
	copyHeader(rw.Header(), resp.Header)
	rw.WriteHeader(resp.StatusCode)
	defer resp.Body.Close()

	_, err := io.Copy(rw, resp.Body)
	return err
}