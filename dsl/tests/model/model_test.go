package model

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/dsl"
	"github.com/yaoapp/gou/dsl/workshop"
)

func TestDSLCompileSimple(t *testing.T) {
	yao := newModel(t, "simple.mod.yao")
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}
	m, has := gou.Models["simple"]
	assert.Equal(t, true, has)
	assert.Equal(t, "simple", m.Name)
	assert.Equal(t, "simple_tab", m.MetaData.Table.Name)
	assert.Equal(t, 14, len(m.ColumnNames))
}

func TestDSLCompileNameSpace(t *testing.T) {
	yao := newModel(t, filepath.Join("admin", "dashboard", "user.mod.yao"))
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}
	m, has := gou.Models["admin.dashboard.user"]
	assert.Equal(t, true, has)
	assert.Equal(t, "admin.dashboard.user", m.Name)
	assert.Equal(t, "yao_user", m.MetaData.Table.Name)
	assert.Equal(t, 16, len(m.ColumnNames))
}

func TestDSLCompileRouterTableName(t *testing.T) {
	yao := newModel(t, filepath.Join("admin", "staff.mod.yao"))
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}
	m, has := gou.Models["admin.staff"]
	assert.Equal(t, true, has)
	assert.Equal(t, "admin.staff", m.Name)
	assert.Equal(t, "yao_admin_staff", m.MetaData.Table.Name)
	assert.Equal(t, 16, len(m.ColumnNames))
}

func TestDSLCheckFatal(t *testing.T) {
	yao := newModel(t, "fatal.mod.yao")
	err := yao.Check()
	assert.Contains(t, err.Error(), "fatal.mod.yao columns[11].name id is existed")
}

func TestDSLRefresh(t *testing.T) {
	yao := newModel(t, "simple.mod.yao")
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	m, has := gou.Models["simple"]
	assert.Equal(t, true, has)
	assert.Equal(t, "simple", m.Name)
	assert.Equal(t, "simple_tab", m.MetaData.Table.Name)
	assert.Equal(t, 14, len(m.ColumnNames))

	// Backup content
	file := yao.Head.File
	backup, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	defer ioutil.WriteFile(file, backup, 0644) // RESET

	// change the content
	err = ioutil.WriteFile(file, []byte(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"FROM": "@github.com/yaoapp/workshop-tests-crm/models/user",
		"table": { "name": "simple_refresh_tab" }
	  }`), 0644)

	if err != nil {
		t.Fatal(err)
	}

	err = yao.Refresh()
	if err != nil {
		t.Fatal(err)
	}

	m, has = gou.Models["simple"]
	assert.Equal(t, true, has)
	assert.Equal(t, "simple", m.Name)
	assert.Equal(t, "simple_refresh_tab", m.MetaData.Table.Name)
	assert.Equal(t, 14, len(m.ColumnNames))

}

func TestDSLRemove(t *testing.T) {
	yao := newModel(t, "simple.mod.yao")
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	m, has := gou.Models["simple"]
	assert.Equal(t, true, has)
	assert.Equal(t, "simple", m.Name)
	assert.Equal(t, "simple_tab", m.MetaData.Table.Name)
	assert.Equal(t, 14, len(m.ColumnNames))

	// Remove
	err = yao.Remove()
	if err != nil {
		t.Fatal(err)
	}

	_, has = gou.Models["simple"]
	assert.Equal(t, false, has)
	assert.Equal(t, map[string]interface{}{}, yao.Content)
}

func TestDSLOnCreate(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := filepath.Join(root, "models", "simple.mod.yao")
	workshop, err := workshop.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	yao := dsl.New(workshop)
	err = yao.On(dsl.CREATE, file)
	if err != nil {
		t.Fatal(err)
	}

	m, has := gou.Models["simple"]
	assert.Equal(t, true, has)
	assert.Equal(t, "simple", m.Name)
	assert.Equal(t, "simple_tab", m.MetaData.Table.Name)
	assert.Equal(t, 14, len(m.ColumnNames))
}

func TestDSLOnChange(t *testing.T) {
	yao := newModel(t, "simple.mod.yao")
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	m, has := gou.Models["simple"]
	assert.Equal(t, true, has)
	assert.Equal(t, "simple", m.Name)
	assert.Equal(t, "simple_tab", m.MetaData.Table.Name)
	assert.Equal(t, 14, len(m.ColumnNames))

	// Backup content
	file := yao.Head.File
	backup, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	defer ioutil.WriteFile(file, backup, 0644) // RESET

	// change the content
	err = ioutil.WriteFile(file, []byte(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"FROM": "@github.com/yaoapp/workshop-tests-crm/models/user",
		"table": { "name": "simple_refresh_tab" }
	  }`), 0644)

	if err != nil {
		t.Fatal(err)
	}

	err = yao.On(dsl.CHANGE, file)
	if err != nil {
		t.Fatal(err)
	}

	m, has = gou.Models["simple"]
	assert.Equal(t, true, has)
	assert.Equal(t, "simple", m.Name)
	assert.Equal(t, "simple_refresh_tab", m.MetaData.Table.Name)
	assert.Equal(t, 14, len(m.ColumnNames))

}

func TestDSLOnRemove(t *testing.T) {
	yao := newModel(t, "simple.mod.yao")
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	m, has := gou.Models["simple"]
	assert.Equal(t, true, has)
	assert.Equal(t, "simple", m.Name)
	assert.Equal(t, "simple_tab", m.MetaData.Table.Name)
	assert.Equal(t, 14, len(m.ColumnNames))

	file := yao.Head.File
	err = yao.On(dsl.REMOVE, file)

	if err != nil {
		t.Fatal(err)
	}

	_, has = gou.Models["simple"]
	assert.Equal(t, false, has)
	assert.Equal(t, map[string]interface{}{}, yao.Content)
}

func newModel(t *testing.T, name string) *dsl.YAO {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := filepath.Join(root, "models", name)
	workshop, err := workshop.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	yao := dsl.New(workshop)
	err = yao.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	return yao
}
