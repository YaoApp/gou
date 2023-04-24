package application

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application/yaz"
	"github.com/yaoapp/gou/application/yaz/ciphers"
	"github.com/yaoapp/kun/maps"
)

func TestLoad(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	Load(app)
	assert.NotNil(t, App)
}

func TestOpenFromDisk(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	testParse(t, app)

	_, err = OpenFromDisk("/path/not-exists")
	assert.NotNil(t, err)
}

func TestOpenFromYazFile(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	aesCipher := ciphers.NewAES([]byte("0123456789123456"))

	file, err := yaz.Pack(root, aesCipher)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	app, err := OpenFromYazFile(file, aesCipher)
	if err != nil {
		t.Fatal(err)
	}

	testParse(t, app)

	_, err = OpenFromYazFile("/path/not-exists", nil)
	assert.NotNil(t, err)
}

func TestOpenFromYaz(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	aesCipher := ciphers.NewAES([]byte("0123456789123456"))

	file, err := yaz.Pack(root, aesCipher)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	reader, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	app, err := OpenFromYaz(reader, file, aesCipher)
	if err != nil {
		t.Fatal(err)
	}

	testParse(t, app)

	app, err = OpenFromYazCache(file, aesCipher)
	if err != nil {
		t.Fatal(err)
	}
	testParse(t, app)
}

func testParse(t *testing.T, app Application) {

	data, err := app.Read("app.yao")
	if err != nil {
		t.Fatal(err)
	}

	v := maps.MapStr{}
	err = Parse("app.yao", data, &v)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "::Demo Application", v.Get("name"))

	// Error
	err = Parse("app.yao", []byte(`{"nade":"s}`), &v)
	assert.NotNil(t, err)

	// Yaml
	v = maps.MapStr{}
	err = Parse("test.yml", []byte(`name: "::Demo Application"`), &v)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "::Demo Application", v.Get("name"))

	// Error ext does not support
	err = Parse("app.xls", []byte(`name: "::Demo Application"`), &v)
	assert.NotNil(t, err)
}
