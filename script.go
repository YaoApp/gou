package gou

import (
	"fmt"
	"strings"

	"github.com/robertkrimen/otto"
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/parser"
)

// Scripts 已加载脚本
var Scripts = map[string]*Script{}

// NewJS 创建JS脚本
func NewJS() ScriptVM {
	return JavaScript{Otto: otto.New()}
}

// LoadScript 加载数据处理脚本
func LoadScript(file string, source string, namespace string) error {
	program, err := parser.ParseFile(nil, file, source, 0)
	if err != nil {
		return err
	}
	script := Script{File: file, Source: source, Functions: map[string]Function{}}
	ast.Walk(script, program)
	Scripts[namespace] = &script
	return nil
}

// Enter 解析脚本文件
func (script Script) Enter(n ast.Node) ast.Visitor {
	if v, ok := n.(*ast.FunctionStatement); ok && v != nil {
		name := v.Function.Name.Name
		idx := int(n.Idx0())
		lines := strings.Split(script.Source[:idx], "\n")
		line := len(lines)
		script.Functions[name] = Function{
			Name:      name,
			Source:    v.Function.Source,
			Line:      line,
			NumOfArgs: len(v.Function.ParameterList.List),
		}
	}
	return script
}

// Exit 解析脚本文件
func (script Script) Exit(n ast.Node) {}

// Compile 脚本预编译
func (vm JavaScript) Compile(script *Script) error {
	// source := ""
	for name, f := range script.Functions {
		numOfArgs := f.NumOfArgs
		argNames := []string{}
		for i := 0; i < numOfArgs; i++ {
			argNames = append(argNames, fmt.Sprintf("arg%d", i))
		}
		call := fmt.Sprintf("%s(%s)", name, strings.Join(argNames, ","))
		compiled, err := vm.Otto.Compile("", fmt.Sprintf("%s\n%s;", script.Source, call))
		if err != nil {
			return err
		}
		f.Compiled = compiled
		script.Functions[name] = f
	}
	return nil
}

// Run 运行 JavaScript 函数
func (vm JavaScript) Run(script *Script, method string, args ...interface{}) (interface{}, error) {

	f, has := script.Functions[method]
	if !has {
		return nil, fmt.Errorf("function  %s does not existed! ", method)
	}

	if f.Compiled == nil {
		return nil, fmt.Errorf("function %s does not compiled! ", method)
	}

	for i, v := range args {
		argName := fmt.Sprintf("arg%d", i)
		vm.Set(argName, v)
	}

	value, err := vm.Otto.Run(f.Compiled)
	if err != nil {
		return nil, err
	}

	resp, err := value.Export()
	if err != nil {
		return nil, err
	}

	return resp, nil
}
