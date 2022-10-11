package gou

import (
	"fmt"
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
	process := "fs.system.readFile"
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
	process := "fs.system.readFileBuffer"
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

	process := "fs.system.writeFile"
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

	process := "fs.system.writeFileBuffer"
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

	err = fs.MkdirAll(stor, root, int(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}
}
