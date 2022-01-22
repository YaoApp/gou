package script

import (
	"strings"

	"github.com/robertkrimen/otto/ast"
)

// NewScript 创建 Script
func NewScript(file string, source string) (*Script, error) {
	// program, err := parser.ParseFile(nil, file, source, 0)
	// if err != nil {
	// 	return nil, err
	// }
	script := Script{File: file, Source: source, Functions: map[string]Function{}}
	// ast.Walk(script, program)
	return &script, nil
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
			Line:      line,
			NumOfArgs: len(v.Function.ParameterList.List),
		}
	}
	return script
}

// Exit 解析脚本文件
func (script Script) Exit(n ast.Node) {}
