package bridge

import (
	"fmt"
	"math/big"

	jsoniter "github.com/json-iterator/go"
	"rogchap.com/v8go"
)

// UndefinedT type of Undefined
type UndefinedT byte

// Undefined jsValue  Undefined
var Undefined UndefinedT = 0x00

// JsValues cast golang values to JavasScript values
func JsValues(ctx *v8go.Context, values []interface{}) ([]*v8go.Value, error) {
	res := []*v8go.Value{}
	for _, value := range values {
		jsValue, err := JsValue(ctx, value)
		if err != nil {
			return nil, err
		}
		res = append(res, jsValue)
	}
	return res, nil
}

// Valuers to interface
func Valuers(values []*v8go.Value) []v8go.Valuer {
	valuers := []v8go.Valuer{}
	for _, value := range values {
		valuers = append(valuers, value)
	}
	return valuers
}

// FreeJsValues release js values
func FreeJsValues(values []*v8go.Value) {
	for i := range values {
		values[i].Release()
	}
}

// JsValue cast golang value to JavasScript value
//
// *  ---------------------------------------------------
// *  | Golang                  | Javascript            |
// *  ---------------------------------------------------
// *  | nil                     | null                  |
// *  | bool                    | boolean               |
// *  | int                     | number(int)           |
// *  | uint                    | number(int)           |
// *  | uint8                   | number(int)           |
// *  | uint16                  | number(int)           |
// *  | uint32                  | number(int)           |
// *  | int8                    | number(int)           |
// *  | int16                   | number(int)           |
// *  | int32                   | number(int)           |
// *  | float32                 | number(float)         |
// *  | float64                 | number(float)         |
// *  | int64                   | bigint                |
// *  | uint64                  | bigint                |
// *  | *big.Int                | bigint                |
// *  | string                  | string                |
// *  | map[string]interface{}  | object                |
// *  | []interface{}           | array                 |
// *  | []byte                  | object(Uint8Array)    |
// *  | struct                  | object                |
// *  | func                    | function              |
// *  ---------------------------------------------------
func JsValue(ctx *v8go.Context, value interface{}) (*v8go.Value, error) {

	if value == nil {
		return v8go.Null(ctx.Isolate()), nil
	}

	switch v := value.(type) {

	case string, int32, uint32, int64, uint64, bool, *big.Int, float64, []byte:
		return v8go.NewValue(ctx.Isolate(), v)

	case int:
		return v8go.NewValue(ctx.Isolate(), int32(v))

	case int8:
		return v8go.NewValue(ctx.Isolate(), int32(v))

	case int16:
		return v8go.NewValue(ctx.Isolate(), int32(v))

	case uint:
		return v8go.NewValue(ctx.Isolate(), int32(v))

	case uint8:
		return v8go.NewValue(ctx.Isolate(), int32(v))

	case uint16:
		return v8go.NewValue(ctx.Isolate(), int32(v))

	case float32:
		return v8go.NewValue(ctx.Isolate(), float64(v))

	case UndefinedT:
		return v8go.Undefined(ctx.Isolate()), nil

	default:
		return jsValueParse(ctx, v)
	}
}

func jsValueParse(ctx *v8go.Context, value interface{}) (*v8go.Value, error) {

	data, err := jsoniter.Marshal(value)
	if err != nil {
		return nil, err
	}

	jsValue, err := v8go.JSONParse(ctx, string(data))
	if err != nil {
		return nil, err
	}

	return jsValue, nil
}

// GoValue cast JavasScript value to Golang value
//
// *  JavaScript -> Golang
// *  ---------------------------------------------------
// *  | JavaScript            | Golang                  |
// *  ---------------------------------------------------
// *  | null                  | nil                     |
// *  | undefined             | bridge.Undefined        |
// *  | boolean               | bool                    |
// *  | number(int)           | int                     |
// *  | number(float)         | float64                 |
// *  | bigint                | int64                   |
// *  | string                | string                  |
// *  | object                | map[string]interface{}  |
// *  | array                 | []interface{}           |
// *  | object(Int8Array)     | []byte                  |
// *  | object(Promise)       | bridge.Promise          |
// *  | function              | bridge.Function         |
// *  ---------------------------------------------------
func GoValue(value *v8go.Value) (interface{}, error) {

	if value.IsNull() {
		return nil, nil
	}

	if value.IsUndefined() {
		return Undefined, nil
	}

	if value.IsString() {
		return value.String(), nil
	}

	if value.IsBoolean() {
		return value.Boolean(), nil
	}

	if value.IsNumber() {

		obj, err := value.AsObject()
		if err != nil {
			return nil, err
		}

		jsValue, err := obj.MethodCall("isInteger")
		if err != nil {
			return nil, err
		}

		if jsValue.Boolean() {
			return value.Int32(), nil
		}

		return value.Number(), nil
	}

	if value.IsBigInt() {
		return value.BigInt().Uint64(), nil
	}

	if value.IsUint8Array() { // bytes
		return value.Uint8Array(), nil
	}

	if value.IsArray() {
		var goValue []interface{}
		return goValueParse(value, goValue)
	}

	if value.IsMap() {
		var goValue map[string]interface{}
		return goValueParse(value, goValue)
	}

	// Map, Array etc.
	var goValue interface{}
	return goValueParse(value, goValue)
}

func goValueParse(value *v8go.Value, v interface{}) (interface{}, error) {

	data, err := value.MarshalJSON()
	if err != nil {
		return nil, err
	}

	ptr := &v
	err = jsoniter.Unmarshal(data, ptr)
	if err != nil {
		fmt.Printf("---\n%s\n---\n", data)
		return nil, err
	}

	return *ptr, nil
}
