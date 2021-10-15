package gou

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSelect(t *testing.T) {
	gou := Open(GetFileName("queries/select.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.buildSelect()
	sql := gou.ToSQL()
	assert.Equal(t, true, len(sql) > 0)
}

func TestBuildFrom(t *testing.T) {
	gou := Open(GetFileName("queries/from.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.buildFrom()
	sql := gou.ToSQL()
	assert.Equal(t, "select * from `table` as `name`", sql)
}

func TestBuildWheres(t *testing.T) {
	gou := Open(GetFileName("queries/wheres.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	assert.Equal(t, "select `*` from `user` as `u` where `score` < ? and `score` > ? and `id` in (?,?) and (`name` like ? and `name` like ?) and `manu_id` in (select `manu_id` as `id` from `manu` where `status` = ?)", sql)
}

func TestBuildOrders(t *testing.T) {
	gou := Open(GetFileName("queries/orders.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	assert.Equal(t, "select `*` from `table` as `name` order by `id` desc, MAX(`id`) desc, `table`.`pin` asc, JSON_EXTRACT(`array`, '$[0].id') asc, JSON_EXTRACT(`object`, '$.arr[0].id') asc", sql)
}
