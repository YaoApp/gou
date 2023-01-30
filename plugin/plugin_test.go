package plugin

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadPlugin(t *testing.T) {

	root := os.Getenv("GOU_TEST_PLUGIN")
	file := path.Join(root, "user.so")
	p, err := Load(file, "user")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Kill()

	user, err := Select("user")
	if err != nil {
		t.Fatal(err)
	}

	res, err := user.Exec("login", "13111021983", "#991832")
	assert.Equal(t, res.MustMap().Dot().Get("name"), "login")
	assert.Equal(t, res.MustMap().Dot().Get("args.0"), "13111021983")
	assert.Equal(t, res.MustMap().Dot().Get("args.1"), "#991832")
	assert.Nil(t, err)
}
