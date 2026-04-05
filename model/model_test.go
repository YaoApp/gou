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

func TestGetMetaData(t *testing.T) {
	prepare(t)
	defer clean()
	meta := GetMetaData("user")
	assert.Equal(t, meta.Name, "User")
	assert.Panics(t, func() {
		GetMetaData("not-found")
	})
}

func TestRead(t *testing.T) {
	prepare(t)
	defer clean()
	source := Read("user")
	assert.NotNil(t, source)
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

func TestModelCount(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	// Test count all pets
	count := Select("pet").MustCount(QueryParam{})
	assert.Equal(t, 4, count)

	// Test count with conditions
	count = Select("pet").MustCount(QueryParam{
		Wheres: []QueryWhere{
			{Column: "name", Value: "Tommy", OP: "eq"},
		},
	})
	assert.Equal(t, 1, count)

	// Test count users
	userCount := Select("user").MustCount(QueryParam{})
	assert.Equal(t, 2, userCount)

	// Test count with multiple conditions
	count = Select("pet").MustCount(QueryParam{
		Wheres: []QueryWhere{
			{Column: "category_id", Value: 1, OP: "eq"},
		},
	})
	assert.GreaterOrEqual(t, count, 0)
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
		"store":    filepath.Join("models", "store.mod.yao"),
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
	case "postgres":
		capsule.AddConn("primary", "postgres", TestDSN).SetAsGlobal()
	default:
		capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
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

func TestModelMustPaginateWithsWheresOrder(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	res := Select("pet").MustPaginate(QueryParam{
		Orders: []QueryOrder{{Column: "id", Option: "desc"}},
		Withs: map[string]With{
			"category": {},
			"owner":    {},
		},
	}, 1, 2)
	dot := res.Dot()
	assert.Equal(t, 4, dot.Get("total"))
	assert.Equal(t, 1, dot.Get("page"))
	assert.Equal(t, 2, dot.Get("pagesize"))
	data := dot.Get("data")
	assert.Equal(t, 2, len(data.([]maps.MapStr)))
}

func TestModelMustCreate(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustCreate(maps.MapStr{
		"name":   "用户创建",
		"type":   "admin",
		"status": "enabled",
		"extra":  maps.MapStr{"sex": "女"},
	})

	row := user.MustFind(id, QueryParam{})
	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()

	assert.Equal(t, "用户创建", row.Get("name"))
	assert.Equal(t, "女", row.Dot().Get("extra.sex"))
}

func TestModelMustSaveNew(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name":   "用户保存",
		"type":   "admin",
		"status": "enabled",
		"extra":  maps.MapStr{"sex": "女"},
	})

	row := user.MustFind(id, QueryParam{})
	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()

	assert.Equal(t, "用户保存", row.Get("name"))
	assert.Equal(t, "女", row.Dot().Get("extra.sex"))
}

func TestModelWithStringPrimary(t *testing.T) {
	prepare(t)
	defer clean()

	store := Select("store")
	key := "key-test"

	err := capsule.Query().Table(store.MetaData.Table.Name).Insert(
		maps.MapStr{"key": key, "data": `["value-test"]`},
	)
	assert.Nil(t, err)

	row := store.MustFind(key, QueryParam{})
	assert.Equal(t, []interface{}{"value-test"}, row.Get("data"))
	assert.Equal(t, 1, len(store.MustGet(QueryParam{})))

	store.MustSave(maps.MapStr{"key": key, "data": []string{"value-updated"}})
	assert.Equal(t, 1, len(store.MustGet(QueryParam{})))

	row2 := store.MustFind(key, QueryParam{})
	assert.Equal(t, []interface{}{"value-updated"}, row2.Get("data"))

	store.MustDestroy(key)
	assert.Equal(t, 0, len(store.MustGet(QueryParam{})))
}

func TestModelMustSaveUpdate(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name":   "SaveUpdate测试",
		"type":   "admin",
		"status": "enabled",
	})

	user.MustSave(maps.MapStr{
		"id":     id,
		"status": "disabled",
	})

	row := user.MustFind(id, QueryParam{})
	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
	assert.Equal(t, "disabled", row.Get("status"))
}

func TestModelMustUpdate(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustCreate(maps.MapStr{
		"name":   "Update测试",
		"type":   "admin",
		"status": "enabled",
	})

	user.MustUpdate(id, maps.MapStr{"status": "disabled"})
	row := user.MustFind(id, QueryParam{})

	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
	assert.Equal(t, "disabled", row.Get("status"))
}

