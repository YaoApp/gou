package bridge

import (
	"fmt"
	"io"
	"math/big"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"rogchap.com/v8go"
)

// UndefinedT type of Undefined
type UndefinedT byte

// FunctionT jsValue Function
type FunctionT struct {
	ctx   *v8go.Context
	value *v8go.Value
}

// PromiseT jsValue Promise
type PromiseT struct {
	ctx   *v8go.Context
	value *v8go.Value
}

// Share share data
type Share struct {
	Iso    string // Isolate ID
	Sid    string
	Root   bool
	Global map[string]interface{}
}

// Valuer is the interface for the value
type Valuer interface {
	JsValue(ctx *v8go.Context) (*v8go.Value, error)
}

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
func GoValues(jsValues []*v8go.Value, ctx *v8go.Context) ([]interface{}, error) {
	goValues := []interface{}{}
	for _, jsValue := range jsValues {
		goValue, err := GoValue(jsValue, ctx)
		if err != nil {
			return nil, err
		}
		goValues = append(goValues, goValue)
	}
	return goValues, nil
}

// JsValue cast golang value to JavasScript value
//
// * |-----------------------------------------------------------
// * |    | Golang                  | JavaScript                |
// * |-----------------------------------------------------------
// * | ✅ | nil                     | null                      |
// * | ✅ | bool                    | boolean                   |
// * | ✅ | int                     | number(int)               |
// * | ✅ | uint                    | number(int)               |
// * | ✅ | uint8                   | number(int)               |
// * | ✅ | uint16                  | number(int)               |
// * | ✅ | uint32                  | number(int)               |
// * | ✅ | int8                    | number(int)               |
// * | ✅ | int16                   | number(int)               |
// * | ✅ | int32                   | number(int)               |
// * | ✅ | float32                 | number(float)             |
// * | ✅ | float64                 | number(float)             |
// * | ✅ | int64                   | number(int)               |
// * | ✅ | uint64                  | number(int)               |
// * | ✅ | *big.Int                | bigint                    |
// * | ✅ | string                  | string                    |
// * | ✅ | map[string]interface{}  | object                    |
// * | ✅ | []interface{}           | array                 	   |
// * | ✅ | []byte                  | object(Uint8Array) 	   |
// * | ✅ | struct                  | object                    |
// * | ✅ | bridge.PromiseT         | object(Promise)           |
// * | ✅ | bridge.FunctionT        | function                  |
// * | ✅ | io.Writer               | object(external)          |
// * | ✅ | io.Reader               | object(external)          |
// * | ✅ | io.ReadCloser           | object(external)          |
// * | ✅ | io.WriteCloser          | object(external)          |
// * | ✅ | *gin.Context            | object(external)          |
// * | ✅ | *gin.ResponseWriter     | object(external)          |
// * | ✅ | bridge.Valuer           | custom value interface     |
// * |-----------------------------------------------------------
func JsValue(ctx *v8go.Context, value interface{}) (*v8go.Value, error) {

	if value == nil {
		return v8go.Null(ctx.Isolate()), nil
	}

	switch v := value.(type) {

	// Custom value interface
	case Valuer:
		return v.JsValue(ctx)

	case string, int32, uint32, bool, *big.Int, float64:
		return v8go.NewValue(ctx.Isolate(), v)

	case []byte:
		newObj, err := ctx.RunScript(fmt.Sprintf("new Uint8Array(%d)", len(v)), "")
		if err != nil {
			return nil, err
		}

		jsObj, err := newObj.AsObject()
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(v); i++ {
			jsObj.SetIdx(uint32(i), uint32(v[i]))
		}
		return jsObj.Value, nil

	case int64:
		return v8go.NewValue(ctx.Isolate(), int32(v))

	case uint64:
		return v8go.NewValue(ctx.Isolate(), int32(v))

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

	case PromiseT:
		return v.value, nil

	case FunctionT:
		return v.value, nil

	case UndefinedT:
		return v8go.Undefined(ctx.Isolate()), nil

	// For share value
	case io.Writer, io.Reader, io.ReadCloser, io.WriteCloser, gin.ResponseWriter:
		return v8go.NewExternal(ctx.Isolate(), v)

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
// *  |-----------------------------------------------------------
// *  |    | JavaScript            | Golang                      |
// *  |-----------------------------------------------------------
// *  | ✅ | null                      | nil                     |
// *  | ✅ | undefined                 | bridge.Undefined        |
// *  | ✅ | boolean                   | bool                    |
// *  | ✅ | number(int)               | int                     |
// *  | ✅ | number(float)             | float64                 |
// *  | ✅ | bigint                    | int64                   |
// *  | ✅ | string               	  | string                  |
// *  | ✅ | object(SharedArrayBuffer) | []byte                  |
// *  | ✅ | object(Uint8Array)        | []byte                  |
// *  | ✅ | object                    | map[string]interface{}  |
// *  | ✅ | array                     | []interface{}           |
// *  | ✅ | object(Promise)           | bridge.PromiseT         |
// *  | ✅ | function                  | bridge.FunctionT        |
// *  | ✅ | object(external)          | interface{}             |
// *  |-----------------------------------------------------------
func GoValue(value *v8go.Value, ctx *v8go.Context) (interface{}, error) {

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

	if value.IsFunction() {
		return FunctionT{value: value, ctx: ctx}, nil
	}

	if value.IsPromise() {
		return PromiseT{value: value, ctx: ctx}, nil
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

	if value.IsSharedArrayBuffer() { // bytes
		buf, cleanup, err := value.SharedArrayBufferGetContents()
		if err != nil {
			return nil, err
		}
		defer cleanup()
		return buf, nil
	}

	if value.IsUint8Array() { // bytes
		arr, err := value.AsObject()
		if err != nil {
			return nil, err
		}

		length, err := arr.Get("length")
		if err != nil {
			return nil, err
		}

		var goValue []byte
		for i := uint32(0); i < length.Uint32(); i++ {
			v, err := arr.GetIdx(i)
			if err != nil {
				return nil, err
			}
			goValue = append(goValue, byte(v.Uint32()))
		}
		return goValue, nil
	}

	if value.IsArray() {
		var goValue []interface{}
		return goValueParse(value, goValue)
	}

	if value.IsMap() {
		var goValue map[string]interface{}
		return goValueParse(value, goValue)
	}

	// YAO External
	if value.IsYaoExternal() {
		goValue, err := value.External()
		if err != nil {
			return nil, err
		}
		return goValue, nil
	}

	// Map, Array etc.
	var goValue interface{}
	return goValueParse(value, goValue)
}

// Unmarshal cast javascript value to golang value
func Unmarshal(value *v8go.Value, v interface{}) error {
	data, err := value.MarshalJSON()
	if err != nil {
		return err
	}
	return jsoniter.Unmarshal(data, v)
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

// SetShareData set share data golang <-> javascript
func SetShareData(ctx *v8go.Context, obj *v8go.Object, share *Share) error {

	goData := map[string]interface{}{
		"SID":  share.Sid,
		"ROOT": share.Root,
		"DATA": share.Global,
		"ISO":  share.Iso,
	}

	jsData, err := JsValue(ctx, goData)
	if err != nil {
		return err
	}

	err = obj.Set("__yao_data", jsData)
	if err != nil {
		return err
	}

	defer func() {
		if !jsData.IsNull() && !jsData.IsUndefined() {
			jsData.Release()
		}
	}()

	return nil
}

// ShareData get share data golang <-> javascript
func ShareData(ctx *v8go.Context) (*Share, error) {
	jsData, err := ctx.Global().Get("__yao_data")
	if err != nil {
		return nil, err
	}

	goData, err := GoValue(jsData, nil)
	if err != nil {
		return nil, err
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

	iso, ok := data["ISO"].(string) // Isolate ID
	if !ok {
		iso = ""
	}

	return &Share{
		Root:   root,
		Sid:    sid,
		Global: global,
		Iso:    iso,
	}, nil
}

// ShareData1 get share data golang <-> javascript
func ShareData1(ctx *v8go.Context) (bool, map[string]interface{}, string, *v8go.Value) {
	jsData, err := ctx.Global().Get("__yao_data")
	if err != nil {
		return false, nil, "", JsException(ctx, err)
	}

	goData, err := GoValue(jsData, nil)
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

func (fun FunctionT) String() string {
	return fmt.Sprintf("[Function: %s]", fun.value.String())
}

func (promise PromiseT) String() string {
	p, err := promise.value.AsPromise()
	if err != nil {
		return fmt.Sprintf("%s", err.Error())
	}

	var state string = "pending"
	switch p.State() {
	case v8go.Fulfilled:
		state = "fulfilled"
	case v8go.Rejected:
		state = "rejected"
	}
	return state
}

func (undefined UndefinedT) String() string {
	return fmt.Sprintf("undefined")
}
