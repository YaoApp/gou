package v8

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCall(t *testing.T) {
	prepare(t)
	time.Sleep(20 * time.Millisecond)
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

func TestCallRelease(t *testing.T) {
	prepare(t)

	SetHeapAvailableSize(2018051350)
	defer SetHeapAvailableSize(524288000)

	DisablePrecompile()
	defer EnablePrecompile()

	basic, err := Select("runtime.basic")
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := basic.NewContext("SID_1020", map[string]interface{}{"name": "testing"})
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, ctx.Iso.health())
	ctx.Close()
	assert.Equal(t, 1, len(chIsoReady))

	time.Sleep(1 * time.Second)
	assert.Equal(t, 2, len(chIsoReady))
}
