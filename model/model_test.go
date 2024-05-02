package model

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/capsule"
)

func TestLoad(t *testing.T) {
	prepare(t)
	defer clean()
	check(t)

	_, err := Load("not-found", "not-found")
	assert.NotNil(t, err)

	_, has := Models["not-found"]
	assert.False(t, has)
}

func TestLoadWithoutDB(t *testing.T) {
	prepare(t)
	defer clean()
	dbclose()
	capsule.Global = nil

	check(t)
	_, err := Load("not-found", "not-found")
	assert.NotNil(t, err)

	_, has := Models["not-found"]
	assert.False(t, has)
}

func TestReload(t *testing.T) {
	prepare(t)
	defer clean()

	user := Select("user")
	_, err := user.Reload()
	assert.Nil(t, err)
}

func TestMigrate(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	users := user.MustGet(QueryParam{Select: []interface{}{"id"}})
	assert.Equal(t, len(users), 2)

	err := user.Migrate(true)
	assert.Nil(t, err)
	users = user.MustGet(QueryParam{Select: []interface{}{"id"}})
	assert.Equal(t, len(users), 0)
}

func TestExists(t *testing.T) {
	prepare(t)
	defer clean()
	assert.True(t, Exists("user"))
	assert.False(t, Exists("not-found"))
}

func TestRead(t *testing.T) {
	prepare(t)
	defer clean()
	dsl := Read("user")
	assert.Equal(t, dsl.Name, "User")
	assert.Panics(t, func() {
		Read("not-found")
	})
}

func TestModelMustFind(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user").MustFind(1, QueryParam{})
	assert.Equal(t, user.Get("mobile"), "1234567890")
	assert.Equal(t, user.Dot().Get("extra.sex"), "male")
}

func TestModelMustFindWiths(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)
	pet := Select("pet").MustFind(1, QueryParam{
		Withs: map[string]With{"category": {Query: QueryParam{
			Select: []interface{}{"id", "name"},
		}}},
	})
	res := pet.Dot()
	assert.Equal(t, res.Get("category_id"), res.Get("category.id"))

	// Multiple withs
	pet = Select("pet").MustFind(1, QueryParam{
		Withs: map[string]With{"category": {}, "owner": {}},
	})
	res = pet.Dot()
	assert.NotEmpty(t, res.Get("category.name"))
	assert.NotEmpty(t, res.Get("owner.name"))
	assert.Equal(t, res.Get("category_id"), res.Get("category.id"))
	assert.Equal(t, res.Get("owner_id"), res.Get("owner.id"))

	// Multiple withs the same model
	pet = Select("pet").MustFind(1, QueryParam{
		Withs: map[string]With{"category": {}, "owner": {}, "doctor": {}},
	})

	res = pet.Dot()
	assert.NotEmpty(t, res.Get("category.name"))
	assert.NotEmpty(t, res.Get("owner.name"))
	assert.NotEmpty(t, res.Get("doctor.name"))
	assert.Equal(t, res.Get("category_id"), res.Get("category.id"))
	assert.Equal(t, res.Get("owner_id"), res.Get("owner.id"))
	assert.Equal(t, res.Get("doctor_id"), res.Get("doctor.id"))

}

func TestModelMustGet(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	pets := Select("pet").MustGet(QueryParam{})
	if len(pets) != 4 {
		t.Fatal("pets length not equal 4")
	}
	res := pets[0].Dot()
	assert.Equal(t, res.Get("name"), "Tommy")
}

