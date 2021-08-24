package gou

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
)

// TestPLGRoot
var TestPLGRoot = "/data/plugins"

func init() {
	TestPLGRoot = os.Getenv("GOU_TEST_PLG_ROOT")
}

func TestLoadPlugin(t *testing.T) {
	cmd := path.Join(TestPLGRoot, "user")
	p := LoadPlugin(cmd, "user")
	defer p.Client.Kill()

	mod := SelectPlugin("user")
	res, err := mod.Get("hello", []byte(`{"foo":"bar"}`))
	v := maps.MakeStrAny()
	err = json.Unmarshal(res, &v)
	utils.Dump(v, err)
}
