package process

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func prepareDynamic(t *testing.T) {
	// Clean up any existing test handlers
	Unregister("dynamic.test.handler")
	Unregister("dynamic.test.prepare")
}

// TestRegisterDynamic tests thread-safe dynamic registration
func TestRegisterDynamic(t *testing.T) {
	prepareDynamic(t)
	// Clean up
	defer func() {
		Unregister("dynamic.test.handler")
	}()

	handler := func(process *Process) interface{} {
		return "dynamic handler result"
	}

	// Test basic registration
	RegisterDynamic("dynamic.test.handler", handler)
	assert.NotNil(t, Handlers["dynamic.test.handler"])

	// Test process execution
	p := New("dynamic.test.handler")
	result := p.Run()
	assert.Equal(t, "dynamic handler result", result)
}

// TestRegisterDynamicConcurrent tests concurrent dynamic registration
func TestRegisterDynamicConcurrent(t *testing.T) {
	prepareDynamic(t)
	var wg sync.WaitGroup
	count := 100

	// Clean up
	defer func() {
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("dynamic.concurrent.handler%d", i)
			Unregister(name)
		}
	}()

	// Register multiple handlers concurrently
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("dynamic.concurrent.handler%d", index)
			handler := func(process *Process) interface{} {
				return index
			}
			RegisterDynamic(name, handler)
		}(i)
	}

	wg.Wait()

	// Verify all handlers were registered
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("dynamic.concurrent.handler%d", i)
		assert.NotNil(t, Handlers[name], "Handler %s should be registered", name)
	}
}

// TestUnregister tests handler removal
func TestUnregister(t *testing.T) {
	handler := func(process *Process) interface{} {
		return "test"
	}

	// Register a handler
	RegisterDynamic("unregister.test.handler", handler)
	assert.NotNil(t, Handlers["unregister.test.handler"])

	// Unregister it
	result := Unregister("unregister.test.handler")
	assert.True(t, result, "Unregister should return true for existing handler")
	assert.Nil(t, Handlers["unregister.test.handler"])

	// Try to unregister again
	result = Unregister("unregister.test.handler")
	assert.False(t, result, "Unregister should return false for non-existing handler")
}

// TestUnregisterConcurrent tests concurrent handler removal
func TestUnregisterConcurrent(t *testing.T) {
	count := 100
	var wg sync.WaitGroup

	// Register handlers
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("unregister.concurrent.handler%d", i)
		handler := func(process *Process) interface{} {
			return i
		}
		RegisterDynamic(name, handler)
	}

	// Unregister concurrently
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("unregister.concurrent.handler%d", index)
			Unregister(name)
		}(i)
	}

	wg.Wait()

	// Verify all handlers were removed
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("unregister.concurrent.handler%d", i)
		assert.Nil(t, Handlers[name], "Handler %s should be unregistered", name)
	}
}

// TestRegisterDynamicGroup tests dynamic group registration
func TestRegisterDynamicGroup(t *testing.T) {
	// Clean up
	defer func() {
		Unregister("dynamic.group.method1")
		Unregister("dynamic.group.method2")
		Unregister("dynamic.group.method3")
	}()

	group := map[string]Handler{
		"Method1": func(process *Process) interface{} {
			return "method1 result"
		},
		"Method2": func(process *Process) interface{} {
			return "method2 result"
		},
		"Method3": func(process *Process) interface{} {
			return "method3 result"
		},
	}

	// Register group dynamically
	RegisterDynamicGroup("dynamic.group", group)

	// Verify all methods are registered
	assert.NotNil(t, Handlers["dynamic.group.method1"])
	assert.NotNil(t, Handlers["dynamic.group.method2"])
	assert.NotNil(t, Handlers["dynamic.group.method3"])

	// Test execution
	p1 := New("dynamic.group.method1")
	result1 := p1.Run()
	assert.Equal(t, "method1 result", result1)

	p2 := New("dynamic.group.method2")
	result2 := p2.Run()
	assert.Equal(t, "method2 result", result2)
}

// TestUnregisterGroup tests removing a group of handlers
func TestUnregisterGroup(t *testing.T) {
	group := map[string]Handler{
		"Method1": func(process *Process) interface{} { return "m1" },
		"Method2": func(process *Process) interface{} { return "m2" },
		"Method3": func(process *Process) interface{} { return "m3" },
	}

	// Register group
	RegisterDynamicGroup("unregister.group", group)

	// Verify registration
	assert.NotNil(t, Handlers["unregister.group.method1"])
	assert.NotNil(t, Handlers["unregister.group.method2"])
	assert.NotNil(t, Handlers["unregister.group.method3"])

	// Unregister group
	count := UnregisterGroup("unregister.group")
	assert.Equal(t, 3, count, "Should unregister 3 handlers")

	// Verify removal
	assert.Nil(t, Handlers["unregister.group.method1"])
	assert.Nil(t, Handlers["unregister.group.method2"])
	assert.Nil(t, Handlers["unregister.group.method3"])
}

// TestMixedRegisterUnregister tests mixed register and unregister operations
func TestMixedRegisterUnregister(t *testing.T) {
	var wg sync.WaitGroup
	count := 50

	handler := func(process *Process) interface{} {
		return "test"
	}

	// Concurrent register and unregister
	for i := 0; i < count; i++ {
		wg.Add(2)

		// Register
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("mixed.test.handler%d", index)
			RegisterDynamic(name, handler)
		}(i)

		// Unregister (might fail if not yet registered)
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("mixed.test.handler%d", index)
			Unregister(name)
		}(i)
	}

	wg.Wait()

	// Clean up any remaining handlers
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("mixed.test.handler%d", i)
		Unregister(name)
	}
}

// TestRegisterDynamicCaseInsensitive tests case insensitive handler names
func TestRegisterDynamicCaseInsensitive(t *testing.T) {
	defer Unregister("case.test.handler")

	handler := func(process *Process) interface{} {
		return "case test"
	}

	// Register with mixed case
	RegisterDynamic("Case.Test.Handler", handler)

	// Should be stored in lowercase
	assert.NotNil(t, Handlers["case.test.handler"])
	assert.Nil(t, Handlers["Case.Test.Handler"])

	// Unregister with different case
	result := Unregister("CASE.TEST.HANDLER")
	assert.True(t, result)
	assert.Nil(t, Handlers["case.test.handler"])
}