func TestModelMustGetWiths(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	pets := Select("pet").MustGet(QueryParam{
		Withs: map[string]With{"category": {Query: QueryParam{
			Select: []interface{}{"id", "name"},
		}}},
	})
	if len(pets) != 4 {
		t.Fatal("pets length not equal 4")
	}
	res := pets[0].Dot()
	assert.Equal(t, res.Get("category_id"), res.Get("category.id"))

	// Multiple withs
	pets = Select("pet").MustGet(QueryParam{
		Withs: map[string]With{"category": {}, "owner": {}},
	})
	if len(pets) != 4 {
		t.Fatal("pets length not equal 4")
	}
	res = pets[0].Dot()
	assert.NotEmpty(t, res.Get("category.name"))
	assert.NotEmpty(t, res.Get("owner.name"))
	assert.Equal(t, res.Get("category_id"), res.Get("category.id"))
	assert.Equal(t, res.Get("owner_id"), res.Get("owner.id"))

	// Multiple withs the same model
	pets = Select("pet").MustGet(QueryParam{
		Withs: map[string]With{"category": {}, "owner": {}, "doctor": {}},
	})
	if len(pets) != 4 {
		t.Fatal("pets length not equal 4")
	}
	res = pets[0].Dot()
	assert.NotEmpty(t, res.Get("category.name"))
	assert.NotEmpty(t, res.Get("owner.name"))
	assert.NotEmpty(t, res.Get("doctor.name"))
	assert.Equal(t, res.Get("category_id"), res.Get("category.id"))
	assert.Equal(t, res.Get("owner_id"), res.Get("owner.id"))
	assert.Equal(t, res.Get("doctor_id"), res.Get("doctor.id"))
}

func TestModelMustPaginate(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	pets := Select("pet").MustPaginate(QueryParam{}, 1, 2)
	assert.Equal(t, pets["total"], 4)
	assert.Equal(t, pets["page"], 1)
	assert.Equal(t, pets["pagecnt"], 2)
	assert.Equal(t, pets["pagesize"], 2)
	assert.Equal(t, pets["next"], 2)
	assert.Equal(t, pets["prev"], -1)

	rows := pets["data"].([]maps.MapStr)
	if len(rows) != 2 {
		t.Fatal("pets length not equal 4")
	}
	res := rows[0].Dot()
	assert.Equal(t, res.Get("name"), "Tommy")

}

func TestModelMustPaginateWiths(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	pets := Select("pet").MustPaginate(QueryParam{
		Withs: map[string]With{"category": {Query: QueryParam{
			Select: []interface{}{"id", "name"},
		}}},
	}, 1, 2)
	assert.Equal(t, pets["total"], 4)
	assert.Equal(t, pets["page"], 1)
	assert.Equal(t, pets["pagecnt"], 2)
	assert.Equal(t, pets["pagesize"], 2)
	assert.Equal(t, pets["next"], 2)
	assert.Equal(t, pets["prev"], -1)

	rows := pets["data"].([]maps.MapStr)
	if len(rows) != 2 {
		t.Fatal("pets length not equal 4")
	}
	res := rows[0].Dot()
	assert.Equal(t, res.Get("name"), "Tommy")

	// Multiple withs
	pets = Select("pet").MustPaginate(QueryParam{
		Withs: map[string]With{"category": {}, "owner": {}},
	}, 1, 2)
	rows = pets["data"].([]maps.MapStr)
	if len(rows) != 2 {
		t.Fatal("pets length not equal 4")
	}
	res = rows[0].Dot()
	assert.NotEmpty(t, res.Get("category.name"))
	assert.NotEmpty(t, res.Get("owner.name"))
	assert.Equal(t, res.Get("category_id"), res.Get("category.id"))
	assert.Equal(t, res.Get("owner_id"), res.Get("owner.id"))

	// Multiple withs the same model
	pets = Select("pet").MustPaginate(QueryParam{
		Withs: map[string]With{"category": {}, "owner": {}, "doctor": {}},
	}, 1, 2)
	rows = pets["data"].([]maps.MapStr)
	if len(rows) != 2 {
		t.Fatal("pets length not equal 4")
	}
	res = rows[0].Dot()
	assert.NotEmpty(t, res.Get("category.name"))
	assert.NotEmpty(t, res.Get("owner.name"))
	assert.NotEmpty(t, res.Get("doctor.name"))
	assert.Equal(t, res.Get("category_id"), res.Get("category.id"))
	assert.Equal(t, res.Get("owner_id"), res.Get("owner.id"))
	assert.Equal(t, res.Get("doctor_id"), res.Get("doctor.id"))
}

