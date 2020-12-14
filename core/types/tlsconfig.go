package types

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
)

//TODO(cjh) a lot of this is copied from quorum-security-plugin-enterprise config/core.go and tls/tls.go with some small alterations.
//  Might make sense to export this logic in the plugin, or move out to quorum-go-utils project

type ServerTLSConfig struct {
	KeyFile  string `toml:"keyFile"`
	CertFile string `toml:"certFile"`
	TlsCfg   *tls.Config
}

func (c *ServerTLSConfig) SetTLSConfig() error {
	var err error
	c.TlsCfg, err = c.TLSConfig()
	return err
}

func (c *ServerTLSConfig) IsValid() error {
	if c.CertFile == "" {
		return errors.New("serverTLSConfig - cert file is empty")
	}
	if c.KeyFile == "" {
		return errors.New("serverTLSConfig - key file is empty")
	}
	err := c.SetTLSConfig()
	return err
}

func (c *ServerTLSConfig) TLSConfig() (*tls.Config, error) {
	certPem, err := ioutil.ReadFile(c.CertFile)
	if err != nil {
		return nil, err
	}
	keyPem, err := ioutil.ReadFile(c.KeyFile)
	if err != nil {
		return nil, err
	}
	cer, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cer},
		// Support only TLS1.2 & Above
		MinVersion: tls.VersionTLS12,
	}
	return tlsConfig, nil
}

type ClientTLSConfig struct {
	CACertFile         string `toml:"caCertFile"`
	InsecureSkipVerify bool   `toml:"insecureSkipVerify"`
	TlsCfg             *tls.Config
}

func (c *ClientTLSConfig) SetTLSConfig() error {
	var err error
	c.TlsCfg, err = c.TLSConfig()
	return err
}

func (c *ClientTLSConfig) IsValid() error {
	if c.CACertFile == "" {
		return errors.New("ClientTLSConfig - certCA file is empty")
	}
	err := c.SetTLSConfig()
	return err
}

func (c *ClientTLSConfig) TLSConfig() (*tls.Config, error) {
	// copied from SecurityPlugin/tls.go::NewHttpClient
	tlsConfig := new(tls.Config)
	tlsConfig.InsecureSkipVerify = c.InsecureSkipVerify
	if !c.InsecureSkipVerify {
		var caPem []byte
		var err error
		// TODO(cjh) make sure we don't need CertFile for the client side of 1-way TLS
		//if c.CertFile != "" {
		//	certPem, err = ioutil.ReadFile(c.CertFile.String())
		//	if err != nil {
		//		return nil, err
		//	}
		//}
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
