package dsl

import (
	"os"
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestOpenWorkshop(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	workshop, err := OpenWorkshop(root)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 8, len(workshop.Require))
	assert.Equal(t, 8, len(workshop.Mapping))

	assert.Equal(t, true, workshop.Require[3].Replaced)
	assert.Equal(t, false, workshop.Require[3].Downloaded)
	assert.Equal(t, "github.com/yaoapp/demo-wms/cloud@v0.0.0-20220223010332-e86eab4c8490", workshop.Require[3].URL)
	assert.Equal(t, "github.com/yaoapp/demo-wms", workshop.Require[3].Repo)
	assert.Equal(t, "github.com", workshop.Require[3].Domain)
	assert.Equal(t, "yaoapp", workshop.Require[3].Team)
	assert.Equal(t, "demo-wms", workshop.Require[3].Project)
	assert.Equal(t, "/cloud", workshop.Require[3].Path)
	assert.Equal(t, "demo-wms.yaoapp.cloud", workshop.Require[3].Name)
	assert.Equal(t, "demo-wms.yaoapp.cloud", workshop.Require[3].Alias)
	assert.Equal(t, "0.0.0-20220223010332-e86eab4c8490", workshop.Require[3].Version.String())
	assert.Equal(t, "e86eab4c8490", workshop.Require[3].Commit)

}
