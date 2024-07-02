package fs

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
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
		err := Mkdir(stor, f["D1_D2"], uint32(os.ModePerm))
		assert.NotNil(t, err, name)

		err = Mkdir(stor, f["D1"], uint32(os.ModePerm))
		assert.Nil(t, err, name)
	}
}

func TestMkdirAll(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
		assert.Nil(t, err, name)

		err = Mkdir(stor, f["D1"], uint32(os.ModePerm))
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
		err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
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

func TestGlob(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Mkdir
		err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
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

		files, err := stor.Glob(f["D1"] + "/*")
		assert.Nil(t, err, name)
		assert.Equal(t, 3, len(files), name)

		files, err = stor.Glob(f["D1"] + "/*/*")
		assert.Nil(t, err, name)
		assert.Equal(t, 2, len(files), name)
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

		err = MkdirAll(stor, f["D1"], uint32(os.ModePerm))
		assert.Nil(t, err, name)
		err = Chmod(stor, f["D1"], 0400)
		assert.Nil(t, err, name)

		l32, err := WriteFile(stor, f["D1_D2_F1"], data, 0644)
		assert.NotNil(t, err, name)
		assert.Equal(t, l32, 0)
	}
}

func TestWrite(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)

		// Write
		f1 := strings.NewReader("Hello F1")
		length, err := Write(stor, f["F1"], f1, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F1"], name)
		checkFileSize(stor, t, f["F1"], length, name)
		checkFileMode(stor, t, f["F1"], 0644, name)

		// Overwrite
		f2 := strings.NewReader("Hello F2")
		l21, err := Write(stor, f["F2"], f2, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F2"], name)
		checkFileSize(stor, t, f["F2"], l21, name)
		checkFileMode(stor, t, f["F2"], 0644, name)

		f3 := strings.NewReader("Hello F3")
		l22, err := Write(stor, f["F2"], f3, 0644)
		assert.Nil(t, err, name)
		checkFileExists(stor, t, f["F2"], name)
		checkFileSize(stor, t, f["F2"], l22, name)
		checkFileMode(stor, t, f["F2"], 0644, name)

		// permission denied
		err = Chmod(stor, f["F2"], 0400)
		assert.Nil(t, err, name)
		l31, err := Write(stor, f["F2"], f2, 0644)
		assert.NotNil(t, err, name)
		assert.Equal(t, l31, 0)

		err = MkdirAll(stor, f["D1"], uint32(os.ModePerm))
		assert.Nil(t, err, name)
		err = Chmod(stor, f["D1"], 0400)
		assert.Nil(t, err, name)

		d1d2f1 := strings.NewReader("Hello D1_D2_F1")
		l32, err := Write(stor, f["D1_D2_F1"], d1d2f1, 0644)
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
		err = MkdirAll(stor, f["D1"], uint32(os.ModePerm))
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
		err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
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
		err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
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
		err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
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
		assert.Equal(t, uint32(0644), mode)

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
		err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
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
		assert.Equal(t, uint32(0644), mode)

		err = Chmod(stor, f["F1"], uint32(0755))
		assert.Nil(t, err, name)

		mode, err = Mode(stor, f["F1"])
		assert.Nil(t, err, name)
		assert.Equal(t, uint32(0755), mode)

		// Chmod Dir
		mode, err = Mode(stor, f["D1_D2"])
		assert.Nil(t, err, name)

		err = Chmod(stor, f["D1"], uint32(0755))
		assert.Nil(t, err, name)

		mode, err = Mode(stor, f["D1_D2"])
		assert.Nil(t, err, name)
		assert.Equal(t, uint32(0755), mode)
	}
}

