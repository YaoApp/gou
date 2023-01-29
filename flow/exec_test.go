package flow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
)

func TestExec(t *testing.T) {
	prepare(t)
	flow, err := Select("basic")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	res, err := flow.Exec(yesterday, tomorrow)
	if err != nil {
		t.Fatal(err)
	}

	r := any.Of(res).MapStr().Dot()
	assert.Equal(t, yesterday, r.Get("params[0]"))
	assert.Equal(t, tomorrow, r.Get("params[1]"))
	assert.Equal(t, "U1", r.Get("data.query[0].name"))
	assert.Equal(t, "Duck", r.Get("data.categories[2].name"))
	assert.Equal(t, "U3", r.Get("data.users[1].name"))
}

// func TestExecQuery(t *testing.T) {
// 	flow, _ := Select("stat")
// 	res := maps.Of(flow.Exec("2000-01-02", "2050-12-31", 1, 2).(map[string]interface{}))
// 	// utils.Dump(res)
// 	assert.Equal(t, res.Dot().Get("data.manus.0.id"), int64(1))
// 	assert.Equal(t, res.Dot().Get("data.manus.0.short_name"), "云道天成")
// 	assert.Equal(t, res.Dot().Get("data.manus.0.type"), "服务商")
// 	assert.Equal(t, res.Dot().Get("data.manus.1.id"), int64(8))
// 	assert.Equal(t, res.Dot().Get("data.users.total"), 3)
// 	assert.Equal(t, res.Dot().Get("data.address.city"), "丰台区")
// 	assert.Equal(t, res.Dot().Get("params.0"), "2000-01-02")
// }

// func TestExecArraySet(t *testing.T) {
// 	args := []map[string]interface{}{
// 		{"name": "hello", "value": "world"},
// 		{"name": "foo", "value": "bar"},
// 	}
// 	flow, _ := Select("arrayset")
// 	res := flow.Exec(args)
// 	utils.Dump(res)
// }

// func TestExecGlobalSession(t *testing.T) {
// 	sid := session.ID()
// 	session.Global().ID(sid).Set("id", 1)
// 	flow, _ := Select("user.info")
// 	flow.WithSID(sid).WithGlobal(map[string]interface{}{"foo": "bar"})
// 	res := maps.Of(flow.Exec().(map[string]interface{})).Dot()
// 	assert.Equal(t, float64(1), res.Get("ID"))
// 	assert.Equal(t, float64(1), res.Get("会话信息.id"))
// 	assert.Equal(t, "admin", res.Get("会话信息.type"))
// 	assert.Equal(t, "bar", res.Get("全局信息.foo"))
// 	assert.Equal(t, "bar", res.Get("全局信息.foo"))
// 	assert.Equal(t, int64(1), res.Get("用户数据.id"))
// 	assert.Equal(t, "管理员", res.Get("用户数据.name"))
// 	assert.Equal(t, "admin", res.Get("用户数据.type"))
// 	assert.Equal(t, "bar", res.Get("脚本数据.global.foo"))
// 	assert.Equal(t, float64(1), res.Get("脚本数据.session.id"))
// 	assert.Equal(t, "admin", res.Get("脚本数据.session.type"))
// }
