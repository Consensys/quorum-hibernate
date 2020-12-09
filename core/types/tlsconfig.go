package types

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
)

//TODO(cjh) a lot of this is copied from quorum-security-plugin-enterprise config/config.go and tls/tls.go with some small alterations.
//  Might make sense to export this logic in the plugin, or move out to quorum-go-utils project

var (
	// copy from crypto/tls/cipher_suites.go per go 1.11.6
	supportedCipherSuites = map[string]uint16{
		"TLS_RSA_WITH_RC4_128_SHA":                tls.TLS_RSA_WITH_RC4_128_SHA,
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":        tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_RC4_128_SHA":          tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":     tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":  tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	}

	defaultCipherSuites = []string{
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	}

	// harden the cipher strength by only using ciphers >=256bits
	defaultCipherSuitesUint16 = []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	}
)

type serverTLSConfig struct {
	KeyFile      string   `toml:"keyFile"`
	CertFile     string   `toml:"certFile"`
	CipherSuites []string `toml:"cipherSuites"`
}

func (c *serverTLSConfig) Convert() (*tls.Config, error) {
	// copied from Quorum/plugin/security/gateway.go::transform

	tlsConfig := &tls.Config{
		// prioritize curve preferences from crypto/tls/common.go#defaultCurvePreferences
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
			tls.X25519,
		},
		// Support only TLS1.2 & Above
		MinVersion: tls.VersionTLS12,
	}

	suites, err := toUint16Array(c.CipherSuites)
	if err != nil {
		return nil, err
	}
	receivedCipherSuites := suites

	cipherSuites := make([]uint16, len(receivedCipherSuites))
	if len(receivedCipherSuites) > 0 {
		for i, cs := range receivedCipherSuites {
			if cs > math.MaxUint16 {
				return nil, errors.New("cipher suite value overflow")
			}
			cipherSuites[i] = uint16(cs)
		}
	} else {
		cipherSuites = defaultCipherSuitesUint16
	}
	tlsConfig.CipherSuites = cipherSuites
	tlsConfig.PreferServerCipherSuites = true

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
	tlsConfig.Certificates = []tls.Certificate{cer}

	return tlsConfig, nil
}

type clientTLSConfig struct {
	CACertFile         string   `toml:"caCertFile"`
	CipherSuites       []string `toml:"cipherSuites"`
	InsecureSkipVerify bool     `toml:"insecureSkipVerify"`
}

func (c *clientTLSConfig) Convert() (*tls.Config, error) {
	// copied from SecurityPlugin/tls.go::NewHttpClient

	tlsConfig := new(tls.Config)

	if len(c.CipherSuites) > 0 {
		suites, err := toUint16Array(c.CipherSuites)
		if err != nil {
			return nil, err
		}
		tlsConfig.CipherSuites = suites
	}
	tlsConfig.InsecureSkipVerify = c.InsecureSkipVerify
	if !c.InsecureSkipVerify {
		//var certPem, caPem []byte
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

func toUint16Array(cipherSuites []string) ([]uint16, error) {
	a := make([]uint16, len(cipherSuites))
	for i, cs := range cipherSuites {
		v, err := toUint16(cs)
		if err != nil {
			return nil, err
		}
		a[i] = v
	}
	return a, nil
}

func toUint16(cipherSuite string) (uint16, error) {
	v, ok := supportedCipherSuites[cipherSuite]
	if ok {
		return v, nil
	}
	return 0, fmt.Errorf("not supported cipher suite %s", cipherSuite)
}
