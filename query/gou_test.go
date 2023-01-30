package query

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGouOpen(t *testing.T) {
	assert.Panics(t, func() {
		GouOpen(path.Join(TestQueryRoot, "not-exists.json"))
	})
	gou := GouOpen(path.Join(TestQueryRoot, "full.json"))

	assert.Equal(t, "user", gou.From.Name)
	assert.Len(t, gou.Orders, 1)
	assert.Len(t, gou.Select, 2)
	assert.Len(t, gou.Wheres, 2)
}
