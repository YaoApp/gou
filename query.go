package gou

import (
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
)

// Query 构建查询栈
func (param QueryParam) Query(root query.Query) []query.Query {
	qbs := []query.Query{}
	if param.Model != "" {
		mod := Select(param.Model)
		param.Table = mod.MetaData.Table.Name
		if param.Alias == "" {
			param.Alias = param.Table
		}

		qb := capsule.Query().Table(param.Table + " as " + param.Table)

		// Select Select All
		if len(param.Select) == 0 {
			param.Select = mod.ColumnNames
		}
		qb.Select(mod.FliterSelect(param.Alias, param.Select)...)

		// Where
		for _, where := range param.Wheres {
			param.Where(where, qb, mod)
		}
		qbs = append(qbs, qb)
	}
	return qbs
}

// Where 查询条件
func (param QueryParam) Where(where QueryWhere, qb query.Query, mod *Model) {
	if where.Method == "" {
		where.Method = "where"
	}

	// Sub wheres
	if where.Wheres != nil {
		qb.Where(func(sub query.Query) {
			for _, subwhere := range where.Wheres {
				param.Where(subwhere, sub, mod)
			}
		})
		return
	}

	column := mod.FliterWhere(param.Alias, where.Column)
	switch where.Method {
	case "where":
		qb.Where(column, where.Value)
		break
	case "orWhere":
		qb.OrWhere(column, where.Value)
		break
	}
}