// prepare test suit
func prepare(t *testing.T) {
	dbconnect()
	root := os.Getenv("GOU_TEST_APPLICATION")
	aesKey := os.Getenv("GOU_TEST_AES_KEY")

	mods := map[string]string{
		"user":     filepath.Join("models", "user.mod.yao"),
		"pet":      filepath.Join("models", "pet.mod.yao"),
		"tag":      filepath.Join("models", "tag.mod.yao"),
		"category": filepath.Join("models", "category.mod.yao"),
		"user.pet": filepath.Join("models", "user", "pet.mod.yao"),
		"pet.tag":  filepath.Join("models", "pet", "tag.mod.yao"),
	}

	WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, aesKey)), "AES")
	WithCrypt([]byte(`{}`), "PASSWORD")

	// Load app
	app, err := application.OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	// load mods
	for id, file := range mods {
		_, err := Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Migrate
	for id := range mods {
		mod := Select(id)
		err := mod.Migrate(true)
		if err != nil {
			t.Fatal(err)
		}
	}

}

func prepareTestData(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	file := filepath.Join(root, "data", "tests.json")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)

	}

	data := map[string][]map[string]interface{}{}
	err = jsoniter.Unmarshal(raw, &data)
	if err != nil {
		t.Fatal(err)
	}

	for id, rows := range data {
		mod := Select(id)
		_, err := mod.EachSave(rows)
		if err != nil {
			t.Fatal(err)
		}
	}

}

func check(t *testing.T) {
	keys := map[string]bool{}
	for id := range Models {
		keys[id] = true
	}
	mods := []string{"user", "pet", "tag", "category", "user.pet", "pet.tag"}
	for _, id := range mods {
		_, has := keys[id]
		assert.True(t, has)
	}
}

func clean() {
	dbclose()
}

func dbclose() {
	if capsule.Global != nil {
		capsule.Global.Connections.Range(func(key, value any) bool {
			if conn, ok := value.(*capsule.Connection); ok {
				conn.Close()
			}
			return true
		})
	}
}

func dbconnect() {

	TestDriver := os.Getenv("GOU_TEST_DB_DRIVER")
	TestDSN := os.Getenv("GOU_TEST_DSN")
	TestAESKey := os.Getenv("GOU_TEST_AES_KEY")

	// connect db
	switch TestDriver {
	case "sqlite3":
		capsule.AddConn("primary", "sqlite3", TestDSN).SetAsGlobal()
		break
	default:
		capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
		break
	}

	// query engine
	query.Register("query-test", &gou.Query{
		Query: capsule.Query(),
		GetTableName: func(s string) string {
			if mod, has := Models[s]; has {
				return mod.MetaData.Table.Name
			}
			exception.New("[query] %s not found", 404).Throw()
			return s
		},
		AESKey: TestAESKey,
	})
}

// func TestModelMustPaginateWithsWheresOrder(t *testing.T) {
// 	user := Select("user").MustPaginate(QueryParam{
// 		Orders: []QueryOrder{
// 			{
// 				Column: "id",
// 				Option: "desc",
// 			},
// 		},
// 		Wheres: []QueryWhere{
// 			{
// 				Wheres: []QueryWhere{
// 					{
// 						Column: "mobile",
// 						Value:  "13900002222",
// 					}, {
// 						Column: "mobile",
// 						Method: "orwhere",
// 						Value:  "13900001111",
// 					},
// 				},
// 			},
// 		},
// 		Withs: map[string]With{
// 			"manu":      {},
// 			"addresses": {},
// 			"mother":    {},
// 		},
// 	}, 1, 2)
// 	userDot := user.Dot()
// 	assert.Equal(t, userDot.Get("total"), 2)
// 	assert.Equal(t, userDot.Get("next"), -1)
// 	assert.Equal(t, userDot.Get("page"), 1)
// 	assert.Equal(t, userDot.Get("data.1.id"), int64(1))
// 	assert.Equal(t, userDot.Get("data.1.manu.name"), "北京云道天成科技有限公司")
// 	assert.Equal(t, userDot.Get("data.1.mother.extra.sex"), "女")
// 	assert.Equal(t, userDot.Get("data.1.extra.sex"), "男")
// 	assert.Equal(t, userDot.Get("data.1.addresses.0.location"), "银海星月9号楼9单元9层1024室")

