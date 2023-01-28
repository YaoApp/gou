package flow

import (
	"context"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

// Exec 运行flow
func (flow *Flow) Exec(args ...interface{}) interface{} {

	res := map[string]interface{}{} // 结果集
	ctx, cancel := context.WithCancel(context.Background())

	flowCtx := &Context{
		Context: &ctx,
		Cancel:  cancel,
		Res:     res,
		In:      args,
	}

	// 运行工作流节点，并处理数据
	flowProcess := "flows." + flow.Name
	for i, node := range flow.Nodes {

		// 死循环检查
		if strings.HasPrefix(node.Process, flowProcess) {
			exception.New("不能调用自身工作流(%s)", 400, node.Process)
		}

		// 执行解析器
		flow.ExecNode(&node, flowCtx, i-1)
	}

	// 结果集输出处理
	return flow.FormatResult(flowCtx)
}

// ExtendIn 展开 in 参数
func (ctx *Context) ExtendIn(data maps.Map) maps.Map {
	if len(ctx.In) < 1 {
		return data
	}

	item, ok := ctx.In[0].(map[string]interface{})
	if !ok {
		return data
	}
	for key, value := range item {
		data["$"+key] = value
	}
	return data
}

// FormatResult 结果集格式化输出
func (flow *Flow) FormatResult(ctx *Context) interface{} {
	if flow.Output == nil {
		return ctx.Res
	}
	data := maps.Map{"$in": ctx.In, "$res": ctx.Res, "$global": flow.Global}
	data = ctx.ExtendIn(data).Dot()
	return share.Bind(flow.Output, data)
}

// ExecNode 运行节点
func (flow *Flow) ExecNode(node *Node, ctx *Context, prev int) []interface{} {
	data := maps.Map{"$in": ctx.In, "$res": ctx.Res, "$global": flow.Global}
	data = ctx.ExtendIn(data).Dot()
	var outs = []interface{}{}

	if node.DSL != nil {
		_, outs = flow.RunQuery(node, ctx, data)
		return outs
	}

	_, outs = flow.RunProcess(node, ctx, data)
	return outs
}

// RunQuery 运行 Query DSL 查询
func (flow *Flow) RunQuery(node *Node, ctx *Context, data maps.Map) (interface{}, []interface{}) {

	var res interface{}
	outs := []interface{}{}
	resp := node.DSL.Run(data)

	if node.Outs == nil || len(node.Outs) == 0 {
		res = resp
	} else {
		data["$out"] = resp
		data = data.Dot()
		for _, value := range node.Outs {
			outs = append(outs, share.Bind(value, data))
		}
		res = outs
	}

	if node.Name != "" {
		ctx.Res[node.Name] = res
	}
	return resp, outs
}

// RunProcess 运行处理器
func (flow *Flow) RunProcess(node *Node, ctx *Context, data maps.Map) (interface{}, []interface{}) {

	args := []interface{}{}
	outs := []interface{}{}
	var resp interface{}
	var res interface{}
	for _, arg := range node.Args {
		args = append(args, share.Bind(arg, data))
	}

	if node.Process != "" {
		process := process.New(node.Process, args...).WithGlobal(flow.Global).WithSID(flow.Sid)
		resp = process.Run()

		// 当使用 Session start 设置SID时
		// 设置SID (这个逻辑需要优化)
		if flow.Sid == "" && process.Sid != "" {
			flow.WithSID(process.Sid)
		}
	}

	if node.Outs == nil || len(node.Outs) == 0 {
		res = resp
	} else {
		data["$out"] = resp
		data = data.Dot()
		for _, value := range node.Outs {
			outs = append(outs, share.Bind(value, data))
		}
		res = outs
	}

	if node.Name != "" {
		ctx.Res[node.Name] = res
	}
	return resp, outs
}
