package http

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/cast"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
)

var fileRoot = ""

// HTTPHandlers the http handlers
var HTTPHandlers = map[string]process.Handler{
	"get":    processHTTPGet,
	"post":   processHTTPPost,
	"put":    processHTTPPut,
	"patch":  processHTTPPatch,
	"delete": processHTTPDelete,
	"head":   processHTTPHead,
	"send":   processHTTPSend,
	"stream": processHTTPStream,
}

func init() {
	process.RegisterGroup("http", HTTPHandlers)
}

// SetFileRoot SetFileRoot
func SetFileRoot(root string) error {
	path, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	fileRoot = path
	return nil
}

// http.Get
// args[0] URL
// args[1] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"}, {"k2":"v2"}], k1=v1&k2=v2&k3=k3
// args[2] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
func processHTTPGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	req, err := processHTTPNew(process, 1)
	if err != nil {
		return err
	}
	return req.Get()
}

// http.Post
// args[0] URL
// args[1] Payload <Optional> {"foo":"bar"} ["foo", "bar", {"k1":"v1"}], "k1=v1&k2=v2", "/path/root/file", ...
// args[2] Files   <Optional> {"foo":"/path/root/file"}
// args[3] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[4] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
func processHTTPPost(process *process.Process) interface{} {

	process.ValidateArgNums(1)
	req, err := processHTTPNew(process, 3)
	if err != nil {
		return err
	}

	var payload interface{}
	if process.NumOfArgs() > 1 {
		payload = process.Args[1]
	}

	// Upload a file via payload
	if req.GetHeader("Content-Type") == "multipart/form-data" {

		if file, ok := payload.(string); ok {
			if fileRoot != "" {
				file = filepath.Join(fileRoot, file)
			}

			fileAbs, err := filepath.Abs(file)
			if err != nil {
				return &Response{
					Status:  400,
					Code:    400,
					Message: fmt.Sprintf("args[%d] parameter error: %s", 2, err.Error()),
					Headers: map[string][]string{},
					Data:    nil,
				}
			}
			payload = fileAbs
		}
	}

	// Upload files via files
	files := process.ArgsMap(2, map[string]interface{}{})
	for name, val := range files {
		if file, ok := val.(string); ok {
			if fileRoot != "" {
				file = filepath.Join(fileRoot, file)
			}

			file, err := filepath.Abs(file)
			if err != nil {
				return &Response{
					Status:  400,
					Code:    400,
					Message: fmt.Sprintf("args[%d] parameter error: %s", 2, err.Error()),
					Headers: map[string][]string{},
					Data:    nil,
				}
			}
			req.AddFile(name, file)
		}
	}

	return req.Post(payload)
}

// http.Put
// args[0] URL
// args[1] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path", "k1=v1&k2=v2" ...
// args[2] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[3] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
func processHTTPPut(process *process.Process) interface{} {

	process.ValidateArgNums(1)
	req, err := processHTTPNew(process, 2)
	if err != nil {
		return err
	}

	var payload interface{}
	if process.NumOfArgs() > 1 {
		payload = process.Args[1]
	}

	return req.Put(payload)
}

// http.Patch
// args[0] URL
// args[1] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path", "k1=v1&k2=v2" ...
// args[2] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[3] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
func processHTTPPatch(process *process.Process) interface{} {

	process.ValidateArgNums(1)
	req, err := processHTTPNew(process, 2)
	if err != nil {
		return err
	}

	var payload interface{}
	if process.NumOfArgs() > 1 {
		payload = process.Args[1]
	}

	return req.Patch(payload)
}

// http.Delete
// args[0] URL
// args[1] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path", "k1=v1&k2=v2" ...
// args[2] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[3] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
func processHTTPDelete(process *process.Process) interface{} {

	process.ValidateArgNums(1)
	req, err := processHTTPNew(process, 2)
	if err != nil {
		return err
	}

	var payload interface{}
	if process.NumOfArgs() > 1 {
		payload = process.Args[1]
	}

	return req.Delete(payload)
}

// http.Head
// args[0] URL
// args[1] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path", "k1=v1&k2=v2" ...
// args[2] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[3] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
func processHTTPHead(process *process.Process) interface{} {

	process.ValidateArgNums(1)
	req, err := processHTTPNew(process, 2)
	if err != nil {
		return err
	}

	var payload interface{}
	if process.NumOfArgs() > 1 {
		payload = process.Args[1]
	}

	return req.Head(payload)
}