// }

// func TestModelMustCreate(t *testing.T) {
// 	user := Select("user")
// 	id := user.MustCreate(maps.MapStr{
// 		"name":     "用户创建",
// 		"manu_id":  2,
// 		"type":     "user",
// 		"idcard":   "23082619820207006X",
// 		"mobile":   "13900004444",
// 		"password": "qV@uT1DI",
// 		"key":      "XZ12MiPp",
// 		"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
// 		"status":   "enabled",
// 		"extra":    maps.MapStr{"sex": "女"},
// 	})

// 	// utils.Dump(id)

// 	row := user.MustFind(id, QueryParam{})

// 	// 清空数据
// 	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()

// 	assert.Equal(t, row.Get("name"), "用户创建")
// 	assert.Equal(t, row.Dot().Get("extra.sex"), "女")

// }

// func TestModelMustSaveNew(t *testing.T) {
// 	user := Select("user")
// 	id := user.MustSave(maps.MapStr{
// 		"name":     "用户创建",
// 		"manu_id":  2,
// 		"type":     "user",
// 		"idcard":   "23082619820207006X",
// 		"mobile":   "13900004444",
// 		"password": "qV@uT1DI",
// 		"key":      "XZ12MiPp",
// 		"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
// 		"status":   "enabled",
// 		"extra":    maps.MapStr{"sex": "女"},
// 	})

// 	row := user.MustFind(id, QueryParam{})

// 	// 清空数据
// 	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
// 	assert.Equal(t, row.Get("name"), "用户创建")
// 	assert.Equal(t, row.Dot().Get("extra.sex"), "女")

// }

// func TestModelWithStringPrimary(t *testing.T) {
// 	store := Select("store")

// 	assert.Equal(t, 2, len(store.MustGet(QueryParam{})))

// 	key := "key-test"
// 	store.MustCreate(maps.MapStr{"key": key, "data": []string{"value-test"}})
// 	row := store.MustFind(key, QueryParam{})
// 	assert.Equal(t, []interface{}{"value-test"}, row.Get("data"))
// 	assert.Equal(t, 3, len(store.MustGet(QueryParam{})))

// 	keyReturn := store.MustSave(maps.MapStr{"key": key, "data": []string{"value-test"}})
// 	assert.Equal(t, key, keyReturn)
// 	assert.Equal(t, 3, len(store.MustGet(QueryParam{})))

// 	store.MustDelete(key)
// 	assert.Equal(t, 2, len(store.MustGet(QueryParam{})))

// 	res, err := store.EachSave([]map[string]interface{}{
// 		{"key": key, "data": []string{"value-test"}},
// 		{"key": "key-1", "data": []string{"value-key-1"}},
// 	})

// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	assert.Equal(t, 3, len(store.MustGet(QueryParam{})))
// 	assert.Equal(t, 2, len(res))
// 	capsule.Query().Table(store.MetaData.Table.Name).Where("key", key).Delete()

// }

// func TestModelMustSaveUpdate(t *testing.T) {
// 	user := Select("user")
// 	id := user.MustSave(maps.MapStr{
// 		"id":      1,
// 		"balance": 200,
// 	})

// 	row := user.MustFind(id, QueryParam{})

// 	// 恢复数据
// 	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Update(maps.MapStr{"balance": 0})
// 	assert.Equal(t, any.Of(row.Get("balance")).CInt(), 200)
// }

// func TestModelMustUpdate(t *testing.T) {
// 	user := Select("user")
// 	user.MustUpdate(1, maps.MapStr{"balance": 200})

// 	row := user.MustFind(1, QueryParam{})

