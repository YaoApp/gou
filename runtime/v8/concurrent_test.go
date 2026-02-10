package v8

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestConcurrentAll(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.HeapSizeLimit = 4294967296
	prepare(t, option)
	defer Stop()

	p, err := process.Of("scripts.runtime.concurrent.TestAll")
	if err != nil {
		t.Fatal(err)
	}

	err = p.Execute()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "PASS", p.Value())
}

func TestConcurrentAllWithError(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.HeapSizeLimit = 4294967296
	prepare(t, option)
	defer Stop()

	p, err := process.Of("scripts.runtime.concurrent.TestAllWithError")
	if err != nil {
		t.Fatal(err)
	}

	err = p.Execute()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "PASS", p.Value())
}

func TestConcurrentAny(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.HeapSizeLimit = 4294967296
	prepare(t, option)
	defer Stop()

	p, err := process.Of("scripts.runtime.concurrent.TestAny")
	if err != nil {
		t.Fatal(err)
	}

	err = p.Execute()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "PASS", p.Value())
}

func TestConcurrentRace(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.HeapSizeLimit = 4294967296
	prepare(t, option)
	defer Stop()

	p, err := process.Of("scripts.runtime.concurrent.TestRace")
	if err != nil {
		t.Fatal(err)
	}

	err = p.Execute()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "PASS", p.Value())
}

func TestConcurrentAllConcurrency(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.HeapSizeLimit = 4294967296
	prepare(t, option)
	defer Stop()

	p, err := process.Of("scripts.runtime.concurrent.TestAllConcurrency")
	if err != nil {
		t.Fatal(err)
	}

	err = p.Execute()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "PASS", p.Value())
}

func TestConcurrentAllEmpty(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.HeapSizeLimit = 4294967296
	prepare(t, option)
	defer Stop()

	p, err := process.Of("scripts.runtime.concurrent.TestAllEmpty")
	if err != nil {
		t.Fatal(err)
	}

	err = p.Execute()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "PASS", p.Value())
}
