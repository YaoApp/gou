package gou

import (
	"os"
	"path"
	"testing"

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
	res, err := mod.Exec("login", "13111021983", "#991832")
	utils.Dump(res.MustMap(), err)
}
