package fs

import (
	"fmt"
	"io"
	iofs "io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs/system"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/types"
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

func TestProcessFsAppendFile(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	data := testFsData(t)

	processName := "fs.system.AppendFile"

	// Append new file
	args := []interface{}{f["F1"], string(data), 0644}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res.(int))

	// Append to existing file
	args = []interface{}{f["F1"], string(data), 0644}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res.(int))

	// Check file content
	processName = "fs.system.ReadFile"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	content, ok := res.(string)
	assert.True(t, ok)
	assert.Equal(t, string(data)+string(data), content)
}

func TestProcessFsAppendBuffer(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	data := testFsData(t)

	processName := "fs.system.AppendFileBuffer"

	// Append new file
	args := []interface{}{f["F1"], data, 0644}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res.(int))

	// Append to existing file
	args = []interface{}{f["F1"], data, 0644}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res.(int))

	// Check file content
	processName = "fs.system.ReadFile"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	content, ok := res.(string)
	assert.True(t, ok)
	assert.Equal(t, string(data)+string(data), content)
}

func TestProcessFsInsertFile(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	data := testFsData(t)

	processName := "fs.system.InsertFile"

	// Insert new file
	args := []interface{}{f["F1"], 0, string(data), 0644}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res.(int))

	// Insert to existing file
	args = []interface{}{f["F1"], 5, string(data), 0644}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res.(int))

	// Check file content
	processName = "fs.system.ReadFile"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	contentDataShouldBe := append(data[:5], append(data, data[5:]...)...)
	content, ok := res.(string)
	assert.True(t, ok)
	assert.Equal(t, string(contentDataShouldBe), content)
}

func TestProcessFsInsertBuffer(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	data := testFsData(t)

	processName := "fs.system.InsertFileBuffer"

	// Insert new file
	args := []interface{}{f["F1"], 0, data, 0644}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res.(int))

	// Insert to existing file
	args = []interface{}{f["F1"], 5, data, 0644}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res.(int))

	// Check file content
	processName = "fs.system.ReadFile"
	args = []interface{}{f["F1"]}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	contentDataShouldBe := append(data[:5], append(data, data[5:]...)...)
	content, ok := res.(string)
	assert.True(t, ok)
	assert.Equal(t, string(contentDataShouldBe), content)
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

