package system

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/yaoapp/kun/log"
	"golang.org/x/image/draw"
)

// File the file
type File struct {
	root      string   // the root path
	allowlist []string // the pattern list https://pkg.go.dev/path/filepath#Match
	denylist  []string // the pattern list https://pkg.go.dev/path/filepath#Match
}

type fileInfoCache struct {
	files      []string
	fileInfos  map[string]os.FileInfo
	lastUpdate time.Time
}

var cache fileInfoCache

// New create a new file struct
func New(root ...string) *File {
	f := &File{allowlist: []string{}, denylist: []string{}}
	if len(root) > 0 {
		f.root = root[0]
	}
	return f
}

// Root get the root path
func (f *File) Root() string {
	return f.root
}

// Allow allow rel path
func (f *File) Allow(patterns ...string) *File {
	if f.root != "" {
		for i := range patterns {
			patterns[i] = filepath.Join(f.root, patterns[i])
		}
	}
	f.allowlist = append(f.allowlist, patterns...)
	return f
}

// Deny deny rel path
func (f *File) Deny(patterns ...string) *File {
	if f.root != "" {
		for i := range patterns {
			patterns[i] = filepath.Join(f.root, patterns[i])
		}
	}
	f.denylist = append(f.denylist, patterns...)
	return f
}

// AllowAbs allow abs path
func (f *File) AllowAbs(patterns ...string) *File {
	f.allowlist = append(f.allowlist, patterns...)
	return f
}

// DenyAbs deny abs path
func (f *File) DenyAbs(patterns ...string) *File {
	f.denylist = append(f.denylist, patterns...)
	return f
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile reads the whole file, it does not treat an EOF from Read as an error to be reported.
func (f *File) ReadFile(file string) ([]byte, error) {
	file, err := f.absPath(file)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(file)
}

// WriteFile writes data to the named file, creating it if necessary.
//
//	If the file does not exist, WriteFile creates it with permissions perm (before umask); otherwise WriteFile truncates it before writing, without changing permissions.
func (f *File) WriteFile(file string, data []byte, perm uint32) (int, error) {
	file, err := f.absPath(file)
	if err != nil {
		return 0, err
	}

	dir := filepath.Dir(file)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return 0, err
	}

	err = os.WriteFile(file, data, fs.FileMode(perm))
	if err != nil {
		return 0, err
	}

	return len(data), err
}

// Write writes data to the named file, creating it if necessary.
func (f *File) Write(file string, reader io.Reader, perm uint32) (int, error) {

	file, err := f.absPath(file)
	if err != nil {
		return 0, err
	}

	dir := filepath.Dir(file)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return 0, err
	}

	outFile, err := os.Create(file)
	if err != nil {
		return 0, err
	}
	defer outFile.Close()

	n, err := io.Copy(outFile, reader)
	if err != nil {
		return 0, err
	}

	err = outFile.Chmod(fs.FileMode(perm))
	if err != nil {
		return 0, err
	}

	return int(n), nil
}

// ReadDir reads the named directory, returning all its directory entries sorted by filename.
// If an error occurs reading the directory, ReadDir returns the entries it was able to read before the error, along with the error.
func (f *File) ReadDir(dir string, recursive bool) ([]string, error) {

	dir, err := f.absPath(dir)
	if err != nil {
		return nil, err
	}

	dirs := []string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		file := filepath.Join(dir, entry.Name())
		dirs = append(dirs, f.relPath(file))
		if recursive && entry.IsDir() {
			subdirs, err := f.ReadDir(f.relPath(file), true)
			if err != nil {
				return nil, err
			}
			dirs = append(dirs, subdirs...)
		}
	}

	return dirs, nil
}

// Glob returns the names of all files matching pattern or nil if there is no matching file.
// The syntax of patterns is the same as in Match. The pattern may describe hierarchical names such as /usr/*/bin/ed (assuming the Separator is '/').
func (f *File) Glob(pattern string) ([]string, error) {
	pattern, err := f.absPath(pattern)
	if err != nil {
		return nil, err
	}
	absDirs, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	dirs := []string{}
	for _, dir := range absDirs {
		dirs = append(dirs, f.relPath(dir))
	}
	return dirs, nil
}

// Mkdir creates a new directory with the specified name and permission bits (before umask).
// If there is an error, it will be of type *PathError.
func (f *File) Mkdir(dir string, perm uint32) error {
	dir, err := f.absPath(dir)
	if err != nil {
		return err
	}
	return os.Mkdir(dir, fs.FileMode(perm))
}

