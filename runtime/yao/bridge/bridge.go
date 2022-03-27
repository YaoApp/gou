package bridge

import (
	"math/big"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// ToInterface Convert *v8go.Value to Interface
func ToInterface(value *v8go.Value) (interface{}, error) {

	if value == nil {
		return nil, nil
	}

	var v interface{} = nil
	if value.IsNull() || value.IsUndefined() {
		return nil, nil
	}

	content, err := value.MarshalJSON()
	if err != nil {
		return nil, err
	}
	err = jsoniter.Unmarshal([]byte(content), &v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// MustAnyToValue Convert any to *v8go.Value
func MustAnyToValue(ctx *v8go.Context, value interface{}) *v8go.Value {
	v, err := AnyToValue(ctx, value)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return v
}

// AnyToValue Convert data to *v8go.Value
func AnyToValue(ctx *v8go.Context, value interface{}) (*v8go.Value, error) {

	switch value.(type) {
	case []byte:
		// Todo: []byte to Uint8Array
		return v8go.NewValue(ctx.Isolate(), string(value.([]byte)))
	case string, int32, uint32, int64, uint64, bool, float64, *big.Int:
		return v8go.NewValue(ctx.Isolate(), value)
	}

	v, err := jsoniter.Marshal(value)
	if err != nil {
		log.Error("AnyToValue error: %s", err)
	}

	return v8go.JSONParse(ctx, string(v))
}