func TestModelMustUpdateWhere(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustCreate(maps.MapStr{
		"name":   "UpdateWhere测试",
		"type":   "admin",
		"status": "enabled",
	})

	effect := user.MustUpdateWhere(
		QueryParam{
			Wheres: []QueryWhere{
				{Column: "id", Value: id},
			},
		},
		maps.MapStr{"status": "disabled"})

	row := user.MustFind(id, QueryParam{})
	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
	assert.Equal(t, "disabled", row.Get("status"))
	assert.Equal(t, 1, effect)
}

func TestModelMustDeleteSoft(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name":   "软删除测试",
		"type":   "admin",
		"status": "enabled",
	})
	user.MustDelete(id)
	row, _ := user.Find(id, QueryParam{})

	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
	assert.Nil(t, row)
}

func TestModelMustDestroy(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name":   "硬删除测试",
		"type":   "admin",
		"status": "enabled",
	})
	user.MustDestroy(id)

	row, err := capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).First()
	assert.True(t, row.IsEmpty())
	assert.Nil(t, err)
}

func TestModelMustInsert(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	columns := []string{"name", "type", "status"}
	rows := [][]interface{}{
		{"批量插入1", "admin", "enabled"},
		{"批量插入2", "staff", "enabled"},
	}
	err := user.Insert(columns, rows)
	assert.Nil(t, err)

	res := user.MustGet(QueryParam{
		Wheres: []QueryWhere{{Column: "name", Value: "批量插入", OP: "match"}},
	})
	capsule.Query().Table(user.MetaData.Table.Name).Where("name", "like", "批量插入%").Delete()
	assert.Equal(t, 2, len(res))
}

func TestModelMustDeleteWhere(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	columns := []string{"name", "type", "status"}
	rows := [][]interface{}{
		{"批量软删1", "admin", "enabled"},
		{"批量软删2", "admin", "enabled"},
		{"批量软删3", "admin", "enabled"},
	}
	user.Insert(columns, rows)

	effect := user.MustDeleteWhere(QueryParam{
		Wheres: []QueryWhere{{Column: "name", Value: "批量软删", OP: "match"}},
	})

	capsule.Query().Table(user.MetaData.Table.Name).Where("name", "like", "批量软删%").Delete()
	assert.Equal(t, 3, effect)
}

func TestModelMustDestroyWhere(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	columns := []string{"name", "type", "status"}
	rows := [][]interface{}{
		{"批量硬删1", "admin", "enabled"},
		{"批量硬删2", "admin", "enabled"},
		{"批量硬删3", "admin", "enabled"},
	}
	user.Insert(columns, rows)

	effect := user.MustDestroyWhere(QueryParam{
		Wheres: []QueryWhere{{Column: "name", Value: "批量硬删", OP: "match"}},
	})
	assert.Equal(t, 3, effect)
}

func TestModelMustEachSave(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	ids, err := user.EachSave([]map[string]interface{}{
		{
			"name":   "EachSave新建1",
			"type":   "admin",
			"status": "enabled",
		},
		{
			"name":   "EachSave新建2",
			"type":   "staff",
			"status": "enabled",
			"extra":  maps.MapStr{"sex": "女"},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(ids))

	row := user.MustFind(ids[1], QueryParam{})
	assert.Equal(t, "EachSave新建2", row.Get("name"))

	for _, id := range ids {
		capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
	}
}

func TestModelAESCrypt(t *testing.T) {
	if os.Getenv("GOU_TEST_DB_DRIVER") == "sqlite3" {
		t.Skip("SQLite3 does not support AES encryption")
	}
	prepare(t)
	defer clean()

	user := Select("user")
	mobile := "13912345678"
	id := user.MustSave(maps.MapStr{
		"name":   "AES加密测试",
		"type":   "admin",
		"status": "enabled",
		"mobile": mobile,
	})

	row := user.MustFind(id, QueryParam{})
	assert.Equal(t, mobile, row.Get("mobile"))

	rawRow, err := capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).First()
	assert.Nil(t, err)
	assert.NotEqual(t, mobile, rawRow.Get("mobile"),
		"raw DB value should be encrypted, not plaintext")

	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
}

func TestModelEachSaveWithExisting(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name":   "EachSave已有",
		"type":   "admin",
		"status": "enabled",
	})

	ids, err := user.EachSave([]map[string]interface{}{
		{"id": id, "name": "EachSave已更新"},
		{"name": "EachSave新增", "type": "staff", "status": "enabled"},
	})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(ids))

	row := user.MustFind(id, QueryParam{})
	assert.Equal(t, "EachSave已更新", row.Get("name"))

	for _, rid := range ids {
		capsule.Query().Table(user.MetaData.Table.Name).Where("id", rid).Delete()
	}
}

