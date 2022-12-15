//Package system system
package system

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

type Options struct {
	Listen   string `json:"listen,omitempty" description:"listen address"`
	Locale   string `json:"locale,omitempty" description:"default locale for site"`
	CAFile   string `json:"caFile,omitempty" description:"ca file path"`
	CertFile string `json:"certFile,omitempty" description:"cert file path"`
	KeyFile  string `json:"keyFile,omitempty" description:"key file path"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Listen:   ":8080",
		Locale:   "",
		CAFile:   "",
		CertFile: "",
		KeyFile:  "",
	}
}

func (o *Options) TLSConfigEnabled() bool {
	return (o.CertFile != "" && o.KeyFile != "")
}

func (o *Options) TLSConfig() (*tls.Config, error) {
	caFile, certFile, keyFile := o.CAFile, o.CertFile, o.KeyFile

	config := &tls.Config{
		ClientCAs: x509.NewCertPool(),
	}

	if caFile != "" {
		caPem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		config.ClientCAs.AppendCertsFromPEM(caPem)
	}

	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	config.Certificates = append(config.Certificates, certificate)

	return config, nil
}
