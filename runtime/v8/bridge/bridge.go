package bridge

import (
	"fmt"
	"math/big"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"rogchap.com/v8go"
)

// UndefinedT type of Undefined
type UndefinedT byte

// Undefined jsValue  Undefined
var Undefined UndefinedT = 0x00

// JsValues Golang -> JavaScript
func JsValues(ctx *v8go.Context, goValues []interface{}) ([]*v8go.Value, error) {
	res := []*v8go.Value{}
	for _, goValue := range goValues {
		jsValue, err := JsValue(ctx, goValue)
		if err != nil {
			return nil, err
		}
		res = append(res, jsValue)
	}
	return res, nil
}

// JsError return javascript error object
func JsError(ctx *v8go.Context, err interface{}) *v8go.Value {

	var message string
	switch v := err.(type) {
	case string:
		message = v
		break

	case error:
		message = v.Error()
		break

	case *exception.Exception:
		message = v.Message
		break

	case exception.Exception:
		message = v.Message
		break

	default:
		message = fmt.Sprintf("%v", err)
	}

	global := ctx.Global()
	errorObj, _ := global.Get("Error")
	if errorObj.IsFunction() {
		fn, _ := errorObj.AsFunction()
		m, _ := v8go.NewValue(ctx.Isolate(), message)
		v, _ := fn.Call(v8go.Undefined(ctx.Isolate()), m)
		return v
	}

	tmpl := v8go.NewObjectTemplate(ctx.Isolate())
	inst, _ := tmpl.NewInstance(ctx)
	inst.Set("message", message)
	return inst.Value
}

// JsException throw javascript Exception
func JsException(ctx *v8go.Context, message interface{}) *v8go.Value {
	return ctx.Isolate().ThrowException(JsError(ctx, message))
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
	if values == nil {
		return
	}

	for i := range values {
		if !values[i].IsNull() && !values[i].IsUndefined() {
			values[i].Release()
		}
	}
}

// GoValues JavaScript -> Golang
func GoValues(jsValues []*v8go.Value) ([]interface{}, error) {
	goValues := []interface{}{}
	for _, jsValue := range jsValues {
		goValue, err := GoValue(jsValue)
		if err != nil {
			return nil, err
		}
		goValues = append(goValues, goValue)
	}
	return goValues, nil
}

// JsValue cast golang value to JavasScript value
//
// * |-------------------------------------------------------
// * |    | Golang                  | JavaScript            |
// * |-------------------------------------------------------
// * | ✅ | nil                     | null                  |
// * | ✅ | bool                    | boolean               |
// * | ✅ | int                     | number(int)           |
// * | ✅ | uint                    | number(int)           |
// * | ✅ | uint8                   | number(int)           |
// * | ✅ | uint16                  | number(int)           |
// * | ✅ | uint32                  | number(int)           |
// * | ✅ | int8                    | number(int)           |
// * | ✅ | int16                   | number(int)           |
// * | ✅ | int32                   | number(int)           |
// * | ✅ | float32                 | number(float)         |
// * | ✅ | float64                 | number(float)         |
// * | ✅ | int64                   | bigint                |
// * | ✅ | uint64                  | bigint                |
// * | ✅ | *big.Int                | bigint                |
// * | ✅ | string                  | string                |
// * | ✅ | map[string]interface{}  | object                |
// * | ✅ | []interface{}           | array                 |
// * | ✅ | []byte                  | object(Uint8Array)    |
// * | ✅ | struct                  | object                |
// * | ❌ | ?func                   | function              |
// * |-------------------------------------------------------
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
// *  |--------------------------------------------------------
// *  |    | JavaScript            | Golang                   |
// *  |--------------------------------------------------------
// *  | ✅ | null                  | nil                     |
// *  | ✅ | undefined             | bridge.Undefined        |
// *  | ✅ | boolean               | bool                    |
// *  | ✅ | number(int)           | int                     |
// *  | ✅ | number(float)         | float64                 |
// *  | ✅ | bigint                | int64                   |
// *  | ✅ | string                | string                  |
// *  | ✅ | object(Int8Array)     | []byte                  |
// *  | ✅ | object                | map[string]interface{}  |
// *  | ✅ | array                 | []interface{}           |
// *  | ❌ | object(Promise)       | bridge.Promise          |
// *  | ❌ | function              | bridge.Function         |
// *  |-------------------------------------------------------
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

		if value.IsInt32() {
			return int(value.Int32()), nil
		}

		return value.Number(), nil
	}

	if value.IsBigInt() {
		return value.BigInt().Int64(), nil
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

// ShareData get share data golang <-> javascript
func ShareData(ctx *v8go.Context) (bool, map[string]interface{}, string, *v8go.Value) {
	jsData, err := ctx.Global().Get("__yao_data")
	if err != nil {
		return false, nil, "", JsException(ctx, err)
	}

	goData, err := GoValue(jsData)
	if err != nil {
		return false, nil, "", JsException(ctx, err)
	}

	data, ok := goData.(map[string]interface{})
	if !ok {
		data = map[string]interface{}{}
	}

	global, ok := data["DATA"].(map[string]interface{})
	if !ok {
		global = map[string]interface{}{}
	}

	sid, ok := data["SID"].(string)
	if !ok {
		sid = ""
	}

	root, ok := data["ROOT"].(bool)
	if !ok {
		root = false
	}

	return root, global, sid, nil
}