func TestModelEachSaveError(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()
		user.EachSave([]map[string]interface{}{
			{"name": "ValidRow", "type": "admin", "status": "enabled"},
			{"name": "InvalidType", "type": "INVALID_TYPE", "status": "enabled"},
		})
	}()
	assert.True(t, didPanic, "EachSave with invalid data should panic via exception.Throw")
	capsule.Query().Table(user.MetaData.Table.Name).Where("name", "ValidRow").Delete()
}

func TestModelUpsert(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name": "Upsert测试", "type": "admin", "status": "enabled",
	})

	row := maps.MapStr{"id": id, "name": "Upsert测试", "type": "admin", "status": "disabled"}
	affected, err := user.Upsert(row, []interface{}{"id"}, []interface{}{"status"})
	assert.Nil(t, err)
	assert.True(t, affected > 0)

	found := user.MustFind(id, QueryParam{})
	assert.Equal(t, "disabled", found.Get("status"))

	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
}

func TestModelUpdate(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name": "Update测试", "type": "admin", "status": "enabled",
	})

	err := user.Update(id, maps.MapStr{"status": "disabled"})
	assert.Nil(t, err)

	row := user.MustFind(id, QueryParam{})
	assert.Equal(t, "disabled", row.Get("status"))

	user.MustUpdate(id, maps.MapStr{"status": "enabled"})
	row2 := user.MustFind(id, QueryParam{})
	assert.Equal(t, "enabled", row2.Get("status"))

	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
}

func TestModelCreate(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id, err := user.Create(maps.MapStr{
		"name": "Create测试", "type": "admin", "status": "enabled",
	})
	assert.Nil(t, err)
	assert.True(t, id > 0)

	row := user.MustFind(id, QueryParam{})
	assert.Equal(t, "Create测试", row.Get("name"))

	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
}

func TestModelAESCryptWhereFilter(t *testing.T) {
	if os.Getenv("GOU_TEST_DB_DRIVER") == "sqlite3" {
		t.Skip("SQLite3 does not support AES encryption")
	}
	prepare(t)
	defer clean()

	user := Select("user")
	mobile := "13800001111"
	id := user.MustSave(maps.MapStr{
		"name": "AES查询", "type": "admin", "status": "enabled", "mobile": mobile,
	})

	res := user.MustGet(QueryParam{
		Wheres: []QueryWhere{{Column: "mobile", Value: mobile}},
	})
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "AES查询", res[0].Get("name"))

	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
}

func TestModelFliterOut(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name": "FliterOut测试", "type": "admin", "status": "enabled",
		"extra": maps.MapStr{"level": 5},
	})

	row := user.MustFind(id, QueryParam{})
	assert.Equal(t, "FliterOut测试", row.Get("name"))
	dot := row.Dot()
	assert.Equal(t, float64(5), dot.Get("extra.level"))

	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
}

func TestEncryptorPasswordEncodeDecode(t *testing.T) {
	enc := &Encryptor{Key: "testkey", Name: "PASSWORD"}
	Encryptors["PASSWORD"] = enc
	pwd := &EncryptorPassword{}
	pwd.Set(enc)

	hash, err := pwd.Encode("mypassword")
	assert.Nil(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "mypassword", hash)

	decoded, err := pwd.Decode(hash)
	assert.Nil(t, err)
	assert.Equal(t, hash, decoded)

	assert.True(t, pwd.Validate(hash, "mypassword"))
	assert.False(t, pwd.Validate(hash, "wrongpassword"))
}

func TestEncryptorAESEncodeDecode(t *testing.T) {
	enc := &Encryptor{Key: "aes-test-key", Name: "AES"}
	Encryptors["AES"] = enc
	aes := &EncryptorAES{}
	aes.Set(enc)

	encoded, err := aes.Encode("hello")
	assert.Nil(t, err)
	assert.Contains(t, encoded, "HEX(AES_ENCRYPT(")

	decoded, err := aes.Decode("mobile")
	assert.Nil(t, err)
	assert.Contains(t, decoded, "AES_DECRYPT(UNHEX(")

	decodedDot, err := aes.Decode("user.mobile")
	assert.Nil(t, err)
	assert.Contains(t, decodedDot, "`user`.`mobile`")

	assert.False(t, aes.Validate("abc", "abc"))
}

