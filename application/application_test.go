package application

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
	_, err := OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}

	_, err = OpenFromDisk("/path/not-exists")
	assert.NotNil(t, err)
}

func TestParse(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}

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
