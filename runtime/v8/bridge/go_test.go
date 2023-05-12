package bridge

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReturnNull(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnNull")
	if err != nil {
		t.Fatal(err)
	}
	checkReturn(t, nil, res)
}

func TestReturnUndefined(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnUndefined")
	if err != nil {
		t.Fatal(err)
	}
	checkReturn(t, Undefined, res)
}

func TestReturnBoolean(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnBoolean")
	if err != nil {
		t.Fatal(err)
	}
	checkReturn(t, true, res)
}

func TestReturnNumberInt(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnNumberInt")
	if err != nil {
		t.Fatal(err)
	}
	checkReturn(t, int(99), res)
}

func TestReturnNumberFloat(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnNumberFloat")
	if err != nil {
		t.Fatal(err)
	}
	checkReturn(t, float64(0.618), res)
}

func TestReturnBigInt(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnBigInt")
	if err != nil {
		t.Fatal(err)
	}
	checkReturn(t, int64(99), res)
}

func TestReturnString(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnString")
	if err != nil {
		t.Fatal(err)
	}
	checkReturn(t, "hello world", res)
}

func TestReturnUint8Array(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnUint8Array")
	if err != nil {
		t.Fatal(err)
	}
	checkReturn(t, []byte{0x2a}, res)
}

func TestReturnObject(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnObject")
	if err != nil {
		t.Fatal(err)
	}
	expect := map[string]interface{}{
		"string": "foo",
		"int":    float64(99),
		"bigint": float64(99),
		"float":  0.618,
		"nests": map[string]interface{}{
			"string": "foo",
			"int":    float64(99),
			"float":  0.618,
			"bigint": float64(99),
		},
	}
	checkReturn(t, true, reflect.DeepEqual(expect, res))
}

func TestReturnArray(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnArray")
	if err != nil {
		t.Fatal(err)
	}

	expect := []interface{}{}
	vMap := map[string]interface{}{
		"string": "foo",
		"int":    float64(99),
		"bigint": float64(99),
		"float":  float64(0.618),
		"nests": map[string]interface{}{
			"string": "foo",
			"int":    float64(99),
			"float":  float64(0.618),
			"bigint": float64(99),
		},
	}

	vArr := []interface{}{"foo", float64(99), 0.618, float64(99), vMap}
	expect = append(expect, vArr...)
	expect = append(expect, expect)
	checkReturn(t, true, reflect.DeepEqual(expect, res))
}

func TestReturnFunction(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnFunction")
	if err != nil {
		t.Fatal(err)
	}

	_, ok := res.(FunctionT)
	checkReturn(t, true, ok)
}

func TestReturnPromise(t *testing.T) {
	ctx := prepare(t)
	defer close(ctx)
	res, err := call(ctx, "ReturnPromiseString", "hello")
	if err != nil {
		t.Fatal(err)
	}

	_, ok := res.(PromiseT)
	checkReturn(t, true, ok)

	res, err = call(ctx, "ReturnPromiseInt", 1)
	if err != nil {
		t.Fatal(err)
	}

	_, ok = res.(PromiseT)
	checkReturn(t, true, ok)
}

func checkReturn(t *testing.T, expect interface{}, value interface{}) {
	assert.Equal(t, expect, value, fmt.Sprintf("Value: %#v expectedValue:%#v", value, expect))
}