func TestEncryptorPGCryptoEncodeDecode(t *testing.T) {
	enc := &Encryptor{Key: "pg-test-key", Name: "AES"}
	Encryptors["AES"] = enc
	pg := &EncryptorPGCrypto{}
	pg.Set(enc)

	encoded, err := pg.Encode("hello")
	assert.Nil(t, err)
	assert.Contains(t, encoded, "pgp_sym_encrypt")
	assert.Contains(t, encoded, "pg-test-key")

	decoded, err := pg.Decode("mobile")
	assert.Nil(t, err)
	assert.Contains(t, decoded, "pgp_sym_decrypt")
	assert.Contains(t, decoded, `"mobile"`)

	decodedDot, err := pg.Decode("user.mobile")
	assert.Nil(t, err)
	assert.Contains(t, decodedDot, `"user"."mobile"`)

	assert.False(t, pg.Validate("abc", "abc"))
}

func TestSelectCrypt(t *testing.T) {
	enc := &Encryptor{Key: "select-key", Name: "AES"}
	Encryptors["AES"] = enc

	icrypt, err := SelectCrypt("AES")
	assert.Nil(t, err)
	assert.NotNil(t, icrypt)

	_, err = SelectCrypt("NONEXISTENT")
	assert.NotNil(t, err)
}

func TestWithCrypt(t *testing.T) {
	data := []byte(`{"key": "wc-key"}`)
	enc, err := WithCrypt(data, "TestCrypt")
	assert.Nil(t, err)
	assert.Equal(t, "wc-key", enc.Key)
	assert.Equal(t, "TestCrypt", enc.Name)

	_, err = WithCrypt([]byte(`invalid`), "Bad")
	assert.NotNil(t, err)
}

func TestEncryptorSQLEscape(t *testing.T) {
	enc := &Encryptor{Key: "key'with\"quote", Name: "AES"}
	Encryptors["AES"] = enc

	aes := &EncryptorAES{}
	aes.Set(enc)
	encoded, _ := aes.Encode("val'ue")
	assert.Contains(t, encoded, "val''ue")
	assert.Contains(t, encoded, "key''with\"quote")
	assert.NotContains(t, encoded, "val'u")

	decoded, _ := aes.Decode("field")
	assert.Contains(t, decoded, "key''with")

	pg := &EncryptorPGCrypto{}
	pg.Set(enc)
	pgEncoded, _ := pg.Encode("val'ue")
	assert.Contains(t, pgEncoded, "val''ue")
	assert.Contains(t, pgEncoded, "key''with")

	pgDecoded, _ := pg.Decode("field")
	assert.Contains(t, pgDecoded, "key''with")
}

func TestModelMustInsertBatch(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	columns := []string{"name", "type", "status"}
	rows := [][]interface{}{
		{"MustInsert1", "admin", "enabled"},
		{"MustInsert2", "staff", "enabled"},
	}

	user.MustInsert(columns, rows)

	res := user.MustGet(QueryParam{
		Wheres: []QueryWhere{{Column: "name", Value: "MustInsert", OP: "match"}},
	})
	for _, r := range res {
		capsule.Query().Table(user.MetaData.Table.Name).Where("id", r.Get("id")).Delete()
	}
	assert.Equal(t, 2, len(res))
}

func TestModelMustEachSaveBatch(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	ids := user.MustEachSave([]map[string]interface{}{
		{"name": "MustEach1", "type": "admin", "status": "enabled"},
		{"name": "MustEach2", "type": "staff", "status": "enabled"},
	})
	assert.Equal(t, 2, len(ids))

	for _, id := range ids {
		capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
	}
}

func TestModelMustUpsert(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	user := Select("user")
	id := user.MustSave(maps.MapStr{
		"name": "MustUpsert", "type": "admin", "status": "enabled",
	})

	affected := user.MustUpsert(
		maps.MapStr{"id": id, "name": "MustUpsert", "type": "admin", "status": "disabled"},
		[]interface{}{"id"},
		[]interface{}{"status"},
	)
	assert.True(t, affected > 0)

	found := user.MustFind(id, QueryParam{})
	assert.Equal(t, "disabled", found.Get("status"))

	capsule.Query().Table(user.MetaData.Table.Name).Where("id", id).Delete()
}

func TestModelFliterOutDirect(t *testing.T) {
	prepare(t)
	defer clean()

	user := Select("user")
	row := maps.MapStrAny{
		"name":   "Direct",
		"extra":  `{"level":3}`,
		"status": "enabled",
	}
	user.FliterOut(row)
	assert.Equal(t, float64(3), row.Dot().Get("extra.level"))
}