// MkdirAll creates a directory named path, along with any necessary parents, and returns nil, or else returns an error.
// The permission bits perm (before umask) are used for all directories that MkdirAll creates. If path is already a directory, MkdirAll does nothing and returns nil.
func (f *File) MkdirAll(dir string, perm uint32) error {
	dir, err := f.absPath(dir)
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, fs.FileMode(perm))
}

// MkdirTemp creates a new temporary directory in the directory dir and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead. If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory. It is the caller's responsibility to remove the directory when it is no longer needed.
func (f *File) MkdirTemp(dir string, pattern string) (string, error) {

	var err error = nil
	if dir != "" {
		dir, err = f.absPath(dir)
		if err != nil {
			return "", err
		}

		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	path, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", err
	}

	if dir != "" {
		path = f.relPath(path)
	}

	return path, err
}

// Remove removes the named file or (empty) directory. If there is an error, it will be of type *PathError.
func (f *File) Remove(name string) error {
	name, err := f.absPath(name)
	if err != nil {
		return err
	}

	err = os.Remove(name)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		log.Warn("[Remove] %s no such file or directory", name)
	}
	return nil
}

// RemoveAll removes path and any children it contains. It removes everything it can but returns the first error it encounters. If the path does not exist, RemoveAll returns nil (no error). If there is an error, it will be of type *PathError.
func (f *File) RemoveAll(name string) error {
	name, err := f.absPath(name)
	if err != nil {
		return err
	}
	return os.RemoveAll(name)
}

// Exists returns a boolean indicating whether the error is known to report that a file or directory already exists.
// It is satisfied by ErrExist as well as some syscall errors.
func (f *File) Exists(name string) (bool, error) {

	name, err := f.absPath(name)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Size return the length in bytes for regular files; system-dependent for others
func (f *File) Size(name string) (int, error) {

	name, err := f.absPath(name)
	if err != nil {
		return 0, err
	}

	info, err := os.Stat(name)
	if err != nil {
		return 0, err
	}
	return int(info.Size()), nil
}

// Mode return the file mode bits
func (f *File) Mode(name string) (uint32, error) {
	name, err := f.absPath(name)
	if err != nil {
		return 0, err
	}

	info, err := os.Stat(name)
	if err != nil {
		return 0, err
	}
	return uint32(info.Mode().Perm()), nil
}

// Chmod changes the mode of the named file to mode. If the file is a symbolic link, it changes the mode of the link's target. If there is an error, it will be of type *PathError.
// A different subset of the mode bits are used, depending on the operating system.
// On Unix, the mode's permission bits, ModeSetuid, ModeSetgid, and ModeSticky are used.
// On Windows, only the 0200 bit (owner writable) of mode is used; it controls whether the file's read-only attribute is set or cleared. The other bits are currently unused.
// For compatibility with Go 1.12 and earlier, use a non-zero mode. Use mode 0400 for a read-only file and 0600 for a readable+writable file.
// On Plan 9, the mode's permission bits, ModeAppend, ModeExclusive, and ModeTemporary are used.
func (f *File) Chmod(name string, mode uint32) error {
	name, err := f.absPath(name)
	if err != nil {
		return err
	}

	return os.Chmod(name, fs.FileMode(mode))
}

// ModTime return the file modification time
func (f *File) ModTime(name string) (time.Time, error) {
	name, err := f.absPath(name)
	if err != nil {
		return time.Time{}, err
	}

	info, err := os.Stat(name)
	if err != nil {
		return time.Now(), err
	}
	return info.ModTime(), nil
}

// IsDir check the given path is dir
func (f *File) IsDir(name string) bool {
	name, err := f.absPath(name)
	if err != nil {
		log.Warn("[IsDir] %s %s", name, err.Error())
		return false
	}

	info, err := os.Stat(name)
	if err != nil {
		log.Warn("[IsDir] %s %s", name, err.Error())
		return false
	}
	return info.IsDir()
}

// IsFile check the given path is file
func (f *File) IsFile(name string) bool {
	name, err := f.absPath(name)
	if err != nil {
		log.Warn("[IsFile] %s %s", name, err.Error())
		return false
	}

	info, err := os.Stat(name)
	if err != nil {
		log.Warn("[IsFile] %s %s", name, err.Error())
		return false
	}
	return !info.IsDir()
}

// IsLink check the given path is symbolic link
func (f *File) IsLink(name string) bool {
	name, err := f.absPath(name)
	if err != nil {
		log.Warn("[IsLink] %s %s", name, err.Error())
		return false
	}
	info, err := os.Stat(name)
	if err != nil {
		log.Warn("[IsLink] %s %s", name, err.Error())
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// Move move from oldpath to newpath
func (f *File) Move(oldpath string, newpath string) error {

	oldpath, err := f.absPath(oldpath)
	if err != nil {
		return err
	}

	newpath, err = f.absPath(newpath)
	if err != nil {
		return err
	}

	err = os.Rename(oldpath, newpath)
	if err != nil && strings.Contains(err.Error(), "invalid cross-device link") {
		return f.copyRemove(f.relPath(oldpath), f.relPath(newpath))
	}
	return err
}

// Copy copy from src to dst
func (f *File) Copy(src string, dest string) error {

	src, err := f.absPath(src)
	if err != nil {
		return err
	}

	dest, err = f.absPath(dest)
	if err != nil {
		return err
	}

	stat, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Copy Link
	if stat.Mode()&os.ModeSymlink != 0 {
		return f.copyLink(f.relPath(src), f.relPath(dest))
	}

	// Copy File
	if !stat.IsDir() {
		return f.copyFile(f.relPath(src), f.relPath(dest))
	}

	// Copy Dir
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())
		if err := f.Copy(f.relPath(sourcePath), f.relPath(destPath)); err != nil {
			return err
		}

	}
	return nil
}

// MimeType return the MimeType
func (f *File) MimeType(name string) (string, error) {
	name, err := f.absPath(name)
	if err != nil {
		return "", err
	}

	mtype, err := mimetype.DetectFile(name)
	if err != nil {
		return "", err
	}
	return mtype.String(), nil
}

// Walk traverse folders and read file contents
func (f *File) Walk(root string, handler func(root, file string, isdir bool) error, patterns ...string) error {
	rootAbs, err := f.absPath(root)
	if err != nil {
		return err
	}

	return filepath.Walk(rootAbs, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error("[fs.Walk] %s %s", filename, err.Error())
			return err
		}

		isdir := info.IsDir()
		if patterns != nil && !isdir && len(patterns) > 0 && patterns[0] != "-" {
			notmatched := true
			basname := filepath.Base(filename)
			for _, pattern := range patterns {
				if matched, _ := filepath.Match(pattern, basname); matched {
					notmatched = false
					break
				}
			}

			if notmatched {
				return nil
			}
		}

		name := strings.TrimPrefix(filename, rootAbs)
		if name == "" && isdir {
			name = string(os.PathSeparator)
		}

		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "/.") || strings.HasPrefix(name, "\\.") {
			return nil
		}

		if !isdir {
			name = filepath.Join(root, name)
		}

		err = handler(root, name, isdir)
		if filepath.SkipDir == err || filepath.SkipAll == err {
			return err
		}

		if err != nil {
			log.Error("[fs.Walk] %s %s", filename, err.Error())
			return err
		}

		return nil
	})
}

