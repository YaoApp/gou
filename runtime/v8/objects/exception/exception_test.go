package exception

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"rogchap.com/v8go"
)

func TestException(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	e := New()
	global := v8go.NewObjectTemplate(iso)
	global.Set("Exception", e.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	v, err := ctx.RunScript(`
		var test = ()=> {
			var res = 1
			try { throw new Exception("hello", 403); } catch(e){  
				res = {	 "isError":e instanceof Error, "message": e.message, "code": e.code, "name":e.name }
			}
			return res;
		}
		test()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	obj, err := v.AsObject()
	if err != nil {
		t.Fatal(err)
	}

	code, err := obj.Get("code")
	if err != nil {
		t.Fatal(err)
	}

	isError, err := obj.Get("isError")
	if err != nil {
		t.Fatal(err)
	}

	message, err := obj.Get("message")
	if err != nil {
		t.Fatal(err)
	}

	name, err := obj.Get("name")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int32(403), code.Int32())
	assert.Equal(t, true, isError.Boolean())
	assert.Equal(t, "hello", message.String())
	assert.Equal(t, "Exception|403", name.String())
}

func TestExceptionWithoutCode(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	e := New()
	global := v8go.NewObjectTemplate(iso)
	global.Set("Exception", e.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	v, err := ctx.RunScript(`
		var test = ()=> {
			var res = 1
			try { throw new Exception("hello"); } catch(e){  
				res = {	 "isError":e instanceof Error, "message": e.message, "code": e.code, "name":e.name }
			}
			return res;
		}
		test()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	obj, err := v.AsObject()
	if err != nil {
		t.Fatal(err)
	}

	code, err := obj.Get("code")
	if err != nil {
		t.Fatal(err)
	}

	isError, err := obj.Get("isError")
	if err != nil {
		t.Fatal(err)
	}

	message, err := obj.Get("message")
	if err != nil {
		t.Fatal(err)
	}

	name, err := obj.Get("name")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int32(500), code.Int32())
	assert.Equal(t, true, isError.Boolean())
	assert.Equal(t, "hello", message.String())
	assert.Equal(t, "Exception|500", name.String())
}

func TestExceptionWithoutThrow(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	e := New()
	global := v8go.NewObjectTemplate(iso)
	global.Set("Exception", e.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	v, err := ctx.RunScript(`
		var ErrorSys = Error
		Error = undefined;
		var test = ()=> {
			var e = new Exception("hello", 403);
			var res = {	 "isError":e instanceof ErrorSys, "message": e.message, "code": e.code, "name":e.name }
			return res;
		}
		test()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	obj, err := v.AsObject()
	if err != nil {
		t.Fatal(err)
	}

	code, err := obj.Get("code")
	if err != nil {
		t.Fatal(err)
	}

	isError, err := obj.Get("isError")
	if err != nil {
		t.Fatal(err)
	}

	message, err := obj.Get("message")
	if err != nil {
		t.Fatal(err)
	}

	name, err := obj.Get("name")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int32(403), code.Int32())
	assert.Equal(t, false, isError.Boolean())
	assert.Equal(t, "hello", message.String())
	assert.Equal(t, "Exception", name.String())
}
