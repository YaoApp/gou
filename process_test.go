package gou

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/capsule"
)

func TestProcessPlugin(t *testing.T) {
	defer SelectPlugin("user").Client.Kill()
	res := NewProcess("plugins.user.Login", 1).Run().(maps.Map)
	res2 := NewProcess("plugins.user.Login", 2).Run().(maps.Map)
	assert.Equal(t, "login", res.Get("name"))
	assert.Equal(t, "login", res2.Get("name"))
	assert.Equal(t, 1, any.Of(res.Dot().Get("args.0")).CInt())
	assert.Equal(t, 2, any.Of(res2.Dot().Get("args.0")).CInt())
}

func TestProcessScript(t *testing.T) {
	res := NewProcess("scripts.app.test.hello", "world").Run()
	assert.Equal(t, "hello:world", res)

	res = NewProcess("scripts.app.test.helloProcess", "Max").Run()
	resdot := any.MapOf(res).MapStrAny.Dot()
	assert.Equal(t, "Max", resdot.Get("name"))
	assert.Equal(t, "login", resdot.Get("out.name"))
	assert.Equal(t, float64(1024), resdot.Get("out.args.0"))

	res = NewProcess("scripts.flows.script.rank.hello").Run()
	assert.Equal(t, "rank hello", res)
}

func TestProcessFlow(t *testing.T) {
	res := maps.Of(NewProcess("flows.latest", "%公司%", "bar").Run().(map[string]interface{}))
	assert.Equal(t, res.Get("params"), []interface{}{"%公司%", "bar"})
	assert.Equal(t, len(res.Dot().Get("data.users").([]maps.Map)), 3)
	assert.Equal(t, len(res.Dot().Get("data.manus").([]interface{})), 4)
	assert.Equal(t, res.Dot().Get("data.count.plugin"), "github")
}

func TestProcessFind(t *testing.T) {
	res := NewProcess("models.user.Find", 1, QueryParam{}).Run().(maps.MapStr)
	assert.Equal(t, 1, any.Of(res.Dot().Get("id")).CInt())
	assert.Equal(t, "男", res.Dot().Get("extra.sex"))
}

func TestProcessGet(t *testing.T) {
	rows := NewProcess("models.user.Get", QueryParam{Limit: 2}).Run().([]maps.MapStr)
	res := maps.Map{"data": rows}.Dot()
	assert.Equal(t, 2, len(rows))
	assert.Equal(t, 1, any.Of(res.Get("data.0.id")).CInt())
	assert.Equal(t, "男", res.Get("data.0.extra.sex"))
	assert.Equal(t, 2, any.Of(res.Get("data.1.id")).CInt())
	assert.Equal(t, "女", res.Get("data.1.extra.sex"))
}

func TestProcessSelectOption(t *testing.T) {
	rows := NewProcess("models.user.SelectOption").Run().([]maps.MapStr)
	res := maps.Map{"data": rows}.Dot()
	assert.Equal(t, 3, len(rows))
	assert.Equal(t, 1, any.Of(res.Get("data.0.id")).CInt())
	assert.Equal(t, "管理员", res.Get("data.0.name"))
	assert.Equal(t, 2, any.Of(res.Get("data.1.id")).CInt())
	assert.Equal(t, "员工", res.Get("data.1.name"))
	assert.Equal(t, 3, any.Of(res.Get("data.2.id")).CInt())
	assert.Equal(t, "用户", res.Get("data.2.name"))

	rows = NewProcess("models.user.SelectOption", "用户").Run().([]maps.MapStr)
	res = maps.Map{"data": rows}.Dot()
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, 3, any.Of(res.Get("data.0.id")).CInt())
	assert.Equal(t, "用户", res.Get("data.0.name"))

	rows = NewProcess("models.user.SelectOption", "", "name", "name").Run().([]maps.MapStr)
	res = maps.Map{"data": rows}.Dot()
	assert.Equal(t, 3, len(rows))
	assert.Equal(t, "管理员", res.Get("data.0.id"))
	assert.Equal(t, "管理员", res.Get("data.0.name"))
	assert.Equal(t, "员工", res.Get("data.1.id"))
	assert.Equal(t, "员工", res.Get("data.1.name"))
	assert.Equal(t, "用户", res.Get("data.2.id"))
	assert.Equal(t, "用户", res.Get("data.2.name"))

}

