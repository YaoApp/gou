package model

import (
	"strings"

	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
)

var opmap map[string]string = map[string]string{
	"like": "like",
	"eq":   "=",
	"ne":   "<>",
	"gt":   ">",
	"lt":   "<",
	"ge":   ">=",
	"le":   "<=",
}

// NewQuery 新建查询栈
func (param QueryParam) NewQuery() *QueryStack {
	return param.Query(nil)
}

// Query 构建查询栈(本版先实现，下一版本根据实际应用场景迭代)
func (param QueryParam) Query(stack *QueryStack, stackParams ...QueryStackParam) *QueryStack {

	if param.Model == "" {
		return stack
	}
	mod := Select(param.Model)
	param.Table = mod.MetaData.Table.Name
	if param.Alias == "" {
		param.Alias = param.Table
	}

	exportPrefix := param.Export
	if stack == nil {
		stack = MakeQueryStack()
		stackParam := QueryStackParam{
			QueryParam: param,
		}
		if len(stackParams) > 0 {
			stackParam = stackParams[0]
		}

		builder := QueryStackBuilder{
			Model:     mod,
			Query:     capsule.Query().Table(param.Table + " as " + param.Alias),
			ColumnMap: map[string]ColumnMap{},
		}

		exportPrefix = ""
		stack.Push(builder, stackParam)
	}

	// Select
	if len(param.Select) == 0 {
		param.Select = mod.ColumnNames // Select All
	}

	selects := mod.Filterselect(param.Alias, param.Select, stack.Builder().ColumnMap, exportPrefix)
	stack.Query().SelectAppend(selects...)

	// Where
	for _, where := range param.Wheres {
		param.Where(where, stack.Query(), mod)
	}

	// 软删除
	if mod.MetaData.Option.SoftDeletes {
		param.Where(QueryWhere{Column: "deleted_at", OP: "null"}, stack.Query(), mod)
	}

	// Order
	for _, order := range param.Orders {
		param.Order(order, stack.Query(), mod)
	}

	// Limit
	if param.Limit > 0 {
		stack.Query().Limit(param.Limit)
	}

	// Withs
	for name, with := range param.Withs {
		param.With(name, stack, with, mod)
	}

	return stack
}

// With 关联查询
func (param QueryParam) With(name string, stack *QueryStack, with With, mod *Model) {
	rel, has := mod.MetaData.Relations[name]
	if !has {
		return
	}

	rel.Name = name
	switch rel.Type {
	case "hasOne":
		param.Export = rel.Name
		param.withHasOne(stack, rel, with)
		return
	case "hasOneThrough":
		param.withHasOneThrough(stack, rel, with)
		return
	case "hasMany":
		param.withHasMany(stack, rel, with)
		return

	}

}

// withHasOne hasOneThrough 关联查询
func (param QueryParam) withHasOneThrough(stack *QueryStack, rel Relation, with With) {
	links := rel.Links
	prev := param
	alias := rel.Name
	if param.Alias != "" {
		alias = param.Alias + "_" + alias
	}
	length := len(links)
	for i, link := range links {
		prev.Export = rel.Name + "." + link.Model
		if i == length-1 {
			prev.Export = rel.Name
		}
		prev.Alias = alias
		prev.withHasOne(stack, link, with)
		prev = link.Query
		prev.Model = link.Model
	}
}

// withHasOne hasOne 关联查询 临时BUG修复，这里整个逻辑需要优化
func (param QueryParam) withHasOne(stack *QueryStack, rel Relation, with With) {
	withModel := Select(rel.Model)
	withParam := with.Query
	withParam.Model = rel.Model
	withParam.Table = withModel.MetaData.Table.Name
	alias := rel.Name
	if rel.Name == "" {
		alias = withParam.Table
	}
	withParam.Alias = alias + "__rel__" // 临时BUG修复，这里整个逻辑需要优化
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

		// 临时BUG修复，这里整个逻辑需要优化
		foreignArr := strings.Split(rel.Foreign, ".")
		foreignLen := len(foreignArr)
		tab := strings.Join(foreignArr[0:foreignLen-1], ".")
		field := foreignArr[foreignLen-1]
		if tab != param.Table {
			foreign = tab + "__rel__" + "." + field
		}
		// fmt.Println(tab, param.Table, rel.Foreign, foreign)
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
		stack.Query().LeftJoinSub(func(sub query.Query) {

			sub.Table(withSubParam.Table)

			// Select
			if len(withParam.Select) == 0 {
				withSubParam.Select = withModel.ColumnNames // Select All
			} else if !withParam.hasSelectColumn(rel.Key) {
				withSubParam.Select = append(withParam.Select, rel.Key)
			}

			// 软删除
			if withModel.MetaData.Option.SoftDeletes && !withSubParam.hasSelectColumn("deleted_at") {
				withSubParam.Select = append(withSubParam.Select, "deleted_at")
			}

			selects := withModel.Filterselect("", withSubParam.Select, nil, "")
			sub.SelectAppend(selects...)

			// Where
			for _, where := range withSubParam.Wheres {
				withSubParam.Where(where, sub, withModel)
			}
		}, withParam.Alias, key, "=", foreign)

		withParam.Wheres = []QueryWhere{}
		withParam.Orders = []QueryOrder{}
	} else {

		// 直接Join
		stack.Query().LeftJoin(
			withParam.Table+" as "+withParam.Alias,
			key,
			"=",
			foreign,
		)
	}

	withParam.Export = param.Export
	withParam.Query(stack)
}

