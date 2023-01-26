package v8

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCall(t *testing.T) {
	prepare(t)
	assert.Equal(t, 1, len(Scripts))
	assert.Equal(t, 1, len(RootScripts))
	assert.Equal(t, 2, len(chIsoReady))

	basic, err := Select("runtime.basic")
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := basic.NewContext("SID_1010", map[string]interface{}{"name": "testing"})
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	res, err := ctx.Call("Hello", "world")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "world", res)
}
