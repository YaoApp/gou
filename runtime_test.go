package gou

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/runtime"
	"github.com/yaoapp/kun/utils"
)

var yao = runtime.Yao().
	AddFunction("UnitTestFn", func(global map[string]interface{}, sid string, args ...interface{}) interface{} {
		utils.Dump(global, sid, args)
		return args
	}).
	AddFunction("Process", func(global map[string]interface{}, sid string, args ...interface{}) interface{} {
		return map[string]interface{}{"global": global, "sid": sid, "args": args}
	}).
	AddObject("console", map[string]func(global map[string]interface{}, sid string, args ...interface{}) interface{}{
		"log": func(global map[string]interface{}, sid string, args ...interface{}) interface{} {
			utils.Dump(args)
			return nil
		},
	})

func TestRuntimeLoad(t *testing.T) {
	err := yao.Load(path.Join(TestScriptRoot, "test.js"), "test")
	assert.Nil(t, err)
}

func TestRuntimeExec(t *testing.T) {
	ctx := context.Background()
	err := yao.Load(path.Join(TestScriptRoot, "test.js"), "test")
	assert.Equal(t, nil, err)
	getArgs := yao.New("test", "getArgs").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)

	v, err := getArgs.Call("world", 1, 0.618, []interface{}{"foo", "bar"}, map[string]interface{}{"foo": "bar", "int": 1})
	assert.Nil(t, err)
	fmt.Println(v)

	getArgs = yao.New("test", "getArgs").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)
	v, err = getArgs.Call("yao", 2, 1.618, []interface{}{"code", "ping"}, map[string]interface{}{"one": "two", "int": 5})
	assert.Nil(t, err)
	fmt.Println(v)
}

func TestRuntimeExecES6(t *testing.T) {
	ctx := context.Background()
	err := yao.Load(path.Join(TestScriptRoot, "es6.js"), "es6")
	assert.Equal(t, nil, err)
	now := yao.New("es6", "now").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)
	v, err := now.Call("world", 1, 0.618, []interface{}{"foo", "bar"}, map[string]interface{}{"foo": "bar", "int": 1})
	assert.Nil(t, err)
	fmt.Println(v)

	promiseTest := yao.New("es6", "promiseTest").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)
	v, err = promiseTest.Call()
	assert.Nil(t, err)
	fmt.Println(v)

	asyncTest := yao.New("es6", "asyncTest").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)
	v, err = asyncTest.Call()
	assert.Nil(t, err)
	fmt.Println(v)

	processTest := yao.New("es6", "processTest").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)
	v, err = processTest.Call()
	assert.Nil(t, err)
	fmt.Println(v)
}
