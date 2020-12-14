package config

import (
	"errors"
	"fmt"
	"strings"
)

type ProxyConfig struct {
	Name         string   `toml:"name"`         // name of node manager process
	Type         string   `toml:"type"`         // proxy scheme - http or ws
	ProxyAddr    string   `toml:"proxyAddr"`    // proxy address
	UpstreamAddr string   `toml:"upstreamAddr"` // upstream address of the proxy address
	ProxyPaths   []string `toml:"proxyPaths"`   // httpRequestURI paths of the upstream address
	// httpRequestURI paths of the upstream address that should be ignored for activity
	IgnorePathsForActivity []string         `toml:"ignorePathsForActivity"`
	ReadTimeout            int              `toml:"readTimeout"`     // readTimeout of the proxy server
	WriteTimeout           int              `toml:"writeTimeout"`    // writeTimeout of the proxy server
	ProxyServerTLSConfig   *ServerTLSConfig `toml:"proxyTLSConfig"`  // proxy server tls config
	ClientTLSConfig        *ClientTLSConfig `toml:"clientTLSConfig"` // reverse proxy client tls config
}

func (c ProxyConfig) IsHttp() bool {
	return strings.ToLower(c.Type) == "http"
}

func (c ProxyConfig) IsWS() bool {
	return strings.ToLower(c.Type) == "ws"
}

// IsValid returns nil if the ProxyConfig is valid else returns error
func (c ProxyConfig) IsValid() error {
	if c.Name == "" {
		return errors.New("name is empty")
	}
	if !c.IsWS() && !c.IsHttp() {
		return namedValidationError{name: c.Name, errMsg: "invalid type. supports only http or ws"}
	}
	if c.ProxyAddr == "" {
		return namedValidationError{name: c.Name, errMsg: "proxyAddr is empty"}
	}
	if c.UpstreamAddr == "" {
		return namedValidationError{name: c.Name, errMsg: "upstreamAddr is empty"}
	}
	if err := isValidUrl(c.ProxyAddr); err != nil {
		return namedValidationError{name: c.Name, errMsg: fmt.Sprintf("invalid proxyAddr: %v", err)}
	}
	if err := isValidUrl(c.UpstreamAddr); err != nil {
		return namedValidationError{name: c.Name, errMsg: fmt.Sprintf("invalid upstreamAddr: %v", err)}
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
