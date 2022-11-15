package bridge

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/runtime/bridge"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// ToInterface JS->GO Convert *v8go.Value to Interface
func ToInterface(value *v8go.Value) (interface{}, error) {

	if value == nil {
		return nil, nil
	}

	var v interface{} = nil
	if value.IsNull() || value.IsUndefined() {
		return nil, nil

	} else if value.IsBigInt() {
		return value.BigInt(), nil

	} else if value.IsBoolean() {
		return value.Boolean(), nil

	} else if value.IsString() {
		return value.String(), nil

	} else if value.IsInt32() {
		return int(value.Int32()), nil

	} else if value.IsUint8Array() || value.IsArrayBufferView() {
		res := []byte{}
		codes := strings.Split(value.String(), ",")
		for _, code := range codes {
			c, err := strconv.Atoi(code)
			if err != nil {
				return nil, err
			}
			res = append(res, byte(c))
		}
		return res, nil
	}

	content, err := value.MarshalJSON()
	if err != nil {
		log.Error("ToInterface MarshalJSON: %#v Error: %s", value, err.Error())
		return nil, err
	}

	err = jsoniter.Unmarshal([]byte(content), &v)
	if err != nil {
		log.Error("ToInterface Unmarshal Value: %#v Content: %#v Error: %s", value, content, err.Error())
		return nil, err
	}
	return v, nil
}

// MustAnyToValue GO->JS Convert any to *v8go.Value
func MustAnyToValue(ctx *v8go.Context, value interface{}) *v8go.Value {
	v, err := AnyToValue(ctx, value)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return v
}

// AnyToValue JS->GO Convert data to *v8go.Value
func AnyToValue(ctx *v8go.Context, value interface{}) (*v8go.Value, error) {

	switch v := value.(type) {
	case []byte:
		return v8go.NewValue(ctx.Isolate(), string(v))

	case bridge.Uint8Array:
		// will be refactored next version
		hexstr := hex.EncodeToString(v)
		res, err := ctx.RunScript(fmt.Sprintf(`
			function _yao_hexToBytes(hex) {
				for (var bytes = [], c = 0; c < hex.length; c += 2) {
					bytes.push(parseInt(hex.substr(c, 2), 16));
				}
				return bytes;
			}
			new Uint8Array(_yao_hexToBytes("%s"));
		`, hexstr), "__temp")

		if err != nil {
			return nil, err
		}
		return res, nil

	case string, int32, uint32, bool, float32, float64, *big.Int:
		return v8go.NewValue(ctx.Isolate(), v)

	// force cast int64 -> int32 ( will be refactored next version )
	case int64:
		return v8go.NewValue(ctx.Isolate(), int32(v))

	case uint64:
		return v8go.NewValue(ctx.Isolate(), int32(v))

	case int:
		return v8go.NewValue(ctx.Isolate(), int32(v))
	}

	v, err := jsoniter.Marshal(value)
	if err != nil {
		log.Error("AnyToValue error: %s", err)
		return nil, err
	}

	return v8go.JSONParse(ctx, string(v))
}

// ArrayToValuers GO->JS Convert []inteface to []v8.Valuer
func ArrayToValuers(ctx *v8go.Context, values []interface{}) ([]v8go.Valuer, error) {
	res := []v8go.Valuer{}
	if ctx == nil {
		return res, fmt.Errorf("Context is nil")
	}

	for i := range values {
		value, err := AnyToValue(ctx, values[i])
		if err != nil {
			log.Error("AnyToValue error: %s", err)
			value, _ = v8go.NewValue(ctx.Isolate(), nil)
		}
		res = append(res, value)
	}
	return res, nil
}

// ValuesToArray JS->GO Convert []*v8go.Value to []interface{}
func ValuesToArray(values []*v8go.Value) []interface{} {
	res := []interface{}{}
	for i := range values {
		var v interface{} = nil
		if values[i].IsNull() || values[i].IsUndefined() {
			res = append(res, nil)
			continue
		}

		v, err := ToInterface(values[i])
		if err != nil {
			log.Error("ValuesToArray Value: %v Error: %s", err.Error(), values[i])
			res = append(res, nil)
			continue
		}

		res = append(res, v)

		// res = append(res, ToInterface(values[i]))
		// content, _ := values[i].MarshalJSON()
		// jsoniter.Unmarshal([]byte(content), &v)
		// res = append(res, v)
	}
	return res
}
