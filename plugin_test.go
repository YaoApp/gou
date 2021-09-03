package gou

import (
	"path"
	"testing"

	"github.com/yaoapp/kun/utils"
)

func TestLoadPlugin(t *testing.T) {
	cmd := path.Join(TestPLGRoot, "user")
	p := LoadPlugin(cmd, "user")
	defer p.Client.Kill()

	mod := SelectPluginModel("user")
	res, err := mod.Exec("login", "13111021983", "#991832")
	utils.Dump(res.MustMap(), err)
}
