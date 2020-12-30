package config

import (
	"encoding/json"
	"fmt"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	rpcAddressField  = "rpcAddress"
	rpcCorsListField = "rpcCorsList"
	rpcvHostsField   = "rpcvHosts"
	tlsConfigField   = "tlsConfig"
)

func minimumValidRPCServer() RPCServer {
	return RPCServer{
		RPCAddr:     "http://url",
		RPCCorsList: nil,
		RPCVHosts:   nil,
		TLSConfig:   nil,
	}
}

func TestRPCServer_Unmarshal(t *testing.T) {
	tests := []struct {
		name, configTemplate string
	}{
		{
			name: "json",
			configTemplate: `
{
	"%v": "http://url",
	"%v": ["http://other"],
	"%v": ["http://another"],
	"%v": {}
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "http://url"
%v = ["http://other"]
%v = ["http://another"]
%v = {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(tt.configTemplate, rpcAddressField, rpcCorsListField, rpcvHostsField, tlsConfigField)

			want := RPCServer{
				RPCAddr:     "http://url",
				RPCCorsList: []string{"http://other"},
				RPCVHosts:   []string{"http://another"},
				TLSConfig:   &ServerTLS{},
			}

			var (
				got RPCServer
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

func TestRPCServer_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidRPCServer()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestRPCServer_IsValid_RPCAddress(t *testing.T) {
	tests := []struct {
		name, rpcAddr, wantErr string
	}{
		{
			name:    "not set",
			rpcAddr: "",
			wantErr: rpcAddressField + " is empty",
		},
		{
			name:    "invalid url",
			rpcAddr: "://no-scheme",
			wantErr: rpcAddressField + ` parse "://no-scheme": missing protocol scheme`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidRPCServer()
			c.RPCAddr = tt.rpcAddr

			err := c.IsValid()

			require.IsType(t, &fieldErr{}, err)
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestRPCServer_IsValid_TLSConfig(t *testing.T) {
	c := minimumValidRPCServer()
	c.TLSConfig = &ServerTLS{}

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, fmt.Sprintf("%v.%v %v", tlsConfigField, certificateFileField, "is empty"))
}
