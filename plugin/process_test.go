package plugin

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/maps"
)

func TestProcessPlugins(t *testing.T) {
	prepare(t)
	defer KillAll()

	p, err := process.Of("plugins.user.login", "13111021983", "#991832")
	if err != nil {
		t.Fatal(err)
	}

	data, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, ok := data.(maps.MapStr)
	if !ok {
		t.Fatal("return data type error")
	}

	assert.Equal(t, res.Dot().Get("name"), "login")
	assert.Equal(t, res.Dot().Get("args.0"), "13111021983")
	assert.Equal(t, res.Dot().Get("args.1"), "#991832")
	assert.Nil(t, err)
}

func prepare(t *testing.T) {
	KillAll()
	root := os.Getenv("GOU_TEST_PLUGIN")
	file := path.Join(root, "user.so")
	_, err := Load(file, "user")
	if err != nil {
		t.Fatal(err)
	}
}
