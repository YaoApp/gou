package dsl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/dsl/workshop"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
)

func TestYaoOpen(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := filepath.Join(root, "models", "user.mod.yao")
	workshop, err := workshop.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	yao := New(workshop)
	err = yao.Open(file)
	if err != nil {
		t.Fatal(err)
	}

	assert.FileExists(t, yao.Head.File)
	assert.Equal(t, "@infra.erp.models.user", yao.Head.From)
	assert.Equal(t, "1.0.0", yao.Head.Lang.String())
	assert.Equal(t, "1.0.0", yao.Head.Version.String())
	assert.Equal(t, Model, yao.Head.Type)
	assert.Equal(t, "user", yao.Head.Name)
	assert.Equal(t, 1, len(yao.Head.Run.APPEND))
	for key, arr := range yao.Head.Run.APPEND[0] {
		assert.Equal(t, "columns", key)
		assert.Equal(t, 1, len(arr))
		v := any.Of(arr[0]).MapStr()
		assert.Equal(t, "Published At", v.Get("comment"))
		assert.Equal(t, "Published At", v.Get("label"))
		assert.Equal(t, "published_at", v.Get("name"))
		assert.Equal(t, "datetime", v.Get("type"))
		assert.Equal(t, true, v.Get("index"))
		assert.Equal(t, true, v.Get("nullable"))

	}

	assert.Equal(t, 1, len(yao.Head.Run.REPLACE))
	for key, value := range yao.Head.Run.REPLACE[0] {
		assert.Equal(t, "table", key)
		assert.Equal(t, "$new.table", value)
	}

	assert.Equal(t, []string{"columns[1]", "columns[2]"}, yao.Head.Run.DELETE)
	assert.Equal(t, 1, len(yao.Head.Run.MERGE))
}

func TestYaoCompileModelFromRemote(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := filepath.Join(root, "models", "from", "remote.mod.yao")
	workshop, err := workshop.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	yao := New(workshop)
	err = yao.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	err = yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(yao.Trace))
}

func TestYaoCompileModelFromRemoteDeep(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := filepath.Join(root, "models", "from", "remote-deep.mod.yao")
	workshop, err := workshop.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	yao := New(workshop)
	err = yao.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	err = yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 3, len(yao.Trace))
}

func TestYaoCompileModelMerge(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := filepath.Join(root, "models", "from", "merge.mod.yao")
	workshop, err := workshop.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	yao := New(workshop)
	err = yao.Open(file)
	if err != nil {
		t.Fatal(err)
	}

	err = yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	res := maps.Of(yao.Compiled).Dot()

	assert.Equal(t, "Merge", res.Get("name"))
	assert.Equal(t, "author_index", res.Get("indexes[0].name"))
	assert.Equal(t, "New User", res.Get("columns[1].label"))
	assert.Equal(t, "New Author {{input}} should be string", res.Get("columns[3].validations[0].message"))
	assert.Equal(t, "author_index", res.Get("indexes[0].name"))
	assert.Equal(t, "user_id_phone_unique", res.Get("indexes[1].name"))
	assert.Equal(t, false, res.Has("tmpl"))

}
