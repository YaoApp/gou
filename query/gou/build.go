package gou

import (
	"fmt"

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
	for _, order := range gou.Orders {
		sql := gou.sqlExpression(*order.Field)
		if sql != nil {
			gou.Query.OrderBy(sql, order.Sort)
		}
	}
	return gou
}
