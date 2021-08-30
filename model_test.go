package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/utils"
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
	for name, mod := range Models {
		utils.Dump(name)
		mod.Migrate(true)
	}
}

func TestModelMustFind(t *testing.T) {
	user := Select("user").MustFind(1)
	assert.Equal(t, user.Get("mobile"), "13900001111")
	assert.Equal(t, user.Dot().Get("extra.sex"), "男")
}

func TestModelMustFindWithHasOne(t *testing.T) {
	user := Select("user").MustFind(1,
		With{Name: "manu"},
		With{Name: "addresses", Query: QueryParam{Page: 2, PageSize: 1}},
		With{Name: "roles"},
		With{
			Name: "mother", Query: QueryParam{
				Withs: map[string]With{"addresses": {
					Name:  "addresses",
					Query: QueryParam{Page: 2, PageSize: 1},
				}},
			}},
	)
	utils.Dump(user)
}
