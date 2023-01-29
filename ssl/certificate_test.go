package ssl

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
)

func TestLoadCertificateFrom(t *testing.T) {
	prepare(t)
	cert, err := Load(certFile("cert.pem"), "cert")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "CERTIFICATE", cert.Type)
	assert.NotNil(t, cert.cert)

	_, has := Certificates["cert"]
	assert.True(t, has)
}

func TestLoadCertificateFromPrivate(t *testing.T) {
	prepare(t)
	cert, err := Load(certFile("private.pem"), "private")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "PRIVATE KEY", cert.Type)
	assert.NotNil(t, cert.pri)

	_, has := Certificates["private"]
	assert.True(t, has)
}

func TestLoadCertificateFromPublic(t *testing.T) {
	prepare(t)
	cert, err := Load(certFile("public.pem"), "public")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "PUBLIC KEY", cert.Type)
	assert.NotNil(t, cert.pub)

	_, has := Certificates["public"]
	assert.True(t, has)
}

func prepare(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := application.OpenFromDisk(root) // Load app
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)
}

func certFile(name string) string {
	return path.Join("certs", name)
}
