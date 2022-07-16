package ssl

import "crypto/x509"

// Certificate the ssl Certificate
type Certificate struct {
	cert   *x509.Certificate
	pub    any
	pri    any
	Type   string
	Format string // PEM
	data   []byte
}
