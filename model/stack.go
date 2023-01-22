package model

import (
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun"
	"github.com/yaoapp/xun/dbal/query"
)

// QueryStack 查询栈
type QueryStack struct {
	Builders []QueryStackBuilder
	Params   []QueryStackParam
	Current  int
}

// QueryStackBuilder 查询构建器
type QueryStackBuilder struct {
	Model     *Model
	Query     query.Query
	ColumnMap map[string]ColumnMap
}

// QueryStackParam QueryStack 查询参数
type QueryStackParam struct {
	QueryParam   QueryParam
	Relation     Relation
	ExportPrefix string // 字段导出前缀
}

// MakeQueryStack 创建查询栈
func MakeQueryStack() *QueryStack {
	return &QueryStack{
		Builders: []QueryStackBuilder{},
		Params:   []QueryStackParam{},
		Current:  -1,
	}
}

// NewQueryStack 新建查询栈
func NewQueryStack(param QueryParam) *QueryStack {
	return param.Query(nil)
}

// Push 添加查询器
func (stack *QueryStack) Push(builder QueryStackBuilder, param QueryStackParam) {
	stack.Builders = append(stack.Builders, builder)
	stack.Params = append(stack.Params, param)
	stack.Current = len(stack.Builders) - 1
}

// Merge 合并 Stack
func (stack *QueryStack) Merge(new *QueryStack) {
	curr := stack.Current
	for i, builder := range new.Builders {
		stack.Builders = append(stack.Builders, builder)
		stack.Params = append(stack.Params, new.Params[i])
	}
	stack.Current = curr
}

// Len 查询器数量
func (stack *QueryStack) Len() int {
	return len(stack.Builders)
}

// Builder 返回当前查询构建器
func (stack *QueryStack) Builder() *QueryStackBuilder {
	if stack.Current < 0 {
		return nil
	}
	return &stack.Builders[stack.Current]
}

// Query 返回当前查询器
func (stack *QueryStack) Query() query.Query {
	if stack.Current < 0 {
		return nil
	}
	return stack.Builders[stack.Current].Query
}

// FirstQuery 返回第一个查询器
func (stack *QueryStack) FirstQuery() query.Query {
	if len(stack.Builders) == 0 {
		return nil
	}
	return stack.Builders[0].Query
}

// QueryParam 返回当前查询参数
func (stack *QueryStack) QueryParam() QueryParam {
	if stack.Current < 0 {
		return QueryParam{}
	}
	return stack.Params[stack.Current].QueryParam
}

// Relation 返回当前查询参数
func (stack *QueryStack) Relation() Relation {
	if stack.Current < 0 {
		return Relation{}
	}
	return stack.Params[stack.Current].Relation
}

// Next 返回下一个查询器
func (stack *QueryStack) Next() int {
	next := stack.Current + 1
	if next < stack.Len() {
		stack.Current = next
		return next
	}
	return -1
}

// Run 执行查询栈
func (stack *QueryStack) Run() []maps.MapStrAny {
	res := [][]maps.MapStrAny{}
	for i, qb := range stack.Builders {
		param := stack.Params[i]
		switch param.Relation.Type {
		case "hasMany":
			stack.runHasMany(&res, qb, param)
			break
		default:
			stack.run(&res, qb, param)
		}
	}

	if len(res) < 0 {
		return nil
	}
	return res[0]
}

// Paginate 执行查询栈(分页查询)
func (stack *QueryStack) Paginate(page int, pagesize int) maps.MapStrAny {
	res := [][]maps.MapStrAny{}
	var pageInfo xun.P
	for i, qb := range stack.Builders {
		param := stack.Params[i]
		if i == 0 {
			pageInfo = stack.paginate(page, pagesize, &res, qb, param)
			continue
		}
		switch param.Relation.Type {
		case "hasMany":
			stack.runHasMany(&res, qb, param)
			break
		default:
			stack.run(&res, qb, param)
		}
	}

	if len(res) < 0 {
		return nil
	}

	response := maps.MapStrAny{}
	response["data"] = res[0]
	response["pagesize"] = pageInfo.PageSize
	response["pagecnt"] = pageInfo.TotalPages
	response["pagesize"] = pageInfo.PageSize
	response["page"] = pageInfo.CurrentPage
	response["next"] = pageInfo.NextPage
	response["prev"] = pageInfo.PreviousPage
	response["total"] = pageInfo.Total
	return response
}

