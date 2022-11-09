package yao

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/gou/runtime/yao/bridge"
	v8 "rogchap.com/v8go"
)

func TestMustAnyToValue(t *testing.T) {
	ctx := v8.NewContext()
	v := bridge.MustAnyToValue(ctx, 0.618)
	assert.True(t, v.IsNumber())
}

func TestLang(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	langRoot := filepath.Join(root, "langs")
	err := lang.Load(langRoot)
	if err != nil {
		t.Fatal(err)
	}

	lang.Pick("zh-cn").AsDefault()
	jsfile := filepath.Join(root, "scripts", "lang.js")

	yao := New(1, "")
	err = yao.Load(jsfile, "lang")
	if err != nil {
		t.Fatal(err)
	}

	v, err := yao.Call(nil, "lang", "TestLang")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "邮政编码", v)
}
