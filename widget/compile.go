package widget

import (
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
)

// CompileSource Execute the compile.js Source function
func (w *Widget) CompileSource() (res map[string][]byte, err error) {
	defer func() { err = exception.Catch(recover(), err) }()
	res = map[string][]byte{}

	value, err := w.CompileExec("Source")
	if err != nil {
		if strings.HasSuffix(err.Error(), "does not exists") {
			err = nil
			return res, nil
		}

		return nil, err
	}

	sources := any.Of(value).Map().MapStrAny
	for name, source := range sources {
		bytes, err := jsoniter.Marshal(source)
		if err != nil {
			return nil, err
		}
		res[name] = bytes
	}
	return res, nil
}

// CompileCompile Execute the compile.js Compile function
func (w *Widget) CompileCompile(name string, dsl map[string]interface{}) (res map[string]interface{}, err error) {
	defer func() { err = exception.Catch(recover(), err) }()
	value, err := w.CompileExec("Compile", name, dsl)
	if err != nil {
		if strings.HasSuffix(err.Error(), "does not exists") {
			err = nil
			return dsl, nil
		}
		return nil, err
	}
	res = any.Of(value).Map().MapStrAny
	return res, err
}

// CompileOnLoad Execute the compile.js OnLoad function
func (w *Widget) CompileOnLoad(name string, dsl map[string]interface{}) (err error) {
	defer func() { err = exception.Catch(recover(), err) }()
	_, err = w.CompileExec("OnLoad", name, dsl)
	if err != nil && strings.HasSuffix(err.Error(), "does not exists") {
		err = nil
	}
	return err
}
