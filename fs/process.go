package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// FileSystemHandlers the file system handlers
var FileSystemHandlers = map[string]process.Handler{
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
	"zip":             processZip,
	"unzip":           processUnzip,
	"glob":            processGlob,
}

func init() {
	process.RegisterGroup("fs", FileSystemHandlers)
}

func stor(process *process.Process) FileSystem {
	name := strings.ToLower(process.ID)
	return MustGet(name)
}

func processReadFile(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	file := process.ArgsString(0)
	data, err := ReadFile(stor, file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return string(data)
}

func processReadFileBuffer(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	file := process.ArgsString(0)
	data, err := ReadFile(stor, file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return data
}

func processWirteFile(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	file := process.ArgsString(0)
	content := process.ArgsString(1)
	perm := process.ArgsUint32(2, uint32(os.ModePerm))
	length, err := WriteFile(stor, file, []byte(content), perm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return length
}

func processWriteFileBuffer(process *process.Process) interface{} {
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

	default:
		exception.New("file content type error", 400).Throw()
	}

	length, err := WriteFile(stor, file, data, perm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return length
}

func processReadDir(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	recursive := process.ArgsBool(1, false)
	dirs, err := ReadDir(stor, dir, recursive)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return dirs
}

func processGlob(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	pattern := process.ArgsString(0)
	absDirs, err := Glob(stor, pattern)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	dirs := []string{}
	for _, dir := range absDirs {
		dirs = append(dirs, strings.Replace(dir, stor.Root(), "", 1))
	}

	return dirs
}

func processMkdir(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	perm := process.ArgsUint32(1, uint32(os.ModePerm))

	err := Mkdir(stor, dir, perm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processMkdirAll(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	perm := process.ArgsUint32(1, uint32(os.ModePerm))

	err := MkdirAll(stor, dir, perm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processMkdirTemp(process *process.Process) interface{} {
	stor := stor(process)
	dir := process.ArgsString(0, "")
	pattern := process.ArgsString(1, "")
	path, err := MkdirTemp(stor, dir, pattern)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return path
}

func processRemove(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	err := Remove(stor, dir)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processRemoveAll(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	err := RemoveAll(stor, dir)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processExists(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	dir := process.ArgsString(0)
	has, err := Exists(stor, dir)
	if err != nil {
		log.Error("[%s] %s", process.ID, err.Error())
		return false
	}
	return has
}

func processIsDir(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	return IsDir(stor, name)
}

func processIsFile(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	return IsFile(stor, name)
}

func processIsLink(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	return IsLink(stor, name)
}

func processChmod(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	name := process.ArgsString(0)
	perm := process.ArgsUint32(1)
	err := Chmod(stor, name, perm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processSize(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	size, err := Size(stor, name)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return size
}

func processMode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	mode, err := Mode(stor, name)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return uint32(mode)
}

func processModTime(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	t, err := ModTime(stor, name)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return int(t.Unix())
}

func processBaseName(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	return BaseName(name)
}

func processDirName(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	return DirName(name)
}

func processExtName(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	return ExtName(name)
}

func processMimeType(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	name := process.ArgsString(0)
	mimetype, err := MimeType(stor, name)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return mimetype
}

func processMove(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	src := process.ArgsString(0)
	dst := process.ArgsString(1)
	err := Move(stor, src, dst)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processZip(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	src := process.ArgsString(0)
	dst := process.ArgsString(1)
	err := Zip(stor, src, dst)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processUnzip(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	src := process.ArgsString(0)
	dst := process.ArgsString(1)
	files, err := Unzip(stor, src, dst)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return files
}

func processCopy(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	src := process.ArgsString(0)
	dst := process.ArgsString(1)
	err := Copy(stor, src, dst)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processUpload(process *process.Process) interface{} {

	process.ValidateArgNums(1)
	tmpfile, ok := process.Args[0].(types.UploadFile)
	if !ok {
		exception.New("parameters error: %v", 400, process.Args[0]).Throw()
	}

	fingerprint := strings.ToUpper(uuid.NewString())
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

func processDownload(process *process.Process) interface{} {

	process.ValidateArgNums(1)
	stor := stor(process)
	file := process.ArgsString(0)
	data, err := ReadFile(stor, file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	mimeType, err := MimeType(stor, file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return map[string]interface{}{
		"content": data,
		"type":    mimeType,
	}
}