// List list the files
func (f *File) List(path string, types []string, page, pageSize int, filter func(string) bool) ([]string, int, int, error) {

	pathAbs, err := f.absPath(path)
	if err != nil {
		return nil, 0, 0, err
	}

	path = pathAbs
	if !cacheNeedsUpdate(path, filter) {
		// If the cache is still valid, use it directly
		totalCount := len(cache.files)
		totalPages := calculateTotalPages(totalCount, pageSize)
		return f.paginateFiles(cache.files, page, pageSize), totalCount, totalPages, nil
	}

	var matchingFiles []string
	fileInfos := make(map[string]os.FileInfo)

	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (len(types) == 0 || containsFileType(filePath, types)) {
			if filter(filePath) {
				matchingFiles = append(matchingFiles, filePath)
				fileInfos[filePath] = info
			}
		}
		return nil
	})

	if err != nil {
		return nil, 0, 0, err
	}

	// Sort files by modification time
	sort.Slice(matchingFiles, func(i, j int) bool {
		timeI := fileInfos[matchingFiles[i]].ModTime()
		timeJ := fileInfos[matchingFiles[j]].ModTime()
		return timeI.After(timeJ)
	})

	cache.files = matchingFiles
	cache.fileInfos = fileInfos
	cache.lastUpdate = time.Now()

	totalCount := len(matchingFiles)
	totalPages := calculateTotalPages(totalCount, pageSize)

	return f.paginateFiles(matchingFiles, page, pageSize), totalCount, totalPages, nil
}

