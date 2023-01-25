package wasm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
)

func TestLoad(t *testing.T) {
	prepare(t)
	check(t)
}

// prepare test suit
func prepare(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	wasms := map[string]string{
		"hi":   filepath.Join("scripts", "hi.wasm"),
		"test": filepath.Join("scripts", "test.wasm"),
	}

	// Load app
	app, err := application.OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	// load wasm
	for id, file := range wasms {
		_, err := Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func check(t *testing.T) {
	keys := map[string]bool{}
	for id := range Instances {
		keys[id] = true
	}
	mods := []string{"hi", "test"}
	for _, id := range mods {
		_, has := keys[id]
		assert.True(t, has)
	}
}