// 	// 恢复数据
// 	capsule.Query().Table(user.MetaData.Table.Name).Where("id", 1).Update(maps.MapStr{"balance": 0})
// 	assert.Equal(t, any.Of(row.Get("balance")).CInt(), 200)
// }

// func TestModelMustUpdateWhere(t *testing.T) {
// 	user := Select("user")
// 	effect := user.MustUpdateWhere(
// 		QueryParam{
// 			Wheres: []QueryWhere{
// 				{
// 					Column: "id",
// 					Value:  1,
// 				},
// 			},
// 		},
// 		maps.MapStr{
// 			"balance": 200,
// 		})

// 	row := user.MustFind(1, QueryParam{})

// 	// 恢复数据
// 	capsule.Query().Table(user.MetaData.Table.Name).Where("id", 1).Update(maps.MapStr{"balance": 0})
// 	assert.Equal(t, any.Of(row.Get("balance")).CInt(), 200)
// 	assert.Equal(t, effect, 1)
// }

// func TestModelMustDeleteSoft(t *testing.T) {
// 	user := Select("user")
// 	id := user.MustSave(maps.MapStr{
// 		"name":     "用户创建",
// 		"manu_id":  2,
// 		"type":     "user",
// 		"idcard":   "23082619820207006X",
// 		"mobile":   "13900004444",
// 		"password": "qV@uT1DI",
// 		"key":      "XZ12MiPp",
// 		"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
// 		"status":   "enabled",
// 		"extra":    maps.MapStr{"sex": "女"},
// 	})
// 	user.MustDelete(id)
// 	row, _ := user.Find(id, QueryParam{})

// 	// 清空数据
// 	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
// 	assert.Nil(t, row)
// }

// func TestModelMustDestroy(t *testing.T) {
// 	user := Select("user")
// 	id := user.MustSave(maps.MapStr{
// 		"name":     "用户创建",
// 		"manu_id":  2,
// 		"type":     "user",
// 		"idcard":   "23082619820207006X",
// 		"mobile":   "13900004444",
// 		"password": "qV@uT1DI",
// 		"key":      "XZ12MiPp",
// 		"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
// 		"status":   "enabled",
// 		"extra":    maps.MapStr{"sex": "女"},
// 	})
// 	user.MustDestroy(id)

// 	row, err := capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).First()
// 	assert.True(t, row.IsEmpty())
// 	assert.Nil(t, err)
// }

// func TestModelMustInsert(t *testing.T) {
// 	columns := []string{"user_id", "province", "city", "location"}
// 	rows := [][]interface{}{
// 		{4, "北京市", "丰台区", "银海星月9号楼9单元9层1024室"},
// 		{4, "天津市", "塘沽区", "益海星云7号楼3单元1003室"},
// 	}
// 	address := Select("address")
// 	err := address.Insert(columns, rows)
// 	assert.Nil(t, err)
// 	capsule.Query().Table(address.MetaData.Table.Name).Where("user_id", 4).Delete()
// }

// func TestModelMustInsertError(t *testing.T) {
// 	columns := []string{"user_id", "province", "city", "location"}
// 	rows := [][]interface{}{
// 		{4, "北京市", "丰台区", "银海星月9号楼9单元9层1024室"},
// 		{4, "天津市", "塘沽区", "益海星云7号楼3单元1003室", 5028},
// 		{4, "天津市", "塘沽区", "益海星云7号楼3单元1002室"},
// 	}
// 	address := Select("address")
// 	assert.Panics(t, func() {
// 		address.Insert(columns, rows)
// 	})
// }

// func TestModelMustDeleteWhere(t *testing.T) {
// 	columns := []string{"name", "manu_id", "type", "idcard", "mobile", "password", "key", "secret", "status"}
// 	rows := [][]interface{}{
// 		{"用户创建1", 5, "user", "23082619820207006X", "13900004444", "qV@uT1DI", "XZ12MiP1", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
// 		{"用户创建2", 5, "user", "33082619820207006X", "13900005555", "qV@uT1DI", "XZ12MiP2", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
// 		{"用户创建3", 5, "user", "43082619820207006X", "13900006666", "qV@uT1DI", "XZ12MiP3", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
// 	}

