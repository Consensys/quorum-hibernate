package config

import (
	"encoding/json"
	"fmt"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	nameField                   = "name"
	typeField                   = "type"
	proxyAddressField           = "proxyAddress"
	upstreamAddressField        = "upstreamAddress"
	proxyPathsField             = "proxyPaths"
	ignorePathsForActivityField = "ignorePathsForActivity"
	readTimeoutField            = "readTimeout"
	writeTimeoutField           = "writeTimeout"
	proxyTlsConfigField         = "proxyTlsConfig"
	clientTlsConfigField        = "clientTlsConfig"
)

func minimumValidProxy() Proxy {
	return Proxy{
		Name:                   "myproxy",
		Type:                   "http",
		ProxyAddr:              "localhost:8080",
		UpstreamAddr:           "http://localhost:9090",
		ProxyPaths:             []string{"/"},
		IgnorePathsForActivity: nil,
		ReadTimeout:            15,
		WriteTimeout:           15,
		ProxyServerTLSConfig:   nil,
		ClientTLSConfig:        nil,
	}
}

func TestProxy_Unmarshal(t *testing.T) {
	tests := []struct {
		name, configTemplate string
	}{
		{
			name: "json",
			configTemplate: `
{
	"%v": "myproxy",
	"%v": "http",
	"%v": "localhost:8080",
	"%v": "http://localhost:9090",
	"%v": ["/"],
	"%v": ["/ignore"],
	"%v": 15,
	"%v": 15,
	"%v": {},
	"%v": {}
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "myproxy"
%v = "http"
%v = "localhost:8080"
%v = "http://localhost:9090"
%v = ["/"]
%v = ["/ignore"]
%v = 15
%v = 15
%v = {}
%v = {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(
				tt.configTemplate,
				nameField,
				typeField,
				proxyAddressField,
				upstreamAddressField,
				proxyPathsField,
				ignorePathsForActivityField,
				readTimeoutField,
				writeTimeoutField,
				proxyTlsConfigField,
				clientTlsConfigField,
			)

			want := Proxy{
				Name:                   "myproxy",
				Type:                   "http",
				ProxyAddr:              "localhost:8080",
				UpstreamAddr:           "http://localhost:9090",
				ProxyPaths:             []string{"/"},
				IgnorePathsForActivity: []string{"/ignore"},
				ReadTimeout:            15,
				WriteTimeout:           15,
				ProxyServerTLSConfig:   &ServerTLS{},
				ClientTLSConfig:        &ClientTLS{},
			}

			var (
				got Proxy
				err error
			)

			if tt.name == "json" {
				err = json.Unmarshal([]byte(conf), &got)
			} else if tt.name == "toml" {
				err = toml.Unmarshal([]byte(conf), &got)
			}

			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestProxy_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidProxy()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestProxy_IsValid_Name(t *testing.T) {
	c := minimumValidProxy()

	c.Name = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, nameField+" is empty")
}

func TestUpcheck_IsValid_Type(t *testing.T) {
	tests := []struct {
		name, proxyType, wantErrMsg string
	}{
		{
			name:       "not set",
			proxyType:  "",
			wantErrMsg: typeField + " must be http or ws",
		},
		{
			name:       "invalid",
			proxyType:  "unix",
			wantErrMsg: typeField + " must be http or ws",
		},
		{
			name:       "http",
			proxyType:  "http",
			wantErrMsg: "",
		},
		{
			name:       "ws",
			proxyType:  "ws",
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidProxy()
			c.Type = tt.proxyType

			err := c.IsValid()

			if tt.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.IsType(t, &fieldErr{}, err)
				require.EqualError(t, err, tt.wantErrMsg)
			}
		})
	}
}

func TestProxy_IsValid_ProxyAddress(t *testing.T) {
	tests := []struct {
		name, proxyAddr, wantErr string
	}{
		{
			name:      "not set",
			proxyAddr: "",
			wantErr:   proxyAddressField + " is empty",
		},
		{
			name:      "invalid url",
			proxyAddr: "://no-scheme",
			wantErr:   proxyAddressField + ` parse "://no-scheme": missing protocol scheme`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidProxy()
			c.ProxyAddr = tt.proxyAddr

			err := c.IsValid()

			require.IsType(t, &fieldErr{}, err)
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestProxy_IsValid_UpstreamAddress(t *testing.T) {
	tests := []struct {
		name, upstreamAddr, wantErr string
	}{
		{
			name:         "not set",
			upstreamAddr: "",
			wantErr:      upstreamAddressField + " is empty",
		},
		{
			name:         "invalid url",
			upstreamAddr: "://no-scheme",
			wantErr:      upstreamAddressField + ` parse "://no-scheme": missing protocol scheme`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidProxy()
			c.UpstreamAddr = tt.upstreamAddr

			err := c.IsValid()

			require.IsType(t, &fieldErr{}, err)
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestProxy_IsValid_ProxyPaths(t *testing.T) {
	tests := []struct {
		name       string
		proxyPaths []string
		wantErr    string
	}{
		{
			name:       "not set",
			proxyPaths: nil,
			wantErr:    proxyPathsField + " is empty",
		},
		{
			name:       "empty",
			proxyPaths: []string{},
			wantErr:    proxyPathsField + " is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidProxy()
			c.ProxyPaths = tt.proxyPaths

			err := c.IsValid()

			require.IsType(t, &fieldErr{}, err)
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestProxy_IsValid_ReadTimeout(t *testing.T) {
	c := minimumValidProxy()
	c.ReadTimeout = 0

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, readTimeoutField+" must be > 0")
}

func TestProxy_IsValid_WriteTimeout(t *testing.T) {
	c := minimumValidProxy()
	c.WriteTimeout = 0

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, writeTimeoutField+" must be > 0")
}

func TestProxy_IsValid_ProxyTLSConfig(t *testing.T) {
	c := minimumValidProxy()
	c.ProxyServerTLSConfig = &ServerTLS{}

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, fmt.Sprintf("%v.%v %v", proxyTlsConfigField, certificateFileField, "is empty"))
}

func TestProxy_IsValid_clientTLSConfig(t *testing.T) {
	c := minimumValidProxy()
	c.ClientTLSConfig = &ClientTLS{}

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, fmt.Sprintf("%v.%v %v", clientTlsConfigField, caCertificateFileField, "is empty"))
}
