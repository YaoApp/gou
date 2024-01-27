package http

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/cast"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// USAGE
// http.Get(...args)
// args[0] URL
// args[1] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"}, {"k2":"v2"}], k1=v1&k2=v2&k3=k3
// args[2] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
//
// http.Post(...args)
// args[0] URL
// args[1] Payload <Optional> {"foo":"bar"} ["foo", "bar", {"k1":"v1"}], "k1=v1&k2=v2", "/path/root/file", ...
// args[2] Files   <Optional> {"foo":"/path/root/file"}
// args[3] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[4] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
//
// http.Put(...args)
// args[0] URL
// args[1] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path", "k1=v1&k2=v2" ...
// args[2] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[3] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
//
// http.Head(...args)
// args[0] URL
// args[1] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path", "k1=v1&k2=v2" ...
// args[2] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[3] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
//
// http.Patch(...args)
// args[0] URL
// args[1] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path", "k1=v1&k2=v2" ...
// args[2] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[3] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
//
// http.Delete(...args)
// args[0] URL
// args[1] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path", "k1=v1&k2=v2" ...
// args[2] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[3] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
//
// http.Send(...args)
// args[0] Method GET/POST/PUT/HEAD/PATCH/DELETE/...
// args[1] URL
// args[2] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path"
// args[3] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[4] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
// args[5] Files   <Optional> [{"name": "field_name", "path": "/path/root/file",  "data": "base64EncodedFileContent" }...]
//
// http.Stream(...args)
// args[0] Method GET/POST/PUT/HEAD/PATCH/DELETE/...
// args[1] URL
// args[2] Callback function ( data ) => { return 1 } / (data) => { return 0 } / (data) => { return -1 }
// args[3] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path"
// args[4] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[5] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
//

// Object Javascript API
type Object struct {
	fileRoot string
}

// New create a new HTTP Object
func New(root string) *Object {
	return &Object{fileRoot: root}
}

// SetFileRoot set the root space of file
func (obj *Object) SetFileRoot(root string) error {
	path, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	obj.fileRoot = path
	return nil
}

// ExportObject http object
func (obj *Object) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("Get", obj.get(iso))
	tmpl.Set("Post", obj.post(iso))
	tmpl.Set("Put", obj.put(iso))
	tmpl.Set("Patch", obj.patch(iso))
	tmpl.Set("Delete", obj.delete(iso))
	tmpl.Set("Send", obj.send(iso))
	tmpl.Set("Stream", obj.stream(iso))
	return tmpl
}

func (obj *Object) get(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		err := obj.validateArgNums(info, 1)
		if err != nil {
			return obj.vReturn(info, err)
		}

		req, err := obj.new(info, 0, 1)
		if err != nil {
			return obj.vReturn(info, err)
		}

		return obj.vReturn(info, req.Get())
	})
}