func TestList(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	stor := stores["system"]
	clear(stor, t)

	// Mkdir
	err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
	assert.Nil(t, err)

	// Write
	_, err = WriteFile(stor, f["F1"], []byte("F1"), 0644)
	assert.Nil(t, err)

	_, err = WriteFile(stor, f["D1_D2_F1"], []byte("D1_D2_F1"), 0644)
	assert.Nil(t, err)

	time.Sleep(200 * time.Millisecond)

	_, err = WriteFile(stor, f["D1_D2_F2"], []byte("D1_D2_F2"), 0644)
	assert.Nil(t, err)

	// List
	files, total, pagecnt, err := stor.List(f["D1"], []string{".file"}, 1, 1, func(s string) bool {
		return true
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, 2, total)
	assert.Equal(t, 2, pagecnt)
	assert.Equal(t, "f2.file", filepath.Base(files[0]))

	files, total, pagecnt, err = stor.List(f["D1"], []string{".file"}, 2, 1, func(s string) bool {
		return true
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, 2, total)
	assert.Equal(t, 2, pagecnt)
	assert.Equal(t, "f1.file", filepath.Base(files[0]))

	files, total, pagecnt, err = stor.List(f["D1"], []string{".file"}, 3, 1, func(s string) bool {
		return true
	})

	assert.Nil(t, err)
	assert.Equal(t, 0, len(files))
	assert.Equal(t, 2, total)
	assert.Equal(t, 2, pagecnt)

	stor.CleanCache()
	files, total, pagecnt, err = stor.List(f["D1"], []string{".file"}, 1, 1, func(s string) bool {
		return strings.Contains(filepath.Base(s), "f2")
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, pagecnt)
	assert.Equal(t, "f2.file", filepath.Base(files[0]))
}

func TestResize(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	stor := stores["system"]
	clear(stor, t)

	blob := blankImg(t)

	// Write
	_, err := WriteFile(stor, f["I1"], blob, 0644)
	assert.Nil(t, err)

	// Resize
	err = stor.Resize(f["I1"], f["I2"], 10, 10)
	assert.Nil(t, err)
	checkFileExists(stor, t, f["I2"], "system")

	newBlob, err := ReadFile(stor, f["I2"])
	if err != nil {
		t.Fatal(err)
	}

	img, _, err := image.Decode(bytes.NewReader(newBlob))
	if err != nil {
		t.Fatal(err)
	}

	bounds := img.Bounds()
	assert.Equal(t, 10, bounds.Dx())
	assert.Equal(t, 10, bounds.Dy())
}

func TestMove(t *testing.T) {
	stores := testStores(t)
	f := testFiles(t)
	for name, stor := range stores {
		clear(stor, t)
		data := testData(t)

		// Mkdir
		err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
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
		err := MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
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

func TestUnzipZip(t *testing.T) {

	stores := testStores(t)
	f := testFiles(t)
	stor := stores["system"]
	clear(stor, t)

	data := testData(t)
	names := []string{"D1_D2_F1", "F1", "F2", "D1_F1", "D1_F2"}
	for _, name := range names {
		_, err := WriteFile(stor, f[name], data, 0644)
		if err != nil {
			t.Fatalf("WriteFile error: %s", err)
		}
	}

	// Zip
	stor = stores["system-relpath"]
	err := Zip(stor, "d1", "d1.zip")
	if err != nil {
		t.Fatalf("Zip error: %s", err)
	}

	// Unzip
	files, err := Unzip(stor, "d1.zip", "d1-unzip")
	if err != nil {
		t.Fatalf("Unzip error: %s", err)
	}

	assert.Len(t, files, 3)
	assert.Contains(t, files, "d1-unzip/d2/f1.file")
	assert.Contains(t, files, "d1-unzip/f1.file")
	assert.Contains(t, files, "d1-unzip/f2.file")
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
		"I1":       filepath.Join(root, "i1.png"),
		"I2":       filepath.Join(root, "i2.png"),
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

func checkFileMode(stor FileSystem, t assert.TestingT, path string, mode uint32, name string) {
	realMode, _ := Mode(stor, path)
	assert.Equal(t, mode, realMode, name)
}

func clear(stor FileSystem, t *testing.T) {

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

func blankImg(t *testing.T) []byte {
	code := `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`
	blob, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		t.Fatal(err)
	}
	return blob
}
