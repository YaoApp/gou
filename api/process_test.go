package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestProcesList(t *testing.T) {

	prepare(t)
	defer clean()

	var apis = process.New("api.list").Run()
	assert.Equal(t, 2, len(apis.(map[string]*API)))
	apis = process.New("api.list", []string{"user"}).Run()
	assert.Equal(t, 1, len(apis.(map[string]*API)))
	assert.Equal(t, "user", apis.(map[string]*API)["user"].ID)
}
