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
	assert.Equal(t, "select `*` from `table` as `name` order by `id` desc, MAX(`id`) desc, `table`.`pin` asc, JSON_EXTRACT(`array`, '$[*].id') asc, JSON_EXTRACT(`object`, '$.arr[0].id') asc", sql)
}

func TestBuildGroups(t *testing.T) {
	gou := Open(GetFileName("queries/groups.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	// utils.Dump(sql)
	assert.Equal(t,
		"select max(`score`) AS `最高分`, IF(GROUPING(`city`),'所有城市',`city`) AS `城市`, IF(GROUPING(`id`),'ID',`id`) AS `id`, `kind` from `table` as `name` group by `kind`, `city` WITH ROLLUP, `id` WITH ROLLUP",
		sql,
	)
}

func TestBuildGroupsArray(t *testing.T) {
	gou := Open(GetFileName("queries/groups.array.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	// utils.Dump(sql)
	assert.Equal(t,
		"select max(`score`) AS `最高分`, IF(GROUPING(`__JSON_T1`.`F1`),'所有城市',`__JSON_T1`.`F1`) AS `citys[*]`, `__JSON_T2`.`f2` as `行业`, IF(GROUPING(`__JSON_T3`.`F3`),'所有行政区',`__JSON_T3`.`F3`) AS `towns[*]`, IF(GROUPING(`__JSON_T4`.`F4`),'合计',`__JSON_T4`.`F4`) AS `goods.sku[*].price`, `__JSON_T5`.`f5` as `goods`, JSON_EXTRACT(`option`, '$.ids[*]') AS `ID` from `table` as `name` JOIN JSON_TABLE(`industries`, '$[*]' columns (`F2` VARCHAR(100) path '$') ) AS `__JSON_T2` JOIN JSON_TABLE(`citys`, '$[*]' columns (`F1` VARCHAR(50) path '$') ) AS `__JSON_T1` JOIN JSON_TABLE(`towns`, '$[*]' columns (`F3` VARCHAR(100) path '$') ) AS `__JSON_T3` JOIN JSON_TABLE(`goods`.`sku`, '$[*]' columns (`F4` DECIMAL(11,2) path '$.price') ) AS `__JSON_T4` JOIN JSON_TABLE(`goods`.`sku`, '$[*]' columns (`F5` INT path '$.gid') ) AS `__JSON_T5` group by `行业`, `ID`, `citys[*]` WITH ROLLUP, `towns[*]` WITH ROLLUP, `goods.sku[*].price` WITH ROLLUP, `goods.sku[*].gid`",
		sql,
	)
}

func TestBuildHavings(t *testing.T) {
	gou := Open(GetFileName("queries/havings.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	// utils.Dump(sql)
	assert.Equal(t, "select max(`score`) AS `最高分`, IF(GROUPING(`city`),'所有城市',`city`) AS `城市`, IF(GROUPING(`id`),'ID',`id`) AS `id`, `kind` from `table` as `name` group by `kind`, `city` WITH ROLLUP, `id` WITH ROLLUP having `城市` = ? or `kind` = ?", sql)
}

func TestBuildUnions(t *testing.T) {
	gou := Open(GetFileName("queries/unions.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	// utils.Dump(sql)
	assert.Contains(t, sql, "union all")
}

func TestBuildSubQueryName(t *testing.T) {
	gou := Open(GetFileName("queries/subquery.name.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	// utils.Dump(sql)
	assert.Equal(t, "select `id`, `厂商`.`name` from (select `id`, `name` from `manu`) as `厂商`", sql)
}

func TestBuildSubQuery(t *testing.T) {
	gou := Open(GetFileName("queries/subquery.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	// utils.Dump(sql)
	assert.Equal(t, "select `id`, `_SUB_`.`name` from (select `id`, `name` from `manu`) as `_SUB_`", sql)
}
