package gou

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	// rows := Gou([]byte{}).With(qb).Get()
	// utils.Dump(rows)
}

func TestFirst(t *testing.T) {
	// Gou([]byte{}).With(qb).First()
}

func TestPaginate(t *testing.T) {
	// Gou([]byte{}).With(qb).Paginate()
}

func TestRun(t *testing.T) {
	// Gou([]byte{}).With(qb).Run()
}

func TestRunSelect(t *testing.T) {
	gou := Open(GetFileName("queries/select.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.runSelect()
	sql := gou.ToSQL()
	assert.Equal(t, true, len(sql) > 0)
}

func TestRunFrom(t *testing.T) {
	gou := Open(GetFileName("queries/from.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.runFrom()
	sql := gou.ToSQL()
	assert.Equal(t, "select * from `table` as `name`", sql)
}
