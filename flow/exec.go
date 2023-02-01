package flow

import (
	"context"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/maps"
)

// Exec execute flow
func (flow *Flow) Exec(args ...interface{}) (interface{}, error) {

	res := map[string]interface{}{} // 结果集
	ctx, cancel := context.WithCancel(context.Background())

	flowCtx := &Context{
		Context: &ctx,
		Cancel:  cancel,
		Res:     res,
		In:      args,
	}

	flowProcess := "flows." + flow.Name
	for i, node := range flow.Nodes {

		if strings.HasPrefix(node.Process, flowProcess) {
			return nil, fmt.Errorf("cannot call self flow(%s)", node.Process)
		}

		_, err := flow.ExecNode(&node, flowCtx, i-1)
		if err != nil {
			return nil, err
		}
	}

	return flow.FormatResult(flowCtx)
}

// ExtendIn Extend params
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

// FormatResult format result
func (flow *Flow) FormatResult(ctx *Context) (interface{}, error) {
	if flow.Output == nil {
		return ctx.Res, nil
	}
	data := maps.Map{"$in": ctx.In, "$res": ctx.Res, "$global": flow.Global}
	data = ctx.ExtendIn(data).Dot()
	return helper.Bind(flow.Output, data), nil
}

// ExecNode Execute node
func (flow *Flow) ExecNode(node *Node, ctx *Context, prev int) ([]interface{}, error) {
	data := maps.Map{"$in": ctx.In, "$res": ctx.Res, "$global": flow.Global}
	data = ctx.ExtendIn(data).Dot()
	var outs = []interface{}{}
	var err error

	if node.DSL != nil {
		_, outs, err = flow.RunQuery(node, ctx, data)
		return outs, err
	}

	_, outs, err = flow.RunProcess(node, ctx, data)
	return outs, err
}

// RunQuery execute Query DSL
func (flow *Flow) RunQuery(node *Node, ctx *Context, data maps.Map) (interface{}, []interface{}, error) {

	var res interface{}
	outs := []interface{}{}
	resp := node.DSL.Run(data)

	if node.Outs == nil || len(node.Outs) == 0 {
		res = resp
	} else {
		data["$out"] = resp
		data = data.Dot()
		for _, value := range node.Outs {
			outs = append(outs, helper.Bind(value, data))
		}
		res = outs
	}

	if node.Name != "" {
		ctx.Res[node.Name] = res
	}
	return resp, outs, nil
}

// RunProcess exec process
func (flow *Flow) RunProcess(node *Node, ctx *Context, data maps.Map) (interface{}, []interface{}, error) {

	args := []interface{}{}
	outs := []interface{}{}
	var resp interface{}
	var res interface{}
	for _, arg := range node.Args {
		args = append(args, helper.Bind(arg, data))
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
			outs = append(outs, helper.Bind(value, data))
		}
		res = outs
	}

	if node.Name != "" {
		ctx.Res[node.Name] = res
	}
	return resp, outs, nil
}
