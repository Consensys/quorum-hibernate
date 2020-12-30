package config

import (
	"encoding/json"
	"fmt"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	keyFileField             = "keyFile"
	certificateFileField     = "certificateFile"
	clientCaCertificateField = "clientCaCertificateFile"
	caCertificateFileField   = "caCertificateFile"
	insecureSkipVerifyField  = "insecureSkipVerify"
)

func minimumValidServerTLS() ServerTLS {
	return ServerTLS{
		KeyFile:          "/path/to/key.pem",
		CertFile:         "/path/to/cert.pem",
		ClientCaCertFile: "",
	}
}

func minimumValidClientTLS() ClientTLS {
	return ClientTLS{
		CACertFile:         "/path/to/cert.pem",
		KeyFile:            "",
		CertFile:           "",
		InsecureSkipVerify: false,
	}
}

func TestServerTLS_Unmarshal_Json(t *testing.T) {
	template := `
{
	"%v": "/path/to/key.pem",
	"%v": "/path/to/cert.pem",
	"%v": "/path/to/ca.pem"
}
`
	conf := fmt.Sprintf(template, keyFileField, certificateFileField, clientCaCertificateField)

	want := ServerTLS{
		KeyFile:          "/path/to/key.pem",
		CertFile:         "/path/to/cert.pem",
		ClientCaCertFile: "/path/to/ca.pem",
	}

	var got ServerTLS
	err := json.Unmarshal([]byte(conf), &got)

	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestServerTLS_Unmarshal_Toml(t *testing.T) {
	template := `
	%v = "/path/to/key.pem"
	%v = "/path/to/cert.pem"
	%v = "/path/to/ca.pem"
`
	conf := fmt.Sprintf(template, keyFileField, certificateFileField, clientCaCertificateField)

	want := ServerTLS{
		KeyFile:          "/path/to/key.pem",
		CertFile:         "/path/to/cert.pem",
		ClientCaCertFile: "/path/to/ca.pem",
	}

	var got ServerTLS
	err := toml.Unmarshal([]byte(conf), &got)

	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestClientTLS_Unmarshal_Json(t *testing.T) {
	template := `
{
	"%v": "/path/to/key.pem",
	"%v": "/path/to/cert.pem",
	"%v": "/path/to/ca.pem",
	"%v": true
}
`
	conf := fmt.Sprintf(template, keyFileField, certificateFileField, caCertificateFileField, insecureSkipVerifyField)

	want := ClientTLS{
		KeyFile:            "/path/to/key.pem",
		CertFile:           "/path/to/cert.pem",
		CACertFile:         "/path/to/ca.pem",
		InsecureSkipVerify: true,
	}

	var got ClientTLS
	err := json.Unmarshal([]byte(conf), &got)

	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestClientTLS_Unmarshal_Toml(t *testing.T) {
	template := `
%v = "/path/to/key.pem"
%v = "/path/to/cert.pem"
%v = "/path/to/ca.pem"
%v = true
`
	conf := fmt.Sprintf(template, keyFileField, certificateFileField, caCertificateFileField, insecureSkipVerifyField)

	want := ClientTLS{
		KeyFile:            "/path/to/key.pem",
		CertFile:           "/path/to/cert.pem",
		CACertFile:         "/path/to/ca.pem",
		InsecureSkipVerify: true,
	}

	var got ClientTLS
	err := toml.Unmarshal([]byte(conf), &got)

	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestServerTLS_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidServerTLS()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestServerTLS_IsValid_CertificateFile(t *testing.T) {
	c := minimumValidServerTLS()
	c.CertFile = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, certificateFileField+" is empty")
}

func TestServerTLS_IsValid_KeyFile(t *testing.T) {
	c := minimumValidServerTLS()
	c.KeyFile = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, keyFileField+" is empty")
}

func TestServerTLS_Load(t *testing.T) {
	require.True(t, false, "implement me")
}

func TestClientTLS_IsValid_CaCertificateFile(t *testing.T) {
	c := minimumValidClientTLS()
	c.CACertFile = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, caCertificateFileField+" is empty")
}

func TestClientTLS_IsValid_CertificateAndKeyFile(t *testing.T) {
	tests := []struct {
		name, certFile, keyFile, wantErrMsg string
	}{
		{
			name:       "both set",
			certFile:   "/path/to/cert.pem",
			keyFile:    "/path/to/key.pem",
			wantErrMsg: "",
		},
		{
			name:       "only keyFile set",
			certFile:   "",
			keyFile:    "/path/to/key.pem",
			wantErrMsg: fmt.Sprintf("%v must be set as %v is set", certificateFileField, keyFileField),
		},
		{
			name:       "only certificateFile set",
			certFile:   "/path/to/cert.pem",
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

func TestClientTLS_Load(t *testing.T) {
	require.True(t, false, "implement me")
}
