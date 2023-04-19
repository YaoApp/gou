package yaz

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/yaoapp/gou/application/yaz/ciphers"
)

func TestCompress(t *testing.T) {
	vars := prepare(t)
	file, err := Compress(vars["root"])
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)
}

func TestCompressTo(t *testing.T) {
	vars := prepare(t)
	tempDir, err := ioutil.TempDir(os.TempDir(), "pack-*")
	if err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(tempDir, "test.yaz")
	err = CompressTo(vars["root"], file)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)
}

func TestPackUnpack(t *testing.T) {
	vars := prepare(t)
	aesCipher := ciphers.NewAES([]byte(vars["aseKey"]))
	file, err := Pack(vars["root"], aesCipher)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	dir, err := Unpack(file, aesCipher)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
}

func TestPackToUnpackTo(t *testing.T) {
	vars := prepare(t)
	tempDir, err := ioutil.TempDir(os.TempDir(), "pack-*")
	if err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(tempDir, "test.yaz")
	aesCipher := ciphers.NewAES([]byte(vars["aseKey"]))
	err = PackTo(vars["root"], file, aesCipher)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	dir, err := ioutil.TempDir(os.TempDir(), "unpack-*")
	if err != nil {
		t.Fatal(err)
	}

	dir = filepath.Join(dir, "unpack")
	err = UnpackTo(file, dir, aesCipher)
	if err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(dir)
}

func prepare(t *testing.T) map[string]string {

	root := os.Getenv("GOU_TEST_APPLICATION")
	if root == "" {
		t.Fatal("GOU_TEST_APPLICATION is not set")
	}

	aseKey := "0123456789123456"
	return map[string]string{"root": root, "aseKey": aseKey}
}
