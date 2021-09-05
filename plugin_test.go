package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadPlugin(t *testing.T) {
	cmd := path.Join(TestPLGRoot, "user")
	p := LoadPlugin(cmd, "user")
	defer p.Client.Kill()

	mod := SelectPluginModel("user")
	res, err := mod.Exec("login", "13111021983", "#991832")
	assert.Equal(t, res.MustMap().Dot().Get("name"), "login")
	assert.Equal(t, res.MustMap().Dot().Get("args.0"), "13111021983")
	assert.Equal(t, res.MustMap().Dot().Get("args.1"), "#991832")
	assert.Nil(t, err)
}
