package fs

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// var fs = new FS("data")
// var dataString	  = fs.ReadFile("/root/path/name.file")
// var dataUnit8Array = fs.ReadFileBuffer("/root/path/name.file")
// var dataBase64	  = fs.ReadFileBase64("/root/path/name.file")
// var handler		  = fs.ReadCloser("/root/path/name.file")
// var res 			  = fs.Download('/data/path/file.txt') // { type: MimeType, content: ReadCloser }
// var length	      = fs.WriteFile("/root/path/name.file", "Hello")
// var length	      = fs.WriteFile("/root/path/name.file", "Hello", 0644 )
// var length	      = fs.WriteFileBuffer("/root/path/name.file", dataUnit8Array)
// var length	      = fs.WriteFileBase64("/root/path/name.file", dataUnit8Array, 0644 )
// var length	      = fs.AppendFile("/root/path/name.file", "Hello")
// var length	      = fs.AppendFileBuffer("/root/path/name.file", dataUnit8Array)
// var length	      = fs.AppendFileBase64("/root/path/name.file", dataBase64)
// var length	      = fs.InsertFile("/root/path/name.file", 20, "Hello", 0644)
// var length	      = fs.InsertFileBuffer("/root/path/name.file", 20, dataUnit8Array, 0644)
// var length	      = fs.InsertFileBase64("/root/path/name.file", 20, dataBase64, 0644)
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
// var file 		  = fs.Abs('/data/path')
// var files 		  = fs.Glob('/data/path/*.txt')

// Object Javascript API
type Object struct{}

// New create a new FS Object
func New() *Object {
	return &Object{}
}

// ExportObject Export as a FS Object
func (obj *Object) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)

	// Check file
	tmpl.Set("Exists", obj.exists(iso))
	tmpl.Set("IsDir", obj.isdir(iso))
	tmpl.Set("IsFile", obj.isfile(iso))
	tmpl.Set("IsLink", obj.islink(iso))

	// Basic file operation
	tmpl.Set("ReadFile", obj.readFile(iso))
	tmpl.Set("ReadFileBuffer", obj.readFileBuffer(iso))
	tmpl.Set("ReadFileBase64", obj.readFileBase64(iso))
	tmpl.Set("ReadCloser", obj.readCloser(iso))
	tmpl.Set("WriteFile", obj.writeFile(iso))
	tmpl.Set("WriteFileBuffer", obj.writeFileBuffer(iso))
	tmpl.Set("WriteFileBase64", obj.writeFileBase64(iso))
	tmpl.Set("AppendFile", obj.appendFile(iso))
	tmpl.Set("AppendBuffer", obj.appendFileBuffer(iso))
	tmpl.Set("AppendBase64", obj.appendFileBase64(iso))
	tmpl.Set("InsertFile", obj.insertFile(iso))
	tmpl.Set("InsertBuffer", obj.insertFileBuffer(iso))
	tmpl.Set("InsertBase64", obj.insertFileBase64(iso))
	tmpl.Set("Remove", obj.remove(iso))
	tmpl.Set("RemoveAll", obj.removeAll(iso))

	// Download
	tmpl.Set("Download", obj.download(iso))

	// Directory
	tmpl.Set("ReadDir", obj.readdir(iso))
	tmpl.Set("Mkdir", obj.mkdir(iso))
	tmpl.Set("MkdirAll", obj.mkdirAll(iso))
	tmpl.Set("MkdirTemp", obj.mkdirTemp(iso))

	// File info
	tmpl.Set("Chmod", obj.chmod(iso))
	tmpl.Set("BaseName", obj.baseName(iso))
	tmpl.Set("DirName", obj.dirName(iso))
	tmpl.Set("ExtName", obj.extName(iso))
	tmpl.Set("MimeType", obj.mimeType(iso))
	tmpl.Set("Mode", obj.mode(iso))
	tmpl.Set("Size", obj.size(iso))
	tmpl.Set("ModTime", obj.modTime(iso))

	// File operation
	tmpl.Set("Move", obj.move(iso))
	tmpl.Set("Copy", obj.copy(iso))
	tmpl.Set("MoveAppend", obj.moveAppend(iso))
	tmpl.Set("MoveInsert", obj.moveInsert(iso))

	// Directory operation
	tmpl.Set("Abs", obj.abs(iso))

	// Compression
	tmpl.Set("Zip", obj.zip(iso))
	tmpl.Set("Unzip", obj.unzip(iso))

	// Glob
	tmpl.Set("Glob", obj.glob(iso))
	return tmpl
}