// 	user := Select("user")
// 	user.Insert(columns, rows)
// 	param := QueryParam{Wheres: []QueryWhere{
// 		{
// 			Column: "manu_id",
// 			Value:  5,
// 		},
// 	}}
// 	effect := user.MustDeleteWhere(param)

// 	// 清理数据
// 	capsule.Query().Table(user.MetaData.Table.Name).Where("name", "like", "用户创建%").Delete()
// 	assert.Equal(t, effect, 3)
// }

// func TestModelMustDestroyWhere(t *testing.T) {
// 	columns := []string{"name", "manu_id", "type", "idcard", "mobile", "password", "key", "secret", "status"}
// 	rows := [][]interface{}{
// 		{"用户创建1", 5, "user", "23082619820207006X", "13900004444", "qV@uT1DI", "XZ12MiP1", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
// 		{"用户创建2", 5, "user", "33082619820207006X", "13900005555", "qV@uT1DI", "XZ12MiP2", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
// 		{"用户创建3", 5, "user", "43082619820207006X", "13900006666", "qV@uT1DI", "XZ12MiP3", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
// 	}

// 	user := Select("user")
// 	user.Insert(columns, rows)
// 	param := QueryParam{Wheres: []QueryWhere{
// 		{
// 			Column: "manu_id",
// 			Value:  5,
// 		},
// 	}}
// 	effect := user.MustDestroyWhere(param)

// 	// 清理数据
// 	assert.Equal(t, effect, 3)
// }

// func TestModelMustEachSave(t *testing.T) {
// 	user := Select("user")
// 	ids := user.MustEachSave([]map[string]interface{}{
// 		{"id": 1, "balance": 200},
// 		{
// 			"name":     "用户创建",
// 			"manu_id":  2,
// 			"type":     "user",
// 			"idcard":   "23082619820207006X",
// 			"mobile":   "13900004444",
// 			"password": "qV@uT1DI",
// 			"key":      "XZ12MiPp",
// 			"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
// 			"status":   "enabled",
// 			"extra":    maps.MapStr{"sex": "女"},
// 		},
// 	})

// 	assert.Equal(t, 2, len(ids))
// 	row := user.MustFind(1, QueryParam{})

// 	// 恢复数据
// 	capsule.Query().Table(user.MetaData.Table.Name).Where("id", 1).Update(maps.MapStr{"balance": 0})
// 	capsule.Query().Table(user.MetaData.Table.Name).Where("id", ids[1]).Delete()
// 	assert.Equal(t, any.Of(row.Get("balance")).CInt(), 200)
// }

// func TestModelMustEachSaveWithIndex(t *testing.T) {
// 	user := Select("user")
// 	ids := user.MustEachSave([]map[string]interface{}{
// 		{
// 			"name":     "用户创建",
// 			"manu_id":  2,
// 			"type":     "user",
// 			"idcard":   "23082619820207006X",
// 			"mobile":   "13900004444",
// 			"password": "qV@uT1DI",
// 			"key":      "XZ12MiPp",
// 			"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
// 			"status":   "enabled",
// 			"extra":    maps.MapStr{"sex": "女"},
// 		}, {
// 			"name":     "用户创建2",
// 			"manu_id":  2,
// 			"type":     "user",
// 			"idcard":   "23012619820207006X",
// 			"mobile":   "13900004443",
// 			"password": "qV@uT1DI",
// 			"key":      "XZ12MiPM",
// 			"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
// 			"status":   "enabled",
// 			"extra":    maps.MapStr{"sex": "男"},
// 		},
// 	}, maps.MapStr{"balance": "$index"})

// 	assert.Equal(t, 2, len(ids))
// 	row := user.MustFind(ids[0], QueryParam{})
// 	row1 := user.MustFind(ids[1], QueryParam{})

