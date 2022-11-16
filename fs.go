package gou

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	"upload":          processUpload,
	"download":        processDownload,
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
	process.ValidateArgNums(2)
	stor := stor(process)
	file := process.ArgsString(0)
	content := process.ArgsString(1)
	perm := process.ArgsUint32(2, uint32(os.ModePerm))
	length, err := fs.WriteFile(stor, file, []byte(content), perm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return length
}

func processWriteFileBuffer(process *Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	file := process.ArgsString(0)
	content := process.Args[1]
	perm := process.ArgsUint32(2, uint32(os.ModePerm))

	data := []byte{}
	switch v := content.(type) {
	case []byte:
		data = v
		break

	case bridge.Uint8Array:
		data = []byte(v)
		break

	default:
		exception.New("file content type error", 400).Throw()
	}

	length, err := fs.WriteFile(stor, file, data, perm)
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
	perm := process.ArgsUint32(1, uint32(os.ModePerm))

	err := fs.Mkdir(stor, dir, perm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processMkdirAll(process *Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	perm := process.ArgsUint32(1, uint32(os.ModePerm))

	err := fs.MkdirAll(stor, dir, perm)
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
	perm := process.ArgsUint32(1)
	err := fs.Chmod(stor, name, perm)
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

func processUpload(process *Process) interface{} {

	process.ValidateArgNums(1)
	tmpfile, ok := process.Args[0].(UploadFile)
	if !ok {
		exception.New("parameters error: %v", 400, process.Args[0]).Throw()
	}

	hash := md5.Sum([]byte(time.Now().Format("20060102-15:04:05")))
	fingerprint := string(hex.EncodeToString(hash[:]))
	fingerprint = strings.ToUpper(fingerprint)

	dir := strings.Join([]string{string(os.PathSeparator), time.Now().Format("20060102")}, "")
	ext := filepath.Ext(tmpfile.Name)
	filename := filepath.Join(dir, fmt.Sprintf("%s%s", fingerprint, ext))

	stor := stor(process)
	content, err := stor.ReadFile(tmpfile.TempFile)
	if err != nil {
		exception.New("unable to read uploaded file %s", 500, err.Error()).Throw()
	}

	_, err = stor.WriteFile(filename, content, uint32(os.ModePerm))
	if err != nil {
		exception.New("failed to save file %s", 500, err.Error()).Throw()
	}

	return filename
}

func processDownload(process *Process) interface{} {

	process.ValidateArgNums(1)
	stor := stor(process)
	file := process.ArgsString(0)
	data, err := fs.ReadFile(stor, file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	mimeType, err := fs.MimeType(stor, file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return map[string]interface{}{
		"content": data,
		"type":    mimeType,
	}
}