func TestProcessFsGlob(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	testFsMakeData(t)
	testFsMakeF1(t)
	testFsMakeD1D2F1(t)

	processName := "fs.system.Glob"
	args := []interface{}{filepath.Join(f["root"], "*")}
	res, err := process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(res.([]string)))

	processName = "fs.system.Glob"
	args = []interface{}{filepath.Join(f["root"], "*.file")}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(res.([]string)))

	processName = "fs.system.Glob"
	args = []interface{}{filepath.Join(f["root"], "d1", "*.file")}
	res, err = process.New(processName, args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(res.([]string)))
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
	if runtime.GOOS != "windows" {
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

func TestProcessMoveAppend(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	testFsMakeData(t)

	// Mkdir
	stor := FileSystems["system"]
	data := testData(t)
	name := "TestProcessMoveAppend"
	err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
	assert.Nil(t, err, "TestProcessMoveAppend")
	checkFileExists(stor, t, f["D1"], name)
	checkFileExists(stor, t, f["D1_D2"], name)

	// Write
	_, err = WriteFile(stor, f["D1_D2_F1"], data, 0644)
	assert.Nil(t, err, name)
	checkFileExists(stor, t, f["D1_D2_F1"], name)

	_, err = WriteFile(stor, f["D1_D2_F2"], data, 0644)
	assert.Nil(t, err, name)
	checkFileExists(stor, t, f["D1_D2_F2"], name)

	processName := "fs.system.MoveAppend"
	args := []interface{}{f["D1_D2_F1"], f["D1_D2_F2"]}
	_, err = process.New(processName, args...).Exec()
	assert.Nil(t, err, name)

	// Check the content
	fileContent, err := ReadFile(stor, f["D1_D2_F2"])
	assert.Nil(t, err, name)
	contentDataShouldBe := append(data, data...)
	assert.Equal(t, contentDataShouldBe, fileContent, name)
	checkFileNotExists(stor, t, f["D1_D2_F1"], name)
}

func TestProcessMoveInsert(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	testFsMakeData(t)

	// Mkdir
	stor := FileSystems["system"]
	data := testData(t)
	name := "TestProcessMoveAppend"
	err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
	assert.Nil(t, err, "TestProcessMoveInsert")
	checkFileExists(stor, t, f["D1"], name)
	checkFileExists(stor, t, f["D1_D2"], name)

	// Write
	_, err = WriteFile(stor, f["D1_D2_F1"], data, 0644)
	assert.Nil(t, err, name)
	checkFileExists(stor, t, f["D1_D2_F1"], name)

	_, err = WriteFile(stor, f["D1_D2_F2"], data, 0644)
	assert.Nil(t, err, name)
	checkFileExists(stor, t, f["D1_D2_F2"], name)

	processName := "fs.system.MoveInsert"
	args := []interface{}{f["D1_D2_F1"], f["D1_D2_F2"], 2}
	_, err = process.New(processName, args...).Exec()
	assert.Nil(t, err, name)

	// Check the content
	fileContent, err := ReadFile(stor, f["D1_D2_F2"])
	assert.Nil(t, err, name)
	contentDataShouldBe := append(data[:2], append(data, data[2:]...)...)
	assert.Equal(t, contentDataShouldBe, fileContent, name)
	checkFileNotExists(stor, t, f["D1_D2_F1"], name)
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

func TestProcessFsAbs(t *testing.T) {
	// Test fs.system.Abs — system root is "/"
	t.Run("fs.system.Abs", func(t *testing.T) {
		processName := "fs.system.Abs"
		args := []interface{}{"/tmp/foo"}
		res, err := process.New(processName, args...).Exec()
		if err != nil {
			t.Fatal(err)
		}

		absPath, ok := res.(string)
		assert.True(t, ok)
		if runtime.GOOS == "windows" {
			assert.Equal(t, "/tmp/foo", absPath)
		} else {
			assert.Equal(t, filepath.Join("/", "/tmp/foo"), absPath)
		}
		t.Logf("fs.system.Abs(\"/tmp/foo\") = %s", absPath)
	})

	// Test fs.data.Abs — data root is app_root + "/data"
	t.Run("fs.data.Abs", func(t *testing.T) {
		appRoot := os.Getenv("GOU_TEST_APP_ROOT")
		if appRoot == "" {
			t.Skip("GOU_TEST_APP_ROOT not set")
		}
		dataRoot := filepath.Join(appRoot, "data")
		Register("data", system.New(dataRoot))

		processName := "fs.data.Abs"
		args := []interface{}{"/tmp/foo"}
		res, err := process.New(processName, args...).Exec()
		if err != nil {
			t.Fatal(err)
		}

		absPath, ok := res.(string)
		assert.True(t, ok)
		expected := filepath.Join(dataRoot, "/tmp/foo")
		assert.Equal(t, expected, absPath)
		t.Logf("fs.data.Abs(\"/tmp/foo\") = %s", absPath)
	})

	// Test fs.dsl.Abs — dsl root is app_root
	t.Run("fs.dsl.Abs", func(t *testing.T) {
		appRoot := os.Getenv("GOU_TEST_APP_ROOT")
		if appRoot == "" {
			t.Skip("GOU_TEST_APP_ROOT not set")
		}
		Register("dsl", system.New(appRoot))

		processName := "fs.dsl.Abs"
		args := []interface{}{"/models/user.yao"}
		res, err := process.New(processName, args...).Exec()
		if err != nil {
			t.Fatal(err)
		}

		absPath, ok := res.(string)
		assert.True(t, ok)
		expected := filepath.Join(appRoot, "/models/user.yao")
		assert.Equal(t, expected, absPath)
		t.Logf("fs.dsl.Abs(\"/models/user.yao\") = %s", absPath)
	})
}

func TestProcessFsIsLink(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	testFsMakeF1(t)

	res, err := process.New("fs.system.IsLink", f["F1"]).Exec()
	assert.Nil(t, err)
	assert.Equal(t, false, res)

	absF1 := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data", "f1.file")
	absLink := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data", "f1.file.link")
	err = os.Symlink(absF1, absLink)
	if err != nil {
		t.Skipf("Cannot create symlink (need privileges on Windows): %v", err)
	}
	defer os.Remove(absLink)

	linkPath := f["F1"] + ".link"
	res, err = process.New("fs.system.IsLink", linkPath).Exec()
	assert.Nil(t, err)
	assert.Equal(t, true, res)
}

func TestProcessFsUpload(t *testing.T) {
	appRoot := os.Getenv("GOU_TEST_APP_ROOT")
	if appRoot == "" {
		t.Skip("GOU_TEST_APP_ROOT not set")
	}
	dataRoot := filepath.Join(appRoot, "data")
	Register("system-upload", system.New(dataRoot))
	testFsClear(FileSystems["system"], t)

	tmpFile := filepath.Join(dataRoot, "upload_tmp.txt")
	err := os.WriteFile(tmpFile, []byte("hello upload"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	uf := types.UploadFile{
		Name:     "test.txt",
		TempFile: "upload_tmp.txt",
	}

	res, err := process.New("fs.system-upload.Upload", uf).Exec()
	assert.Nil(t, err)
	filename, ok := res.(string)
	assert.True(t, ok)
	assert.NotEmpty(t, filename)
	assert.True(t, strings.HasSuffix(filename, ".txt"))
}

func TestProcessFsUploadChunkSync(t *testing.T) {
	appRoot := os.Getenv("GOU_TEST_APP_ROOT")
	if appRoot == "" {
		t.Skip("GOU_TEST_APP_ROOT not set")
	}
	dataRoot := filepath.Join(appRoot, "data")
	Register("system-upload", system.New(dataRoot))
	testFsClear(FileSystems["system"], t)

	chunk1 := filepath.Join(dataRoot, "chunk1.tmp")
	err := os.WriteFile(chunk1, []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	uf1 := types.UploadFile{
		UID:      "test-chunk-uid",
		Name:     "test.txt",
		TempFile: "chunk1.tmp",
		Range:    "bytes 0-4/10",
		Sync:     true,
	}

	res, err := process.New("fs.system-upload.Upload", uf1).Exec()
	assert.Nil(t, err)
	resMap, ok := res.(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, resMap["progress"])

	chunk2 := filepath.Join(dataRoot, "chunk2.tmp")
	err = os.WriteFile(chunk2, []byte("world"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	uf2 := types.UploadFile{
		UID:      "test-chunk-uid",
		Name:     "test.txt",
		TempFile: "chunk2.tmp",
		Range:    "bytes 5-9/10",
		Sync:     true,
	}

	res, err = process.New("fs.system-upload.Upload", uf2).Exec()
	assert.Nil(t, err)
	filename, ok := res.(string)
	assert.True(t, ok)
	assert.NotEmpty(t, filename)
}

func TestProcessFsDownload(t *testing.T) {
	f := testFsFiles(t)
	testFsClear(FileSystems["system"], t)
	testFsMakeF1(t)

	res, err := process.New("fs.system.Download", f["F1"]).Exec()
	assert.Nil(t, err)

	resMap, ok := res.(map[string]interface{})
	assert.True(t, ok)

	content, ok := resMap["content"].(io.ReadCloser)
	assert.True(t, ok)
	if content != nil {
		content.Close()
	}

	mimeType, ok := resMap["type"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, mimeType)
}

func TestParseFileSize(t *testing.T) {
	size, err := parseFileSize(nil)
	assert.Nil(t, err)
	assert.Equal(t, int64(1024*1024), size)

	size, err = parseFileSize(int(1024))
	assert.Nil(t, err)
	assert.Equal(t, int64(1024), size)

	size, err = parseFileSize(int64(2048))
	assert.Nil(t, err)
	assert.Equal(t, int64(2048), size)

	size, err = parseFileSize("10M")
	assert.Nil(t, err)
	assert.Equal(t, int64(10*1024*1024), size)

	size, err = parseFileSize("5K")
	assert.Nil(t, err)
	assert.Equal(t, int64(5*1024), size)

	_, err = parseFileSize(true)
	assert.NotNil(t, err)
}

func TestValidateAcceptType(t *testing.T) {
	testFsClear(FileSystems["system"], t)
	stor := FileSystems["system"]

	root := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data")
	txtFile := filepath.Join(root, "test.txt")
	err := os.WriteFile(txtFile, []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotPanics(t, func() {
		validateAcceptType(stor, txtFile, ".txt", false)
	})

	assert.NotPanics(t, func() {
		validateAcceptType(stor, txtFile, "text/plain; charset=utf-8", true)
	})

	assert.NotPanics(t, func() {
		validateAcceptType(stor, txtFile, "text/*", true)
	})

	txtFile2 := filepath.Join(root, "test2.txt")
	err = os.WriteFile(txtFile2, []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	assert.Panics(t, func() {
		validateAcceptType(stor, txtFile2, ".pdf", true)
	})

	assert.Panics(t, func() {
		validateAcceptType(stor, txtFile, 12345, false)
	})
}

func TestValidateFileSize(t *testing.T) {
	testFsClear(FileSystems["system"], t)
	stor := FileSystems["system"]

	root := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data")
	txtFile := filepath.Join(root, "test_size.txt")
	err := os.WriteFile(txtFile, []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotPanics(t, func() {
		validateFileSize(stor, txtFile, "1M")
	})

	txtFile2 := filepath.Join(root, "test_size2.txt")
	err = os.WriteFile(txtFile2, []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	assert.Panics(t, func() {
		validateFileSize(stor, txtFile2, int(1))
	})
}

func TestProcessFsArgValidation(t *testing.T) {
	testFsClear(FileSystems["system"], t)

	handlers := []string{
		"fs.system.ReadFile",
		"fs.system.ReadFileBuffer",
		"fs.system.WriteFile",
		"fs.system.WriteFileBuffer",
		"fs.system.AppendFile",
		"fs.system.AppendFileBuffer",
		"fs.system.InsertFile",
		"fs.system.InsertFileBuffer",
		"fs.system.ReadDir",
		"fs.system.Mkdir",
		"fs.system.MkdirAll",
		"fs.system.Remove",
		"fs.system.RemoveAll",
		"fs.system.Exists",
		"fs.system.IsDir",
		"fs.system.IsFile",
		"fs.system.IsLink",
		"fs.system.Chmod",
		"fs.system.Size",
		"fs.system.Mode",
		"fs.system.ModTime",
		"fs.system.BaseName",
		"fs.system.DirName",
		"fs.system.ExtName",
		"fs.system.MimeType",
		"fs.system.Move",
		"fs.system.MoveAppend",
		"fs.system.MoveInsert",
		"fs.system.Copy",
		"fs.system.Upload",
		"fs.system.Download",
		"fs.system.Zip",
		"fs.system.Unzip",
		"fs.system.Glob",
		"fs.system.Abs",
	}

	for _, name := range handlers {
		t.Run(name, func(t *testing.T) {
			_, err := process.New(name).Exec()
			assert.NotNil(t, err, "should error with 0 args: %s", name)
		})
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
