package yaz

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application/yaz/ciphers"
	"github.com/yaoapp/kun/utils"
)

func TestOpen(t *testing.T) {
	vars := data(t)
	defer clean(t, vars)

	compress, err := os.Open(vars["compress"])
	if err != nil {
		t.Fatal(err)
	}
	defer compress.Close()

	_, err = Open(compress, vars["compress"], nil)
	assert.Nil(t, err)

	pack, err := os.Open(vars["pack"])
	if err != nil {
		t.Fatal(err)
	}
	defer pack.Close()

	_, err = Open(pack, vars["pack"], ciphers.NewAES([]byte(vars["aseKey"])))
	assert.Nil(t, err)
}

func TestOpenFile(t *testing.T) {
	vars := data(t)
	defer clean(t, vars)

	_, err := OpenFile(vars["compress"], nil)
	assert.Nil(t, err)

	_, err = OpenFile(vars["pack"], ciphers.NewAES([]byte(vars["aseKey"])))
	assert.Nil(t, err)

	_, err = OpenFile("not exists", nil)
	assert.NotNil(t, err)
}

func TestGlob(t *testing.T) {
	vars := data(t)
	defer clean(t, vars)

	app, err := OpenFile(vars["compress"], nil)
	if err != nil {
		t.Fatal(err)
	}

	matches, err := app.Glob("models/*.mod.yao")
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(matches), 1)
	utils.Dump(matches)

	matches, err = app.Glob("/models/*.mod.yao")
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(matches), 1)

	matches, err = app.Glob("/models/*.tab.yao")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, matches)
}

func TestWalk(t *testing.T) {
	vars := data(t)
	defer clean(t, vars)

	app, err := OpenFile(vars["compress"], nil)
	if err != nil {
		t.Fatal(err)
	}

	files := []string{}
	err = app.Walk("models", func(root, filename string, isdir bool) error {
		files = append(files, filepath.Join(filename))
		assert.IsType(t, true, isdir)
		assert.IsType(t, "string", filename)
		assert.Equal(t, "models", root)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(files), 1)
}

func TestWalkWithPatterns(t *testing.T) {
	vars := data(t)
	defer clean(t, vars)

	app, err := OpenFile(vars["compress"], nil)
	if err != nil {
		t.Fatal(err)
	}

	files := []string{}
	err = app.Walk("scripts", func(root, filename string, isdir bool) error {
		files = append(files, filepath.Join(filename))
		assert.IsType(t, true, isdir)
		assert.IsType(t, "string", filename)
		assert.Equal(t, "scripts", root)
		if !isdir {
			ext := filepath.Ext(filename)
			assert.True(t, ext == ".ts" || ext == ".js")
		}
		return nil
	}, "*.js", "*.ts")

	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(files), 1)
}

func TestRead(t *testing.T) {

	vars := data(t)
	defer clean(t, vars)

	// test compress
	app, err := OpenFile(vars["compress"], nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := app.Read(filepath.Join("models", "user.mod.yao"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(data), 1)
	assert.Contains(t, string(data), "columns")

	data, err = app.Read(filepath.Join("/", "models", "user.mod.yao"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(data), 1)
	assert.Contains(t, string(data), "columns")

	_, err = app.Read(filepath.Join("/", "models", "user.mod.yao-not-exists"))
	assert.NotNil(t, err)

	// test pack without cipher
	app, err = OpenFile(vars["pack"], nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err = app.Read(filepath.Join("models", "user.mod.yao"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(data), 1)
	assert.NotContains(t, string(data), "columns")

	data, err = app.Read(filepath.Join("/", "models", "user.mod.yao"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(data), 1)
	assert.NotContains(t, string(data), "columns")

	_, err = app.Read(filepath.Join("/", "models", "user.mod.yao-not-exists"))
	assert.NotNil(t, err)

	// test pack with cipher
	aesCipher := ciphers.NewAES([]byte(vars["aseKey"]))
	app, err = OpenFile(vars["pack"], aesCipher)
	if err != nil {
		t.Fatal(err)
	}

	data, err = app.Read(filepath.Join("models", "user.mod.yao"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(data), 1)
	assert.Contains(t, string(data), "columns")

	data, err = app.Read(filepath.Join("/", "models", "user.mod.yao"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(data), 1)
	assert.Contains(t, string(data), "columns")

	_, err = app.Read(filepath.Join("/", "models", "user.mod.yao-not-exists"))
	assert.NotNil(t, err)

}

func TestWriteRemoveWatchExist(t *testing.T) {
	vars := data(t)
	defer clean(t, vars)

	// test compress
	app, err := OpenFile(vars["compress"], nil)
	if err != nil {
		t.Fatal(err)
	}

	err = app.Write(filepath.Join("models", "user.mod.yao"), []byte("test"))
	assert.NotNil(t, err)

	err = app.Remove(filepath.Join("models", "user.mod.yao"))
	assert.NotNil(t, err)

	err = app.Watch(func(event, name string) {}, nil)
	assert.NotNil(t, err)

	exists, _ := app.Exists(filepath.Join("models", "user.mod.yao"))
	assert.True(t, exists)

	exists, _ = app.Exists(filepath.Join("models", "temp.mod.yao"))
	assert.False(t, exists)
}

func data(t *testing.T) map[string]string {

	vars := prepare(t)
	compress, err := Compress(vars["root"])
	if err != nil {
		t.Fatal(err)
	}

	aesCipher := ciphers.NewAES([]byte(vars["aseKey"]))
	pack, err := Pack(vars["root"], aesCipher)
	if err != nil {
		t.Fatal(err)
	}

	vars["pack"] = pack
	vars["compress"] = compress
	return vars
}

func clean(t *testing.T, vars map[string]string) {
	os.Remove(vars["pack"])
	os.Remove(vars["compress"])
}
