package v8

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCall(t *testing.T) {
	prepare(t)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 2, len(Scripts))
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

func TestCallWith(t *testing.T) {
	prepare(t)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 2, len(Scripts))
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

	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	res, err := ctx.CallWith(context, "Cancel", "hello")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "hello", res)
}

func TestCallWithCancel(t *testing.T) {
	prepare(t)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 2, len(Scripts))
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

	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	_, err = ctx.CallWith(context, "Cancel", "hello")
	assert.Contains(t, err.Error(), "context canceled")
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
