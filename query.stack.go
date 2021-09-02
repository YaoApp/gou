package gou

import (
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
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
	QueryParam QueryParam
	Relation   Relation
}

// NewQueryStack 创建查询栈
func NewQueryStack() *QueryStack {
	return &QueryStack{
		Builders: []QueryStackBuilder{},
		Params:   []QueryStackParam{},
		Current:  -1,
	}
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

// Prev 返回上一个查询器
func (stack *QueryStack) Prev() int {
	prev := stack.Current - 1
	if prev > 0 {
		stack.Current = prev
		return prev
	}
	return -1
}

// PrevModel 上一个查询的 Model
func (stack *QueryStack) PrevModel() *Model {
	prev := stack.Current - 1
	if prev >= 0 {
		return stack.Builders[prev].Model
	}
	return nil
}

// PrevParam 上一个查询的 Param
func (stack *QueryStack) PrevParam() *QueryStackParam {
	prev := stack.Current - 1
	if prev >= 0 {
		return &stack.Params[prev]
	}
	return nil
}

// Run 执行查询栈
func (stack *QueryStack) Run() {
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
}

func (stack *QueryStack) run(res *[][]maps.MapStrAny, builder QueryStackBuilder, param QueryStackParam) {
	rows := builder.Query.MustGet()
	fmtRows := []maps.MapStr{}
	for _, row := range rows {
		fmtRow := maps.MapStr{}
		for key, value := range row {
			if cmap, has := builder.ColumnMap[key]; has {
				prefix := cmap.Model.Name + "."
				name := prefix + cmap.Column.Name
				fmtRow[name] = value
				cmap.Column.FliterOut(value, fmtRow, prefix)
				continue
			}
			fmtRow[key] = value
		}

		fmtRows = append(fmtRows, fmtRow)
	}
	*res = append(*res, fmtRows)
	stack.Next()
}

func (stack *QueryStack) runHasMany(res *[][]maps.MapStrAny, builder QueryStackBuilder, param QueryStackParam) {

	// 获取上次查询结果，拼接结果集ID
	rel := stack.Relation()
	foreignIDs := []interface{}{}
	prevModel := stack.PrevModel().Name
	prevRows := (*res)[len(*res)-1]
	for _, row := range prevRows {
		id := row.Get(prevModel + "." + rel.Foreign)
		foreignIDs = append(foreignIDs, id)
	}

	// 添加 WhereIn 查询数据
	name := rel.Key
	if param.QueryParam.Alias != "" {
		name = param.QueryParam.Alias + "." + name
	}
	builder.Query.WhereIn(name, foreignIDs)
	rows := builder.Query.MustGet()

	// 格式化数据
	fmtRowMap := map[interface{}]maps.MapStr{}
	fmtRows := []maps.MapStr{}
	currModel := builder.Model.Name
	for _, row := range rows {
		fmtRow := maps.MapStr{}
		for key, value := range row {
			if cmap, has := builder.ColumnMap[key]; has {
				prefix := cmap.Model.Name + "."
				name := prefix + cmap.Column.Name
				fmtRow[name] = value
				cmap.Column.FliterOut(value, fmtRow, prefix)
				continue
			}
			fmtRow[key] = value
		}

		relKey := currModel + "." + rel.Key
		relVal := fmtRow.Get(relKey)
		if relVal != nil {
			fmtRows = append(fmtRows, fmtRow)
			fmtRowMap[relVal] = fmtRow
		}
	}

	// 追加到上一层
	varname := currModel
	for idx, prow := range prevRows {
		id := prow.Get(prevModel + "." + rel.Foreign)
		if row, has := fmtRowMap[id]; has {
			if _, has := prevRows[idx][varname]; !has {
				prevRows[idx][varname] = []maps.MapStr{}
			}
			prevRows[idx][varname] = append(prevRows[idx][varname].([]maps.MapStr), row)
		}
	}

	*res = append(*res, fmtRows)
	utils.Dump(res)
}
