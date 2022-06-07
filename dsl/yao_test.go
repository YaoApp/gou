package dsl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYaoOpen(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := filepath.Join(root, "models", "user.mod.yao")
	yao := New()
	err := yao.Open(file)
	if err != nil {
		t.Fatal(err)
	}

	assert.FileExists(t, yao.Head.File)
	assert.Equal(t, "@infra.erp.models.user", yao.Head.From)
	assert.Equal(t, "1.0.0", yao.Head.Lang.String())
	assert.Equal(t, "1.0.0", yao.Head.Version.String())
	assert.Equal(t, Model, yao.Head.Type)
	assert.Equal(t, "user", yao.Head.Name)
	assert.Equal(t, []string{"columns", "indexes"}, yao.Head.Run.APPEND)
	assert.Equal(t, []string{"columns.1", "columns.2"}, yao.Head.Run.DELETE)
	assert.Equal(t, []string{"table"}, yao.Head.Run.REPLACE)
	assert.Equal(t, 1, len(yao.Head.Run.MERGE))

}
