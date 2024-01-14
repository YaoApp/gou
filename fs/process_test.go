package fs

import (
	"fmt"
	iofs "io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestProcessFsReadFile(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	data := testFsMakeF1(t)
	processName := "fs.system.ReadFile"
	args := []interface{}{f["F1"]}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	content, ok := res.(string)
	assert.True(t, ok)
	assert.Equal(t, string(data), content)
}

func TestProcessFsReadFileBuffer(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	data := testFsMakeF1(t)
	processName := "fs.system.ReadFileBuffer"
	args := []interface{}{f["F1"]}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, data, res)
}

func TestProcessFsWriteFile(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	data := testFsData(t)

	processName := "fs.system.WriteFile"
	args := []interface{}{f["F1"], string(data), 0644}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, len(data), res.(int))
}

func TestProcessFsWriteBuffer(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	data := testFsData(t)

	processName := "fs.system.WriteFileBuffer"
	args := []interface{}{f["F1"], data, 0644}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	testFsClear(FileSystems["system"], t)
	args = []interface{}{f["F1"], data, 0644}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res.(int))

	testFsClear(FileSystems["system"], t)
	args = []interface{}{f["F1"], string(data), 0644}
	res, err = process.New(processName, args...).Exec()
	assert.NotNil(t, err)
}

func TestProcessFsDir(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	processName := "fs.system.Mkdir"
	args := []interface{}{f["D2"]}
	_, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	processName = "fs.system.MkdirAll"
	args = []interface{}{f["D1_D2"]}
	_, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	processName = "fs.system.MkdirTemp"
	args = []interface{}{}
	_, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	processName = "fs.system.MkdirTemp"
	args = []interface{}{f["D1"]}
	_, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	processName = "fs.system.MkdirTemp"
	args = []interface{}{f["D1"], "*-logs"}
	_, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	processName = "fs.system.ReadDir"
	args = []interface{}{f["root"]}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(res.([]string)))

	processName = "fs.system.ReadDir"
	args = []interface{}{f["root"], true}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 5, len(res.([]string)))
}

func TestProcessFsExistRemove(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	testFsMakeF1(t)
	testFsMakeD1D2F1(t)

	// Exists
	processName := "fs.system.Exists"
	args := []interface{}{f["F1"]}
	ok, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ok.(bool))

	processName = "fs.system.Exists"
	args = []interface{}{f["F2"]}
	ok, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, ok.(bool))

	// IsDir
	processName = "fs.system.IsDir"
	args = []interface{}{f["D1"]}
	ok, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ok.(bool))

	processName = "fs.system.IsDir"
	args = []interface{}{f["F1"]}
	ok, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, ok.(bool))

	// IsFile
	processName = "fs.system.IsFile"
	args = []interface{}{f["F1"]}
	ok, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ok.(bool))

	processName = "fs.system.IsFile"
	args = []interface{}{f["D1"]}
	ok, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, ok.(bool))

	// Remove
	processName = "fs.system.Remove"
	args = []interface{}{f["F1"]}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	processName = "fs.system.Remove"
	args = []interface{}{f["F2"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	processName = "fs.system.Remove"
	args = []interface{}{f["D1"]}
	res, err = process.New(processName, args...).Exec()
	assert.NotNil(t, err)

	// RemoveAll
	processName = "fs.system.RemoveAll"
	args = []interface{}{f["D1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	processName = "fs.system.RemoveAll"
	args = []interface{}{f["D1_D2"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

}

func TestProcessFsFileInfo(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	data := testFsMakeF1(t)

	processName := "fs.system.BaseName"
	args := []interface{}{f["F1"]}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "f1.file", res)

	processName = "fs.system.DirName"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, f["root"], res)

	processName = "fs.system.ExtName"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "file", res)

	processName = "fs.system.MimeType"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "text/plain; charset=utf-8", res)

	processName = "fs.system.Size"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res)

	processName = "fs.system.ModTime"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, int(time.Now().Unix()) >= res.(int))

	processName = "fs.system.Mode"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, iofs.FileMode(0644), iofs.FileMode(res.(uint32)))

	processName = "fs.system.Chmod"
	args = []interface{}{f["F1"], 0755}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	processName = "fs.system.Mode"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, iofs.FileMode(0755), iofs.FileMode(res.(uint32)))
}

func TestProcessFsMove(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	testFsMakeData(t)

	processName := "fs.system.Move"
	args := []interface{}{f["D1_D2"], f["D2"]}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	stor := FileSystems["system"]
	dirs, err := ReadDir(stor, f["D2"], true)
	assert.Nil(t, err)
	assert.Nil(t, res)
	assert.Equal(t, 2, len(dirs))
	processCheckFileNotExists(t, f["D1_D2"])
	processCheckFileExists(t, f["D2_F1"])
	processCheckFileExists(t, f["D2_F2"])
}

func TestProcessFsZip(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	testFsMakeData(t)

	processName := "fs.system.Zip"
	zipfile := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data", "test.zip")
	args := []interface{}{f["D1_D2"], zipfile}
	_, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Unzip
	processName = "fs.system.Unzip"
	unzipdir := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data", "test")
	args = []interface{}{zipfile, unzipdir}
	files, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(files.([]string)))

}

func TestProcessFsCopy(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	testFsMakeData(t)

	processName := "fs.system.Copy"
	args := []interface{}{f["D1_D2"], f["D2"]}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	stor := FileSystems["system"]
	dirs, err := ReadDir(stor, f["D2"], true)
	assert.Nil(t, err)
	assert.Nil(t, res)
	assert.Equal(t, 2, len(dirs))
	processCheckFileExists(t, f["D1_D2"])
	processCheckFileExists(t, f["D2_F1"])
	processCheckFileExists(t, f["D2_F2"])
}

func testFsMakeF1(t *testing.T) []byte {
	data := testFsData(t)
	f := testFsFiles(t)

	// Write
	_, err := WriteFile(FileSystems["system"], f["F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func testFsMakeD1D2F1(t *testing.T) []byte {
	data := testFsData(t)
	f := testFsFiles(t)

	// Write
	_, err := WriteFile(FileSystems["system"], f["D1_D2_F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func processCheckFileExists(t assert.TestingT, path string) {
	stor := FileSystems["system"]
	exist, _ := Exists(stor, path)
	assert.True(t, exist)
}

func processCheckFileNotExists(t assert.TestingT, path string) {
	stor := FileSystems["system"]
	exist, _ := Exists(stor, path)
	assert.False(t, exist)
}
func testFsMakeData(t *testing.T) {

	stor := FileSystems["system"]
	f := testFsFiles(t)
	data := testFsData(t)

	err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}

	// Write
	_, err = WriteFile(stor, f["D1_F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = WriteFile(stor, f["D1_F2"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = WriteFile(stor, f["D1_D2_F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = WriteFile(stor, f["D1_D2_F2"], data, 0644)
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

func testFsClear(stor FileSystem, t *testing.T) {

	root := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data")
	err := os.RemoveAll(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	err = MkdirAll(stor, root, uint32(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}
}
