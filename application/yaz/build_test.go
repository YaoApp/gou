package yaz

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application/yaz/ciphers"
)

func TestBuildSaveTo(t *testing.T) {
	vars := prepare(t)
	tempDir, err := ioutil.TempDir(os.TempDir(), "pack-*")
	if err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(tempDir, "test.yaz")
	err = BuildSaveTo(vars["root"], file)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)
}

func TestEncryptAndDecryptBytes(t *testing.T) {

	vars := prepare(t)
	file, err := Build(vars["root"])
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	aesCipher := ciphers.NewAES([]byte(vars["aseKey"]))
	dataEncrypt, err := EncryptBytes(aesCipher, data)
	if err != nil {
		t.Fatal(err)
	}

	dataDecrypt, err := DecryptBytes(aesCipher, dataEncrypt)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, data, dataDecrypt)
	assert.NotEqual(t, dataEncrypt, dataDecrypt)
}

func TestEncryptAndDecryptFile(t *testing.T) {

	vars := prepare(t)
	file, err := Build(vars["root"])
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)
	fmt.Println(file)

	tempDir := filepath.Dir(file)

	aesCipher := ciphers.NewAES([]byte(vars["aseKey"]))
	fileEncrypt := filepath.Join(tempDir, "test.yax")
	err = EncryptFile(aesCipher, file, fileEncrypt)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileEncrypt)

	fileDecrypt := filepath.Join(tempDir, "test-new.yaz")
	err = DecryptFile(aesCipher, fileEncrypt, fileDecrypt)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileDecrypt)

	dataEncrypt, err := os.ReadFile(fileEncrypt)
	if err != nil {
		t.Fatal(err)
	}

	dataDecrypt, err := os.ReadFile(fileDecrypt)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, data, dataDecrypt)
	assert.NotEqual(t, dataEncrypt, dataDecrypt)
}

func prepare(t *testing.T) map[string]string {

	root := os.Getenv("GOU_TEST_APPLICATION")
	if root == "" {
		t.Fatal("GOU_TEST_APPLICATION is not set")
	}

	aseKey := "0123456789123456"
	return map[string]string{"root": root, "aseKey": aseKey}
}
