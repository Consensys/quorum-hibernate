package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
)

//TODO(cjh) a lot of this is copied from quorum-security-plugin-enterprise config/core.go and tls/tls.go with some small alterations.
//  Might make sense to export this logic in the plugin, or move out to quorum-go-utils project

type ServerTLS struct {
	KeyFile          string `toml:"keyFile" json:"keyFile"`
	CertFile         string `toml:"certificateFile" json:"certificateFile"`
	ClientCaCertFile string `toml:"clientCaCertificateFile" json:"clientCaCertificateFile"`
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

func (c *ServerTLS) TLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// Support only TLS1.2 & Above
		MinVersion: tls.VersionTLS12,
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
	CertFile           string `toml:"certificateFile" json:"certificateFile"`
	KeyFile            string `toml:"keyFile" json:"keyFile"`
	CACertFile         string `toml:"caCertificateFile" json:"caCertificateFile"`
	InsecureSkipVerify bool   `toml:"insecureSkipVerify" json:"insecureSkipVerify"`
	TlsCfg             *tls.Config
}

func (c *ClientTLS) SetTLSConfig() error {
	var err error
	c.TlsCfg, err = c.TLSConfig()
	return err
}

func (c *ClientTLS) IsValid() error {
	if c.CACertFile == "" {
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
	tlsConfig := new(tls.Config)
	tlsConfig.InsecureSkipVerify = c.InsecureSkipVerify
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
