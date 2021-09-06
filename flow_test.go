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
