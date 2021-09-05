package gou

import (
	"context"

	"github.com/yaoapp/kun/maps"
)

// Exec 运行flow
func (flow *Flow) Exec(args ...interface{}) interface{} {

	// 数据定义
	res := map[string]interface{}{} // 结果集
	ctx, cancel := context.WithCancel(context.Background())

	flowCtx := &FlowContext{
		Context: &ctx,
		Cancel:  cancel,
		Res:     res,
		In:      args,
	}

	// 运行工作流节点，并处理数据
	for i, node := range flow.Nodes {
		flow.ExecNode(&node, flowCtx, i-1)
	}
	return res
}

// ExecNode 运行节点
func (flow *Flow) ExecNode(node *FlowNode, ctx *FlowContext, prev int) []interface{} {
	var out interface{}
	outs := []interface{}{}
	data := maps.Map{
		"$in":  ctx.In,
		"$res": ctx.Res,
	}.Dot()
	for i := range node.Args {
		node.Args[i] = Bind(node.Args[i], data)
	}

	// 运行处理器
	if node.Process != "" {
		process := NewProcess(node.Process, node.Args...)
		out = process.Run()

		// 赋值
		if node.Name != "" {
			if node.Outs == nil || len(node.Outs) == 0 {
				ctx.Res[node.Name] = out
			} else {
				outs := []interface{}{}
				data["$out"] = out
				for _, outval := range node.Outs {
					outs = append(outs, Bind(outval, data))
				}
				ctx.Res[node.Name] = outs
			}
		}
	}

	// 运行数据处理脚本
	if node.Script != "" {
		flow.ExecScript(node.Script, outs, node, ctx, prev)
	}

	return outs
}

// ExecScript 运行数据处理脚本
func (flow *Flow) ExecScript(name string, outs []interface{}, node *FlowNode, ctx *FlowContext, prev int) []interface{} {
	_, has := flow.Scripts[name]
	if !has {
		return outs
	}

	// fmt.Println(script)
	return outs
}
