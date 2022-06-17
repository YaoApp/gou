package flow

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/dsl"
	"github.com/yaoapp/gou/dsl/workshop"
)

func TestDSLCompilePing(t *testing.T) {
	yao := newFlow(t, "ping.flow.yao")
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	flow, has := gou.Flows["ping"]
	assert.Equal(t, true, has)
	assert.Equal(t, "ping", flow.Name)
	assert.Equal(t, "PONG", flow.Output)
}

func TestDSLCompileUserSearch(t *testing.T) {
	yao := newFlow(t, filepath.Join("user", "search.flow.yao"))
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	flow, has := gou.Flows["user.search"]
	assert.Equal(t, true, has)
	assert.Equal(t, "user.search", flow.Name)
	assert.Equal(t, "?:$res.data", flow.Output.(map[string]interface{})["data"])
	assert.Equal(t, "?:$res.user", flow.Output.(map[string]interface{})["user"])
}

func TestDSLCompileRefresh(t *testing.T) {
	yao := newFlow(t, "ping.flow.yao")
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	flow, has := gou.Flows["ping"]
	assert.Equal(t, true, has)
	assert.Equal(t, "ping", flow.Name)
	assert.Equal(t, "PONG", flow.Output)

	// Backup content
	file := yao.Head.File
	backup, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	defer ioutil.WriteFile(file, backup, 0644) // RESET

	// change the content
	err = ioutil.WriteFile(file, []byte(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"FROM": "@github.com/YaoApp/workshop-tests-erp/flows/sys/ping",
		"output": "DONG"
	  }`), 0644)

	if err != nil {
		t.Fatal(err)
	}

	err = yao.Refresh()
	if err != nil {
		t.Fatal(err)
	}

	flow, has = gou.Flows["ping"]
	assert.Equal(t, true, has)
	assert.Equal(t, "ping", flow.Name)
	assert.Equal(t, "DONG", flow.Output)

}

func TestDSLRemove(t *testing.T) {
	yao := newFlow(t, "ping.flow.yao")
	err := yao.Compile()
	if err != nil {
		t.Fatal(err)
	}

	flow, has := gou.Flows["ping"]
	assert.Equal(t, true, has)
	assert.Equal(t, "ping", flow.Name)
	assert.Equal(t, "PONG", flow.Output)

	// Remove
	err = yao.Remove()
	if err != nil {
		t.Fatal(err)
	}

	_, has = gou.Models["ping"]
	assert.Equal(t, false, has)
	assert.Equal(t, map[string]interface{}{}, yao.Content)
}

func newFlow(t *testing.T, name string) *dsl.YAO {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := filepath.Join(root, "flows", name)
	workshop, err := workshop.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	yao := dsl.New(workshop)
	err = yao.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	return yao
}