// 	// 恢复数据
// 	capsule.Query().Table(user.MetaData.Table.Name).WhereIn("id", ids).Delete()
// 	assert.Equal(t, any.Of(row.Get("balance")).CInt(), 0)
// 	assert.Equal(t, any.Of(row1.Get("balance")).CInt(), 1)
// }

// func TestModelExportImport(t *testing.T) {
// 	columns := []string{"name", "manu_id", "type", "idcard", "mobile", "password", "key", "secret", "status", "updated_at"}
// 	rows := [][]interface{}{
// 		{"用户创建1", 5, "user", "23082619820207006X", "13900004444", "qV@uT1DI", "XZ12MiP1", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled", "2022-06-13T10:09:01+08:00"},
// 		{"用户创建2", 5, "user", "33082619820207006X", "13900005555", "qV@uT1DI", "XZ12MiP2", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled", "2022-06-13 10:09:01"},
// 		{"用户创建3", 5, "user", "43082619820207006X", "13900006666", "qV@uT1DI", "XZ12MiP3", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled", "2022-06-13T10:09:01Z"},
// 	}

// 	user := Select("uimport")
// 	err := user.Migrate(true)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	err = user.Insert(columns, rows)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer capsule.Query().Table(user.MetaData.Table.Name).Where("name", "like", "用户创建%").MustDelete()

// 	files, err := user.Export(2, func(curr, total int) {
// 		fmt.Printf("Export: %d/%d\n", curr, total)
// 	})

// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	assert.Greater(t, len(files), 0)
// 	capsule.Query().Table(user.MetaData.Table.Name).MustDelete()
// 	for _, file := range files {
// 		err = user.Import(file)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 	}

// 	res := user.MustGet(QueryParam{
// 		Wheres: []QueryWhere{
// 			{Column: "name", Value: "用户创建", OP: "match"},
// 		},
// 	})
// 	assert.Equal(t, 3, len(res))
// }

// func TestModelLang(t *testing.T) {
// 	root := os.Getenv("GOU_TEST_APP_ROOT")
// 	rootLang := filepath.Join(root, "langs")
// 	err := lang.Load(rootLang)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	modelFile := filepath.Join(root, "models", "demo.mod.json")
// 	mod, err := Load(modelFile, "demo")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	dict := lang.Pick("zh-cn")
// 	dict.ReplaceAll([]string{fmt.Sprintf("model.%s", mod.ID)}, &mod)
// 	assert.Equal(t, mod.MetaData.Name, "演示")
// 	assert.Equal(t, mod.Columns["action"].Label, "动作")

// 	// Reload
// 	mod.Reload()
// 	dict = lang.Pick("zh-hk")
// 	dict.ReplaceAll([]string{fmt.Sprintf("model.%s", mod.ID)}, &mod)
// 	assert.Equal(t, mod.MetaData.Name, "演示")
// 	assert.Equal(t, mod.Columns["action"].Label, "動作")

// 	// Reload
// 	mod.Reload()
// 	dict = lang.Pick("zh-cn")
// 	new, err := dict.ReplaceClone([]string{fmt.Sprintf("model.%s", mod.ID)}, mod)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	newMod := new.(*Model)
// 	assert.Equal(t, newMod.MetaData.Name, "演示")
// 	assert.Equal(t, newMod.Columns["action"].Label, "动作")
// 	assert.Equal(t, mod.MetaData.Name, "::Demo")
// 	assert.Equal(t, mod.Columns["action"].Label, "::Action")

// 	mod.Reload()
// 	dict = lang.Pick("zh-hk")
// 	new, err = dict.ReplaceClone([]string{fmt.Sprintf("model.%s", mod.ID)}, mod)
// 	newMod = new.(*Model)
// 	assert.Equal(t, newMod.MetaData.Name, "演示")
// 	assert.Equal(t, newMod.Columns["action"].Label, "動作")
// 	assert.Equal(t, mod.MetaData.Name, "::Demo")
// 	assert.Equal(t, mod.Columns["action"].Label, "::Action")

// }