func (stack *QueryStack) paginate(page int, pagesize int, res *[][]maps.MapStrAny, builder QueryStackBuilder, param QueryStackParam) xun.P {

	rows := []xun.R{}
	pageRes := builder.Query.MustPaginate(pagesize, page)
	for _, item := range pageRes.Items {
		rows = append(rows, xun.MakeR(item))
	}

	fmtRows := []maps.MapStr{}
	for _, row := range rows {
		fmtRow := maps.MapStr{}
		for key, value := range row {
			if cmap, has := builder.ColumnMap[key]; has {
				fmtRow[cmap.Export] = value
				cmap.Column.FliterOut(value, fmtRow, cmap.Export)
				continue
			}
			fmtRow[key] = value
		}

		fmtRows = append(fmtRows, fmtRow.UnDot())
	}
	*res = append(*res, fmtRows)
	stack.Next()
	return pageRes
}

func (stack *QueryStack) run(res *[][]maps.MapStrAny, builder QueryStackBuilder, param QueryStackParam) {

	limit := 100
	if param.QueryParam.Limit > 0 {
		limit = param.QueryParam.Limit
	}

	defer log.
		With(log.F{
			"sql":      builder.Query.Limit(limit).ToSQL(),
			"bindings": builder.Query.Limit(limit).GetBindings()}).
		Trace("QueryStack run()")

	rows := builder.Query.Limit(limit).MustGet()
	fmtRows := []maps.MapStr{}
	for _, row := range rows {
		fmtRow := maps.MapStr{}
		for key, value := range row {
			if cmap, has := builder.ColumnMap[key]; has {
				fmtRow[cmap.Export] = value
				cmap.Column.FliterOut(value, fmtRow, cmap.Export)
				continue
			}
			fmtRow[key] = value
		}
		fmtRows = append(fmtRows, fmtRow.UnDot())
	}
	*res = append(*res, fmtRows)
	stack.Next()
}

func (stack *QueryStack) runHasMany(res *[][]maps.MapStrAny, builder QueryStackBuilder, param QueryStackParam) {

	// 获取上次查询结果，拼接结果集ID
	rel := stack.Relation()
	foreignIDs := []interface{}{}
	prevRows := (*res)[len(*res)-1]
	for _, row := range prevRows {
		id := row.Get(rel.Foreign)
		foreignIDs = append(foreignIDs, id)
	}

	// 添加 WhereIn 查询数据
	name := rel.Key
	if param.QueryParam.Alias != "" {
		name = param.QueryParam.Alias + "." + name
	}

	// 空数据
	if len(foreignIDs) == 0 {
		*res = append(*res, []maps.MapStr{})
		varname := rel.Name
		for idx := range prevRows {
			prevRows[idx][varname] = []maps.MapStr{}
		}
		return
	}

	limit := 100
	if param.QueryParam.Limit > 0 {
		limit = param.QueryParam.Limit
	}
	builder.Query.WhereIn(name, foreignIDs).Limit(limit)
	rows := builder.Query.MustGet()

	// 格式化数据
	fmtRowMap := map[interface{}][]maps.MapStr{}
	fmtRows := []maps.MapStr{}
	for _, row := range rows {
		fmtRow := maps.MapStr{}
		for key, value := range row {
			if cmap, has := builder.ColumnMap[key]; has {
				fmtRow[cmap.Export] = value
				cmap.Column.FliterOut(value, fmtRow, cmap.Export)
				continue
			}
			fmtRow[key] = value
		}
		relKey := rel.Key
		relVal := fmtRow.Get(relKey)
		if relVal != nil {
			unDotRows := fmtRow.UnDot()
			fmtRows = append(fmtRows, unDotRows)
			if _, has := fmtRowMap[relVal]; !has {
				fmtRowMap[relVal] = []maps.MapStr{}
			}
			fmtRowMap[relVal] = append(fmtRowMap[relVal], unDotRows)
		}
	}

	// 追加到上一层
	varname := rel.Name
	// utils.Dump(fmtRows, rel.Foreign, varname, fmtRowMap, prevRows)
	for idx, prow := range prevRows {
		id := prow.Get(rel.Foreign)
		if rows, has := fmtRowMap[id]; has {
			if _, has := prevRows[idx][varname]; !has {
				prevRows[idx][varname] = []maps.MapStr{}
			}
			prevRows[idx][varname] = append(prevRows[idx][varname].([]maps.MapStr), rows...)
		}
	}

	*res = append(*res, fmtRows)
}
