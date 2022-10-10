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

func TestMkdir(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		err := stor.Mkdir(f["D1_D2"], int(os.ModePerm))
		assert.NotNil(t, err, name)

		err = stor.Mkdir(f["D1"], int(os.ModePerm))
		assert.Nil(t, err, name)
	}
}

func TestMkdirAll(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		err := stor.MkdirAll(f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)

		err = stor.Mkdir(f["D1"], int(os.ModePerm))
		assert.NotNil(t, err, name)
	}
}

func TestMkdirTemp(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)
		tempPath, err := stor.MkdirTemp("", "")
		assert.Nil(t, err, name)
		checkFileExists(stor, t, tempPath, name)

		tempPath, err = stor.MkdirTemp("", "*-logs")
		assert.Nil(t, err, name)
		assert.True(t, strings.HasSuffix(tempPath, "-logs"))
		checkFileExists(stor, t, tempPath, name)

		tempPath, err = stor.MkdirTemp("", "logs-")
		assert.Nil(t, err, name)
		assert.True(t, strings.HasPrefix(filepath.Base(tempPath), "logs-"))
		checkFileExists(stor, t, tempPath, name)

		tempPath, err = stor.MkdirTemp(f["D1_D2"], "")
		assert.Nil(t, err, name)
		assert.True(t, strings.HasPrefix(tempPath, f["D1_D2"]))
		checkFileExists(stor, t, tempPath, name)

		tempPath, err = stor.MkdirTemp(f["D1_D2"], "*-logs")
		assert.Nil(t, err, name)
		assert.True(t, strings.HasPrefix(tempPath, f["D1_D2"]))
		assert.True(t, strings.HasSuffix(tempPath, "-logs"))
		checkFileExists(stor, t, tempPath, name)

		tempPath, err = stor.MkdirTemp(f["D1_D2"], "logs-")
		assert.Nil(t, err, name)
		assert.True(t, strings.HasPrefix(tempPath, f["D1_D2"]))
		assert.True(t, strings.HasPrefix(filepath.Base(tempPath), "logs-"))
		checkFileExists(stor, t, tempPath, name)

		_, err = stor.WriteFile(f["F1"], data, 0644)
		assert.Nil(t, err, name)
		tempPath, err = stor.MkdirTemp(f["F1"], "logs-")
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
		err := stor.MkdirAll(f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = stor.WriteFile(f["D1_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F1"], name)

		_, err = stor.WriteFile(f["D1_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F2"], name)

		_, err = stor.WriteFile(f["D1_D2_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F1"], name)

		_, err = stor.WriteFile(f["D1_D2_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F2"], name)

		dirs, err := stor.ReadDir(f["D1"], false)
		assert.Nil(t, err, name)
		assert.Equal(t, 3, len(dirs))

		dirs, err = stor.ReadDir(f["D1"], true)
		assert.Equal(t, 5, len(dirs))
	}
}

func TestWriteFile(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Write
		length, err := stor.WriteFile(f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)
		checkFileSize(stor, t, f["F1"], length, name)
		checkFileMode(stor, t, f["F1"], 0644, name)

		// Overwrite
		data = testData(t)
		l21, err := stor.WriteFile(f["F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F2"], name)
		checkFileSize(stor, t, f["F2"], l21, name)
		checkFileMode(stor, t, f["F2"], 0644, name)

		data = testData(t)
		l22, err := stor.WriteFile(f["F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F2"], name)
		checkFileSize(stor, t, f["F2"], l22, name)
		checkFileMode(stor, t, f["F2"], 0644, name)
	}
}

