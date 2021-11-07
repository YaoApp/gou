package gou

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
)

func getSource() string {
	file, err := os.Open(path.Join(TestScriptRoot, "test.js"))
	if err != nil {
		exception.Err(err, 400).Throw()
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		exception.Err(err, 400).Throw()
	}
	return string(content)
}

func TestLoadScript(t *testing.T) {

	err := LoadScript("test.js", getSource(), "test")
	if err != nil {
		panic(err)
	}
	test := any.MapOf(Scripts["test"]).Dot()
	assert.True(t, test.Has("functions.hello"))
	assert.True(t, test.Has("functions.lastYear"))
	assert.True(t, test.Has("functions.now"))
	assert.True(t, test.Has("functions.main"))
}

func TestJavaScriptCompile(t *testing.T) {
	err := LoadScript("test.js", getSource(), "test")
	if err != nil {
		panic(err)
	}
	vm := NewVM()
	test := Scripts["test"]
	err = vm.Compile(test)
	if err != nil {
		panic(err)
	}
	for _, f := range test.Functions {
		assert.NotNil(t, f.Compiled)
	}
}

func TestJavaScriptRun(t *testing.T) {
	err := LoadScript("test.js", getSource(), "test")
	if err != nil {
		panic(err)
	}
	vm := NewVM()
	test := Scripts["test"]
	err = vm.Compile(test)
	if err != nil {
		panic(err)
	}

	res, err := vm.Run(test, "hello", "foo")
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "hello:foo", res)
	res, err = vm.Run(test, "hello", "bar")
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "hello:bar", res)

	res, err = vm.Run(test, "main", []interface{}{"world"})
	resdot := any.MapOf(res).MapStrAny.Dot()
	assert.Equal(t, "world", resdot.Get("args.0"))
	assert.Equal(t, "hello:world", resdot.Get("hello"))
	assert.True(t, resdot.Has("lastYear"))
	assert.True(t, resdot.Has("now"))
}

func TestJavaScriptRunWithProcess(t *testing.T) {
	err := LoadScript("test.js", getSource(), "test")
	if err != nil {
		panic(err)
	}
	vm := NewVM()
	test := Scripts["test"]
	err = vm.Compile(test)
	if err != nil {
		panic(err)
	}
	res, err := vm.WithProcess("*").Run(test, "helloProcess", "foo")
	if err != nil {
		panic(err)
	}

	resdot := any.MapOf(res).MapStrAny.Dot()
	assert.Equal(t, "foo", resdot.Get("name"))
	assert.Equal(t, "plugins.user.Login", resdot.Get("out.args.0"))
	assert.Equal(t, float64(1024), resdot.Get("out.args.1"))
	assert.Equal(t, "foo", resdot.Get("out.args.2"))
	assert.Equal(t, "login", resdot.Get("out.name"))
}
