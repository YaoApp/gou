package gou

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
)

// Flows 已加载工作流列表
var Flows = map[string]*Flow{}

// LoadFlow 载入数据接口
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
		Name:         name,
		Source:       source,
		Scripts:      map[string]string{},
		ScriptSource: map[string]string{},
	}
	err := helper.UnmarshalFile(input, &flow)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	// 预加载 Query DSL
	for i, node := range flow.Nodes {
		if node.Query == nil {
			continue
		}
		if engine, has := Engines[node.Engine]; has {
			flow.Nodes[i].DSL, err = engine.Load(node.Query)
			if err != nil {
				log.Printf("%s 数据分析查询解析错误", node.Engine)
				log.Println(node.Query)
			}
			continue
		}
		log.Printf("%s 数据分析引擎尚未注册", node.Engine)
	}

	Flows[name] = &flow
	return Flows[name]
}

// LoadScript 载入脚本
func (flow *Flow) LoadScript(source string, name string) *Flow {
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

	content, err := ioutil.ReadAll(input)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	flow.Scripts[name] = string(content)
	flow.ScriptSource[name] = source
	return flow
}

// Reload 重新载入API
func (flow *Flow) Reload() *Flow {
	new := LoadFlow(flow.Source, flow.Name)
	for name, source := range flow.ScriptSource {
		new.LoadScript(source, name)
	}
	flow = new
	Flows[flow.Name] = new
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
