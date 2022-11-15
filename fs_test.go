package gou

import (
	"fmt"
	iofs "io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/runtime/bridge"
)

func TestProcessFsReadFile(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(fs.FileSystems["system"], t)
	data := testFsMakeF1(t)
	process := "fs.system.ReadFile"
	args := []interface{}{f["F1"]}
	res, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	content, ok := res.(string)
	assert.True(t, ok)
	assert.Equal(t, string(data), content)
}

func TestProcessFsReadFileBuffer(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(fs.FileSystems["system"], t)
	data := testFsMakeF1(t)
	process := "fs.system.ReadFileBuffer"
	args := []interface{}{f["F1"]}
	res, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	content, ok := res.(bridge.Uint8Array)
	assert.True(t, ok)
	assert.Equal(t, bridge.Uint8Array(data), content)
}

func TestProcessFsWriteFile(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(fs.FileSystems["system"], t)
	data := testFsData(t)

	process := "fs.system.WriteFile"
	args := []interface{}{f["F1"], string(data), 0644}
	res, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, len(data), res.(int))
}

func TestProcessFsWriteBuffer(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(fs.FileSystems["system"], t)
	data := testFsData(t)

	process := "fs.system.WriteFileBuffer"
	args := []interface{}{f["F1"], data, 0644}
	res, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	testFsClear(fs.FileSystems["system"], t)
	args = []interface{}{f["F1"], bridge.Uint8Array(data), 0644}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res.(int))

	testFsClear(fs.FileSystems["system"], t)
	args = []interface{}{f["F1"], string(data), 0644}
	res, err = NewProcess(process, args...).Exec()
	assert.NotNil(t, err)
}

func TestProcessFsDir(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(fs.FileSystems["system"], t)
	process := "fs.system.Mkdir"
	args := []interface{}{f["D2"]}
	_, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	process = "fs.system.MkdirAll"
	args = []interface{}{f["D1_D2"]}
	_, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	process = "fs.system.MkdirTemp"
	args = []interface{}{}
	_, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	process = "fs.system.MkdirTemp"
	args = []interface{}{f["D1"]}
	_, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	process = "fs.system.MkdirTemp"
	args = []interface{}{f["D1"], "*-logs"}
	_, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	process = "fs.system.ReadDir"
	args = []interface{}{f["root"]}
	res, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(res.([]string)))

	process = "fs.system.ReadDir"
	args = []interface{}{f["root"], true}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 5, len(res.([]string)))
}

func TestProcessFsExistRemove(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(fs.FileSystems["system"], t)
	testFsMakeF1(t)
	testFsMakeD1D2F1(t)

	// Exists
	process := "fs.system.Exists"
	args := []interface{}{f["F1"]}
	ok, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ok.(bool))

	process = "fs.system.Exists"
	args = []interface{}{f["F2"]}
	ok, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, ok.(bool))

	// IsDir
	process = "fs.system.IsDir"
	args = []interface{}{f["D1"]}
	ok, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ok.(bool))

	process = "fs.system.IsDir"
	args = []interface{}{f["F1"]}
	ok, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, ok.(bool))

	// IsFile
	process = "fs.system.IsFile"
	args = []interface{}{f["F1"]}
	ok, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ok.(bool))

	process = "fs.system.IsFile"
	args = []interface{}{f["D1"]}
	ok, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, ok.(bool))

	// Remove
	process = "fs.system.Remove"
	args = []interface{}{f["F1"]}
	res, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	process = "fs.system.Remove"
	args = []interface{}{f["F2"]}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	process = "fs.system.Remove"
	args = []interface{}{f["D1"]}
	res, err = NewProcess(process, args...).Exec()
	assert.NotNil(t, err)

	// RemoveAll
	process = "fs.system.RemoveAll"
	args = []interface{}{f["D1"]}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	process = "fs.system.RemoveAll"
	args = []interface{}{f["D1_D2"]}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

}

