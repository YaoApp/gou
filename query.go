package gou

import (
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
)

// Query 构建查询栈
func (param QueryParam) Query(root query.Query) []query.Query {
	qbs := []query.Query{}
	if param.Model == "" {
		return qbs
	}

	mod := Select(param.Model)
	param.Table = mod.MetaData.Table.Name
	if param.Alias == "" {
		param.Alias = param.Table
	}

	if root == nil {
		root = capsule.Query().Table(param.Table + " as " + param.Table)
	}

	// Select
	if len(param.Select) == 0 {
		param.Select = mod.ColumnNames // Select All
	}
	root.SelectAppend(mod.FliterSelect(param.Alias, param.Select)...)

	// Where
	for _, where := range param.Wheres {
		param.Where(where, root, mod)
	}

	qbs = append(qbs, root)

	// Withs
	for _, with := range param.Withs {
		param.With(with, root, mod)
	}

	return qbs
}

// With 关联查询
func (param QueryParam) With(with With, qb query.Query, mod *Model) []query.Query {
	qbs := []query.Query{}
	rel, has := mod.MetaData.Relations[with.Name]
	if !has {
		return qbs
	}

	switch rel.Type {
	case "hasOne":
		return param.withHasOne(rel, with, qb, mod)
	}

	return qbs
}

// withHasOne hasOne 关联查询
func (param QueryParam) withHasOne(rel Relation, with With, qb query.Query, mod *Model) []query.Query {
	qbs := []query.Query{}
	hasOneMod := Select(rel.Model)
	hasOneQueryParam := with.Query
	hasOneQueryParam.Model = rel.Model
	hasOneQueryParam.Alias = with.Name
	hasOneQueryParam.Table = hasOneMod.MetaData.Table.Name

	if len(hasOneQueryParam.Wheres) > 0 || len(hasOneQueryParam.Orders) > 0 {

		// SubQuery
		qb.LeftJoinSub(func(sub query.Query) {
			sub.Table(hasOneQueryParam.Table)

			// Select
			if len(hasOneQueryParam.Select) == 0 {
				hasOneQueryParam.Select = hasOneMod.ColumnNames // Select All
			} else if !hasOneQueryParam.hasSelectColumn(rel.Key) {
				hasOneQueryParam.Select = append(hasOneQueryParam.Select, rel.Key)
			}
			sub.SelectAppend(hasOneMod.FliterSelect("", hasOneQueryParam.Select)...)

			// Where
			for _, where := range hasOneQueryParam.Wheres {
				hasOneQueryParam.Where(where, sub, hasOneMod)
			}

		}, hasOneQueryParam.Alias,
			hasOneQueryParam.Alias+"."+rel.Key, "=",
			param.Alias+"."+rel.Foreign)

		hasOneQueryParam.Wheres = []QueryWhere{}
		hasOneQueryParam.Orders = []QueryOrder{}
	} else {

		// 直接Join
		qb.LeftJoin(
			hasOneQueryParam.Table+" as "+hasOneQueryParam.Alias,
			hasOneQueryParam.Alias+"."+rel.Key,
			"=",
			param.Alias+"."+rel.Foreign,
		)
	}

	qbs = hasOneQueryParam.Query(qb)
	return qbs
}

func (param QueryParam) hasSelectColumn(column interface{}) bool {
	for _, col := range param.Select {
		if col == column {
			return true
		}
	}
	return false
}

// Where 查询条件
func (param QueryParam) Where(where QueryWhere, qb query.Query, mod *Model) {

	alias := param.Alias
	m := mod
	if where.Rel != "" {

		// 忽略未关联关系查询
		if _, has := param.Withs[where.Rel]; !has {
			return
		}

		rel, has := mod.MetaData.Relations[where.Rel]
		if !has {
			return
		}

		alias = where.Rel
		m = Select(rel.Model)
	}

	if where.Method == "" {
		where.Method = "where"
	}

	// Sub wheres
	if where.Wheres != nil {
		qb.Where(func(sub query.Query) {
			for _, subwhere := range where.Wheres {
				param.Where(subwhere, sub, m)
			}
		})
		return
	}

	column := m.FliterWhere(alias, where.Column)
	switch where.Method {
	case "where":
		qb.Where(column, where.Value)
		break
	case "orWhere":
		qb.OrWhere(column, where.Value)
		break
	}
}