// ExportFunction Export as a javascript FS function
// var fs = new FS("mongo");
// var fs = new FS();  // same with new FS("system");
func (obj *Object) ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	object := obj.ExportObject(iso)
	tmpl := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		name := "system"
		args := info.Args()
		if len(args) > 0 {
			name = args[0].String()
		}

		share, err := bridge.ShareData(info.Context())
		if err != nil {
			return obj.errorString(info, fmt.Sprintf("%s", err.Error()))
		}

		if share.Root {
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

func (obj *Object) move(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) moveAppend(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		err = fs.MoveAppend(stor, args[0].String(), args[1].String())
		if err != nil {
			return obj.error(info, err)
		}

		return v8go.Null(iso)
	})
}

func (obj *Object) moveInsert(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 3 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		err = fs.MoveInsert(stor, args[0].String(), args[1].String(), args[2].Integer())
		if err != nil {
			return obj.error(info, err)
		}

		return v8go.Null(iso)
	})
}

func (obj *Object) zip(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		err = fs.Zip(stor, args[0].String(), args[1].String())
		if err != nil {
			return obj.error(info, err)
		}

		return v8go.Null(iso)
	})
}

func (obj *Object) unzip(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		files, err := fs.Unzip(stor, args[0].String(), args[1].String())
		if err != nil {
			return obj.error(info, err)
		}

		return obj.stringArrayValue(info, files)
	})
}

func (obj *Object) copy(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) baseName(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		res := fs.BaseName(args[0].String())
		return obj.stringValue(info, res)
	})
}

func (obj *Object) dirName(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		res := fs.DirName(args[0].String())
		return obj.stringValue(info, res)
	})
}

func (obj *Object) extName(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		res := fs.ExtName(args[0].String())
		return obj.stringValue(info, res)
	})
}

func (obj *Object) mimeType(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) size(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) mode(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) modTime(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) chmod(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) remove(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) removeAll(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) exists(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) abs(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		file := filepath.Join(stor.Root(), args[0].String())
		return obj.stringValue(info, file)
	})
}

func (obj *Object) isdir(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) isfile(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) islink(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) glob(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		files, err := fs.Glob(stor, args[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return obj.stringArrayValue(info, files)
	})
}

func (obj *Object) readdir(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) mkdir(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) mkdirAll(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) mkdirTemp(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) readFile(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) readFileBuffer(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) readFileBase64(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

		return obj.stringValue(info, base64.StdEncoding.EncodeToString(data))
	})
}

func (obj *Object) readCloser(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		readCloser, err := fs.ReadCloser(stor, args[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		return obj.readCloserValue(info, readCloser)
	})
}

func (obj *Object) download(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return obj.errorString(info, "Missing parameters")
		}

		stor, err := obj.getFS(info)
		if err != nil {
			return obj.error(info, err)
		}

		var file = args[0].String()

		mimetype, err := fs.MimeType(stor, file)
		if err != nil {
			return obj.error(info, err)
		}

		readCloser, err := fs.ReadCloser(stor, args[0].String())
		if err != nil {
			return obj.error(info, err)
		}

		var res = map[string]interface{}{"type": mimetype, "content": readCloser}
		return obj.downloadValue(info, res)
	})
}

func (obj *Object) writeFile(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

func (obj *Object) writeFileBuffer(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

// writeFileBase64 writes a base64 encoded string to a file
func (obj *Object) writeFileBase64(iso *v8go.Isolate) *v8go.FunctionTemplate {
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
		data := args[1].String()

		// Decode base64
		dataDecode, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return obj.error(info, err)
		}

		perm := uint32(os.ModePerm)
		if len(args) > 2 {
			perm = args[2].Uint32()
		}

		length, err := fs.WriteFile(stor, name, dataDecode, perm)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(length))
	})
}

func (obj *Object) appendFile(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

		length, err := fs.AppendFile(stor, name, data, perm)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(length))
	})
}

func (obj *Object) appendFileBuffer(iso *v8go.Isolate) *v8go.FunctionTemplate {
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

		length, err := fs.AppendFile(stor, name, data, perm)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(length))
	})
}

