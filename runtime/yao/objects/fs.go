package objects

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/runtime/yao/values"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// var fs = new FS("system")
// var dataString	  = fs.ReadFile("/root/path/name.file")
// var dataUnit8Array = fs.ReadFileBuffer("/root/path/name.file")
// var length	      = fs.WriteFile("/root/path/name.file", "Hello")
// var length	      = fs.WriteFile("/root/path/name.file", "Hello", 0644 )
// var length	      = fs.WriteFileBuffer("/root/path/name.file", dataUnit8Array)
// var length	      = fs.WriteFileBuffer("/root/path/name.file", dataUnit8Array, 0644 )
// var dirs 		  = fs.ReadDir("/root/path");
// var dirs 		  = fs.ReadDir("/root/path", true);  // recursive
// var err 		      = fs.Mkdir("/root/path");
// var err 		      = fs.Mkdir("/root/path", 0644);
// var err 		      = fs.MkdirAll("/root/path/dir");
// var err 		      = fs.MkdirAll("/root/path/dir", 0644);
// var temp 		  = fs.MkdirTemp();
// var temp 		  = fs.MkdirTemp("/root/path/dir");
// var temp 		  = fs.MkdirTemp("/root/path/dir", "*-logs");

// FSOBJ Javascript API
type FSOBJ struct{}

// NewFS create a new FS Object
func NewFS() *FSOBJ {
	return &FSOBJ{}
}

// ExportObject Export as a FS Object
func (obj *FSOBJ) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("ReadFile", obj.readFile(iso))
	tmpl.Set("ReadFileBuffer", obj.readFileBuffer(iso))
	tmpl.Set("WriteFile", obj.writeFile(iso))
	tmpl.Set("WriteFileBuffer", obj.writeFileBuffer(iso))

	tmpl.Set("ReadDir", obj.readdir(iso))
	tmpl.Set("Mkdir", obj.mkdir(iso))
	tmpl.Set("MkdirAll", obj.mkdirAll(iso))
	tmpl.Set("MkdirTemp", obj.mkdirTemp(iso))
	return tmpl
}

// ExportFunction Export as a javascript FS function
// var fs = new FS("mongo");
// var fs = new FS();  // same with new FS("system");
func (obj *FSOBJ) ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	object := obj.ExportObject(iso)
	tmpl := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		name := "system"
		args := info.Args()
		if len(args) > 0 {
			name = args[0].String()
		}

		if _, has := fs.FileSystems[name]; !has {
			return obj.errorString(info, fmt.Sprintf("%s does not loaded", name))
		}

		this, err := object.NewInstance(info.Context())
		if err != nil {
			return obj.error(info, err)
		}

		this.Set("name", name)
		return this.Value
	})
	return tmpl
}

func (obj *FSOBJ) readdir(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		name := args[0].String()
		recursive := false
		if len(args) > 1 {
			recursive = args[1].Boolean()
		}

		dirs, err := fs.ReadDir(stor, name, recursive)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.stringArrayValue(info, dirs)
	})
}

func (obj *FSOBJ) mkdir(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		name := args[0].String()
		pterm := int(os.ModePerm)
		if len(args) > 1 {
			pterm = int(args[1].Int32())
		}

		err = fs.Mkdir(stor, name, pterm)
		if err != nil {
			return obj.error(info, err)
		}

		return v8go.Null(iso)
	})
}

func (obj *FSOBJ) mkdirAll(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		name := args[0].String()
		pterm := int(os.ModePerm)
		if len(args) > 1 {
			pterm = int(args[1].Int32())
		}

		err = fs.MkdirAll(stor, name, pterm)
		if err != nil {
			return obj.error(info, err)
		}

		return v8go.Null(iso)
	})
}

func (obj *FSOBJ) mkdirTemp(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		name := ""
		if len(args) > 0 {
			name = args[0].String()
		}

		pattern := ""
		if len(args) > 1 {
			pattern = args[1].String()
		}

		path, err := fs.MkdirTemp(stor, name, pattern)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.stringValue(info, path)
	})
}

func (obj *FSOBJ) readFile(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		data, err := fs.ReadFile(stor, info.Args()[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return obj.stringValue(info, string(data))
	})
}

func (obj *FSOBJ) readFileBuffer(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		data, err := fs.ReadFile(stor, info.Args()[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return obj.arrayBufferValue(info, data)
	})
}

func (obj *FSOBJ) writeFile(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		name := args[0].String()
		data := []byte(args[1].String())
		pterm := int(os.ModePerm)
		if len(args) > 2 {
			pterm = int(args[2].Int32())
		}

		length, err := fs.WriteFile(stor, name, data, pterm)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(length))
	})
}

func (obj *FSOBJ) writeFileBuffer(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 3 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		name := args[0].String()
		data := []byte{}
		if info.Args()[1].IsUint32Array() || info.Args()[1].IsArrayBufferView() {
			codes := strings.Split(info.Args()[1].String(), ",")
			for _, code := range codes {
				c, err := strconv.Atoi(code)
				if err != nil {
					return obj.error(info, err)
				}
				data = append(data, byte(c))
			}
		} else {
			data = []byte(info.Args()[1].String())
		}

		pterm := int(os.ModePerm)
		if len(args) > 2 {
			pterm = int(args[2].Int32())
		}

		length, err := fs.WriteFile(stor, name, data, pterm)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(length))
	})
}

func (obj *FSOBJ) getFS(info *v8go.FunctionCallbackInfo) (fs.FileSystem, error) {
	name, err := info.This().Get("name")
	if err != nil {
		return nil, err
	}
	return fs.MustGet(name.String()), nil
}

func (obj *FSOBJ) stringValue(info *v8go.FunctionCallbackInfo, value string) *v8go.Value {
	res, err := v8go.NewValue(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *FSOBJ) stringArrayValue(info *v8go.FunctionCallbackInfo, value []string) *v8go.Value {

	v, err := jsoniter.Marshal(value)
	if err != nil {
		return obj.error(info, err)
	}

	val, err := v8go.JSONParse(info.Context(), string(v))
	if err != nil {
		return obj.error(info, err)
	}
	return val
}

func (obj *FSOBJ) intValue(info *v8go.FunctionCallbackInfo, value int32) *v8go.Value {
	res, err := v8go.NewValue(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *FSOBJ) arrayBufferValue(info *v8go.FunctionCallbackInfo, value []byte) *v8go.Value {
	hexstr := hex.EncodeToString(value)
	res, err := info.Context().RunScript(fmt.Sprintf(`
		function _yao_hexToBytes(hex) {
			for (var bytes = [], c = 0; c < hex.length; c += 2) {
				bytes.push(parseInt(hex.substr(c, 2), 16));
			}
			return bytes;
	  	}
		new Uint8Array(_yao_hexToBytes("%s"));
	`, hexstr), "__temp")

	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *FSOBJ) error(info *v8go.FunctionCallbackInfo, err error) *v8go.Value {
	return obj.errorString(info, fmt.Sprintf("File System: %s", err.Error()))
}

func (obj *FSOBJ) errorString(info *v8go.FunctionCallbackInfo, err string) *v8go.Value {
	msg := fmt.Sprintf("FS: %s", err)
	log.Error(msg)
	return info.Context().Isolate().ThrowException(values.Error(info.Context(), msg))
}
