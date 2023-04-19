package ciphers

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAES(t *testing.T) {
	data := []byte("hello world")
	aes := NewAES([]byte("0123456789123456"))
	reader := bytes.NewReader(data)
	buffer := &bytes.Buffer{}
	err := aes.Encrypt(reader, buffer)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := buffer.Bytes()

	buffer = &bytes.Buffer{}
	err = aes.Decrypt(bytes.NewReader(encrypted), buffer)
	if err != nil {
		t.Fatal(err)
	}
	decrypted := buffer.Bytes()
	assert.Equal(t, data, decrypted)
}
