package gou

import (
	"fmt"
	"strings"

	"github.com/yaoapp/kun/exception"
)

type whereArgs struct {
	Method string
	OR     bool
	Field  interface{}
	Args   []interface{}
}

// Build 设定查询条件
func (gou *Query) Build() {
	errs := gou.Validate()
	if len(errs) > 0 {
		exception.New("查询条件错误", 400).Ctx(errs).Throw()
	}
	gou.buildSelect()
	gou.buildFrom()
	gou.buildWheres()
	gou.buildOrders()
	gou.buildGroups()
}

// buildSelect Select
func (gou *Query) buildSelect() *Query {
	fields := []interface{}{}
	for _, exp := range gou.Select {
		sql := gou.sqlExpression(exp, true)
		if sql != nil {
			fields = append(fields, sql)
		}
	}
	gou.Query.Select(fields...)
	return gou
}

// buildFrom From
func (gou *Query) buildFrom() *Query {

	if gou.From != nil {
		table := gou.From.Name

		if gou.From.IsModel {
			table = gou.GetTableName(table)
		}
		if gou.From.Alias != "" {
			gou.Query.From(fmt.Sprintf("%s AS %s", table, gou.From.Alias))
			return gou
		}
		gou.Query.From(table)
	}
	return gou
}

// buildWheres Wheres
func (gou *Query) buildWheres() *Query {

	if gou.Wheres == nil {
		return gou
	}

	for _, where := range gou.Wheres {
		gou.buildWhere(where)
	}
	return gou
}

// buildWheres where
func (gou *Query) buildWhere(where Where) {
	args := gou.parseWhereArgs(where)
	switch args.Method {
	case "where":
		gou.setWhere(args.OR, args.Field, args.Args...)
		break
	case "whereIn":
		gou.setWhereIn(args.OR, args.Field, args.Args[1])
		break
	case "whereNull":
		gou.setWhereNull(args.OR, args.Field)
		break
	case "whereNotNull":
		gou.setWhereNotNull(args.OR, args.Field)
		break
	case "wheres":
		gou.setWhere(args.OR, args.Field)
		break
	}
}

// buildOrders Orders
func (gou *Query) buildOrders() *Query {
	if gou.Orders == nil {
		return gou
	}

	for _, order := range gou.Orders {
		sql := gou.sqlExpression(*order.Field)
		if sql != nil {
			gou.Query.OrderBy(sql, order.Sort)
		}
	}
	return gou
}

// buildGroups Groups
func (gou *Query) buildGroups() *Query {

	if gou.Groups == nil {
		return gou
	}

	selects := gou.mapOfSelect()
	fields := []string{}
	jsonTables := []string{}
	for _, group := range *gou.Groups {
		field, joins, updates := gou.sqlGroup(group, selects)
		fields = append(fields, field)
		jsonTables = append(jsonTables, joins...)
		// 更新已选字段
		for i, exp := range updates {
			gou.Select[i] = exp
		}
	}

	// Joins
	for _, table := range jsonTables {
		gou.Query.JoinRaw(fmt.Sprintf("JOIN %s", table))
	}

	// Update Select
	gou.buildSelect()

	// Groupby
	gou.Query.GroupByRaw(strings.Join(fields, ", "))

	return gou
}

// selectMap 读取 Select 字段映射表
func (gou *Query) mapOfSelect() map[string]FieldNode {
	res := map[string]FieldNode{}
	for i, exp := range gou.Select {
		if exp.Field != "" {
			res[gou.ID(exp)] = FieldNode{
				Index: i,
				Field: &gou.Select[i],
			}
		}
		if exp.Alias != "" {
			res[exp.Alias] = FieldNode{
				Index: i,
				Field: &gou.Select[i],
			}
		}
	}
	return res
}

// ID 字段唯一标识
func (gou *Query) ID(exp Expression) string {
	table := exp.Table
	if exp.IsModel {
		table = gou.GetTableName(table)
	}
	id := fmt.Sprintf("%s.%s.%s", table, exp.Field, exp.FullPath())
	return id
}

// NameOf 字段名称
func (gou *Query) NameOf(exp Expression) string {

	if exp.Table != "" {
		table := exp.Table
		if exp.IsModel {
			table = gou.GetTableName(table)
		}
		return fmt.Sprintf("%s.%s", table, exp.Field)
	}
	return exp.Field
}

// WrapNameOf 字段名称
func (gou *Query) WrapNameOf(exp Expression) string {
	if exp.Table != "" {
		table := exp.Table
		if exp.IsModel {
			table = gou.GetTableName(table)
		}
		return fmt.Sprintf("`%s`.`%s`", exp.Table, exp.Field)
	}

	return fmt.Sprintf("`%s`", exp.Field)
}
