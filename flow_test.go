package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
)

func TestLoadFlow(t *testing.T) {
	latestFlow := LoadFlow("file://"+path.Join(TestFLWRoot, "latest.flow.json"), "latest").
		LoadScript("file://"+path.Join(TestFLWRoot, "latest.rank.js"), "rank").
		LoadScript("file://"+path.Join(TestFLWRoot, "latest.count.js"), "count")
	latestFlow.Reload()
	assert.Equal(t, latestFlow.Label, "最新信息")
	assert.Equal(t, latestFlow.Name, "latest")
	assert.Equal(t, len(latestFlow.Nodes), 4)
}

func TestSelectFlow(t *testing.T) {
	latestFlow := SelectFlow("latest")
	latestFlow.Reload()
	assert.Equal(t, latestFlow.Label, "最新信息")
	assert.Equal(t, latestFlow.Name, "latest")
	assert.Equal(t, len(latestFlow.Nodes), 4)
}

func TestFlowExec(t *testing.T) {
	flow := SelectFlow("latest")
	res := maps.Of(flow.Exec("%公司%", "bar").(map[string]interface{}))

	assert.Equal(t, res.Get("params"), []interface{}{"%公司%", "bar"})
	assert.Equal(t, len(res.Dot().Get("data.users").([]maps.MapStrAny)), 3)
	assert.Equal(t, len(res.Dot().Get("data.manus").([]interface{})), 4)
	// assert.Equal(t, res.Dot().Get("data.users.0.id"), int64(3))
	// assert.Equal(t, res.Dot().Get("data.manus.1.id"), int64(3))
	assert.Equal(t, res.Dot().Get("data.count.plugin"), "github")
}

func TestFlowExecQuery(t *testing.T) {
	flow := SelectFlow("stat")
	res := maps.Of(flow.Exec("2000-01-02", "2050-12-31", 1, 2).(map[string]interface{}))
	// utils.Dump(res)
	assert.Equal(t, res.Dot().Get("data.manus.0.id"), int64(1))
	assert.Equal(t, res.Dot().Get("data.manus.0.short_name"), "云道天成")
	assert.Equal(t, res.Dot().Get("data.manus.0.type"), "服务商")
	assert.Equal(t, res.Dot().Get("data.manus.1.id"), int64(8))
	assert.Equal(t, res.Dot().Get("data.users.total"), 3)
	assert.Equal(t, res.Dot().Get("data.address.city"), "丰台区")
	assert.Equal(t, res.Dot().Get("params.0"), "2000-01-02")
}

func TestFlowExecScript(t *testing.T) {
	flow := SelectFlow("script")
	res := maps.Of(flow.Exec("world").(map[string]interface{}))
	assert.Equal(t, "world", res.Dot().Get("全局脚本.args.0"))
	assert.Equal(t, "hello:world", res.Dot().Get("全局脚本.hello"))
	assert.Equal(t, "rank", res.Dot().Get("Flow脚本.name"))
	assert.Equal(t, "sort", res.Dot().Get("Flow完整引用.name"))
	assert.Equal(t, "rank hello", res.Dot().Get("Flow脚本.hello"))
	assert.Equal(t, "sort hello", res.Dot().Get("Flow完整引用.hello"))

	assert.Equal(t, "login", res.Dot().Get("Flow脚本.user.name"))
	assert.Equal(t, float64(1024), res.Dot().Get("Flow脚本.user.args.0"))
}

func TestFlowExecArraySet(t *testing.T) {
	args := []map[string]interface{}{
		{"name": "hello", "value": "world"},
		{"name": "foo", "value": "bar"},
	}
	flow := SelectFlow("arrayset")
	res := flow.Exec(args)
	utils.Dump(res)
}

func TestFlowExecGlobalSession(t *testing.T) {
	sid := session.ID()
	session.Global().ID(sid).Set("id", 1)
	flow := SelectFlow("user.info").WithSID(sid).WithGlobal(map[string]interface{}{"foo": "bar"})
	res := maps.Of(flow.Exec().(map[string]interface{})).Dot()
	assert.Equal(t, float64(1), res.Get("ID"))
	assert.Equal(t, float64(1), res.Get("会话信息.id"))
	assert.Equal(t, "admin", res.Get("会话信息.type"))
	assert.Equal(t, "bar", res.Get("全局信息.foo"))
	assert.Equal(t, "bar", res.Get("全局信息.foo"))
	assert.Equal(t, int64(1), res.Get("用户数据.id"))
	assert.Equal(t, "管理员", res.Get("用户数据.name"))
	assert.Equal(t, "admin", res.Get("用户数据.type"))
	assert.Equal(t, "bar", res.Get("脚本数据.global.foo"))
	assert.Equal(t, float64(1), res.Get("脚本数据.session.id"))
	assert.Equal(t, "admin", res.Get("脚本数据.session.type"))
}
