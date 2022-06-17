package gou

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// MakeFlow make a flow instance
func MakeFlow() *Flow {
	return &Flow{}
}

// DSLCompile compile the FLow
func (flow *Flow) DSLCompile(root string, file string, source map[string]interface{}) error {

	data, err := jsoniter.Marshal(source)
	if err != nil {
		return err
	}

	fullname, _, _ := flow.nameRouter(root, file)
	new := Flow{
		Name:    fullname,
		Source:  string(data),
		Scripts: map[string]string{},
	}

	err = jsoniter.Unmarshal(data, &new)
	if err != nil {
		return err
	}

	flow.Prepare()
	Flows[fullname] = &new
	return nil
}

// DSLCheck check the FLow DSL
func (flow *Flow) DSLCheck(source map[string]interface{}) error { return nil }

// DSLRefresh refresh the FLow
func (flow *Flow) DSLRefresh(root string, file string, source map[string]interface{}) error {
	fullname, _, _ := flow.nameRouter(root, file)
	delete(Flows, fullname)
	return flow.DSLCompile(root, file, source)
}

// DSLRemove the DSL
func (flow *Flow) DSLRemove(root string, file string) error {
	fullname, _, _ := flow.nameRouter(root, file)
	delete(Flows, fullname)
	return nil
}

// nameRouter get the model name from router
func (flow *Flow) nameRouter(root string, file string) (fullname string, namespace string, name string) {
	dir, filename := filepath.Split(strings.TrimPrefix(file, filepath.Join(root, "flows")))
	name = strings.TrimRight(filename, ".flow.yao")
	namespace = strings.ReplaceAll(strings.Trim(filepath.ToSlash(dir), "/"), "/", ".")
	fullname = name
	if namespace != "" {
		fullname = fmt.Sprintf("%s.%s", namespace, name)
	}
	return fullname, namespace, name
}
