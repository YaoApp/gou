package gou

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunSelect(t *testing.T) {
	gou := Open(GetFileName("queries/select.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.runSelect()
	sql := gou.ToSQL()
	assert.Equal(t, true, len(sql) > 0)
}
