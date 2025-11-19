package bridge

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueOfNull(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)
	res, err := call(ctx, "ValueOfNull", nil)
	if err != nil {
		t.Fatal(err)
	}
	checkValueOf(t, res, "object", nil)
}

func TestValueOfUndefined(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)
	res, err := call(ctx, "ValueOfUndefined", Undefined)
	if err != nil {
		t.Fatal(err)
	}
	checkValueOf(t, res, "undefined", Undefined)
}

func TestValueOfBoolean(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)
	res, err := call(ctx, "ValueOfBoolean", true)
	if err != nil {
		t.Fatal(err)
	}
	checkValueOf(t, res, "boolean", true)

	res, err = call(ctx, "ValueOfBoolean", false)
	if err != nil {
		t.Fatal(err)
	}
	checkValueOf(t, res, "boolean", false)
}

func TestValueOfNumberInt(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)

	values := []interface{}{
		99, int(99), int8(99), int16(99), int32(99),
		uint(99), uint8(99), uint16(99), uint32(99),
		uint64(99), int64(99),
	}

	for _, value := range values {
		res, err := call(ctx, "ValueOfNumberInt", value)
		if err != nil {
			t.Fatal(err)
		}
		checkValueOf(t, res, "number", value)
	}
}

func TestValueOfNumberFloat(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)

	values := []interface{}{0.618, float32(0.618), float64(0.618)}

	for _, value := range values {
		res, err := call(ctx, "ValueOfNumberFloat", value)
		if err != nil {
			t.Fatal(err)
		}
		checkValueOf(t, res, "number", value)
	}
}

func TestValueOfBigInt(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)

	values := []interface{}{big.NewInt(99)}

	for _, value := range values {
		res, err := call(ctx, "ValueOfBigInt", value)
		if err != nil {
			t.Fatal(err)
		}
		checkValueOf(t, res, "bigint", value)
	}
}

func TestValueOfString(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)
	res, err := call(ctx, "ValueOfString", "hello world")
	if err != nil {
		t.Fatal(err)
	}
	checkValueOf(t, res, "string", "hello world")
}

func TestValueOfObject(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)

	value := map[string]interface{}{
		"string": "foo",
		"int":    99,
		"bigint": int64(99),
		"float":  float64(0.618),
		"nests": map[string]interface{}{
			"string": "foo",
			"int":    99,
			"float":  float64(0.618),
			"bigint": int64(99),
		},
	}
	res, err := call(ctx, "ValueOfObject", value)
	if err != nil {
		t.Fatal(err)
	}
	checkValueOf(t, res, "object", value)
}

func TestValueOfArray(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)
	value := []interface{}{}
	vMap := map[string]interface{}{
		"string": "foo",
		"int":    99,
		"bigint": int64(99),
		"float":  float64(0.618),
		"nests": map[string]interface{}{
			"string": "foo",
			"int":    99,
			"float":  float64(0.618),
			"bigint": int64(99),
		},
	}

	vArr := []interface{}{"foo", 99, 0.618, int64(99), vMap}
	value = append(value, vArr...)
	value = append(value, value)
	res, err := call(ctx, "ValueOfArray", value)
	if err != nil {
		t.Fatal(err)
	}
	checkValueOf(t, res, "object", value)
}

func TestValueUint8Array(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)

	value := []byte{0x1a, 0x2a}
	res, err := call(ctx, "ValueUint8Array", value)
	if err != nil {
		t.Fatal(err)
	}
	checkValueOf(t, res, "object", value)
}

func TestValueOfStruct(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)

	type Basic struct {
		String string
		Int    int
	}

	type Value struct {
		String string
		Int    int
		Basic  *Basic
	}

	value := Value{String: "foo", Int: 99, Basic: &Basic{String: "bar", Int: 66}}
	res, err := call(ctx, "ValueOfStruct", value)
	if err != nil {
		t.Fatal(err)
	}
	checkValueOf(t, res, "object", value)
}

func TestValueOfFunction(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)

	cb, err := call(ctx, "ReturnFunction")
	if err != nil {
		t.Fatal(err)
	}

	res, err := call(ctx, "ValueOfFunction", cb)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hello", res)
}
func TestValueOfPromise(t *testing.T) {
	ctx := prepare(t)
	defer closeContext(ctx)

	promise, err := call(ctx, "ReturnPromiseString", "hello")
	if err != nil {
		t.Fatal(err)
	}

	res, err := call(ctx, "ValueOfPromise", promise)
	if err != nil {
		t.Fatal(err)
	}

	v, ok := res.(PromiseT)
	assert.Equal(t, true, ok)

	p, err := v.value.AsPromise()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "hello", p.Result().String())

	promise, err = call(ctx, "ReturnPromiseInt", 1)
	if err != nil {
		t.Fatal(err)
	}

	res, err = call(ctx, "ValueOfPromise", promise)
	if err != nil {
		t.Fatal(err)
	}

	pro, ok := res.(PromiseT)
	assert.Equal(t, true, ok)

	res, err = pro.Result()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, res)
}

func checkValueOf(t *testing.T, res interface{}, typeof string, goValue interface{}) {
	value, ok := res.(map[string]interface{})
	if !ok {
		t.Fatal(fmt.Errorf("res type error: %#v", res))
	}

	assert.Equal(t, true, value["check"], fmt.Sprintf("GoValue: %#v Res:%#v", goValue, res))
	assert.Equal(t, typeof, value["typeof"], fmt.Sprintf("GoValue: %#v  Res:%#v", goValue, res))
}
