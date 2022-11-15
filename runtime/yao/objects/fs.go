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
// var ok 		      = fs.Exists("/root/path");
// var ok 		      = fs.IsDir("/root/path");
// var ok 		      = fs.IsFile("/root/path");
// var ok 		      = fs.Remove("/root/path");
// var ok 		      = fs.RemoveAll("/root/path");
// var err 			  = fs.Chmod("/root/path", 0755)
// var res 			  = fs.BaseName("/root/path/name.file")
// var res 			  = fs.DirName("/root/path/name.file")
// var res 			  = fs.ExtName("/root/path/name.file")
// var res 			  = fs.MimeType("/root/path/name.file")
// var res 			  = fs.Mode("/root/path/name.file")
// var res 			  = fs.Size("/root/path/name.file")
// var res 			  = fs.ModTime("/root/path/name.file")
// var res 			  = fs.Copy("/root/path/foo.file", "/root/path/bar.file")
// var res 			  = fs.Copy("/root/path", "/root/new")
// var res 			  = fs.Move("/root/path/foo.file", "/root/path/bar.file")
// var res 			  = fs.Move("/root/path", "/root/new")

// FSOBJ Javascript API
type FSOBJ struct{}

// NewFS create a new FS Object
func NewFS() *FSOBJ {
	return &FSOBJ{}
}

// ExportObject Export as a FS Object
func (obj *FSOBJ) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("Exists", obj.exists(iso))
	tmpl.Set("IsDir", obj.isdir(iso))
	tmpl.Set("IsFile", obj.isfile(iso))
	tmpl.Set("IsLink", obj.islink(iso))

	tmpl.Set("ReadFile", obj.readFile(iso))
	tmpl.Set("ReadFileBuffer", obj.readFileBuffer(iso))
	tmpl.Set("WriteFile", obj.writeFile(iso))
	tmpl.Set("WriteFileBuffer", obj.writeFileBuffer(iso))
	tmpl.Set("Remove", obj.remove(iso))
	tmpl.Set("RemoveAll", obj.removeAll(iso))

	tmpl.Set("ReadDir", obj.readdir(iso))
	tmpl.Set("Mkdir", obj.mkdir(iso))
	tmpl.Set("MkdirAll", obj.mkdirAll(iso))
	tmpl.Set("MkdirTemp", obj.mkdirTemp(iso))

	tmpl.Set("Chmod", obj.chmod(iso))
	tmpl.Set("BaseName", obj.baseName(iso))
	tmpl.Set("DirName", obj.dirName(iso))
	tmpl.Set("ExtName", obj.extName(iso))
	tmpl.Set("MimeType", obj.mimeType(iso))
	tmpl.Set("Mode", obj.mode(iso))
	tmpl.Set("Size", obj.size(iso))
	tmpl.Set("ModTime", obj.modTime(iso))

	tmpl.Set("Move", obj.move(iso))
	tmpl.Set("Copy", obj.copy(iso))
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

		v, err := info.Context().Global().Get("__YAO_SU_ROOT")
		if err != nil {
			return obj.error(info, err)
		}

		if v.Boolean() {
			_, err := fs.RootGet(name)
			if err != nil {
				return obj.errorString(info, fmt.Sprintf("%s does not loaded", name))
			}
		} else {
			_, err := fs.Get(name)
			if err != nil {
				return obj.errorString(info, fmt.Sprintf("%s does not loaded", name))
			}
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

func (obj *FSOBJ) move(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		err = fs.Move(stor, args[0].String(), args[1].String())
		if err != nil {
			return obj.error(info, err)
		}

		return v8go.Null(iso)
	})
}

func (obj *FSOBJ) copy(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		err = fs.Copy(stor, args[0].String(), args[1].String())
		if err != nil {
			return obj.error(info, err)
		}

		return v8go.Null(iso)
	})
}

func (obj *FSOBJ) baseName(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		res := fs.BaseName(args[0].String())
		return obj.stringValue(info, res)
	})
}

func (obj *FSOBJ) dirName(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		res := fs.DirName(args[0].String())
		return obj.stringValue(info, res)
	})
}

func (obj *FSOBJ) extName(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		res := fs.ExtName(args[0].String())
		return obj.stringValue(info, res)
	})
}

func (obj *FSOBJ) mimeType(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		res, err := fs.MimeType(stor, args[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return obj.stringValue(info, res)
	})
}

func (obj *FSOBJ) size(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		res, err := fs.Size(stor, args[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(res))
	})
}

func (obj *FSOBJ) mode(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		res, err := fs.Mode(stor, args[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return obj.uint32Value(info, uint32(res))
	})
}

func (obj *FSOBJ) modTime(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		res, err := fs.ModTime(stor, args[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(res.Unix()))
	})
}

func (obj *FSOBJ) chmod(iso *v8go.Isolate) *v8go.FunctionTemplate {
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
		perm := args[1].Uint32()

		err = fs.Chmod(stor, name, perm)
		if err != nil {
			return obj.error(info, err)
		}

		return v8go.Null(iso)
	})
}

func (obj *FSOBJ) remove(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		err = fs.Remove(stor, args[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return v8go.Null(iso)
	})
}

func (obj *FSOBJ) removeAll(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		err = fs.RemoveAll(stor, args[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return v8go.Null(iso)
	})
}

func (obj *FSOBJ) exists(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		ok, err := fs.Exists(stor, args[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return obj.boolValue(info, ok)
	})
}

func (obj *FSOBJ) isdir(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		ok := fs.IsDir(stor, args[0].String())
		return obj.boolValue(info, ok)
	})
}

func (obj *FSOBJ) isfile(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		ok := fs.IsFile(stor, args[0].String())
		return obj.boolValue(info, ok)
	})
}

func (obj *FSOBJ) islink(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		ok := fs.IsLink(stor, args[0].String())
		return obj.boolValue(info, ok)
	})
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
		perm := uint32(os.ModePerm)
		if len(args) > 1 {
			perm = args[1].Uint32()
		}

		err = fs.Mkdir(stor, name, perm)
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
		perm := uint32(os.ModePerm)
		if len(args) > 1 {
			perm = args[1].Uint32()
		}

		err = fs.MkdirAll(stor, name, perm)
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

		data, err := fs.ReadFile(stor, args[0].String())
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

		data, err := fs.ReadFile(stor, args[0].String())
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
		perm := uint32(os.ModePerm)
		if len(args) > 2 {
			perm = args[2].Uint32()
		}

		length, err := fs.WriteFile(stor, name, data, perm)
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

		perm := uint32(os.ModePerm)
		if len(args) > 2 {
			perm = args[2].Uint32()
		}

		length, err := fs.WriteFile(stor, name, data, perm)
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

	v, err := info.Context().Global().Get("__YAO_SU_ROOT")
	if err != nil {
		return nil, err
	}

	if v.Boolean() {
		return fs.RootGet(name.String())
	}

	return fs.Get(name.String())
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

func (obj *FSOBJ) int64Value(info *v8go.FunctionCallbackInfo, value int64) *v8go.Value {
	res, err := v8go.NewValue(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *FSOBJ) uint32Value(info *v8go.FunctionCallbackInfo, value uint32) *v8go.Value {
	res, err := v8go.NewValue(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *FSOBJ) boolValue(info *v8go.FunctionCallbackInfo, value bool) *v8go.Value {
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
