package ssl

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

var hashTypes = map[string]crypto.Hash{
	"MD4":         crypto.MD4,
	"MD5":         crypto.MD5,
	"SHA1":        crypto.SHA1,
	"SHA224":      crypto.SHA224,
	"SHA256":      crypto.SHA256,
	"SHA384":      crypto.SHA384,
	"SHA512":      crypto.SHA512,
	"MD5SHA1":     crypto.MD5SHA1,
	"RIPEMD160":   crypto.RIPEMD160,
	"SHA3_224":    crypto.SHA3_224,
	"SHA3_256":    crypto.SHA3_256,
	"SHA3_384":    crypto.SHA3_384,
	"SHA3_512":    crypto.SHA3_512,
	"SHA512_224":  crypto.SHA512_224,
	"SHA512_256":  crypto.SHA512_256,
	"BLAKE2s_256": crypto.BLAKE2s_256,
	"BLAKE2b_256": crypto.BLAKE2b_256,
	"BLAKE2b_384": crypto.BLAKE2b_384,
	"BLAKE2b_512": crypto.BLAKE2b_512,
}

// Sign computes a signature for the specified data by generating a cryptographic digital signature
// using the private key associated with private_key.
func Sign(data []byte, cert *Certificate, algorithm string) ([]byte, error) {

	if cert.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("The certificate type is %s, should be a PRIVATE KEY", cert.Type)
	}

	hash, has := hashTypes[algorithm]
	if !has {
		return nil, fmt.Errorf("The algorithm type %s does not support", algorithm)
	}

	privateKey, ok := cert.pri.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("The PRIVATE KEY should be a RSA PRIVATE KEY")
	}

	h := hash.New()
	_, err := h.Write(data)
	if err != nil {
		return nil, err
	}

	hashed := h.Sum(nil)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, hash, hashed)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// SignBase64 Generate signature
func SignBase64(data []byte, cert *Certificate, algorithm string) (string, error) {
	signature, err := Sign(data, cert, algorithm)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// SignStrBase64 Generate signature
func SignStrBase64(data string, cert *Certificate, algorithm string) (string, error) {
	return SignBase64([]byte(data), cert, algorithm)
}

// SignHexBase64 Generate signature
func SignHexBase64(data string, cert *Certificate, algorithm string) (string, error) {
	bytes, err := hex.DecodeString(data)
	if err != nil {
		return "", err
	}
	return SignBase64(bytes, cert, algorithm)
}

// Verify verifies that the signature is correct for the specified data
// using the public key associated with public_key.
// This must be the public key corresponding to the private key used for signing.
func Verify(data []byte, signature []byte, cert *Certificate, algorithm string) (bool, error) {

	if cert.Type != "PUBLIC KEY" && cert.Type != "CERTIFICATE" {
		return false, fmt.Errorf("The certificate type is %s, should be a CERTIFICATE or PUBLIC KEY", cert.Type)
	}

	hash, has := hashTypes[algorithm]
	if !has {
		return false, fmt.Errorf("The algorithm type %s does not support", algorithm)
	}

	publicKey, ok := cert.pub.(*rsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("The PUBLIC KEY should be a RSA PUBLIC KEY")
	}

	h := hash.New()
	_, err := h.Write(data)
	if err != nil {
		return false, err
	}
	hashed := h.Sum(nil)

	err = rsa.VerifyPKCS1v15(publicKey, hash, hashed, signature)
	if err != nil {
		return false, err
	}

	return true, err
}

// VerifyBase64 Verify signature
func VerifyBase64(data []byte, signature string, cert *Certificate, algorithm string) (bool, error) {

	bytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return Verify(data, bytes, cert, algorithm)
}

// VerifyStrBase64 Verify signature
func VerifyStrBase64(data string, signature string, cert *Certificate, algorithm string) (bool, error) {
	return VerifyBase64([]byte(data), signature, cert, algorithm)
}

// VerifyHexBase64 Verify signature
func VerifyHexBase64(data string, signature string, cert *Certificate, algorithm string) (bool, error) {
	bytes, err := hex.DecodeString(data)
	if err != nil {
		return false, err
	}
	return VerifyBase64(bytes, signature, cert, algorithm)
}

// IsCertificateExpired check if the certificate was expired
func IsCertificateExpired(data string) (bool, error) {
	return false, nil
}

// IsCertificateValid check if the certificate is expired at time
func IsCertificateValid(data string, now time.Time) (bool, error) {
	return false, nil
}

// CertificateSerialNumber get the SN of the certificate
func CertificateSerialNumber(data string) (string, error) {
	return "", nil
}
