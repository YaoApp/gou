package gou

import (
	"context"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

// Exec 运行flow
func (flow *Flow) Exec(args ...interface{}) interface{} {

	res := map[string]interface{}{} // 结果集
	ctx, cancel := context.WithCancel(context.Background())

	flowCtx := &FlowContext{
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
func (ctx *FlowContext) ExtendIn(data maps.Map) maps.Map {
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
func (flow *Flow) FormatResult(ctx *FlowContext) interface{} {
	if flow.Output == nil {
		return ctx.Res
	}
	data := maps.Map{"$in": ctx.In, "$res": ctx.Res, "$global": flow.Global}
	data = ctx.ExtendIn(data).Dot()
	return share.Bind(flow.Output, data)
}

// ExecNode 运行节点
func (flow *Flow) ExecNode(node *FlowNode, ctx *FlowContext, prev int) []interface{} {
	data := maps.Map{"$in": ctx.In, "$res": ctx.Res, "$global": flow.Global}
	data = ctx.ExtendIn(data).Dot()
	var outs = []interface{}{}
	var resp interface{}

	if node.DSL != nil {
		resp, outs = flow.RunQuery(node, ctx, data)
	} else {
		resp, outs = flow.RunProcess(node, ctx, data)
	}

	_, outs = flow.RunScript(node, ctx, data, resp, outs)
	return outs
}

// RunQuery 运行 Query DSL 查询
func (flow *Flow) RunQuery(node *FlowNode, ctx *FlowContext, data maps.Map) (interface{}, []interface{}) {

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
func (flow *Flow) RunProcess(node *FlowNode, ctx *FlowContext, data maps.Map) (interface{}, []interface{}) {

	args := []interface{}{}
	outs := []interface{}{}
	var resp interface{}
	var res interface{}
	for _, arg := range node.Args {
		args = append(args, share.Bind(arg, data))
	}

	if node.Process != "" {
		process := NewProcess(node.Process, args...).WithGlobal(flow.Global).WithSID(flow.Sid)
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

// RunScript 运行数据处理脚本
func (flow *Flow) RunScript(node *FlowNode, ctx *FlowContext, data maps.Map, processResp interface{}, processOuts []interface{}) (interface{}, []interface{}) {
	var resp, res interface{}
	if node.Script == "" {
		return processResp, processOuts
	}

	name := node.Script // 全局脚本引用
	if !JavaScriptVM.Has(name) {
		name = fmt.Sprintf("flows.%s.%s", flow.Name, node.Script) // Node 脚本(兼容旧版)
	}

	in := []interface{}{}
	last := map[string]interface{}{}
	for key, value := range ctx.Res {
		last[key] = value
	}

	for _, value := range ctx.In {
		in = append(in, value)
	}

	resp, err := JavaScriptVM.
		WithProcess("*").
		WithGlobal(flow.Global).
		WithSID(flow.Sid).
		Run(name, "main", in, processResp, last, flow.Global)

	if err != nil {
		exception.New("%s 脚本错误: %s", 500, node.Script, err.Error()).Ctx(map[string]interface{}{
			"$in":     in,
			"$out":    last,
			"$res":    processOuts,
			"$global": flow.Global,
			"$sid":    flow.Sid,
		}).Throw()
	}

	if node.Outs == nil || len(node.Outs) == 0 {
		res = resp
	} else {
		var outs = []interface{}{}
		data["$out"] = resp
		data = data.Dot()
		for _, value := range node.Outs {
			outs = append(outs, share.Bind(value, data))
		}
		res = outs
		processOuts = outs
	}

	if node.Name != "" {
		ctx.Res[node.Name] = res
	}

	return resp, processOuts
}
