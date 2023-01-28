package wasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallTestAdd(t *testing.T) {
	prepare(t)
	test, err := Select("test")
	if err != nil {
		t.Fatal(err)
	}

	var res int
	err = test.Call("add", &res, 6, 8)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 14, res)
}

func TestCallTestEcho(t *testing.T) {
	prepare(t)
	test, err := Select("test")
	if err != nil {
		t.Fatal(err)
	}

	var res string
	err = test.Call("echo", &res, "hello world")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "echo: hello world", res)
}

func TestCallTestBytes(t *testing.T) {
	prepare(t)
	test, err := Select("test")
	if err != nil {
		t.Fatal(err)
	}

	var str = []byte("hello world")
	var res []byte
	err = test.Call("bytes", &res, str, 5)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte("hello"), res)
}
