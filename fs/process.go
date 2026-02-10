package fs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

// FileSystemHandlers the file system handlers
var FileSystemHandlers = map[string]process.Handler{
	"readfile":         processReadFile,
	"readfilebuffer":   processReadFileBuffer,
	"writefile":        processWirteFile,
	"writefilebuffer":  processWriteFileBuffer,
	"appendfile":       processAppendFile,
	"appendfilebuffer": processAppendFileBuffer,
	"insertfile":       processInsertFile,
	"insertfilebuffer": processInsertFileBuffer,
	"readdir":          processReadDir,
	"mkdir":            processMkdir,
	"mkdirall":         processMkdirAll,
	"mkdirtemp":        processMkdirTemp,
	"remove":           processRemove,
	"removeall":        processRemoveAll,
	"exists":           processExists,
	"isdir":            processIsDir,
	"isfile":           processIsFile,
	"islink":           processIsLink,
	"chmod":            processChmod,
	"size":             processSize,
	"mode":             processMode,
	"modtime":          processModTime,
	"basename":         processBaseName,
	"dirname":          processDirName,
	"extname":          processExtName,
	"mimetype":         processMimeType,
	"move":             processMove,
	"moveappend":       processMoveAppend,
	"moveinsert":       processMoveInsert,
	"copy":             processCopy,
	"upload":           processUpload,
	"download":         processDownload,
	"zip":              processZip,
	"unzip":            processUnzip,
	"glob":             processGlob,
	"abs":              processAbs,
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

func processAppendFile(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	file := process.ArgsString(0)
	content := process.ArgsString(1)
	perm := process.ArgsUint32(2, uint32(os.ModePerm))
	length, err := AppendFile(stor, file, []byte(content), perm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return length
}

func processAppendFileBuffer(process *process.Process) interface{} {
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

	length, err := AppendFile(stor, file, data, perm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return length
}

func processInsertFile(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	stor := stor(process)
	file := process.ArgsString(0)
	offset := process.ArgsInt(1)
	content := process.ArgsString(2)
	perm := process.ArgsUint32(3, uint32(os.ModePerm))
	length, err := InsertFile(stor, file, int64(offset), []byte(content), perm)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return length
}

func processInsertFileBuffer(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	stor := stor(process)
	file := process.ArgsString(0)
	offset := process.ArgsInt(1)
	content := process.Args[2]
	perm := process.ArgsUint32(3, uint32(os.ModePerm))

	data := []byte{}
	switch v := content.(type) {
	case []byte:
		data = v
		break

	default:
		exception.New("file content type error", 400).Throw()
	}

	length, err := InsertFile(stor, file, int64(offset), data, perm)
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

func processAbs(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	stor := stor(process)
	file := process.ArgsString(0)
	root := stor.Root()
	if root == "" {
		return file
	}
	return filepath.Join(root, file)
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

func processMoveAppend(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	stor := stor(process)
	src := process.ArgsString(0)
	dst := process.ArgsString(1)
	err := MoveAppend(stor, src, dst)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func processMoveInsert(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	stor := stor(process)
	src := process.ArgsString(0)
	dst := process.ArgsString(1)
	offset := process.ArgsInt(2)
	err := MoveInsert(stor, src, dst, int64(offset))
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

	uid := tmpfile.Hash()
	dir := strings.Join([]string{string(os.PathSeparator), time.Now().Format("20060102")}, "")
	ext := filepath.Ext(tmpfile.Name)
	filename := filepath.Join(dir, fmt.Sprintf("%s%s", uid, ext))

	stor := stor(process)
	props := process.ArgsMap(1, maps.Map{}) // Get the props from the process, for validate the file type and size.

	// For chunk upload.
	if tmpfile.IsChunk() {

		// Sync upload, the chunk file will be merged directly.
		if tmpfile.Sync {
			err := MoveAppend(stor, tmpfile.TempFile, filename)
			if err != nil {
				exception.New(err.Error(), 500).Throw()
			}

			// Validate the file size
			if props.Has("maxFilesize") {
				validateFileSize(stor, filename, props.Get("maxFilesize"))
			}

			// Cheek the file is exists
			size, err := stor.Size(filename)
			if err != nil {
				exception.New(err.Error(), 500).Throw()
			}

			total := tmpfile.TotalSize()
			if int64(size) == total {
				if props.Has("accept") {
					validateAcceptType(stor, filename, props.Get("accept"), true)
				}
				return filename
			}

			// Return the file path and the upload progress
			progress := types.UploadProgress{
				Total:     total,
				Uploaded:  int64(size),
				Completed: false,
			}

			return map[string]interface{}{
				"path":     filename,
				"uid":      tmpfile.UID,
				"progress": progress,
			}
		}

		// Async upload, the chunk file will be saved to the temp directory.
		tmpDir := path.Join("upload", "tmp", uid)
		err := stor.MkdirAll(tmpDir, uint32(os.ModePerm))
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		chunkFile := filepath.Join(tmpDir, tmpfile.ChunkFileName())
		err = stor.Move(tmpfile.TempFile, chunkFile)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		// Validate the file size
		if props.Has("maxFilesize") {
			validateFileSize(stor, filename, props.Get("maxFilesize"))
		}

		// Check if all chunks are uploaded.
		progress, err := uploadProgress(stor, tmpDir)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		// Remove the temp directory.
		if progress.Completed {
			defer stor.RemoveAll(tmpDir)

			// Validate the file type
			if props.Has("accept") {
				validateAcceptType(stor, filename, props.Get("accept"), true)
			}

			// Get Files
			files, err := getChunkFiles(stor, tmpDir, true)
			if err != nil {
				exception.New(err.Error(), 500).Throw()
			}

			// Merge the chunk files.
			for _, file := range files {
				err := MoveAppend(stor, file, filename)
				if err != nil {
					exception.New(err.Error(), 500).Throw()
				}
			}
			return filename
		}

		return map[string]interface{}{
			"path":     filename,
			"uid":      tmpfile.UID,
			"progress": progress,
		}
	}

	// Validate the file type
	if props.Has("accept") {
		validateAcceptType(stor, tmpfile.TempFile, props.Get("accept"), true)
	}

	// Validate the file size
	if props.Has("maxFilesize") {
		validateFileSize(stor, tmpfile.TempFile, props.Get("maxFilesize"))
	}

	// For normal upload.
	err := stor.Move(tmpfile.TempFile, filename)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return filename
}

func processDownload(process *process.Process) interface{} {

	process.ValidateArgNums(1)
	stor := stor(process)
	file := process.ArgsString(0)

	// Get the file mime type
	mimeType, err := MimeType(stor, file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	// Get the file reader
	reader, err := ReadCloser(stor, file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return map[string]interface{}{
		"content": reader,
		"type":    mimeType,
	}
}

func validateAcceptType(stor FileSystem, file string, accept interface{}, checkMime bool) {

	// Check the file type
	acceptstr, ok := accept.(string)
	if !ok {
		exception.New("the accept type is invalid", 400).Throw()
	}

	ext := filepath.Ext(file)
	acceptList := strings.Split(acceptstr, ",")
	for _, accept := range acceptList {
		accept = strings.TrimSpace(accept)

		// Check the file extension
		if strings.HasPrefix(accept, ".") {
			if ext == accept {
				return
			}
		}

		// Check the file mime type
		if !checkMime {
			continue
		}

		mime, err := MimeType(stor, file)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		if mime == accept {
			return
		}

		// accept =  image/*, video/*, audio/*, application/* ...
		if strings.HasSuffix(accept, "/*") {
			if strings.HasPrefix(mime, strings.TrimRight(accept, "/*")) {
				return
			}
		}
	}

	// Remove the file
	stor.Remove(file)
	exception.New("File type should be %v", 415, acceptstr).Throw()
}

func validateFileSize(stor FileSystem, file string, hmSize interface{}) {
	// Get the file size
	maxFilesize, err := parseFileSize(hmSize)
	if err != nil {
		defer stor.Remove(file)
		exception.New(err.Error(), 500).Throw()
	}

	// Get the file size
	size, err := stor.Size(file)
	if err != nil {
		defer stor.Remove(file)
		exception.New(err.Error(), 500).Throw()
	}

	if size > int(maxFilesize) {
		defer stor.Remove(file)
		exception.New("File size too large, max size is %v", 413, hmSize).Throw()
	}
}

func parseFileSize(hmSize interface{}) (int64, error) {
	if hmSize == nil {
		return 1024 * 1024, nil // 1MB
	}

	switch v := hmSize.(type) {
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		unit := strings.ToUpper(v[len(v)-1:])
		size, err := strconv.ParseInt(v[:len(v)-1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid chunk size: %v", err)
		}

		switch unit {
		case "M":
			return size * 1024 * 1024, nil // MB
		case "K":
			return size * 1024, nil // KB
		default:
			return size, nil // bytes
		}
	default:
		return 0, fmt.Errorf("invalid type for hmSize")
	}
}

func uploadProgress(stor FileSystem, path string) (types.UploadProgress, error) {
	files, err := getChunkFiles(stor, path, true)
	if err != nil {
		return types.UploadProgress{}, err
	}

	// Check the chunk files
	progress := types.UploadProgress{
		Total:     0,
		Uploaded:  0,
		Completed: false,
	}

	for i, file := range files {
		if i == 0 {
			baseName := strings.ReplaceAll(file, ".chunk", "")
			nameInfo := strings.Split(baseName, "_")
			if len(nameInfo) != 2 {
				return progress, fmt.Errorf("the chunk file %s is invalid", file)
			}

			// Get the chunk file size
			var total int64
			fmt.Sscanf(nameInfo[len(nameInfo)-1], "%d", &total)
			progress.Total = total
		}

		// Get the chunk file size
		size, err := Size(stor, file)
		if err != nil {
			return progress, err
		}

		progress.Uploaded += int64(size)
	}

	// Check the upload is completed or not
	progress.Completed = progress.Uploaded == progress.Total
	return progress, nil
}

func getChunkFiles(stor FileSystem, path string, sortable bool) ([]string, error) {

	// Validate the path is exists
	if _, err := stor.Exists(path); err != nil {
		return nil, fmt.Errorf("the file %s is not exists", path)
	}

	// Validate the path is a directory
	if !stor.IsDir(path) {
		return nil, fmt.Errorf("the file %s is not a directory", path)
	}

	// Get the chunk files in the directory
	files, err := stor.ReadDir(path, false)
	if err != nil {
		return nil, err
	}

	if !sortable {
		return files, nil
	}

	chunkMap := map[string]int64{}
	for _, file := range files {

		// Get the chunk file name
		baseName := strings.ReplaceAll(filepath.Base(file), ".chunk", "")
		nameInfo := strings.Split(baseName, "-")
		if len(nameInfo) != 2 {
			continue
		}

		// Get the chunk file size
		var from int64
		fmt.Sscanf(nameInfo[0], "%d", &from)
		chunkMap[file] = from
	}

	// Sort the chunk files
	sort.Slice(files, func(i, j int) bool {
		return chunkMap[files[i]] < chunkMap[files[j]]
	})

	return files, nil
}
