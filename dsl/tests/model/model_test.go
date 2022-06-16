package model

import (
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

func TestDSLRefresh(t *testing.T) {}

func TestDSLChange(t *testing.T) {}

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
