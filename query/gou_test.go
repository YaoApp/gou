package query

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/utils"
)

func TestGouOpen(t *testing.T) {
	assert.Panics(t, func() {
		GouOpen(path.Join(TestQueryRoot, "not-exists.json"))
	})
	gou := GouOpen(path.Join(TestQueryRoot, "full.json"))
	utils.Dump(gou.QueryDSL)
}
