package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadModel(t *testing.T) {
	source := "file://" + path.Join(TestModRoot, "user.json")
	user := LoadModel(source, "user")
	assert.Equal(t, user.MetaData.Name, "用户")
	assert.Equal(t, user.Name, "user")
	assert.Equal(t, user.Source, source)
}

func TestModelReload(t *testing.T) {
	user := Select("user")
	user.Reload()
	assert.Equal(t, user.MetaData.Name, "用户")
	assert.Equal(t, user.Name, "user")
}

func TestModelMigrate(t *testing.T) {
	Select("user").Migrate(true)
	Select("manu").Migrate(true)
}

func TestModelMustFind(t *testing.T) {
	user := Select("user").MustFind(1)
	assert.Equal(t, user.Get("mobile"), "13900001111")
	assert.Equal(t, user.Dot().Get("extra.sex"), "男")
}
