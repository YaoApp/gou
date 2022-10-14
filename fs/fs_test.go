package fs

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs/system"
)

func TestMustGet(t *testing.T) {
	testStores(t)
	system := MustGet("system")
	assert.NotNil(t, system)

	rel := MustGet("system-relpath")
	assert.NotNil(t, rel)

	assert.Panics(t, func() { MustGet("not-found") })
	assert.Panics(t, func() { MustGet("system-root") })
}

func TestMustRootGet(t *testing.T) {
	testStores(t)
	system := MustRootGet("system")
	assert.NotNil(t, system)

	rel := MustRootGet("system-relpath")
	assert.NotNil(t, rel)

	root := MustRootGet("system-root")
	assert.NotNil(t, root)

	assert.Panics(t, func() { MustRootGet("not-found") })
}

func TestMkdir(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		err := Mkdir(stor, f["D1_D2"], int(os.ModePerm))
		assert.NotNil(t, err, name)

		err = Mkdir(stor, f["D1"], int(os.ModePerm))
		assert.Nil(t, err, name)
	}
}

func TestMkdirAll(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		err := MkdirAll(stor, f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)

		err = Mkdir(stor, f["D1"], int(os.ModePerm))
		assert.NotNil(t, err, name)
	}
}

func TestMkdirTemp(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)
		tempPath, err := MkdirTemp(stor, "", "")
		assert.Nil(t, err, name)
		checkFileExists(stor, t, tempPath, name)

		tempPath, err = MkdirTemp(stor, "", "*-logs")
		assert.Nil(t, err, name)
		assert.True(t, strings.HasSuffix(tempPath, "-logs"))
		checkFileExists(stor, t, tempPath, name)

		tempPath, err = MkdirTemp(stor, "", "logs-")
		assert.Nil(t, err, name)
		assert.True(t, strings.HasPrefix(filepath.Base(tempPath), "logs-"))
		checkFileExists(stor, t, tempPath, name)

		tempPath, err = MkdirTemp(stor, f["D1_D2"], "")
		assert.Nil(t, err, name)
		assert.True(t, strings.HasPrefix(tempPath, f["D1_D2"]))
		checkFileExists(stor, t, tempPath, name)

		tempPath, err = MkdirTemp(stor, f["D1_D2"], "*-logs")
		assert.Nil(t, err, name)
		assert.True(t, strings.HasPrefix(tempPath, f["D1_D2"]))
		assert.True(t, strings.HasSuffix(tempPath, "-logs"))
		checkFileExists(stor, t, tempPath, name)

		tempPath, err = MkdirTemp(stor, f["D1_D2"], "logs-")
		assert.Nil(t, err, name)
		assert.True(t, strings.HasPrefix(tempPath, f["D1_D2"]))
		assert.True(t, strings.HasPrefix(filepath.Base(tempPath), "logs-"))
		checkFileExists(stor, t, tempPath, name)

		_, err = WriteFile(stor, f["F1"], data, 0644)
		assert.Nil(t, err, name)
		tempPath, err = MkdirTemp(stor, f["F1"], "logs-")
		assert.NotNil(t, err, name)
	}
}

func TestReadDir(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Mkdir
		err := MkdirAll(stor, f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = WriteFile(stor, f["D1_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F1"], name)

		_, err = WriteFile(stor, f["D1_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F2"], name)

		_, err = WriteFile(stor, f["D1_D2_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F1"], name)

		_, err = WriteFile(stor, f["D1_D2_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F2"], name)

		dirs, err := ReadDir(stor, f["D1"], false)
		assert.Nil(t, err, name)
		assert.Equal(t, 3, len(dirs), name)

		dirs, err = ReadDir(stor, f["D1"], true)
		assert.Equal(t, 5, len(dirs), name)
	}
}

func TestWriteFile(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Write
		length, err := WriteFile(stor, f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)
		checkFileSize(stor, t, f["F1"], length, name)
		checkFileMode(stor, t, f["F1"], 0644, name)

		// Overwrite
		data = testData(t)
		l21, err := WriteFile(stor, f["F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F2"], name)
		checkFileSize(stor, t, f["F2"], l21, name)
		checkFileMode(stor, t, f["F2"], 0644, name)

		data = testData(t)
		l22, err := WriteFile(stor, f["F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F2"], name)
		checkFileSize(stor, t, f["F2"], l22, name)
		checkFileMode(stor, t, f["F2"], 0644, name)

		// permission denied
		err = Chmod(stor, f["F2"], 0400)
		assert.Nil(t, err, name)
		l31, err := WriteFile(stor, f["F2"], data, 0644)
		assert.NotNil(t, err, name)
		assert.Equal(t, l31, 0)

		err = MkdirAll(stor, f["D1"], int(os.ModePerm))
		assert.Nil(t, err, name)
		err = Chmod(stor, f["D1"], 0400)
		assert.Nil(t, err, name)

		l32, err := WriteFile(stor, f["D1_D2_F1"], data, 0644)
		assert.NotNil(t, err, name)
		assert.Equal(t, l32, 0)
	}
}