func TestProcessFsFileInfo(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(fs.FileSystems["system"], t)
	data := testFsMakeF1(t)

	process := "fs.system.BaseName"
	args := []interface{}{f["F1"]}
	res, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "f1.file", res)

	process = "fs.system.DirName"
	args = []interface{}{f["F1"]}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, f["root"], res)

	process = "fs.system.ExtName"
	args = []interface{}{f["F1"]}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "file", res)

	process = "fs.system.MimeType"
	args = []interface{}{f["F1"]}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "text/plain; charset=utf-8", res)

	process = "fs.system.Size"
	args = []interface{}{f["F1"]}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res)

	process = "fs.system.ModTime"
	args = []interface{}{f["F1"]}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, int(time.Now().Unix()) >= res.(int))

	process = "fs.system.Mode"
	args = []interface{}{f["F1"]}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, iofs.FileMode(0644), iofs.FileMode(res.(uint32)))

	process = "fs.system.Chmod"
	args = []interface{}{f["F1"], 0755}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	process = "fs.system.Mode"
	args = []interface{}{f["F1"]}
	res, err = NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, iofs.FileMode(0755), iofs.FileMode(res.(uint32)))
}

func TestProcessFsMove(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(fs.FileSystems["system"], t)
	testFsMakeData(t)

	process := "fs.system.Move"
	args := []interface{}{f["D1_D2"], f["D2"]}
	res, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	stor := fs.FileSystems["system"]
	dirs, err := fs.ReadDir(stor, f["D2"], true)
	assert.Nil(t, err)
	assert.Nil(t, res)
	assert.Equal(t, 2, len(dirs))
	checkFileNotExists(t, f["D1_D2"])
	checkFileExists(t, f["D2_F1"])
	checkFileExists(t, f["D2_F2"])
}

func TestProcessFsCopy(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(fs.FileSystems["system"], t)
	testFsMakeData(t)

	process := "fs.system.Copy"
	args := []interface{}{f["D1_D2"], f["D2"]}
	res, err := NewProcess(process, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	stor := fs.FileSystems["system"]
	dirs, err := fs.ReadDir(stor, f["D2"], true)
	assert.Nil(t, err)
	assert.Nil(t, res)
	assert.Equal(t, 2, len(dirs))
	checkFileExists(t, f["D1_D2"])
	checkFileExists(t, f["D2_F1"])
	checkFileExists(t, f["D2_F2"])
}

func testFsMakeF1(t *testing.T) []byte {
	data := testFsData(t)
	f := testFsFiles(t)

	// Write
	_, err := fs.WriteFile(fs.FileSystems["system"], f["F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func testFsMakeD1D2F1(t *testing.T) []byte {
	data := testFsData(t)
	f := testFsFiles(t)

	// Write
	_, err := fs.WriteFile(fs.FileSystems["system"], f["D1_D2_F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func checkFileExists(t assert.TestingT, path string) {
	stor := fs.FileSystems["system"]
	exist, _ := fs.Exists(stor, path)
	assert.True(t, exist)
}

func checkFileNotExists(t assert.TestingT, path string) {
	stor := fs.FileSystems["system"]
	exist, _ := fs.Exists(stor, path)
	assert.False(t, exist)
}
func testFsMakeData(t *testing.T) {

	stor := fs.FileSystems["system"]
	f := testFsFiles(t)
	data := testFsData(t)

	err := fs.MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}

	// Write
	_, err = fs.WriteFile(stor, f["D1_F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.WriteFile(stor, f["D1_F2"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.WriteFile(stor, f["D1_D2_F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.WriteFile(stor, f["D1_D2_F2"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func testFsData(t *testing.T) []byte {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(10) + 1
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return []byte(fmt.Sprintf("HELLO WORLD %s", string(b)))
}

func testFsFiles(t *testing.T) map[string]string {

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

func testFsClear(stor fs.FileSystem, t *testing.T) {

	root := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data")
	err := os.RemoveAll(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	err = fs.MkdirAll(stor, root, uint32(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}
}
