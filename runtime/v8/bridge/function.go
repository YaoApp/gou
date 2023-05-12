package bridge

import "fmt"

// Call jsValue Function
func (f FunctionT) Call(args ...interface{}) (interface{}, error) {

	if f.ctx == nil {
		return nil, fmt.Errorf("invalid context")
	}

	cb, err := f.value.AsFunction()
	if err != nil {
		return nil, err
	}

	jsArgs, err := JsValues(f.ctx, args)
	if err != nil {
		return nil, err
	}
	defer FreeJsValues(jsArgs)

	value, err := cb.Call(f.ctx.Global(), Valuers(jsArgs)...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if value != nil {
			value.Release()
		}
	}()

	goValue, err := GoValue(value, f.ctx)
	if err != nil {
		return nil, err
	}

	return goValue, nil
}
