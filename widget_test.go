package gou

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/widget"
	"github.com/yaoapp/kun/any"
)

func TestWidgetReload(t *testing.T) {
	load(t)
	v, err := NewProcess("widgets.dyform.Reload", "pad", "{}").Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, nil, v)
}

func TestWidgetCustomProcess(t *testing.T) {
	load(t)
	v, err := NewProcess("widgets.dyform.Save", "pad", "foo").Exec()
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(v).Map()
	assert.Equal(t, "pad", res.Get("instance"))
	assert.Equal(t, "foo", res.Get("payload"))
}

func moduleRegister() widget.ModuleRegister {
	return widget.ModuleRegister{
		"Models": func(name string, source []byte) error {
			fmt.Printf("Model %s Registered\n", name)
			return nil
		},
		"Flows": func(name string, source []byte) error {
			fmt.Printf("Flow %s Registered\n", name)
			return nil
		},
		"Apis": func(name string, source []byte) error {
			fmt.Printf("API %s Registered\n", name)
			return nil
		},
	}
}

func load(t *testing.T) *widget.Widget {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	path := filepath.Join(root, "widgets", "dyform")
	w, err := LoadWidget(path, "dyform", moduleRegister())
	if err != nil {
		t.Fatal(err)
	}
	err = w.Load()
	if err != nil {
		t.Fatal(err)
	}
	return w
}