func TestReadFile(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Write
		length, err := WriteFile(stor, f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)
		checkFileSize(stor, t, f["F1"], length, name)
		checkFileMode(stor, t, f["F1"], 0644, name)

		// Read
		content, err := ReadFile(stor, f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, data, content, name)

		// file does not exist
		content, err = ReadFile(stor, f["F2"])
		assert.NotNil(t, err, name)

		// file is a directory
		err = MkdirAll(stor, f["D1"], int(os.ModePerm))
		assert.Nil(t, err, name)
		content, err = ReadFile(stor, f["D1"])
		assert.NotNil(t, err, name)
	}
}

func TestRemove(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Mkdir
		err := MkdirAll(stor, f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = WriteFile(stor, f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)

		// Remove
		err = Remove(stor, f["F1"])
		assert.Nil(t, err, name)
		checkFileNotExists(stor, t, f["F1"], name)

		// Remove Dir not empty
		err = Remove(stor, f["D1"])
		assert.NotNil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)

		// Remove Dir
		err = Remove(stor, f["D1_D2"])
		assert.Nil(t, err, name)
		checkFileNotExists(stor, t, f["D1_D2"], name)

		// Remove not exist
		err = Remove(stor, f["F2"])
		assert.Nil(t, err, name)
	}
}

func TestRemoveAll(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Mkdir
		err := MkdirAll(stor, f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = WriteFile(stor, f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)

		// Remove
		err = RemoveAll(stor, f["F1"])
		assert.Nil(t, err, name)
		checkFileNotExists(stor, t, f["F1"], name)

		// Remove Dir not empty
		err = RemoveAll(stor, f["D1"])
		assert.Nil(t, err, name)
		checkFileNotExists(stor, t, f["D1"], name)
		checkFileNotExists(stor, t, f["D1_D2"], name)

		// Remove not exist
		err = RemoveAll(stor, f["F2"])
		assert.Nil(t, err, name)
	}
}

func TestFileInfo(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Mkdir
		err := MkdirAll(stor, f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = WriteFile(stor, f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)

		_, err = WriteFile(stor, f["D1_D2_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F1"], name)

		// IsDir IsFile
		assert.True(t, IsDir(stor, f["D1_D2"]), name)
		assert.False(t, IsDir(stor, f["F1"]), name)
		assert.True(t, IsFile(stor, f["F1"]), name)
		assert.False(t, IsFile(stor, f["D1_D2"]), name)

		// Mode
		mode, err := Mode(stor, f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, 0644, mode)

		// ModTime Time
		modTime, err := ModTime(stor, f["F1"])
		assert.Nil(t, err, name)
		assert.Less(t, time.Now().UnixMicro()-modTime.UnixMicro(), int64(10000))
	}
}

func TestChmod(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Mkdir
		err := MkdirAll(stor, f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = WriteFile(stor, f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)

		// Chmod file
		mode, err := Mode(stor, f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, 0644, mode)

		err = Chmod(stor, f["F1"], 0755)
		assert.Nil(t, err, name)

		mode, err = Mode(stor, f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, 0755, mode)

		// Chmod Dir
		mode, err = Mode(stor, f["D1_D2"])
		assert.Nil(t, err, name)

		err = Chmod(stor, f["D1"], 0755)
		assert.Nil(t, err, name)

		mode, err = Mode(stor, f["D1_D2"])
		assert.Nil(t, err, name)
		assert.Equal(t, 0755, mode)
	}
}

