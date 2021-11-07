package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
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
	assert.Equal(t, len(res.Dot().Get("data.users").([]maps.Map)), 3)
	assert.Equal(t, len(res.Dot().Get("data.manus").([]maps.Map)), 4)
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
}
