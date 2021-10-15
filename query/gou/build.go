package gou

import "fmt"

// buildSelect Select
func (gou *Query) buildSelect() *Query {
	fields := []interface{}{}
	for _, exp := range gou.Select {
		sql := gou.sqlSelect(exp)
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