func TestMove(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Mkdir
		err := MkdirAll(stor, f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = WriteFile(stor, f["D1_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F1"], name)

		_, err = WriteFile(stor, f["D1_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F2"], name)

		_, err = WriteFile(stor, f["D1_D2_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F1"], name)

		_, err = WriteFile(stor, f["D1_D2_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F2"], name)

		err = Move(stor, f["D1_D2"], f["D2"])
		assert.Nil(t, err, name)
		checkFileNotExists(stor, t, f["D1_D2"], name)

		dirs, err := ReadDir(stor, f["D2"], true)
		assert.Nil(t, err, name)
		assert.Equal(t, 2, len(dirs))
		checkFileExists(stor, t, f["D2_F1"], name)
		checkFileExists(stor, t, f["D2_F2"], name)

	}
}

func TestCopy(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Mkdir
		err := MkdirAll(stor, f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = WriteFile(stor, f["D1_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F1"], name)

		_, err = WriteFile(stor, f["D1_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F2"], name)

		_, err = WriteFile(stor, f["D1_D2_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F1"], name)

		_, err = WriteFile(stor, f["D1_D2_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F2"], name)

		err = Copy(stor, f["D1_D2"], f["D2"])
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2"], name)

		dirs, err := ReadDir(stor, f["D2"], true)
		assert.Nil(t, err, name)
		assert.Equal(t, 2, len(dirs))
		checkFileExists(stor, t, f["D2_F1"], name)
		checkFileExists(stor, t, f["D2_F2"], name)

	}
}

func TestMimeType(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)

		_, err := WriteFile(stor, f["F1"], []byte(`<html></html>`), 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)

		_, err = WriteFile(stor, f["F3"], []byte(`HELLO WORLD`), 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F3"], name)

		mime, err := MimeType(stor, f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, "text/html; charset=utf-8", mime)

		mime, err = MimeType(stor, f["F2"])
		assert.NotNil(t, err, name)

		mime, err = MimeType(stor, f["F3"])
		assert.Nil(t, err, name)
		assert.Equal(t, "text/plain; charset=utf-8", mime)

	}
}

func TestBase(t *testing.T) {
	f := testFiles(t)
	assert.Equal(t, "f1.file", BaseName(f["F1"]))
	assert.Equal(t, "d1", BaseName(f["D1"]))
	assert.Equal(t, "f1.file", BaseName(f["F1"]))
	assert.Equal(t, f["D1"], DirName(f["D1_F1"]))
	assert.Equal(t, f["D1"], DirName(f["D1_D2"]))
	assert.Equal(t, "file", ExtName(f["F1"]))
	assert.Equal(t, "", ExtName(f["D1"]))
}

func testStores(t *testing.T) map[string]FileSystem {
	Register("system", system.New())
	Register("system-relpath", system.New(filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data")))
	RootRegister("system-root", system.New())
	return FileSystems
}

func testData(t *testing.T) []byte {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(10) + 1
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return []byte(fmt.Sprintf("HELLO WORLD %s", string(b)))
}

func testFiles(t *testing.T) map[string]string {

	root := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data")
	return map[string]string{
		"root":     root,
		"F1":       filepath.Join(root, "f1.file"),
		"F2":       filepath.Join(root, "f2.file"),
		"F3":       filepath.Join(root, "f3.js"),
		"D1_F1":    filepath.Join(root, "d1", "f1.file"),
		"D1_F2":    filepath.Join(root, "d1", "f2.file"),
		"D2_F1":    filepath.Join(root, "d2", "f1.file"),
		"D2_F2":    filepath.Join(root, "d2", "f2.file"),
		"D1_D2_F1": filepath.Join(root, "d1", "d2", "f1.file"),
		"D1_D2_F2": filepath.Join(root, "d1", "d2", "f2.file"),
		"D1":       filepath.Join(root, "d1"),
		"D2":       filepath.Join(root, "d2"),
		"D1_D2":    filepath.Join(root, "d1", "d2"),
	}

}

func checkFileExists(stor FileSystem, t assert.TestingT, path string, name string) {
	exist, _ := Exists(stor, path)
	assert.True(t, exist, name)
}

func checkFileNotExists(stor FileSystem, t assert.TestingT, path string, name string) {
	exist, _ := Exists(stor, path)
	assert.False(t, exist, name)
}

func checkFileSize(stor FileSystem, t assert.TestingT, path string, size int, name string) {
	realSize, _ := Size(stor, path)
	assert.Equal(t, size, realSize, name)
}

func checkFileMode(stor FileSystem, t assert.TestingT, path string, mode int, name string) {
	realMode, _ := Mode(stor, path)
	assert.Equal(t, mode, realMode, name)
}

func clear(stor FileSystem, t *testing.T) {

	root := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data")
	err := os.RemoveAll(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	err = MkdirAll(stor, root, int(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}
}
