package proxy

// Proxy represents a proxy server that can be started / stopped
type Proxy interface {
	Start()
	Stop()
}