// http.Send
// args[0] Method GET/POST/PUT/HEAD/PATCH/DELETE/...
// args[1] URL
// args[2] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path"
// args[3] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[4] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
// args[5] Files   <Optional> [{"name": "field_name", "path": "/path/root/file",  "data": "base64EncodedFileContent" }...]
func processHTTPSend(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	method := process.ArgsString(0)
	var payload interface{}
	if process.NumOfArgs() > 2 {
		payload = process.Args[2]
	}

	req := New(process.ArgsString(1))

	if process.NumOfArgs() > 3 {
		values, err := cast.AnyToURLValues(process.Args[3])
		if err != nil {
			return &Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", 3, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}
		req.WithQuery(values)
	}

	if process.NumOfArgs() > 4 {
		headers, err := cast.AnyToHeaders(process.Args[4])
		if err != nil {
			return &Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", 4, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}
		req.WithHeader(headers)
	}

	if process.NumOfArgs() > 5 {
		var files []File
		bytes, err := jsoniter.Marshal(process.Args[5])
		if err != nil {
			return &Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", 5, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}

		err = jsoniter.Unmarshal(bytes, &files)
		if err != nil {
			return &Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", 5, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}

		for _, file := range files {
			data, err := base64.StdEncoding.DecodeString(file.Data)
			if err != nil {
				return &Response{
					Status:  400,
					Code:    400,
					Message: fmt.Sprintf("args[%d] parameter error: %s", 5, err.Error()),
					Headers: map[string][]string{},
					Data:    nil,
				}
			}
			req.AddFileBytes(file.Name, file.Path, data)
		}
	}

	return req.Send(method, payload)
}

// http.Stream
// args[0] Method GET/POST/PUT/PATCH/DELETE/...
// args[1] URL
// args[2] Handler procsss name
// args[3] Payload <Optional> "Foo", {"foo":"bar"}, ["foo", "bar", {"k1":"v1"}], "/root/path"
// args[4] Query Params <Optional> {"k1":"v1", "k2":"v2"}, ["k1=v1","k1"="v11","k2"="v2"], [{"k1":"v1"},{"k1":"v11"},{"k2":"v2"}], k1=v1&k1=v11&k2=k2
// args[5] Headers <Optional> {"K1":"V1","K2":"V2"}  [{"K1":"V1"},{"K1":"V11"},{"K2":"V2"}]
func processHTTPStream(p *process.Process) interface{} {
	p.ValidateArgNums(3)
	method := p.ArgsString(0)
	url := p.ArgsString(1)
	handler := p.ArgsString(2)

	var payload interface{}
	if p.NumOfArgs() > 3 {
		payload = p.Args[3]
	}

	req := New(url)
	if p.NumOfArgs() > 4 {
		values, err := cast.AnyToURLValues(p.Args[4])
		if err != nil {
			return &Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", 4, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}
		req.WithQuery(values)
	}

	if p.NumOfArgs() > 5 {
		headers, err := cast.AnyToHeaders(p.Args[5])
		if err != nil {
			return &Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", 5, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}
		req.WithHeader(headers)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return req.Stream(ctx, method, payload, func(data []byte) int {

		procesHandler, err := process.Of(handler, string(data))
		if err != nil {
			log.Error("[http.Stream] %s %s", handler, err.Error())
			return HandlerReturnError
		}

		res, err := procesHandler.WithSID(p.Sid).WithGlobal(p.Global).Exec()
		if err != nil {
			log.Error("[http.Stream] %s %s", handler, err.Error())
			return HandlerReturnError
		}

		if v, ok := res.(int); ok {
			return v
		}

		return HandlerReturnOk
	})
}

// make a *Request
func processHTTPNew(process *process.Process, from int) (*Request, *Response) {

	req := New(process.ArgsString(0))

	if process.NumOfArgs() > from {
		values, err := cast.AnyToURLValues(process.Args[from])
		if err != nil {
			return nil, &Response{
				Status:  400,
				Code:    400,
				Message: fmt.Sprintf("args[%d] parameter error: %s", from, err.Error()),
				Headers: map[string][]string{},
				Data:    nil,
			}
		}
		req.WithQuery(values)
	}

	if process.NumOfArgs() > from+1 {
		headers, err := cast.AnyToHeaders(process.Args[from+1])
		if err != nil {
			return nil, &Response{
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