// Resize resize the image
func (f *File) Resize(inputPath, outputPath string, width, height uint) error {

	inputPath, err := f.absPath(inputPath)
	if err != nil {
		return err
	}

	outputPath, err = f.absPath(outputPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(outputPath)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Open the image file
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	// Calculate the aspect ratio
	aspectRatio := float64(img.Bounds().Dx()) / float64(img.Bounds().Dy())

	// Calculate the new dimensions based on the target width and aspect ratio
	newWidth := int(width)
	newHeight := int(float64(newWidth) / aspectRatio)

	// If the target height is specified, use it instead
	if height != 0 {
		newHeight = int(height)
		newWidth = int(float64(newHeight) * aspectRatio)
	}

	// Create a new image with the new dimensions
	resizedImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.ApproxBiLinear.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Create the output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Save the resized image
	err = f.encodeImage(outFile, outputPath, resizedImg)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) encodeImage(outFile *os.File, outputPath string, img image.Image) error {
	// Determine the image format based on the file extension
	format := f.imageFormatFromExtension(outputPath)

	// Encode and save the image
	switch format {
	case "jpeg":
		return jpeg.Encode(outFile, img, nil)
	case "png":
		return png.Encode(outFile, img)
	default:
		return fmt.Errorf("unsupported image format: %s", format)
	}
}

func (f *File) imageFormatFromExtension(filename string) string {
	switch filepath.Ext(filename) {
	case ".jpeg", ".jpg":
		return "jpeg"
	case ".png":
		return "png"
	default:
		return ""
	}
}

// CleanCache clean the cache
func (f *File) CleanCache() {
	cache = fileInfoCache{}
}

func containsFileType(filePath string, types []string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, t := range types {
		if ext == strings.ToLower(t) {
			return true
		}
	}
	return false
}

func (f *File) paginateFiles(absfiles []string, page, pageSize int) []string {

	files := []string{}
	for _, file := range absfiles {
		files = append(files, f.relPath(file))
	}

	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize
	if endIndex > len(files) {
		endIndex = len(files)
	}
	return files[startIndex:endIndex]
}

func calculateTotalPages(totalCount, pageSize int) int {
	if pageSize == 0 {
		return 0
	}
	return (totalCount + pageSize - 1) / pageSize
}

func cacheNeedsUpdate(path string, filter func(string) bool) bool {
	// Check if the cache is empty or expired
	if len(cache.files) == 0 || time.Since(cache.lastUpdate) > 10*time.Minute {
		return true
	}

	// Check if the filter has changed
	return !filterMatchesCache(path, filter)
}

func filterMatchesCache(path string, filter func(string) bool) bool {
	for _, file := range cache.files {
		if !filter(file) {
			return true
		}
	}
	return false
}

func (f *File) isTemp(path string) bool {
	return strings.HasPrefix(path, os.TempDir())
}

// absPath returns an absolute representation of path
func (f *File) absPath(path string) (string, error) {
	if f.root != "" {
		if !f.isTemp(path) {
			path = filepath.Join(f.root, path)
		}
	}

	if !pathSafe(path) {
		return "", fmt.Errorf("%s is not safe", path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if err := f.validate(absPath); err != nil {
		return "", err
	}

	return absPath, nil
}

// relPath returns an relative representation of path
func (f *File) relPath(path string) string {
	if f.root == "" {
		return path
	}
	return strings.TrimPrefix(path, strings.TrimRight(f.root, string(os.PathSeparator)))
}

// pathSafe returns true if the path is safe
func pathSafe(path string) bool {
	return !strings.Contains(path, "..")
}

func (f *File) validate(absPath string) error {

	// check the allow list
	for _, pattern := range f.allowlist {
		match, err := filepath.Match(pattern, absPath)
		if err != nil {
			return fmt.Errorf("%s checking allowlist error %s (%s)", f.relPath(absPath), err.Error(), pattern)
		}

		if match {
			return nil
		}
	}

	// check the deny list
	for _, pattern := range f.denylist {
		match, err := filepath.Match(pattern, absPath)
		if err != nil {
			return err
		}

		if match {
			return fmt.Errorf("%s is denied (%s)", f.relPath(absPath), pattern)
		}
	}

	return nil
}

func (f *File) copyFile(src string, dest string) error {
	src, err := f.absPath(src)
	if err != nil {
		return err
	}

	dest, err = f.absPath(dest)
	if err != nil {
		return err
	}

	dir := filepath.Dir(dest)
	err = os.MkdirAll(dir, fs.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer out.Close()

	in, err := os.Open(src)
	defer in.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) copyLink(src string, dest string) error {

	src, err := f.absPath(src)
	if err != nil {
		return err
	}

	dest, err = f.absPath(dest)
	if err != nil {
		return err
	}

	link, err := os.Readlink(src)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
}

// copyRemove copy oldpath to newpath then remove oldpath
func (f *File) copyRemove(oldpath string, newpath string) error {
	err := f.Copy(oldpath, newpath)
	if err != nil {
		return err
	}

	return f.RemoveAll(oldpath)
}
