package gou

import (
	"strings"

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
		root = capsule.Query().Table(param.Table + " as " + param.Alias)
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
		return param.withHasOne(rel, with, qb)
	case "hasOneThrough":
		return param.withHasOneThrough(rel, with, qb)
	}

	return qbs
}

// withHasOne hasOneThrough 关联查询
func (param QueryParam) withHasOneThrough(rel Relation, with With, qb query.Query) []query.Query {
	qbs := []query.Query{}
	links := rel.Links
	prev := param
	alias := with.Name
	if param.Alias != "" {
		alias = param.Alias + "_" + alias
	}

	for _, link := range links {
		prev.Alias = alias
		qbs = prev.withHasOne(link, with, qb)
		qb = qbs[0]
		prev = link.Query
		prev.Model = link.Model
	}
	return qbs
}

// withHasOne hasOne 关联查询
func (param QueryParam) withHasOne(rel Relation, with With, qb query.Query) []query.Query {
	qbs := []query.Query{}
	withModel := Select(rel.Model)
	withParam := with.Query
	withParam.Model = rel.Model
	withParam.Table = withModel.MetaData.Table.Name
	withParam.Alias = withParam.Table
	if param.Alias != "" {
		withParam.Alias = param.Alias + "_" + withParam.Alias
	}

	key := withParam.Alias + "." + rel.Key
	if strings.Contains(rel.Key, ".") {
		key = rel.Key
	}

	foreign := param.Alias + "." + rel.Foreign
	if strings.Contains(rel.Foreign, ".") {
		foreign = rel.Foreign
	}

	if len(withParam.Wheres) == 0 && len(rel.Query.Wheres) > 0 {
		withParam.Wheres = rel.Query.Wheres
	}

	if len(withParam.Select) == 0 && len(rel.Query.Select) > 0 {
		withParam.Select = rel.Query.Select
	}

	if len(withParam.Wheres) > 0 || len(withParam.Orders) > 0 {

		withSubParam := withParam
		withSubParam.Alias = withParam.Table

		// SubQuery
		qb.LeftJoinSub(func(sub query.Query) {

			sub.Table(withSubParam.Table)

			// Select
			if len(withParam.Select) == 0 {
				withSubParam.Select = withModel.ColumnNames // Select All
			} else if !withParam.hasSelectColumn(rel.Key) {
				withSubParam.Select = append(withParam.Select, rel.Key)
			}
			sub.SelectAppend(withModel.FliterSelect("", withSubParam.Select)...)

			// Where
			for _, where := range withSubParam.Wheres {
				withSubParam.Where(where, sub, withModel)
			}
		}, withParam.Alias, key, "=", foreign)

		withParam.Wheres = []QueryWhere{}
		withParam.Orders = []QueryOrder{}
	} else {

		// 直接Join
		qb.LeftJoin(
			withParam.Table+" as "+withParam.Alias,
			key,
			"=",
			foreign,
		)
	}

	qbs = withParam.Query(qb)
	return qbs
}

// Where 查询条件
func (param QueryParam) Where(where QueryWhere, qb query.Query, mod *Model) {

	alias := param.Alias
	m := mod
	if where.Rel != "" {

		if strings.Contains(where.Rel, ".") { // mother.friends

			rels := strings.Split(where.Rel, ".")
			rel, has := mod.MetaData.Relations[rels[0]]
			if !has {
				return
			}

			has = false
			for _, link := range rel.Links {
				if link.Model == rels[1] {
					has = true
					rel = link
					break
				}
			}

			if !has {
				return
			}

			alias = strings.ReplaceAll(where.Rel, ".", "_")
			if param.Alias != "" {
				alias = param.Alias + "_" + alias
			}
			m = Select(rel.Model)

		} else { // manu
			rel, has := mod.MetaData.Relations[where.Rel]
			if !has {
				return
			}

			alias = where.Rel
			if param.Alias != "" {
				alias = param.Alias + "_" + alias
			}

			m = Select(rel.Model)
		}

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

// hasSelectColumn 检查字段是否已存在
func (param QueryParam) hasSelectColumn(column interface{}) bool {
	for _, col := range param.Select {
		if col == column {
			return true
		}
	}
	return false
}
