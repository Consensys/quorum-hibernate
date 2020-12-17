package config

import (
	"errors"
	"fmt"
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
		return errors.New("name is empty")
	}
	if !c.IsWS() && !c.IsHttp() {
		return namedValidationError{name: c.Name, errMsg: "invalid type. supports only http or ws"}
	}
	if c.ProxyAddr == "" {
		return namedValidationError{name: c.Name, errMsg: "proxyAddress is empty"}
	}
	if c.UpstreamAddr == "" {
		return namedValidationError{name: c.Name, errMsg: "upstreamAddress is empty"}
	}
	if err := isValidUrl(c.ProxyAddr); err != nil {
		return namedValidationError{name: c.Name, errMsg: fmt.Sprintf("invalid proxyAddress: %v", err)}
	}
	if err := isValidUrl(c.UpstreamAddr); err != nil {
		return namedValidationError{name: c.Name, errMsg: fmt.Sprintf("invalid upstreamAddress: %v", err)}
	}
	if len(c.ProxyPaths) == 0 {
		return namedValidationError{name: c.Name, errMsg: "proxyPaths is empty"}
	}
	if c.ReadTimeout == 0 {
		return namedValidationError{name: c.Name, errMsg: "readTimeout is zero"}
	}
	if c.WriteTimeout == 0 {
		return namedValidationError{name: c.Name, errMsg: "writeTimeout is zero"}
	}

	if c.ProxyServerTLSConfig != nil {
		if err := c.ProxyServerTLSConfig.IsValid(); err != nil {
			return err
		}
	}

	if c.ClientTLSConfig != nil {
		if err := c.ClientTLSConfig.IsValid(); err != nil {
			return err
		}
	}

	return nil
}