func TestProcessPaginate(t *testing.T) {
	res := NewProcess("models.user.Paginate", QueryParam{}, 1, 2).Run().(maps.MapStr).Dot()
	assert.Equal(t, 3, res.Get("total"))
	assert.Equal(t, 1, res.Get("page"))
	assert.Equal(t, 2, res.Get("pagesize"))
	assert.Equal(t, 2, res.Get("pagecnt"))
	assert.Equal(t, 2, res.Get("next"))
	assert.Equal(t, -1, res.Get("prev"))
	assert.Equal(t, 1, any.Of(res.Get("data.0.id")).CInt())
	assert.Equal(t, "男", res.Get("data.0.extra.sex"))
	assert.Equal(t, 2, any.Of(res.Get("data.1.id")).CInt())
	assert.Equal(t, "女", res.Get("data.1.extra.sex"))
}

func TestProcessCreate(t *testing.T) {
	row := maps.MapStr{
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
	}
	id := NewProcess("models.user.Create", row).Run().(int)
	assert.Greater(t, id, 0)

	// 清空数据
	capsule.Query().Table(Select("user").MetaData.Table.Name).Where("id", id).Delete()
}

func TestProcessUpdate(t *testing.T) {
	id := NewProcess("models.user.Update", 1, maps.MapStr{"balance": 200}).Run()
	assert.Nil(t, id)

	// 恢复数据
	capsule.Query().Table(Select("user").MetaData.Table.Name).Where("id", 1).Update(maps.MapStr{"balance": 0})
}

func TestProcessSave(t *testing.T) {
	row := maps.MapStr{
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
	}
	id := NewProcess("models.user.Save", row).Run().(int)
	assert.Greater(t, id, 0)

	// 清空数据
	capsule.Query().Table(Select("user").MetaData.Table.Name).Where("id", id).Delete()
}

func TestProcessDelete(t *testing.T) {
	row := maps.MapStr{
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
	}

	user := Select("user")
	id := user.MustSave(row)
	NewProcess("models.user.Delete", id).Run()

	// 清空数据
	capsule.Query().Table(Select("user").MetaData.Table.Name).Where("id", id).Delete()
}

func TestProcessDestroy(t *testing.T) {
	row := maps.MapStr{
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
	}

	user := Select("user")
	id := user.MustSave(row)
	NewProcess("models.user.Destroy", id).Run()

	// 清空数据
	capsule.Query().Table(Select("user").MetaData.Table.Name).Where("id", id).Delete()
}

func TestProcessInsert(t *testing.T) {

	content := `{
		"columns": ["user_id", "province", "city", "location"],
		"rows": [
			[4, "北京市", "丰台区", "银海星月9号楼9单元9层1024室"],
			[4, "天津市", "塘沽区", "益海星云7号楼3单元1003室"]
		]
	}`

	payload := map[string]interface{}{}
	err := jsoniter.UnmarshalFromString(content, &payload)
	if err != nil {
		assert.Nil(t, err)
		return
	}

	NewProcess("models.address.Insert", payload["columns"], payload["rows"]).Run()

	// 清理数据
	address := Select("address")
	capsule.Query().Table(address.MetaData.Table.Name).Where("user_id", 4).Delete()
}

