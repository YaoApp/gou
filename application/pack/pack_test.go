package pack

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTo(t *testing.T) {

	env := prepare(t)
	pack, err := New(env["root"])
	if err != nil {
		t.Fatal(err)
	}

	tempDir, err := ioutil.TempDir(os.TempDir(), "pack-*")
	if err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(tempDir, "test.pkg")
	err = pack.WithPublicKeyString(env["public"]).BuildTo(file)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	unpack, err := pack.WithPrivateKeyString(env["private"]).decode(file)
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(unpack)
}

func TestBuildCert(t *testing.T) {

	env := prepare(t)
	pack, err := New(env["root"])
	if err != nil {
		t.Fatal(err)
	}

	file, err := pack.WithPublicKeyString(env["cert"]).Build()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	unpack, err := pack.WithPrivateKeyString(env["private"]).decode(file)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(unpack)

}

func prepare(t *testing.T) map[string]string {

	root := os.Getenv("GOU_TEST_APPLICATION")
	if root == "" {
		t.Fatal("GOU_TEST_APPLICATION is not set")
	}

	pub, err := ioutil.ReadFile(filepath.Join(root, "certs", "public.pem"))
	if err != nil {
		t.Fatal(err)
	}

	pri, err := ioutil.ReadFile(filepath.Join(root, "certs", "private.pem"))
	if err != nil {
		t.Fatal(err)
	}

	crt, err := ioutil.ReadFile(filepath.Join(root, "certs", "public.pem"))
	if err != nil {
		t.Fatal(err)
	}

	return map[string]string{"root": root, "cert": string(crt), "public": string(pub), "private": string(pri)}
}