// Order 排序条件
func (param QueryParam) Order(order QueryOrder, qb query.Query, mod *Model) {

	alias := param.Alias
	m := mod
	if order.Rel != "" {

		if strings.Contains(order.Rel, ".") { // mother.friends

			rels := strings.Split(order.Rel, ".")
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

			alias = strings.ReplaceAll(order.Rel, ".", "_") + "__rel__" //  这里逻辑需要重构
			if param.Alias != "" {
				alias = param.Alias + "_" + alias
			}
			m = Select(rel.Model)

		} else { // manu
			rel, has := mod.MetaData.Relations[order.Rel]
			if !has {
				return
			}

			alias = order.Rel + "__rel__" //  这里逻辑需要重构
			if param.Alias != "" {
				alias = param.Alias + "_" + alias
			}

			m = Select(rel.Model)
		}

	}

	if order.Option == "" {
		order.Option = "asc"
	}

	column := m.FliterWhere(alias, order.Column)
	qb.OrderBy(column, order.Option)
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

			alias = strings.ReplaceAll(where.Rel, ".", "_") + "__rel__" //  这里逻辑需要重构
			if param.Alias != "" {
				alias = param.Alias + "_" + alias
			}
			m = Select(rel.Model)

		} else { // manu
			rel, has := mod.MetaData.Relations[where.Rel]
			if !has {
				return
			}

			alias = where.Rel + "__rel__" //  这里逻辑需要重构
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
	switch strings.ToLower(where.Method) {
	case "where":
		switch where.OP {
		case "null":
			qb.WhereNull(column)
			break
		case "notnull":
			qb.WhereNotNull(column)
			break
		case "match":
			if value, ok := where.Value.(string); ok {
				qb.Where(column, "like", "%"+value+"%")
			}
			break
		case "in":
			if value, ok := where.Value.(string); ok {
				where.Value = strings.Split(value, ",")
			}
			qb.WhereIn(column, where.Value)
			break
		default:
			op, has := opmap[where.OP]
			if !has {
				op = "="
			}
			qb.Where(column, op, where.Value)
		}
		break
	case "orwhere":
		switch where.OP {
		case "null":
			qb.OrWhereNull(column)
			break
		case "notnull":
			qb.OrWhereNotNull(column)
			break
		case "match":
			if value, ok := where.Value.(string); ok {
				qb.OrWhere(column, "like", "%"+value+"%")
			}
			break
		case "in":
			if value, ok := where.Value.(string); ok {
				where.Value = strings.Split(value, ",")
			}
			qb.OrWhereIn(column, where.Value)
		default:
			op, has := opmap[where.OP]
			if !has {
				op = "="
			}
			qb.OrWhere(column, op, where.Value)
		}
		break
	}
}

// withHasMany hasMany 关联查询
func (param QueryParam) withHasMany(stack *QueryStack, rel Relation, with With) {

	withModel := Select(rel.Model)
	withParam := with.Query
	withParam.Model = rel.Model
	withParam.Table = withModel.MetaData.Table.Name
	withParam.Alias = withParam.Table
	withParam.Alias = withParam.Table
	if param.Alias != "" {
		withParam.Alias = param.Alias + "_" + withParam.Alias
	}

	// Select & 添加关联主键
	if len(withParam.Select) == 0 {
		withParam.Select = withModel.ColumnNames // Select all
	} else if !withParam.hasSelectColumn(rel.Key) {
		withParam.Select = append(withParam.Select, rel.Key) // 添加关联主键
	}

	// 添加关联外键
	if !param.hasSelectColumn(rel.Foreign) {
		mod := Select(param.Model)
		selects := mod.Filterselect(param.Alias, []interface{}{rel.Foreign}, stack.Builder().ColumnMap, "")
		stack.Query().SelectAppend(selects...)
	}

	stackParam := QueryStackParam{
		QueryParam: withParam,
		Relation:   rel,
	}
	newStack := withParam.Query(nil, stackParam)
	stack.Merge(newStack)
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
