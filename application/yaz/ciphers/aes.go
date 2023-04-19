package ciphers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"io"
)

// AES is a type that represents the AES cipher.
type AES struct {
	key []byte
}

// NewAES creates a new AES cipher.
func NewAES(key []byte) AES {
	return AES{key: key}
}

// Encrypt encrypts a byte slice.
func (aesCipher AES) Encrypt(reader io.Reader, writer io.Writer) error {
	const blockSize = aes.BlockSize
	key := make([]byte, blockSize)
	copy(key, aesCipher.key)

	var iv [blockSize]byte
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	stream := cipher.NewCFBEncrypter(block, iv[:])
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		stream.XORKeyStream(buf[:n], buf[:n])
		if _, err := writer.Write(buf[:n]); err != nil {
			return err
		}
	}

	return nil
}

// Decrypt decrypts a byte slice.
func (aesCipher AES) Decrypt(reader io.Reader, writer io.Writer) error {

	const blockSize = aes.BlockSize
	key := make([]byte, blockSize)
	copy(key, aesCipher.key)

	var iv [blockSize]byte
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	stream := cipher.NewCFBDecrypter(block, iv[:])
	buf := make([]byte, 4096)

	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		stream.XORKeyStream(buf[:n], buf[:n])
		if _, err := writer.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}

// padding
func (aesCipher AES) pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// unpading
func (aesCipher AES) pkcs5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
