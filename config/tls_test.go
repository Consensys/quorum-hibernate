package config

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
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

func TestDefaultCipherSuites(t *testing.T) {
	insecureCipherSuites := tls.InsecureCipherSuites()

	for _, s := range defaultCipherSuites {
		for _, insecure := range insecureCipherSuites {
			require.NotEqual(t, insecure.Name, s, "%v should not be a default cipher suite as it is insecure", s)
		}
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
	"%v": "/path/to/ca.pem",
	"%v": [
		"myciphersuite"
]
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "/path/to/key.pem"
%v = "/path/to/cert.pem"
%v = "/path/to/ca.pem"
%v = [
	"myciphersuite"
]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(tt.configTemplate, keyFileField, certificateFileField, clientCaCertificateField, cipherSuitesField)

			want := ServerTLS{
				KeyFile:          "/path/to/key.pem",
				CertFile:         "/path/to/cert.pem",
				ClientCaCertFile: "/path/to/ca.pem",
				CipherSuites:     []string{"myciphersuite"},
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
	"%v": true,
	"%v": [
		"myciphersuite"
	]
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "/path/to/key.pem"
%v = "/path/to/cert.pem"
%v = "/path/to/ca.pem"
%v = true
%v = [
	"myciphersuite"
]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			conf := fmt.Sprintf(tt.configTemplate, keyFileField, certificateFileField, caCertificateFileField, insecureSkipVerifyField, cipherSuitesField)

			want := ClientTLS{
				KeyFile:            "/path/to/key.pem",
				CertFile:           "/path/to/cert.pem",
				CACertFile:         "/path/to/ca.pem",
				InsecureSkipVerify: true,
				CipherSuites:       []string{"myciphersuite"},
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

func TestServerTLS_IsValid_LoadsTLSConfig_Defaults(t *testing.T) {
	c := minimumValidServerTLS()

	require.Nil(t, c.TlsCfg)

	_ = c.IsValid()

	wantCipherSuites := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	}

	wantCurves := []tls.CurveID{
		tls.CurveP521,
		tls.CurveP384,
		tls.CurveP256,
		tls.X25519,
	}

	require.NotNil(t, c.TlsCfg)
	require.Equal(t, wantCipherSuites, c.TlsCfg.CipherSuites)
	require.Equal(t, uint16(tls.VersionTLS12), c.TlsCfg.MinVersion)
	require.Equal(t, wantCurves, c.TlsCfg.CurvePreferences)
	require.True(t, c.TlsCfg.PreferServerCipherSuites)
	require.Nil(t, c.TlsCfg.ClientCAs)
	require.Zero(t, c.TlsCfg.ClientAuth)
	require.NotNil(t, c.TlsCfg.Certificates)
}

func TestServerTLS_IsValid_LoadsTLSConfig_ConfiguredCipherSuites(t *testing.T) {
	c := minimumValidServerTLS()

	c.CipherSuites = []string{
		"TLS_CHACHA20_POLY1305_SHA256",
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
	}

	require.Nil(t, c.TlsCfg)

	_ = c.IsValid()

	wantCipherSuites := []uint16{
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	}

	require.NotNil(t, c.TlsCfg)
	require.Equal(t, wantCipherSuites, c.TlsCfg.CipherSuites)
}

func TestServerTLS_IsValid_LoadsTLSConfig_ClientCA(t *testing.T) {
	c := minimumValidServerTLS()

	c.ClientCaCertFile = certFile

	require.Nil(t, c.TlsCfg)

	_ = c.IsValid()

	require.NotNil(t, c.TlsCfg.ClientCAs)
	require.Equal(t, tls.RequireAndVerifyClientCert, c.TlsCfg.ClientAuth)
}

func TestServerTLS_IsValid_LoadsTLSConfig_EmptyClientCA(t *testing.T) {
	c := minimumValidServerTLS()

	f, err := ioutil.TempFile("", "emptyfile")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	c.ClientCaCertFile = f.Name()

	require.Nil(t, c.TlsCfg)

	_ = c.IsValid()

	require.Nil(t, c.TlsCfg.ClientCAs)
	require.Zero(t, c.TlsCfg.ClientAuth)
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

func TestClientTLS_IsValid_LoadsTLSConfig_Defaults(t *testing.T) {
	c := minimumValidClientTLS()

	require.Nil(t, c.TlsCfg)

	_ = c.IsValid()

	wantCipherSuites := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	}

	require.NotNil(t, c.TlsCfg)
	require.Equal(t, wantCipherSuites, c.TlsCfg.CipherSuites)
	require.Nil(t, c.TlsCfg.ClientCAs)
	require.False(t, c.TlsCfg.InsecureSkipVerify)
	require.NotNil(t, c.TlsCfg.RootCAs)
	require.Nil(t, c.TlsCfg.Certificates)
}

func TestClientTLS_IsValid_LoadsTLSConfig_ConfiguredCipherSuites(t *testing.T) {
	c := minimumValidClientTLS()

	c.CipherSuites = []string{
		"TLS_CHACHA20_POLY1305_SHA256",
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
	}

	require.Nil(t, c.TlsCfg)

	_ = c.IsValid()

	wantCipherSuites := []uint16{
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	}

	require.NotNil(t, c.TlsCfg)
	require.Equal(t, wantCipherSuites, c.TlsCfg.CipherSuites)
}

func TestClientTLS_IsValid_LoadsTLSConfig_InsecureSkipVerify(t *testing.T) {
	c := minimumValidClientTLS()

	c.InsecureSkipVerify = true

	require.Nil(t, c.TlsCfg)

	_ = c.IsValid()

	require.NotNil(t, c.TlsCfg)
	require.True(t, c.TlsCfg.InsecureSkipVerify)
}