func TestReadFile(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Write
		length, err := stor.WriteFile(f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)
		checkFileSize(stor, t, f["F1"], length, name)
		checkFileMode(stor, t, f["F1"], 0644, name)

		// Read
		content, err := stor.ReadFile(f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, data, content, name)

		// file does not exist
		content, err = stor.ReadFile(f["F2"])
		assert.NotNil(t, err, name)

		// file is a directory
		err = stor.MkdirAll(f["D1"], int(os.ModePerm))
		assert.Nil(t, err, name)
		content, err = stor.ReadFile(f["D1"])
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
		err := stor.MkdirAll(f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = stor.WriteFile(f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)

		// Remove
		err = stor.Remove(f["F1"])
		assert.Nil(t, err, name)
		checkFileNotExists(stor, t, f["F1"], name)

		// Remove Dir not empty
		err = stor.Remove(f["D1"])
		assert.NotNil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)

		// Remove Dir
		err = stor.Remove(f["D1_D2"])
		assert.Nil(t, err, name)
		checkFileNotExists(stor, t, f["D1_D2"], name)

		// Remove not exist
		err = stor.Remove(f["F2"])
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
		err := stor.MkdirAll(f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = stor.WriteFile(f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)

		// Remove
		err = stor.RemoveAll(f["F1"])
		assert.Nil(t, err, name)
		checkFileNotExists(stor, t, f["F1"], name)

		// Remove Dir not empty
		err = stor.RemoveAll(f["D1"])
		assert.Nil(t, err, name)
		checkFileNotExists(stor, t, f["D1"], name)
		checkFileNotExists(stor, t, f["D1_D2"], name)

		// Remove not exist
		err = stor.RemoveAll(f["F2"])
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
		err := stor.MkdirAll(f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = stor.WriteFile(f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)

		_, err = stor.WriteFile(f["D1_D2_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F1"], name)

		// IsDir IsFile
		assert.True(t, stor.IsDir(f["D1_D2"]), name)
		assert.False(t, stor.IsDir(f["F1"]), name)
		assert.True(t, stor.IsFile(f["F1"]), name)
		assert.False(t, stor.IsFile(f["D1_D2"]), name)

		// Mode
		mode, err := stor.Mode(f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, 0644, mode)

		// ModTime Time
		modTime, err := stor.ModTime(f["F1"])
		assert.Nil(t, err, name)
		assert.Less(t, time.Now().UnixMicro()-modTime.UnixMicro(), int64(1000))
	}
}

func TestChmod(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Mkdir
		err := stor.MkdirAll(f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = stor.WriteFile(f["F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)

		// Chmod file
		mode, err := stor.Mode(f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, 0644, mode)

		err = stor.Chmod(f["F1"], 0755)
		assert.Nil(t, err, name)

		mode, err = stor.Mode(f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, 0755, mode)

		// Chmod Dir
		mode, err = stor.Mode(f["D1_D2"])
		assert.Nil(t, err, name)

		err = stor.Chmod(f["D1"], 0755)
		assert.Nil(t, err, name)

		mode, err = stor.Mode(f["D1_D2"])
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
		err := stor.MkdirAll(f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = stor.WriteFile(f["D1_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F1"], name)

		_, err = stor.WriteFile(f["D1_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F2"], name)

		_, err = stor.WriteFile(f["D1_D2_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F1"], name)

		_, err = stor.WriteFile(f["D1_D2_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F2"], name)

		err = stor.Move(f["D1_D2"], f["D2"])
		assert.Nil(t, err, name)
		checkFileNotExists(stor, t, f["D1_D2"], name)

		dirs, err := stor.ReadDir(f["D2"], true)
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
		err := stor.MkdirAll(f["D1_D2"], int(os.ModePerm))
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1"], name)
		checkFileExists(stor, t, f["D1_D2"], name)

		// Write
		_, err = stor.WriteFile(f["D1_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F1"], name)

		_, err = stor.WriteFile(f["D1_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_F2"], name)

		_, err = stor.WriteFile(f["D1_D2_F1"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F1"], name)

		_, err = stor.WriteFile(f["D1_D2_F2"], data, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2_F2"], name)

		err = stor.Copy(f["D1_D2"], f["D2"])
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["D1_D2"], name)

		dirs, err := stor.ReadDir(f["D2"], true)
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

		_, err := stor.WriteFile(f["F1"], []byte(`<html></html>`), 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)

		_, err = stor.WriteFile(f["F3"], []byte(`HELLO WORLD`), 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F3"], name)

		mime, err := stor.MimeType(f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, "text/html; charset=utf-8", mime)

		mime, err = stor.MimeType(f["F2"])
		assert.NotNil(t, err, name)

		mime, err = stor.MimeType(f["F3"])
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
	return map[string]FileSystem{
		"system":       system.New(),
		"system-test2": system.New(),
	}
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
	exist, _ := stor.Exists(path)
	assert.True(t, exist, name)
}

func checkFileNotExists(stor FileSystem, t assert.TestingT, path string, name string) {
	exist, _ := stor.Exists(path)
	assert.False(t, exist, name)
}

func checkFileSize(stor FileSystem, t assert.TestingT, path string, size int, name string) {
	realSize, _ := stor.Size(path)
	assert.Equal(t, size, realSize, name)
}

func checkFileMode(stor FileSystem, t assert.TestingT, path string, mode int, name string) {
	realMode, _ := stor.Mode(path)
	assert.Equal(t, mode, realMode, name)
}

func clear(stor FileSystem, t *testing.T) {

	root := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data")
	err := os.RemoveAll(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	err = stor.MkdirAll(root, int(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}
}
