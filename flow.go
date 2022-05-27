package gou

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yaoapp/kun/log"

	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/runtime"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/utils"
)

// Flows 已加载工作流列表
var Flows = map[string]*Flow{}

// Yao JavaScript 运行环境
var Yao = runtime.Yao(1)

func init() {
	Yao.
		AddFunction("Process", func(global map[string]interface{}, sid string, args ...interface{}) interface{} {
			if len(args) < 1 {
				return map[string]interface{}{"code": 400, "message": "缺少处理器名称"}
			}
			name, ok := args[0].(string)
			if !ok {
				return map[string]interface{}{"code": 400, "message": "处理器参数不正确"}
			}

			in := []interface{}{}
			if len(args) > 1 {
				in = args[1:]
			}

			value, err := NewProcess(name, in...).WithGlobal(global).WithSID(sid).Exec()
			if err != nil {
				return map[string]interface{}{"code": 500, "message": err.Error()}
			}
			return value
		}).
		AddObject("console", map[string]func(global map[string]interface{}, sid string, args ...interface{}) interface{}{
			"log": func(global map[string]interface{}, sid string, args ...interface{}) interface{} {
				utils.Dump(args...)
				return nil
			},
		}).
		Init()
}

// LoadFlowReturn 加载数据流
func LoadFlowReturn(source string, name string) (flow *Flow, err error) {
	defer func() { err = exception.Catch(recover()) }()
	flow = LoadFlow(source, name)
	return flow, nil
}

// LoadFlow 加载数据流
func LoadFlow(source string, name string) *Flow {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	flow := Flow{
		Name:    name,
		Source:  source,
		Scripts: map[string]string{},
	}
	err := helper.UnmarshalFile(input, &flow)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	flow.Prepare()
	Flows[name] = &flow
	return Flows[name]
}

// Prepare 预加载 Query DSL
func (flow *Flow) Prepare() {

	if flow.Scripts == nil {
		flow.Scripts = map[string]string{}
	}

	for i, node := range flow.Nodes {
		if node.Query == nil {
			continue
		}

		if node.Engine == "" {
			log.Error("Node %s: 未指定数据查询分析引擎", node.Name)
			continue
		}

		if engine, has := query.Engines[node.Engine]; has {
			var err error
			flow.Nodes[i].DSL, err = engine.Load(node.Query)
			if err != nil {
				log.With(log.F{"query": node.Query}).Error("Node %s: %s 数据分析查询解析错误", node.Name, node.Engine)
			}
			continue
		}
		log.Error("Node %s: %s 数据分析引擎尚未注册", node.Name, node.Engine)
	}
}

// LoadScriptReturn 加载载入脚本
func (flow *Flow) LoadScriptReturn(source string, name string) (new *Flow, err error) {
	defer func() { err = exception.Catch(recover()) }()
	new = flow.LoadScript(source, name)
	return new, nil
}

// LoadScript 载入脚本
func (flow *Flow) LoadScript(source string, name string) *Flow {
	var input io.Reader = nil
	name = fmt.Sprintf("flows.%s.%s", flow.Name, name)
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		// err := JavaScriptVM.Load(filename, name)
		err := Yao.Load(filename, name)
		if err != nil {
			log.Error("加载数据脚本失败 %s: %s", filename, name)
		}
	} else {
		input = strings.NewReader(source)
		// err := JavaScriptVM.LoadSource("", input, name)
		err := Yao.LoadReader(input, name)
		if err != nil {
			log.Error("加载数据脚本失败 %s", name)
		}
	}
	flow.Scripts[name] = source
	return flow
}

// Reload 重新载入API
func (flow *Flow) Reload() *Flow {
	new := LoadFlow(flow.Source, flow.Name)
	for name, source := range flow.Scripts {
		new.LoadScript(source, name)
	}
	flow = new
	Flows[flow.Name] = new
	return flow
}

// WithSID 设定会话ID
func (flow *Flow) WithSID(sid string) *Flow {
	flow.Sid = sid
	return flow
}

// WithGlobal 设定全局变量
func (flow *Flow) WithGlobal(global map[string]interface{}) *Flow {
	flow.Global = global
	return flow
}

// SelectFlow 读取已加载Flow
func SelectFlow(name string) *Flow {
	flow, has := Flows[name]
	if !has {
		exception.New(
			fmt.Sprintf("Flow:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return flow
}