// appendFileBase64 appends a base64 encoded string to a file
func (obj *Object) appendFileBase64(iso *v8go.Isolate) *v8go.FunctionTemplate {
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
		data := args[1].String()

		// Decode base64
		dataDecode, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return obj.error(info, err)
		}

		perm := uint32(os.ModePerm)
		if len(args) > 2 {
			perm = args[2].Uint32()
		}

		length, err := fs.AppendFile(stor, name, dataDecode, perm)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(length))
	})
}

func (obj *Object) insertFile(iso *v8go.Isolate) *v8go.FunctionTemplate {
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
		offset := args[1].Integer()
		data := []byte(args[2].String())
		perm := uint32(os.ModePerm)
		if len(args) > 3 {
			perm = args[3].Uint32()
		}

		length, err := fs.InsertFile(stor, name, offset, data, perm)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(length))
	})
}

func (obj *Object) insertFileBuffer(iso *v8go.Isolate) *v8go.FunctionTemplate {
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
		offset := args[1].Integer()
		data := []byte{}
		if info.Args()[2].IsUint32Array() || info.Args()[2].IsArrayBufferView() {
			codes := strings.Split(info.Args()[2].String(), ",")
			for _, code := range codes {
				c, err := strconv.Atoi(code)
				if err != nil {
					return obj.error(info, err)
				}
				data = append(data, byte(c))
			}
		} else {
			data = []byte(info.Args()[2].String())
		}

		perm := uint32(os.ModePerm)
		if len(args) > 3 {
			perm = args[3].Uint32()
		}

		length, err := fs.InsertFile(stor, name, offset, data, perm)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(length))
	})
}

func (obj *Object) insertFileBase64(iso *v8go.Isolate) *v8go.FunctionTemplate {
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
		offset := args[1].Integer()
		data := args[2].String()

		// Decode base64
		dataDecode, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return obj.error(info, err)
		}

		perm := uint32(os.ModePerm)
		if len(args) > 3 {
			perm = args[3].Uint32()
		}

		length, err := fs.InsertFile(stor, name, offset, dataDecode, perm)
		if err != nil {
			return obj.error(info, err)
		}

		return obj.intValue(info, int32(length))
	})
}

func (obj *Object) getFS(info *v8go.FunctionCallbackInfo) (fs.FileSystem, error) {
	name, err := info.This().Get("name")
	if err != nil {
		return nil, err
	}

	share, _ := bridge.ShareData(info.Context())
	if share.Root {
		return fs.RootGet(name.String())
	}

	return fs.Get(name.String())
}

func (obj *Object) stringValue(info *v8go.FunctionCallbackInfo, value string) *v8go.Value {
	res, err := v8go.NewValue(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *Object) stringArrayValue(info *v8go.FunctionCallbackInfo, value []string) *v8go.Value {

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

func (obj *Object) intValue(info *v8go.FunctionCallbackInfo, value int32) *v8go.Value {
	res, err := v8go.NewValue(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *Object) int64Value(info *v8go.FunctionCallbackInfo, value int64) *v8go.Value {
	res, err := v8go.NewValue(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *Object) uint32Value(info *v8go.FunctionCallbackInfo, value uint32) *v8go.Value {
	res, err := v8go.NewValue(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *Object) boolValue(info *v8go.FunctionCallbackInfo, value bool) *v8go.Value {
	res, err := v8go.NewValue(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *Object) readCloserValue(info *v8go.FunctionCallbackInfo, value io.ReadCloser) *v8go.Value {
	res, err := v8go.NewExternal(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *Object) downloadValue(info *v8go.FunctionCallbackInfo, value map[string]interface{}) *v8go.Value {
	res, err := v8go.NewExternal(info.Context().Isolate(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *Object) arrayBufferValue(info *v8go.FunctionCallbackInfo, value []byte) *v8go.Value {
	res, err := bridge.JsValue(info.Context(), value)
	if err != nil {
		return obj.error(info, err)
	}
	return res
}

func (obj *Object) error(info *v8go.FunctionCallbackInfo, err error) *v8go.Value {
	return obj.errorString(info, fmt.Sprintf("File System: %s", err.Error()))
}

func (obj *Object) errorString(info *v8go.FunctionCallbackInfo, err string) *v8go.Value {
	msg := fmt.Sprintf("FS: %s", err)
	log.Error(msg)
	return bridge.JsException(info.Context(), msg)
}
