package gou

import (
	"strings"

	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
)

// Query 构建查询栈
func (param QueryParam) Query(root query.Query, prefix string) []query.Query {
	qbs := []query.Query{}
	if param.Model == "" {
		return qbs
	}

	mod := Select(param.Model)
	param.Table = mod.MetaData.Table.Name
	if param.Alias == "" {
		param.Alias = prefix + param.Table
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
		param.With(with, root, mod, prefix)
	}

	return qbs
}

// With 关联查询
func (param QueryParam) With(with With, qb query.Query, mod *Model, prefix string) []query.Query {
	qbs := []query.Query{}
	rel, has := mod.MetaData.Relations[with.Name]
	if !has {
		return qbs
	}

	switch rel.Type {
	case "hasOne":
		return param.withHasOne(rel, with, qb, mod, prefix)
	case "hasOneThrough":
		return param.withHasOneThrough(rel, with, qb, mod)
	}

	return qbs
}

func (param QueryParam) withHasOneThrough(rel Relation, with With, qb query.Query, mod *Model) []query.Query {
	qbs := []query.Query{}
	links := rel.Links
	for _, link := range links {
		qbs = param.withHasOne(link, with, qb, mod, with.Name+"_")
		qb = qbs[0]
	}
	return qbs
}

// withHasOne hasOne 关联查询
func (param QueryParam) withHasOne(rel Relation, with With, qb query.Query, mod *Model, prefix string) []query.Query {
	qbs := []query.Query{}
	hasOneMod := Select(rel.Model)
	hasOneQueryParam := with.Query
	hasOneQueryParam.Model = rel.Model
	hasOneQueryParam.Table = hasOneMod.MetaData.Table.Name
	hasOneQueryParam.Alias = prefix + hasOneQueryParam.Table

	key := hasOneQueryParam.Alias + "." + rel.Key
	if strings.Contains(rel.Key, ".") {
		key = rel.Key
	}
	foreign := param.Alias + "." + rel.Foreign
	if strings.Contains(rel.Foreign, ".") {
		foreign = rel.Foreign
	}

	if len(hasOneQueryParam.Wheres) == 0 && len(rel.Query.Wheres) > 0 {
		hasOneQueryParam.Wheres = rel.Query.Wheres
	}

	if len(hasOneQueryParam.Select) == 0 && len(rel.Query.Select) > 0 {
		hasOneQueryParam.Select = rel.Query.Select
	}

	if len(hasOneQueryParam.Wheres) > 0 || len(hasOneQueryParam.Orders) > 0 {

		hasOneQueryParamSub := hasOneQueryParam
		hasOneQueryParamSub.Alias = hasOneQueryParam.Table

		// SubQuery
		qb.LeftJoinSub(func(sub query.Query) {

			sub.Table(hasOneQueryParamSub.Table)

			// Select
			if len(hasOneQueryParam.Select) == 0 {
				hasOneQueryParamSub.Select = hasOneMod.ColumnNames // Select All
			} else if !hasOneQueryParam.hasSelectColumn(rel.Key) {
				hasOneQueryParamSub.Select = append(hasOneQueryParam.Select, rel.Key)
			}
			sub.SelectAppend(hasOneMod.FliterSelect("", hasOneQueryParamSub.Select)...)

			// Where
			for _, where := range hasOneQueryParamSub.Wheres {
				hasOneQueryParamSub.Where(where, sub, hasOneMod)
			}

		}, hasOneQueryParam.Alias, key, "=", foreign)

		hasOneQueryParam.Wheres = []QueryWhere{}
		hasOneQueryParam.Orders = []QueryOrder{}
	} else {

		// 直接Join
		qb.LeftJoin(
			hasOneQueryParam.Table+" as "+hasOneQueryParam.Alias,
			key,
			"=",
			foreign,
		)
	}

	qbs = hasOneQueryParam.Query(qb, "")
	return qbs
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

// hasSelectColumn 检查字段是否已存在
func (param QueryParam) hasSelectColumn(column interface{}) bool {
	for _, col := range param.Select {
		if col == column {
			return true
		}
	}
	return false
}
