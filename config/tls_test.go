package config

import (
	"github.com/stretchr/testify/require"
	"testing"
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
	require.EqualError(t, err, "certificateFile is empty")
}

func TestServerTLS_IsValid_KeyFile(t *testing.T) {
	c := minimumValidServerTLS()
	c.KeyFile = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, "keyFile is empty")
}

func TestServerTLS_Load(t *testing.T) {
	require.True(t, false, "implement me")
}

func TestClientTLS_IsValid_CaCertificateFile(t *testing.T) {
	c := minimumValidClientTLS()
	c.CACertFile = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, "caCertificateFile is empty")
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
			wantErrMsg: "certificateFile must be set as keyFile is set",
		},
		{
			name:       "only certificateFile set",
			certFile:   "/path/to/cert.pem",
			keyFile:    "",
			wantErrMsg: "keyFile must be set as certificateFile is set",
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
