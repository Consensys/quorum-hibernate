package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
)

type ServerTLS struct {
	KeyFile          string   `toml:"keyFile" json:"keyFile"`
	CertFile         string   `toml:"certificateFile" json:"certificateFile"`
	ClientCaCertFile string   `toml:"clientCaCertificateFile" json:"clientCaCertificateFile"`
	CipherSuites     []string `toml:"cipherSuites" json:"cipherSuites"`
	TlsCfg           *tls.Config
}

func (c *ServerTLS) SetTLSConfig() error {
	var err error
	c.TlsCfg, err = c.TLSConfig()
	return err
}

func (c *ServerTLS) IsValid() error {
	if c.CertFile == "" {
		return newFieldErr("certificateFile", isEmptyErr)
	}
	if c.KeyFile == "" {
		return newFieldErr("keyFile", isEmptyErr)
	}
	if err := c.SetTLSConfig(); err != nil {
		return err
	}
	return nil
}

var (
	// defaultCipherSuites are suites determined to be safe enough for use in most cases (256 bit keys, ECDH key sharing).
	// These are the same ciphers used by default in the quorum security plugin.
	defaultCipherSuites = []string{
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	}
)

func (c *ServerTLS) TLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// Support only TLS1.2 & Above
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
			tls.X25519,
		},
		CipherSuites:             cipherSuitesOrDefault(c.CipherSuites),
		PreferServerCipherSuites: true,
	}

	var caPem []byte
	if c.ClientCaCertFile != "" {
		caPem, err = ioutil.ReadFile(c.ClientCaCertFile)
		if err != nil {
			return nil, err
		}
	}
	if len(caPem) != 0 {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			certPool = x509.NewCertPool()
		}
		certPool.AppendCertsFromPEM(caPem)
		tlsConfig.ClientCAs = certPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

type ClientTLS struct {
	CertFile           string   `toml:"certificateFile" json:"certificateFile"`
	KeyFile            string   `toml:"keyFile" json:"keyFile"`
	CACertFile         string   `toml:"caCertificateFile" json:"caCertificateFile"`
	InsecureSkipVerify bool     `toml:"insecureSkipVerify" json:"insecureSkipVerify"`
	CipherSuites       []string `toml:"cipherSuites" json:"cipherSuites"`
	TlsCfg             *tls.Config
}

func (c *ClientTLS) SetTLSConfig() error {
	var err error
	c.TlsCfg, err = c.TLSConfig()
	return err
}

func (c *ClientTLS) IsValid() error {
	if !c.InsecureSkipVerify && c.CACertFile == "" {
		return newFieldErr("caCertificateFile", isEmptyErr)
	}
	if c.CertFile != "" && c.KeyFile == "" {
		return newFieldErr("keyFile", errors.New("must be set as certificateFile is set"))
	}
	if c.KeyFile != "" && c.CertFile == "" {
		return newFieldErr("certificateFile", errors.New("must be set as keyFile is set"))
	}
	if err := c.SetTLSConfig(); err != nil {
		return err
	}
	return nil
}

func (c *ClientTLS) TLSConfig() (*tls.Config, error) {
	// copied from SecurityPlugin/tls.go::NewHttpClient
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.InsecureSkipVerify,
		CipherSuites:       cipherSuitesOrDefault(c.CipherSuites),
	}

	if !c.InsecureSkipVerify {
		var caPem []byte
		var err error

		if c.CertFile != "" && c.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		if c.CACertFile != "" {
			caPem, err = ioutil.ReadFile(c.CACertFile)
			if err != nil {
				return nil, err
			}
		}
		if len(caPem) != 0 {
			certPool, err := x509.SystemCertPool()
			if err != nil {
				certPool = x509.NewCertPool()
			}
			certPool.AppendCertsFromPEM(caPem)
			tlsConfig.RootCAs = certPool
		}
	}
	return tlsConfig, nil
}

// cipherSuitesOrDefault converts the provided cipher suite names to uint16 IDs if supported.
// Defaults are used if no configured cipher suites are provided.
func cipherSuitesOrDefault(configured []string) []uint16 {
	supportedCipherSuites := tls.CipherSuites()

	var names []string
	if configured != nil {
		names = configured
	} else {
		names = defaultCipherSuites
	}

	var cipherSuites []uint16

	for _, n := range names {
		for _, cc := range supportedCipherSuites {
			if cc.Name == n && !cc.Insecure {
				cipherSuites = append(cipherSuites, cc.ID)
			}
		}
	}

	return cipherSuites
}
