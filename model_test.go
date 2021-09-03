package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xun/capsule"
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
	assert.Equal(t, user.Get("mobile"), "13900001111")
	assert.Equal(t, user.Dot().Get("extra.sex"), "男")
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

	userDot := user.Dot()
	assert.Equal(t, userDot.Get("mobile"), "13900001111")
	assert.Equal(t, userDot.Get("extra.sex"), "男")
	assert.Equal(t, userDot.Get("manu.name"), "北京云道天成科技有限公司")
	assert.Equal(t, userDot.Get("addresses.0.location"), "北京国家数字出版基地A103")
	assert.Equal(t, userDot.Get("mother.extra.sex"), "女")
	assert.Equal(t, userDot.Get("mother.friends.friend_id"), int64(2))
	assert.Equal(t, userDot.Get("mother.friends.type"), "monther")
}

func TestModelMustSearch(t *testing.T) {
	user := Select("user").MustSearch(QueryParam{}, 1, 2)
	userDot := user.Dot()
	assert.Equal(t, userDot.Get("total"), 3)
	assert.Equal(t, userDot.Get("next"), 2)
	assert.Equal(t, userDot.Get("page"), 1)
	assert.Equal(t, userDot.Get("data.0.id"), int64(1))
	assert.Equal(t, userDot.Get("data.1.id"), int64(2))
}

func TestModelMustSearchWiths(t *testing.T) {
	user := Select("user").MustSearch(QueryParam{
		Withs: map[string]With{
			"manu":      {},
			"addresses": {},
			"mother":    {},
		},
	}, 1, 2)
	userDot := user.Dot()
	assert.Equal(t, userDot.Get("total"), 3)
	assert.Equal(t, userDot.Get("next"), 2)
	assert.Equal(t, userDot.Get("page"), 1)
	assert.Equal(t, userDot.Get("data.0.id"), int64(1))
	assert.Equal(t, userDot.Get("data.0.manu.name"), "北京云道天成科技有限公司")
	assert.Equal(t, userDot.Get("data.0.mother.extra.sex"), "女")
	assert.Equal(t, userDot.Get("data.0.extra.sex"), "男")
	assert.Equal(t, userDot.Get("data.0.addresses.0.location"), "北京国家数字出版基地A103")
	assert.Equal(t, userDot.Get("data.1.id"), int64(2))
}

func TestModelMustSearchWithsWhere(t *testing.T) {
	user := Select("user").MustSearch(QueryParam{
		Wheres: []QueryWhere{
			{
				Column: "mobile",
				Value:  "13900001111",
			},
		},
		Withs: map[string]With{
			"manu":      {},
			"addresses": {},
			"mother":    {},
		},
	}, 1, 2)
	userDot := user.Dot()
	assert.Equal(t, userDot.Get("total"), 1)
	assert.Equal(t, userDot.Get("next"), -1)
	assert.Equal(t, userDot.Get("page"), 1)
	assert.Equal(t, userDot.Get("data.0.id"), int64(1))
	assert.Equal(t, userDot.Get("data.0.manu.name"), "北京云道天成科技有限公司")
	assert.Equal(t, userDot.Get("data.0.mother.extra.sex"), "女")
	assert.Equal(t, userDot.Get("data.0.extra.sex"), "男")
	assert.Equal(t, userDot.Get("data.0.addresses.0.location"), "北京国家数字出版基地A103")

}

func TestModelMustSearchWithsWheresOrder(t *testing.T) {
	user := Select("user").MustSearch(QueryParam{
		Orders: []QueryOrder{
			{
				Column: "id",
				Option: "desc",
			},
		},
		Wheres: []QueryWhere{
			{
				Wheres: []QueryWhere{
					{
						Column: "mobile",
						Value:  "13900002222",
					}, {
						Column: "mobile",
						Method: "orwhere",
						Value:  "13900001111",
					},
				},
			},
		},
		Withs: map[string]With{
			"manu":      {},
			"addresses": {},
			"mother":    {},
		},
	}, 1, 2)
	userDot := user.Dot()
	assert.Equal(t, userDot.Get("total"), 2)
	assert.Equal(t, userDot.Get("next"), -1)
	assert.Equal(t, userDot.Get("page"), 1)
	assert.Equal(t, userDot.Get("data.1.id"), int64(1))
	assert.Equal(t, userDot.Get("data.1.manu.name"), "北京云道天成科技有限公司")
	assert.Equal(t, userDot.Get("data.1.mother.extra.sex"), "女")
	assert.Equal(t, userDot.Get("data.1.extra.sex"), "男")
	assert.Equal(t, userDot.Get("data.1.addresses.0.location"), "北京国家数字出版基地A103")

}

func TestModelMustSaveNew(t *testing.T) {
	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name":     "用户创建",
		"manu_id":  2,
		"type":     "user",
		"idcard":   "23082619820207006X",
		"mobile":   "13900004444",
		"password": "qV@uT1DI",
		"key":      "XZ12MiPp",
		"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
		"status":   "enabled",
		"extra":    maps.MapStr{"sex": "女"},
	})

	row := user.MustFind(id, QueryParam{})

	// 清空数据
	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()

	assert.Equal(t, row.Get("name"), "用户创建")
	assert.Equal(t, row.Dot().Get("extra.sex"), "女")

}

func TestModelMustSaveUpdate(t *testing.T) {
	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"id":      1,
		"balance": 200,
	})

	row := user.MustFind(id, QueryParam{})

	// 恢复数据
	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Update(maps.MapStr{"balance": 0})
	assert.Equal(t, any.Of(row.Get("balance")).CInt(), 200)
}

func TestModelMustDeleteSoft(t *testing.T) {
	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name":     "用户创建",
		"manu_id":  2,
		"type":     "user",
		"idcard":   "23082619820207006X",
		"mobile":   "13900004444",
		"password": "qV@uT1DI",
		"key":      "XZ12MiPp",
		"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
		"status":   "enabled",
		"extra":    maps.MapStr{"sex": "女"},
	})
	err := user.Delete(id)
	row, _ := user.Find(id, QueryParam{})

	// 清空数据
	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
	assert.Nil(t, row)
	assert.Nil(t, err)
}

func TestModelMustDestory(t *testing.T) {
	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name":     "用户创建",
		"manu_id":  2,
		"type":     "user",
		"idcard":   "23082619820207006X",
		"mobile":   "13900004444",
		"password": "qV@uT1DI",
		"key":      "XZ12MiPp",
		"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
		"status":   "enabled",
		"extra":    maps.MapStr{"sex": "女"},
	})
	err := user.Destroy(id)
	assert.Nil(t, err)

	row, err := capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).First()
	assert.True(t, row.IsEmpty())
	assert.Nil(t, err)
}
