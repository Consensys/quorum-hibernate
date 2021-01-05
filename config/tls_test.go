package config

import (
	"encoding/json"
	"fmt"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	certFile = "resources/cert.pem"
	keyFile  = "resources/key.pem"
)

func minimumValidServerTLS() ServerTLS {
	return ServerTLS{
		KeyFile:          keyFile,
		CertFile:         certFile,
		ClientCaCertFile: "",
	}
}

func minimumValidClientTLS() ClientTLS {
	return ClientTLS{
		CACertFile:         certFile,
		KeyFile:            "",
		CertFile:           "",
		InsecureSkipVerify: false,
	}
}

func TestServerTLS_Unmarshal(t *testing.T) {
	tests := []struct {
		name, configTemplate string
	}{
		{
			name: "json",
			configTemplate: `
{
	"%v": "/path/to/key.pem",
	"%v": "/path/to/cert.pem",
	"%v": "/path/to/ca.pem"
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "/path/to/key.pem"
%v = "/path/to/cert.pem"
%v = "/path/to/ca.pem"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(tt.configTemplate, keyFileField, certificateFileField, clientCaCertificateField)

			want := ServerTLS{
				KeyFile:          "/path/to/key.pem",
				CertFile:         "/path/to/cert.pem",
				ClientCaCertFile: "/path/to/ca.pem",
			}

			var (
				got ServerTLS
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

func TestClientTLS_Unmarshal(t *testing.T) {
	tests := []struct {
		name, configTemplate string
	}{
		{
			name: "json",
			configTemplate: `
{
	"%v": "/path/to/key.pem",
	"%v": "/path/to/cert.pem",
	"%v": "/path/to/ca.pem",
	"%v": true
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "/path/to/key.pem"
%v = "/path/to/cert.pem"
%v = "/path/to/ca.pem"
%v = true`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			conf := fmt.Sprintf(tt.configTemplate, keyFileField, certificateFileField, caCertificateFileField, insecureSkipVerifyField)

			want := ClientTLS{
				KeyFile:            "/path/to/key.pem",
				CertFile:           "/path/to/cert.pem",
				CACertFile:         "/path/to/ca.pem",
				InsecureSkipVerify: true,
			}

			var (
				got ClientTLS
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

func TestServerTLS_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidServerTLS()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestServerTLS_IsValid_CertificateFile_NotSet(t *testing.T) {
	c := minimumValidServerTLS()
	c.CertFile = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, certificateFileField+" is empty")
}

func TestServerTLS_IsValid_CertificateFile_NotFound(t *testing.T) {
	c := minimumValidServerTLS()
	c.CertFile = "notfound.pem"

	err := c.IsValid()

	require.EqualError(t, err, "open notfound.pem: no such file or directory")
}

func TestServerTLS_IsValid_KeyFile_NotSet(t *testing.T) {
	c := minimumValidServerTLS()
	c.KeyFile = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, keyFileField+" is empty")
}

func TestServerTLS_IsValid_KeyFile_NotFound(t *testing.T) {
	c := minimumValidServerTLS()
	c.KeyFile = "notfound.pem"

	err := c.IsValid()

	require.EqualError(t, err, "open notfound.pem: no such file or directory")
}

func TestServerTLS_IsValid_ClientCACertificateFile_NotFound(t *testing.T) {
	c := minimumValidServerTLS()
	c.ClientCaCertFile = "notfound.pem"

	err := c.IsValid()

	require.EqualError(t, err, "open notfound.pem: no such file or directory")
}

func TestServerTLS_IsValid_LoadsConfig(t *testing.T) {
	c := minimumValidServerTLS()

	require.Nil(t, c.TlsCfg)

	c.IsValid()

	require.NotNil(t, c.TlsCfg)
}

func TestClientTLS_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidClientTLS()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestClientTLS_IsValid_CaCertificateFile_NotSet(t *testing.T) {
	c := minimumValidClientTLS()
	c.CACertFile = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, caCertificateFileField+" is empty")
}

func TestClientTLS_IsValid_CaCertificateFile_NotFound(t *testing.T) {
	c := minimumValidClientTLS()
	c.CACertFile = "notfound.pem"

	err := c.IsValid()

	require.EqualError(t, err, "open notfound.pem: no such file or directory")
}

func TestClientTLS_IsValid_CertificateAndKeyFile_NotSetCombinations(t *testing.T) {
	tests := []struct {
		name, certFile, keyFile, wantErrMsg string
	}{
		{
			name:       "both set",
			certFile:   certFile,
			keyFile:    keyFile,
			wantErrMsg: "",
		},
		{
			name:       "only keyFile set",
			certFile:   "",
			keyFile:    keyFile,
			wantErrMsg: fmt.Sprintf("%v must be set as %v is set", certificateFileField, keyFileField),
		},
		{
			name:       "only certificateFile set",
			certFile:   certFile,
			keyFile:    "",
			wantErrMsg: fmt.Sprintf("%v must be set as %v is set", keyFileField, certificateFileField),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidClientTLS()
			c.CertFile = tt.certFile
			c.KeyFile = tt.keyFile

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

func TestClientTLS_IsValid_CertificateFile_NotFound(t *testing.T) {
	c := minimumValidClientTLS()
	c.CertFile = "notfound.pem"
	c.KeyFile = keyFile

	err := c.IsValid()

	require.EqualError(t, err, "open notfound.pem: no such file or directory")
}

func TestClientTLS_IsValid_KeyFile_NotFound(t *testing.T) {
	c := minimumValidClientTLS()
	c.KeyFile = "notfound.pem"
	c.CertFile = certFile

	err := c.IsValid()

	require.EqualError(t, err, "open notfound.pem: no such file or directory")
}

func TestClientTLS_IsValid_LoadsConfig(t *testing.T) {
	c := minimumValidClientTLS()

	require.Nil(t, c.TlsCfg)

	c.IsValid()

	require.NotNil(t, c.TlsCfg)
}
