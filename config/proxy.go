package config

import (
	"errors"
	"strings"
)

type Proxy struct {
	Name                   string     `toml:"name" json:"name"`                                     // name of node manager process
	Type                   string     `toml:"type" json:"type"`                                     // proxy scheme - http or ws
	ProxyAddr              string     `toml:"proxyAddress" json:"proxyAddress"`                     // proxy address
	UpstreamAddr           string     `toml:"upstreamAddress" json:"upstreamAddress"`               // upstream address of the proxy address
	ProxyPaths             []string   `toml:"proxyPaths" json:"proxyPaths"`                         // httpRequestURI paths of the upstream address
	IgnorePathsForActivity []string   `toml:"ignorePathsForActivity" json:"ignorePathsForActivity"` // httpRequestURI paths of the upstream address that should be ignored for activity
	ReadTimeout            int        `toml:"readTimeout" json:"readTimeout"`                       // readTimeout of the proxy server
	WriteTimeout           int        `toml:"writeTimeout" json:"writeTimeout"`                     // writeTimeout of the proxy server
	ProxyServerTLSConfig   *ServerTLS `toml:"proxyTlsConfig" json:"proxyTlsConfig"`                 // proxy server tls config
	ClientTLSConfig        *ClientTLS `toml:"clientTlsConfig" json:"clientTlsConfig"`               // reverse proxy client tls config
}

func (c Proxy) IsHttp() bool {
	return strings.ToLower(c.Type) == "http"
}

func (c Proxy) IsWS() bool {
	return strings.ToLower(c.Type) == "ws"
}

// IsValid returns nil if the Proxy is valid else returns error
func (c Proxy) IsValid() error {
	if c.Name == "" {
		return newFieldErr("name", isEmptyErr)
	}
	if !c.IsWS() && !c.IsHttp() {
		return newFieldErr("type", errors.New("must be http or ws"))
	}
	if c.ProxyAddr == "" {
		return newFieldErr("proxyAddress", isEmptyErr)
	}
	if c.UpstreamAddr == "" {
		return newFieldErr("upstreamAddress", isEmptyErr)
	}
	if err := isValidUrl(c.ProxyAddr); err != nil {
		return newFieldErr("proxyAddress", err)
	}
	if err := isValidUrl(c.UpstreamAddr); err != nil {
		return newFieldErr("upstreamAddress", err)
	}
	if len(c.ProxyPaths) == 0 {
		return newFieldErr("proxyPaths", isEmptyErr)
	}
	if c.ReadTimeout == 0 {
		return newFieldErr("readTimeout", isNotGreaterThanZeroErr)
	}
	if c.WriteTimeout == 0 {
		return newFieldErr("writeTimeout", isNotGreaterThanZeroErr)
	}

	if c.ProxyServerTLSConfig != nil {
		if err := c.ProxyServerTLSConfig.IsValid(); err != nil {
			return newFieldErr("proxyTlsConfig", err)
		}
	}

	if c.ClientTLSConfig != nil {
		if err := c.ClientTLSConfig.IsValid(); err != nil {
			return newFieldErr("clientTlsConfig", err)
		}
	}

	return nil
}
