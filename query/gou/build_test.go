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
	assert.Equal(t, Q("select * from `table` as `name`"), sql)
}

func TestBuildWheres(t *testing.T) {
	gou := Open(GetFileName("queries/wheres.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	assert.Equal(t, Q("select `*` from `user` as `u` where `score` < ? and `score` > ? and `id` in (?,?) and (`name` like ? or `name` like ?) and `manu_id` in (select `manu_id` AS `id` from `manu` where `status` = ?)"), sql)
}

func TestBuildOrders(t *testing.T) {
	gou := Open(GetFileName("queries/orders.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	if TestDriver == "postgres" {
		assert.Equal(t, Q(`select "*" from "table" as "name" order by "id" desc, MAX("id") desc, "table"."pin" asc, "array"::jsonb#>>'{,id}' asc, "object"::jsonb->>'arr[0].id' asc`), sql)
	} else {
		assert.Equal(t, Q("select `*` from `table` as `name` order by `id` desc, MAX(`id`) desc, `table`.`pin` asc, JSON_EXTRACT(`array`, '$[*].id') asc, JSON_EXTRACT(`object`, '$.arr[0].id') asc"), sql)
	}
}

func TestBuildGroups(t *testing.T) {
	gou := Open(GetFileName("queries/groups.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	if TestDriver == "postgres" {
		assert.Equal(t,
			Q(`select max("score") AS "最高分", CASE WHEN GROUPING("city") = 1 THEN '所有城市' ELSE "city" END AS "城市", CASE WHEN GROUPING("id") = 1 THEN 'ID' ELSE "id" END AS "id", "kind" from "table" as "name" group by "kind", ROLLUP("city", "id")`),
			sql,
		)
	} else {
		assert.Equal(t,
			Q("select max(`score`) AS `最高分`, IF(GROUPING(`city`),'所有城市',`city`) AS `城市`, IF(GROUPING(`id`),'ID',`id`) AS `id`, `kind` from `table` as `name` group by `kind`, `city` WITH ROLLUP, `id` WITH ROLLUP"),
			sql,
		)
	}
}

func TestBuildGroupsArray(t *testing.T) {
	gou := Open(GetFileName("queries/groups.array.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	if TestDriver == "postgres" {
		// PG uses LATERAL jsonb_array_elements + CROSS JOIN + GROUP BY ROLLUP(...)
		assert.Contains(t, sql, "LATERAL jsonb_array_elements")
		assert.Contains(t, sql, "CROSS JOIN")
		assert.Contains(t, sql, "ROLLUP(")
		assert.Contains(t, sql, "CASE WHEN GROUPING")
	} else {
		assert.Equal(t,
			Q("select max(`score`) AS `最高分`, IF(GROUPING(`__JSON_T1`.`F1`),'所有城市',`__JSON_T1`.`F1`) AS `citys[*]`, `__JSON_T2`.`F2` AS `行业`, IF(GROUPING(`__JSON_T3`.`F3`),'所有行政区',`__JSON_T3`.`F3`) AS `towns[*]`, IF(GROUPING(`__JSON_T4`.`F4`),'合计',`__JSON_T4`.`F4`) AS `goods.sku[*].price`, `__JSON_T5`.`F5` AS `goods.sku[*].gid`, JSON_EXTRACT(`option`, '$.ids[*]') AS `ID` from `table` as `name` JOIN JSON_TABLE(`industries`, '$[*]' columns (`F2` VARCHAR(100) path '$') ) AS `__JSON_T2` JOIN JSON_TABLE(`citys`, '$[*]' columns (`F1` VARCHAR(50) path '$') ) AS `__JSON_T1` JOIN JSON_TABLE(`towns`, '$[*]' columns (`F3` VARCHAR(100) path '$') ) AS `__JSON_T3` JOIN JSON_TABLE(`goods`.`sku`, '$[*]' columns (`F4` DECIMAL(11,2) path '$.price') ) AS `__JSON_T4` JOIN JSON_TABLE(`goods`.`sku`, '$[*]' columns (`F5` INT path '$.gid') ) AS `__JSON_T5` group by `行业`, `ID`, `citys[*]` WITH ROLLUP, `towns[*]` WITH ROLLUP, `goods.sku[*].price` WITH ROLLUP, `goods.sku[*].gid`"),
			sql,
		)
	}
}

func TestBuildHavings(t *testing.T) {
	gou := Open(GetFileName("queries/havings.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	if TestDriver == "postgres" {
		assert.Contains(t, sql, "CASE WHEN GROUPING")
		assert.Contains(t, sql, "ROLLUP(")
		assert.Contains(t, sql, "having")
	} else {
		assert.Equal(t, Q("select max(`score`) AS `最高分`, IF(GROUPING(`city`),'所有城市',`city`) AS `城市`, IF(GROUPING(`id`),'ID',`id`) AS `id`, `kind` from `table` as `name` group by `kind`, `city` WITH ROLLUP, `id` WITH ROLLUP having `城市` = ? or `kind` = ?"), sql)
	}
}

func TestBuildUnions(t *testing.T) {
	gou := Open(GetFileName("queries/unions.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	assert.Contains(t, sql, "union all")
}

func TestBuildSubQueryName(t *testing.T) {
	gou := Open(GetFileName("queries/subquery.name.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	assert.Equal(t, Q("select `id`, `厂商`.`name` from (select `id`, `name` from `manu`) as `厂商`"), sql)
}

func TestBuildSubQuery(t *testing.T) {
	gou := Open(GetFileName("queries/subquery.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	assert.Equal(t, Q("select `id`, `_SUB_`.`name` from (select `id`, `name` from `manu`) as `_SUB_`"), sql)
}

func TestBuildJoins(t *testing.T) {
	gou := Open(GetFileName("queries/joins.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	assert.Equal(t, Q("select `id`, `name`, `t2`.`name2`, `t3`.`name3`, `t4`.`name4` from `t1` left join `table2` as `t2` on `t2`.`id` = `t1`.`t2_id` right join `table3` as `t3` on `t3`.`id` = `t2`.`t3_id` inner join `table4` as `t4` on `t4`.`id` = `t2`.`t4_id`"), sql)
}

func TestBuildLimit(t *testing.T) {
	gou := Open(GetFileName("queries/limit.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	assert.Equal(t, Q("select `*` from `table` as `name`"), sql)
}

func TestBuildSQL(t *testing.T) {
	gou := Open(GetFileName("queries/sql.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	bindings := gou.GetBindings()
	assert.Equal(t, "SELECT * FROM user WHERE name = ? AND type = ?", sql)
	assert.Equal(t, []Any{"张三", F(20)}, bindings)
}

func TestSqlTypeOf(t *testing.T) {
	gou := Open(GetFileName("queries/select.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)

	cases := []struct {
		name      string
		length    int
		precision int
		scale     int
		mysql     string
		pg        string
	}{
		{"string", 100, 0, 0, "VARCHAR(100)", "VARCHAR(100)"},
		{"string", 0, 0, 0, "VARCHAR(255)", "VARCHAR(255)"},
		{"char", 50, 0, 0, "VARCHAR(50)", "VARCHAR(50)"},
		{"integer", 0, 0, 0, "INT", "INTEGER"},
		{"boolean", 0, 0, 0, "BOOLEAN", "BOOLEAN"},
		{"date", 0, 0, 0, "DATE", "DATE"},
		{"time", 0, 0, 0, "TIME", "TIME"},
		{"datetime", 0, 0, 0, "DATETIME", "TIMESTAMP"},
		{"timestamp", 0, 0, 0, "TIMESTAMP", "TIMESTAMP"},
		{"double", 0, 0, 0, "DOUBLE(10,2)", "DOUBLE PRECISION"},
		{"double", 0, 8, 3, "DOUBLE(8,3)", "DOUBLE PRECISION"},
		{"float", 0, 0, 0, "FLOAT(10,2)", "REAL"},
		{"float", 0, 6, 2, "FLOAT(6,2)", "REAL"},
		{"decimal", 0, 0, 0, "DECIMAL(10,2)", "DECIMAL(10,2)"},
		{"decimal", 0, 12, 4, "DECIMAL(12,4)", "DECIMAL(12,4)"},
	}

	for _, c := range cases {
		exp := Expression{
			Type: &FieldType{Name: c.name, Length: c.length, Precision: c.precision, Scale: c.scale},
		}
		result := gou.sqlTypeOf(exp)
		if TestDriver == "postgres" {
			assert.Equal(t, c.pg, result, "sqlTypeOf(%s) on PG", c.name)
		} else {
			assert.Equal(t, c.mysql, result, "sqlTypeOf(%s) on MySQL", c.name)
		}
	}

	expNoType := Expression{Type: nil}
	assert.Equal(t, "VARCHAR(255)", gou.sqlTypeOf(expNoType))
}

func TestPgCastType(t *testing.T) {
	gou := Open(GetFileName("queries/select.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)

	cases := []struct {
		name      string
		precision int
		scale     int
		expected  string
	}{
		{"string", 0, 0, "text"},
		{"char", 0, 0, "text"},
		{"integer", 0, 0, "integer"},
		{"boolean", 0, 0, "boolean"},
		{"double", 0, 0, "double precision"},
		{"float", 0, 0, "real"},
		{"decimal", 0, 0, "numeric(10,2)"},
		{"decimal", 12, 4, "numeric(12,4)"},
		{"unknown_type", 0, 0, "text"},
	}

	for _, c := range cases {
		exp := Expression{
			Type: &FieldType{Name: c.name, Precision: c.precision, Scale: c.scale},
		}
		assert.Equal(t, c.expected, gou.pgCastType(exp), "pgCastType(%s)", c.name)
	}

	assert.Equal(t, "text", gou.pgCastType(Expression{Type: nil}))
}

func TestBuildSelectAES(t *testing.T) {
	gou := Open(GetFileName("queries/select.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	gou.Build()
	sql := gou.ToSQL()
	if TestDriver == "postgres" {
		assert.Contains(t, sql, "pgp_sym_decrypt")
		assert.Contains(t, sql, "decode(")
	} else {
		assert.Contains(t, sql, "AES_DECRYPT")
		assert.Contains(t, sql, "UNHEX(")
	}
}
