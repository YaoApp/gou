package gou

import (
	"io/ioutil"
	"net/http"
	"path"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/grpc"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/capsule"
)

func TestLoadAPI(t *testing.T) {
	user := LoadAPI("file://"+path.Join(TestAPIRoot, "user.http.json"), "user")
	user.Reload()
}

func TestSelectAPI(t *testing.T) {
	user := SelectAPI("user")
	user.Reload()
}

func TestServeHTTP(t *testing.T) {

	go ServeHTTP(Server{
		Debug:  true,
		Host:   "127.0.0.1",
		Port:   5001,
		Allows: []string{"a.com", "b.com"},
	})

	// 发送请求
	request := func() (maps.MapStr, error) {
		time.Sleep(time.Microsecond * 100)
		resp, err := http.Get("http://127.0.0.1:5001/user/info/1?select=id,name")
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		res := maps.MakeMapStr()
		err = jsoniter.Unmarshal(body, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// 等待服务启动
	times := 0
	for times < 20 { // 2秒超时
		times++
		res, err := request()
		if err != nil {
			continue
		}
		assert.Equal(t, 1, any.Of(res.Get("id")).CInt())
		assert.Equal(t, "管理员", res.Get("name"))
		return
	}

	assert.True(t, false)
}

func TestCallerExec(t *testing.T) {
	defer SelectPlugin("user").Client.Kill()
	res := NewCaller("plugins.user.Login", 1).Run().(*grpc.Response).MustMap()
	res2 := NewCaller("plugins.user.Login", 2).Run().(*grpc.Response).MustMap()
	assert.Equal(t, "login", res.Get("name"))
	assert.Equal(t, "login", res2.Get("name"))
	assert.Equal(t, 1, any.Of(res.Dot().Get("args.0")).CInt())
	assert.Equal(t, 2, any.Of(res2.Dot().Get("args.0")).CInt())
}

func TestCallerFind(t *testing.T) {
	res := NewCaller("models.user.Find", 1, QueryParam{}).Run().(maps.MapStr)
	assert.Equal(t, 1, any.Of(res.Dot().Get("id")).CInt())
	assert.Equal(t, "男", res.Dot().Get("extra.sex"))
}

func TestCallerGet(t *testing.T) {
	rows := NewCaller("models.user.Get", QueryParam{Limit: 2}).Run().([]maps.MapStr)
	res := maps.Map{"data": rows}.Dot()
	assert.Equal(t, 2, len(rows))
	assert.Equal(t, 1, any.Of(res.Get("data.0.id")).CInt())
	assert.Equal(t, "男", res.Get("data.0.extra.sex"))
	assert.Equal(t, 2, any.Of(res.Get("data.1.id")).CInt())
	assert.Equal(t, "女", res.Get("data.1.extra.sex"))
}

func TestCallerPaginate(t *testing.T) {
	res := NewCaller("models.user.Paginate", QueryParam{}, 1, 2).Run().(maps.MapStr).Dot()
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

func TestCallerCreate(t *testing.T) {
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
	id := NewCaller("models.user.Create", row).Run().(int)
	assert.Greater(t, id, 0)

	// 清空数据
	capsule.Query().Table(Select("user").MetaData.Table.Name).Where("id", id).Delete()
}

func TestCallerUpdate(t *testing.T) {
	id := NewCaller("models.user.Update", 1, maps.MapStr{"balance": 200}).Run()
	assert.Nil(t, id)

	// 恢复数据
	capsule.Query().Table(Select("user").MetaData.Table.Name).Where("id", 1).Update(maps.MapStr{"balance": 0})
}

func TestCallerSave(t *testing.T) {
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
	id := NewCaller("models.user.Save", row).Run().(int)
	assert.Greater(t, id, 0)

	// 清空数据
	capsule.Query().Table(Select("user").MetaData.Table.Name).Where("id", id).Delete()
}

func TestCallerDelete(t *testing.T) {
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
	NewCaller("models.user.Delete", id).Run()

	// 清空数据
	capsule.Query().Table(Select("user").MetaData.Table.Name).Where("id", id).Delete()
}

func TestCallerDestroy(t *testing.T) {
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
	NewCaller("models.user.Destroy", id).Run()

	// 清空数据
	capsule.Query().Table(Select("user").MetaData.Table.Name).Where("id", id).Delete()
}

func TestCallerInsert(t *testing.T) {

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

	NewCaller("models.address.Insert", payload["columns"], payload["rows"]).Run()

	// 清理数据
	address := Select("address")
	capsule.Query().Table(address.MetaData.Table.Name).Where("user_id", 4).Delete()
}

func TestCallerUpdateWhere(t *testing.T) {
	effect := NewCaller("models.user.UpdateWhere",
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
