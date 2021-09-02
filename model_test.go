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
	user := Select("user").MustFind(1, QueryParam{})
	utils.Dump(user)
	// assert.Equal(t, user.Get("mobile"), "13900001111")
	// assert.Equal(t, user.Dot().Get("extra.sex"), "男")
}

func TestModelMustFindWiths(t *testing.T) {
	user := Select("user").MustFind(1,
		QueryParam{
			Withs: map[string]With{
				"manu":      {},
				"addresses": {},
				"roles":     {}, // 暂未实现（ 下一版支持 )
				"mother": {
					Query: QueryParam{ // 数据归集存在BUG（ 下一版修复 )
						Withs: map[string]With{
							// "addresses": {},
							// "manu": {},
						},
					},
				},
			},
		})
	utils.Dump(user)
}