func TestProcessUpdateWhere(t *testing.T) {
	effect := NewProcess("models.user.UpdateWhere",
		QueryParam{
			Limit: 1,
			Wheres: []QueryWhere{
				{
					Column: "id",
					Value:  1,
				},
			},
		},
		maps.MapStr{
			"balance": 200,
		},
	).Run().(int)

	user := Select("user")
	row := user.MustFind(1, QueryParam{})

	// 恢复数据
	capsule.Query().Table(user.MetaData.Table.Name).Where("id", 1).Update(maps.MapStr{"balance": 0})
	assert.Equal(t, any.Of(row.Get("balance")).CInt(), 200)
	assert.Equal(t, 1, effect)
}
func TestProcessDeleteWhere(t *testing.T) {

	columns := []string{"name", "manu_id", "type", "idcard", "mobile", "password", "key", "secret", "status"}
	rows := [][]interface{}{
		{"用户创建1", 5, "user", "23082619820207006X", "13900004444", "qV@uT1DI", "XZ12MiP1", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
		{"用户创建2", 5, "user", "33082619820207006X", "13900005555", "qV@uT1DI", "XZ12MiP2", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
		{"用户创建3", 5, "user", "43082619820207006X", "13900006666", "qV@uT1DI", "XZ12MiP3", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
	}

	user := Select("user")
	user.Insert(columns, rows)
	param := QueryParam{Wheres: []QueryWhere{
		{
			Column: "manu_id",
			Value:  5,
		},
	}}
	effect := NewProcess("models.user.DeleteWhere", param).Run().(int)

	// 清理数据
	capsule.Query().Table(user.MetaData.Table.Name).Where("name", "like", "用户创建%").Delete()
	assert.Equal(t, effect, 3)
}

func TestProcessDestroyWhere(t *testing.T) {

	columns := []string{"name", "manu_id", "type", "idcard", "mobile", "password", "key", "secret", "status"}
	rows := [][]interface{}{
		{"用户创建1", 5, "user", "23082619820207006X", "13900004444", "qV@uT1DI", "XZ12MiP1", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
		{"用户创建2", 5, "user", "33082619820207006X", "13900005555", "qV@uT1DI", "XZ12MiP2", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
		{"用户创建3", 5, "user", "43082619820207006X", "13900006666", "qV@uT1DI", "XZ12MiP3", "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN", "enabled"},
	}

	user := Select("user")
	user.Insert(columns, rows)
	param := QueryParam{Wheres: []QueryWhere{
		{
			Column: "manu_id",
			Value:  5,
		},
	}}
	effect := NewProcess("models.user.DestroyWhere", param).Run().(int)

	// 清理数据
	assert.Equal(t, effect, 3)
}

func TestProcessRegisterProcessHandler(t *testing.T) {

	RegisterProcessHandler("charts", func(process *Process) interface{} {
		return maps.Map{
			"name":   "charts",
			"class":  process.Class,
			"method": process.Method,
			"args":   process.Args,
		}
	})

	RegisterProcessHandler("charts.user.world", func(process *Process) interface{} {
		return maps.Map{
			"name":   "charts.user.world",
			"class":  process.Class,
			"method": process.Method,
			"args":   process.Args,
		}
	})

	res := NewProcess("charts.user.Hello", "foo", "bar").Run().(maps.Map)
	assert.Equal(t, "charts", res.Get("name"))
	assert.Equal(t, "user", res.Get("class"))
	assert.Equal(t, "hello", res.Get("method"))
	assert.Equal(t, []interface{}{"foo", "bar"}, res.Get("args"))

	res2 := NewProcess("charts.user.World", "bar", "foo").Run().(maps.Map)
	assert.Equal(t, "charts.user.world", res2.Get("name"))
	assert.Equal(t, "user", res2.Get("class"))
	assert.Equal(t, "world", res2.Get("method"))
	assert.Equal(t, []interface{}{"bar", "foo"}, res2.Get("args"))
}

func TestProcessSession(t *testing.T) {
	id := session.ID()
	global := map[string]interface{}{"hello": "world"}
	process := NewProcess("session.Set", "foo", "bar").WithSID(id).WithGlobal(global)
	process.Run()

	process = NewProcess("session.Get", "foo").WithSID(id).WithGlobal(global)
	foo := process.Run()
	assert.Equal(t, "bar", foo)

	process = NewProcess("session.Dump").WithSID(id).WithGlobal(global)
	data := process.Run()
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, data)
}

func TestProcessMustEachSaveWithIndex(t *testing.T) {
	user := Select("user")
	args := []interface{}{
		[]map[string]interface{}{
			{
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
			}, {
				"name":     "用户创建2",
				"manu_id":  2,
				"type":     "user",
				"idcard":   "23012619820207006X",
				"mobile":   "13900004443",
				"password": "qV@uT1DI",
				"key":      "XZ12MiPM",
				"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
				"status":   "enabled",
				"extra":    maps.MapStr{"sex": "男"},
			},
		},
		maps.MapStr{"balance": "$index"},
	}

	process := NewProcess("models.user.EachSave", args...)
	res := process.Run()
	ids, ok := res.([]int)
	assert.True(t, ok)
	assert.Equal(t, 2, len(ids))
	row := user.MustFind(ids[0], QueryParam{})
	row1 := user.MustFind(ids[1], QueryParam{})

	// 恢复数据
	capsule.Query().Table(user.MetaData.Table.Name).WhereIn("id", ids).Delete()
	assert.Equal(t, any.Of(row.Get("balance")).CInt(), 0)
	assert.Equal(t, any.Of(row1.Get("balance")).CInt(), 1)
}

func TestProcessMustEachSaveAfterDelete(t *testing.T) {
	user := Select("user")
	id := user.MustSave(maps.MapStrAny{
		"name":     "用户创建",
		"manu_id":  2,
		"type":     "user",
		"idcard":   "23082619820207000X",
		"mobile":   "13211132230",
		"password": "qV@uT1DI",
		"key":      "10120iP2",
		"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
		"status":   "enabled",
		"extra":    maps.MapStr{"sex": "女"},
	})
	assert.True(t, id > 0)

	args := []interface{}{
		[]int{id},
		[]map[string]interface{}{
			{
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
			}, {
				"name":     "用户创建2",
				"manu_id":  2,
				"type":     "user",
				"idcard":   "23012619820207006X",
				"mobile":   "13900004443",
				"password": "qV@uT1DI",
				"key":      "XZ12MiPM",
				"secret":   "wBeYjL7FjbcvpAdBrxtDFfjydsoPKhRN",
				"status":   "enabled",
				"extra":    maps.MapStr{"sex": "男"},
			},
		},
		maps.MapStr{"balance": "$index"},
	}

	process := NewProcess("models.user.EachSaveAfterDelete", args...)
	res := process.Run()

	_, err := user.Find(id, QueryParam{})
	assert.NotNil(t, err)

	ids, ok := res.([]int)
	assert.True(t, ok)
	assert.Equal(t, 2, len(ids))
	row := user.MustFind(ids[0], QueryParam{})
	row1 := user.MustFind(ids[1], QueryParam{})

	// 恢复数据
	ids = append(ids, id)
	capsule.Query().Table(user.MetaData.Table.Name).WhereIn("id", ids).Delete()
	assert.Equal(t, any.Of(row.Get("balance")).CInt(), 0)
	assert.Equal(t, any.Of(row1.Get("balance")).CInt(), 1)
}

func TestProcessOf(t *testing.T) {
	process, err := ProcessOf("models.user.Find", 1, QueryParam{})
	assert.Nil(t, err)
	assert.Equal(t, "models.user.find", process.Name)

	process, err = ProcessOf("not exists", 1, QueryParam{})
	assert.Nil(t, process)
	assert.Equal(t, "Process:not exists format error", err.Error())
}

func TestProcessExec(t *testing.T) {
	value, err := NewProcess("models.user.Find", 1, QueryParam{}).Exec()
	res := value.(maps.MapStr)
	assert.Nil(t, err)
	assert.Equal(t, 1, any.Of(res.Dot().Get("id")).CInt())
	assert.Equal(t, "男", res.Dot().Get("extra.sex"))

	value, err = NewProcess("models.user.Find", 100, QueryParam{}).Exec()
	assert.Nil(t, value)
	assert.Equal(t, "ID=100的数据不存在", err.Error())
}

func TestProcessLang(t *testing.T) {

	lang := NewProcess("models.user.Find", 1, QueryParam{}).Lang()
	assert.Equal(t, "", lang)

	sid := session.ID()
	session.Global().ID(sid).Set("__yao_lang", "zh-cn")

	lang = NewProcess("models.user.Find", 1, QueryParam{}).WithSID(sid).Lang()
	assert.Equal(t, "zh-cn", lang)

}
