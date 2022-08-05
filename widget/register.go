package widget

import (
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
)

// Call by name
func (r ModuleRegister) Call(method string, name string, source []byte) error {
	if fn, has := r[method]; has {
		return fn(name, source)
	}
	return nil
}

// RegisterAPI Execute the export.js API function and register the Apis
func (w *Widget) RegisterAPI() (err error) {

	if w.ModuleRegister == nil {
		return nil
	}

	res, err := w.Export("Apis", "", nil)
	if err != nil {
		return err
	}

	for name, bytes := range res {
		err = w.ModuleRegister.Call("Apis", name, bytes)
		if err != nil {
			return err
		}
	}

	return nil
}

// RegisterModule Execute the export.js name function and register the modules
func (w *Widget) RegisterModule(module, name string, dsl map[string]interface{}) (err error) {

	if w.ModuleRegister == nil || w.ModuleRegister[module] == nil || module == "Apis" {
		return nil
	}

	res, err := w.Export(module, name, dsl)
	if err != nil {
		return err
	}

	for name, bytes := range res {
		err = w.ModuleRegister.Call(module, name, bytes)
		if err != nil {
			return err
		}
	}

	return nil
}

// RegisterProcess Execute the process.js Export function and register the process
func (w *Widget) RegisterProcess() (err error) {
	if w.ProcessRegister == nil {
		return nil
	}

	value, err := w.ProcessExec("Export")
	if err != nil {
		if strings.HasSuffix(err.Error(), "does not exists") {
			err = nil
			return nil
		}
		return err
	}

	if !any.Of(value).IsMap() {
		return nil
	}

	resp := any.Of(value).Map().MapStrAny
	for name, methodAny := range resp {
		if method, ok := methodAny.(string); ok {
			w.ProcessRegister(w.Name, name, func(args ...interface{}) interface{} {
				value, err := w.ProcessExec(method, args...)
				if err != nil {
					exception.New(err.Error(), 500).Throw()
					return nil
				}
				return value
			})
		}
	}

	return nil
}

// Export  Execute the export.js Models function and return map[string][]byte
func (w *Widget) Export(method, name string, dsl map[string]interface{}) (res map[string][]byte, err error) {
	defer func() { err = exception.Catch(recover(), err) }()
	res = map[string][]byte{}
	value, err := w.ExportExec(method, name, dsl)
	if err != nil {
		if strings.HasSuffix(err.Error(), "does not exists") {
			err = nil
			return nil, nil
		}
		return nil, err
	}

	if value == nil {
		return res, nil
	}

	if !any.Of(value).IsMap() {
		return res, nil
	}

	resp := any.Of(value).Map().MapStrAny
	for name, val := range resp {
		bytes, err := jsoniter.Marshal(val)
		if err != nil {
			return nil, err
		}
		res[name] = bytes
	}
	return res, nil
}
