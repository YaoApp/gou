package pack

import "crypto/rsa"

// Pack is a type that represents a package.
type Pack struct {
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey
	Root       string
}