func (obj *Object) post(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		err := obj.validateArgNums(info, 1)
		if err != nil {
			return obj.vReturn(info, err)
		}

		args := info.Args()
		var payload interface{}
		if len(args) > 1 {
			value, err := bridge.GoValue(args[1], info.Context())
			if err != nil {
				return obj.vReturn(info, &http.Response{
					Status:  400,
					Code:    400,
					Message: fmt.Sprintf("args[%d] parameter error: %s", 1, err.Error()),
					Headers: map[string][]string{},
					Data:    nil,
				})
			}
			payload = value
		}

		req, err := obj.new(info, 0, 3)
		if err != nil {
			return obj.vReturn(info, err)
		}

		// Upload a file via payload
		if req.GetHeader("Content-Type") == "multipart/form-data" {

			if file, ok := payload.(string); ok {

				if obj.fileRoot != "" {
					file = filepath.Join(obj.fileRoot, file)
				}

				fileAbs, err := filepath.Abs(file)
				if err != nil {
					return obj.vReturn(info, &http.Response{
						Status:  400,
						Code:    400,
						Message: fmt.Sprintf("args[%d] parameter error: %s", 2, err.Error()),
						Headers: map[string][]string{},
						Data:    nil,
					})
				}
				payload = fileAbs
			}
		}

		// Upload files via files
		files := map[string]interface{}{}
		if len(args) > 2 {
			data, err := args[2].MarshalJSON()
			if err != nil {
				return obj.vReturn(info, &http.Response{
					Status:  400,
					Code:    400,
					Message: fmt.Sprintf("args[%d] parameter error: %s", 3, err.Error()),
					Headers: map[string][]string{},
					Data:    nil,
				})
			}

			err = jsoniter.Unmarshal(data, &files)
			if err != nil {
				return obj.vReturn(info, &http.Response{
					Status:  400,
					Code:    400,
					Message: fmt.Sprintf("args[%d] parameter error: %s", 3, err.Error()),
					Headers: map[string][]string{},
					Data:    nil,
				})
			}
		}

		for name, val := range files {
			if file, ok := val.(string); ok {
				if obj.fileRoot != "" {
					file = filepath.Join(obj.fileRoot, file)
				}

				file, err := filepath.Abs(file)
				if err != nil {
					return obj.vReturn(info, &http.Response{
						Status:  400,
						Code:    400,
						Message: fmt.Sprintf("args[%d] parameter error: %s", 2, err.Error()),
						Headers: map[string][]string{},
						Data:    nil,
					})
				}
				req.AddFile(name, file)
			}
		}

		return obj.vReturn(info, req.Post(payload))
	})
}

func (obj *Object) put(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		err := obj.validateArgNums(info, 1)
		if err != nil {
			return obj.vReturn(info, err)
		}

		args := info.Args()
		var payload interface{}
		if len(args) > 1 {
			payload = args[1]
		}

		req, err := obj.new(info, 0, 2)
		if err != nil {
			return obj.vReturn(info, err)
		}

		return obj.vReturn(info, req.Put(payload))
	})
}

func (obj *Object) patch(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		err := obj.validateArgNums(info, 1)
		if err != nil {
			return obj.vReturn(info, err)
		}

		args := info.Args()
		var payload interface{}
		if len(args) > 1 {
			payload = args[1]
		}

		req, err := obj.new(info, 0, 2)
		if err != nil {
			return obj.vReturn(info, err)
		}

		return obj.vReturn(info, req.Patch(payload))
	})
}

func (obj *Object) delete(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		err := obj.validateArgNums(info, 1)
		if err != nil {
			return obj.vReturn(info, err)
		}

		args := info.Args()
		var payload interface{}
		if len(args) > 1 {
			payload = args[1]
		}

		req, err := obj.new(info, 0, 2)
		if err != nil {
			return obj.vReturn(info, err)
		}

		return obj.vReturn(info, req.Delete(payload))
	})
}

func (obj *Object) send(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		err := obj.validateArgNums(info, 2)
		if err != nil {
			return obj.vReturn(info, err)
		}

		args := info.Args()
		method := args[0].String()
		var payload interface{}
		if len(args) > 2 {
			payload = args[2]
		}

		req, err := obj.new(info, 1, 3)
		if err != nil {
			return obj.vReturn(info, err)
		}

		// Upload a file via payload
		if method == "POST" && len(args) > 5 {
			var files []http.File
			bytes, err := jsoniter.Marshal(args[5])
			if err != nil {
				return obj.vReturn(info,
					&http.Response{
						Status:  400,
						Code:    400,
						Message: fmt.Sprintf("args[%d] parameter error: %s", 5, err.Error()),
						Headers: map[string][]string{},
						Data:    nil,
					})
			}

			err = jsoniter.Unmarshal(bytes, &files)
			if err != nil {
				return obj.vReturn(info,
					&http.Response{
						Status:  400,
						Code:    400,
						Message: fmt.Sprintf("args[%d] parameter error: %s", 5, err.Error()),
						Headers: map[string][]string{},
						Data:    nil,
					})
			}

			for _, file := range files {
				data, err := base64.StdEncoding.DecodeString(file.Data)
				if err != nil {
					return obj.vReturn(info,
						&http.Response{
							Status:  400,
							Code:    400,
							Message: fmt.Sprintf("args[%d] parameter error: %s", 5, err.Error()),
							Headers: map[string][]string{},
							Data:    nil,
						})
				}
				req.AddFileBytes(file.Name, file.Path, data)
			}
		}

		return obj.vReturn(info, req.Send(method, payload))
	})
}

