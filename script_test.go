package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
)

func TestLoadScript(t *testing.T) {
	vm := NewJavaScriptVM().MustLoad(path.Join(TestScriptRoot, "test.js"), "test")
	test := any.MapOf(vm.MustGet("test")).Dot()
	assert.True(t, test.Has("functions.hello"))
	assert.True(t, test.Has("functions.lastYear"))
	assert.True(t, test.Has("functions.now"))
	assert.True(t, test.Has("functions.main"))
	assert.True(t, vm.Has("test"))
	assert.False(t, vm.Has("test.x"))
}

func TestJavaScriptCompile(t *testing.T) {
	test, err := NewScript(path.Join(TestScriptRoot, "test.js"), "test")
	if err != nil {
		panic(err)
	}
	vm := NewJavaScriptVM()
	err = vm.Compile(test)
	if err != nil {
		panic(err)
	}
	for _, f := range test.Functions {
		assert.NotNil(t, f.Compiled)
	}
}

func TestJavaScriptRun(t *testing.T) {
	vm := NewJavaScriptVM().MustLoad(path.Join(TestScriptRoot, "test.js"), "test")
	res, err := vm.Run("test", "hello", "foo")
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "hello:foo", res)
	res, err = vm.Run("test", "hello", "bar")
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "hello:bar", res)

	res, err = vm.Run("test", "main", []interface{}{"world"})
	resdot := any.MapOf(res).MapStrAny.Dot()
	assert.Equal(t, "world", resdot.Get("args.0"))
	assert.Equal(t, "hello:world", resdot.Get("hello"))
	assert.True(t, resdot.Has("lastYear"))
	assert.True(t, resdot.Has("now"))
}

func TestJavaScriptRunWithProcess(t *testing.T) {
	vm := NewJavaScriptVM().MustLoad(path.Join(TestScriptRoot, "test.js"), "test")
	res, err := vm.WithProcess("*").Run("test", "helloProcess", "foo")
	if err != nil {
		panic(err)
	}
	resdot := any.MapOf(res).MapStrAny.Dot()
	assert.Equal(t, "foo", resdot.Get("name"))
	assert.Equal(t, float64(1024), resdot.Get("out.args.0"))
	assert.Equal(t, "foo", resdot.Get("out.args.1"))
	assert.Equal(t, "login", resdot.Get("out.name"))
}

func TestJavaScriptRunWithGlobal(t *testing.T) {
	vm := NewJavaScriptVM().MustLoad(path.Join(TestScriptRoot, "test.js"), "test").
		WithGlobal(map[string]interface{}{"hello": "global"})
	res, err := vm.WithProcess("*").Run("test", "helloGlobal", "foo")
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "hello:foo,global:global", res)
}

func TestJavaScriptRunWithSession(t *testing.T) {
	vm := NewJavaScriptVM().MustLoad(path.Join(TestScriptRoot, "test.js"), "test").
		WithSID(session.ID())
	res, err := vm.WithProcess("*").Run("test", "helloSession", "bar")
	if err != nil {
		panic(err)
	}
	resdot := any.MapOf(res).MapStrAny.Dot()
	assert.Equal(t, "bar", resdot.Get("out"))
	assert.Equal(t, "bar", resdot.Get("input"))
}
