package ssl

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadCertificateFrom(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := path.Join(root, "certs", "cert.pem")
	cert, err := LoadCertificateFrom(file, "cert")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "CERTIFICATE", cert.Type)
	assert.NotNil(t, cert.cert)

	_, has := Certificates["cert"]
	assert.True(t, has)
}

func TestLoadCertificateFromPrivate(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := path.Join(root, "certs", "private.pem")
	cert, err := LoadCertificateFrom(file, "private")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "PRIVATE KEY", cert.Type)
	assert.NotNil(t, cert.pri)

	_, has := Certificates["private"]
	assert.True(t, has)
}

func TestLoadCertificateFromPublic(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := path.Join(root, "certs", "public.pem")
	cert, err := LoadCertificateFrom(file, "public")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "PUBLIC KEY", cert.Type)
	assert.NotNil(t, cert.pub)

	_, has := Certificates["public"]
	assert.True(t, has)
}