func (obj *Object) stream(iso *v8go.Isolate) *v8go.FunctionTemplate {

	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		err := obj.validateArgNums(info, 2)
		if err != nil {
			return obj.vReturn(info, err)
		}

		args := info.Args()
		method := args[0].String()
		cb, v8err := args[2].AsFunction()

		if v8err != nil {
			return obj.vReturn(info, &http.Response{
				Status:  400,
				Code:    400,
				Message: v8err.Error(),
				Headers: map[string][]string{},
				Data:    nil,
			})
		}

		var payload interface{}
		if len(args) > 3 {
			payload = args[3]
		}

		req, err := obj.new(info, 1, 4)
		if err != nil {
			return obj.vReturn(info, err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		v8err = req.Stream(ctx, method, payload, func(data []byte) int {
			v, err := v8go.NewValue(iso, string(data))
			if err != nil {
				log.Error("[http.Stream] %s %s %s", method, args[2].String(), err.Error())
				return http.HandlerReturnError
			}

			ret, err := cb.Call(info.This(), v)
			if err != nil {
				log.Error("[http.Stream] %s %s %s", method, args[2].String(), err.Error())
				return http.HandlerReturnError
			}

			return int(ret.Integer())
		})

		if v8err != nil {
			return obj.vReturn(info, &http.Response{
				Status:  400,
				Code:    400,
				Message: v8err.Error(),
				Headers: map[string][]string{},
				Data:    nil,
			})
		}

		return obj.vReturn(info, &http.Response{
			Status:  200,
			Code:    200,
			Message: "ok",
			Headers: map[string][]string{},
			Data:    nil,
		})
	})
}

func (obj *Object) callback(info *v8go.FunctionCallbackInfo) {

}

func (obj *Object) vReturn(info *v8go.FunctionCallbackInfo, resp *http.Response) *v8go.Value {

	jsValue, err := bridge.JsValue(info.Context(), resp)
	if err != nil {
		return bridge.JsError(info.Context(), err)
	}

	return jsValue
}

func (obj *Object) validateArgNums(info *v8go.FunctionCallbackInfo, length int) *http.Response {
	if len(info.Args()) < length {
		msg := fmt.Sprintf("Log: %s", "Missing parameters")
		return &http.Response{
			Status:  400,
			Code:    400,
			Message: msg,
			Headers: map[string][]string{},
			Data:    nil,
		}
	}
	return nil
}

// make a *http.Request
func (obj *Object) new(info *v8go.FunctionCallbackInfo, idx, from int) (*http.Request, *http.Response) {

	args := info.Args()
	req := http.New(args[idx].String())

	if len(args) > from {
		input, err := bridge.GoValue(args[from], info.Context())
		if err != nil {
			return nil, &http.Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", from, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}

		values, err := cast.AnyToURLValues(input)
		if err != nil {
			return nil, &http.Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", from, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}

		req.WithQuery(values)
	}

	if len(args) > from+1 {
		input, err := bridge.GoValue(args[from+1], info.Context())
		if err != nil {
			return nil, &http.Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", from, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}

		headers, err := cast.AnyToHeaders(input)
		if err != nil {
			return nil, &http.Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", from+1, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}
		req.WithHeader(headers)
	}

	return req, nil
}
