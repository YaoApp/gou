package gou

import (
	"os"
	"strings"

	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/runtime/bridge"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// FileSystemHandlers the file system handlers
var FileSystemHandlers = map[string]ProcessHandler{
	"readfile":        processReadFile,
	"readfilebuffer":  processReadFileBuffer,
	"writefile":       processWirteFile,
	"writefilebuffer": processWriteFileBuffer,
	"readdir":         processReadDir,
	"mkdir":           processMkdir,
	"mkdirall":        processMkdirAll,
	"mkdirtemp":       processMkdirTemp,
	"remove":          processRemove,
	"removeall":       processRemoveAll,
	"exists":          processExists,
	"isdir":           processIsDir,
	"isfile":          processIsFile,
	"islink":          processIsLink,
	"chmod":           processChmod,
	"size":            processSize,
	"mode":            processMode,
	"modtime":         processModTime,
	"basename":        processBaseName,
	"dirname":         processDirName,
	"extname":         processExtName,
	"mimetype":        processMimeType,
	"move":            processMove,
	"copy":            processCopy,
}

func init() {
	RegisterProcessGroup("fs", FileSystemHandlers)
}

func stor(process *Process) fs.FileSystem {
	name := strings.ToLower(process.Class)
	return fs.MustGet(name)
}

func processReadFile(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	file := process.ArgsString(0)
	data, err := fs.ReadFile(stor, file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return string(data)
}

func processReadFileBuffer(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	file := process.ArgsString(0)
	data, err := fs.ReadFile(stor, file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return bridge.Uint8Array(data)
}

func processWirteFile(process *Process) interface{} {
	process.ValidateArgNums(3)
	stor := stor(process)
	file := process.ArgsString(0)
	content := process.ArgsString(1)
	pterm := process.ArgsInt(2)
	length, err := fs.WriteFile(stor, file, []byte(content), pterm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return length
}

func processWriteFileBuffer(process *Process) interface{} {
	process.ValidateArgNums(3)
	stor := stor(process)
	file := process.ArgsString(0)
	content := process.Args[1]
	pterm := process.ArgsInt(2)
	data := []byte{}
	switch content.(type) {
	case []byte:
		data = content.([]byte)
		break

	case bridge.Uint8Array:
		data = []byte(content.(bridge.Uint8Array))
		break

	default:
		exception.New("file content type error", 400).Throw()
	}

	length, err := fs.WriteFile(stor, file, data, pterm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return length
}

func processReadDir(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	recursive := process.ArgsBool(1, false)
	dirs, err := fs.ReadDir(stor, dir, recursive)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return dirs
}

func processMkdir(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	pterm := process.ArgsInt(1, int(os.ModePerm))

	err := fs.Mkdir(stor, dir, pterm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processMkdirAll(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	pterm := process.ArgsInt(1, int(os.ModePerm))

	err := fs.MkdirAll(stor, dir, pterm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processMkdirTemp(process *Process) interface{} {
	stor := stor(process)
	dir := process.ArgsString(0, "")
	pattern := process.ArgsString(1, "")
	path, err := fs.MkdirTemp(stor, dir, pattern)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return path
}

func processRemove(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	err := fs.Remove(stor, dir)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processRemoveAll(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	err := fs.RemoveAll(stor, dir)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processExists(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	has, err := fs.Exists(stor, dir)
	if err != nil {
		log.Error("[%s] %s", process.Class, err.Error())
		return false
	}
	return has
}

func processIsDir(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	return fs.IsDir(stor, name)
}

func processIsFile(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	return fs.IsFile(stor, name)
}

func processIsLink(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	return fs.IsLink(stor, name)
}

func processChmod(process *Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	name := process.ArgsString(0)
	pterm := process.ArgsInt(1)
	err := fs.Chmod(stor, name, pterm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processSize(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	size, err := fs.Size(stor, name)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return size
}

func processMode(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	mode, err := fs.Mode(stor, name)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return uint32(mode)
}

func processModTime(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	t, err := fs.ModTime(stor, name)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return int(t.Unix())
}

func processBaseName(process *Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	return fs.BaseName(name)
}

func processDirName(process *Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	return fs.DirName(name)
}

func processExtName(process *Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	return fs.ExtName(name)
}

func processMimeType(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	mimetype, err := fs.MimeType(stor, name)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return mimetype
}

func processMove(process *Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	src := process.ArgsString(0)
	dst := process.ArgsString(1)
	err := fs.Move(stor, src, dst)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processCopy(process *Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	src := process.ArgsString(0)
	dst := process.ArgsString(1)
	err := fs.Copy(stor, src, dst)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}
