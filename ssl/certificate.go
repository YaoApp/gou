package ssl

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/yaoapp/gou/application"
)

// Certificates loaded
var Certificates = map[string]*Certificate{}

// Load Load the certificate from the given file
func Load(file string, name string) (*Certificate, error) {
	data, err := application.App.Read(file)
	if err != nil {
		return nil, fmt.Errorf("read certificate pem file err:%s", err.Error())
	}
	return LoadCertificate(data, name)
}

// LoadCertificate load the certificate from the given file
func LoadCertificate(data []byte, name string) (*Certificate, error) {
	cert, err := NewCertificate(data)
	if err != nil {
		return nil, err
	}
	Certificates[name] = cert
	return cert, nil
}

// NewCertificate creat a new certificate from file
func NewCertificate(data []byte) (*Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode certificate error")
	}

	switch block.Type {
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse certificate err:%s", err.Error())
		}
		return &Certificate{cert: cert, pub: cert.PublicKey, data: data, Type: block.Type, Format: "PEM"}, nil

	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse certificate err:%s", err.Error())
		}
		return &Certificate{pri: key, data: data, Type: block.Type, Format: "PEM"}, nil

	case "PUBLIC KEY":
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse certificate err:%s", err.Error())
		}
		return &Certificate{pub: key, data: data, Type: block.Type, Format: "PEM"}, nil
	}

	return nil, fmt.Errorf("the kind of PEM should be CERTIFICATE")

}
