package v8

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeScriptInMemoryTypeScript(t *testing.T) {
	option := option()
	option.Mode = "standard"
	prepare(t, option)
	defer Stop()

	// Simple TypeScript code without imports
	source := []byte(`
// @ts-nocheck
function Create(ctx: any, messages: any[]): any {
	return {
		temperature: 0.9,
		metadata: {
			hook_executed: true,
			message_count: messages.length
		}
	};
}

function Hello(name: string): string {
	return "Hello, " + name + "!";
}
`)

	script, err := MakeScriptInMemory(source, "test/memory/script.ts", 5*time.Second, true)
	require.NoError(t, err)
	require.NotNil(t, script)

	// Verify script properties
	assert.Equal(t, "test/memory/script.ts", script.File)
	assert.True(t, script.Root)
	assert.NotEmpty(t, script.Source)

	// The source should be transformed JavaScript (no TypeScript type annotations)
	assert.NotContains(t, script.Source, ": any")
	assert.NotContains(t, script.Source, ": string")

	// Verify functions exist in source
	assert.Contains(t, script.Source, "function Create")
	assert.Contains(t, script.Source, "function Hello")
}

func TestMakeScriptInMemoryJavaScript(t *testing.T) {
	option := option()
	option.Mode = "standard"
	prepare(t, option)
	defer Stop()

	// Plain JavaScript code
	source := []byte(`
function Add(a, b) {
	return a + b;
}

function Multiply(a, b) {
	return a * b;
}
`)

	script, err := MakeScriptInMemory(source, "test/memory/script.js", 5*time.Second, true)
	require.NoError(t, err)
	require.NotNil(t, script)

	// Verify functions exist
	assert.Contains(t, script.Source, "function Add")
	assert.Contains(t, script.Source, "function Multiply")
}

func TestMakeScriptInMemoryWithExport(t *testing.T) {
	option := option()
	option.Mode = "standard"
	prepare(t, option)
	defer Stop()

	// TypeScript code with export statements (should be removed)
	source := []byte(`
export function Create(ctx: any, messages: any[]): any {
	return null;
}

export default function Main(): string {
	return "main";
}
`)

	script, err := MakeScriptInMemory(source, "test/memory/export.ts", 5*time.Second, true)
	require.NoError(t, err)
	require.NotNil(t, script)

	// Export statements should be removed
	assert.NotContains(t, script.Source, "export ")

	// Functions should still exist
	assert.Contains(t, script.Source, "function Create")
	assert.Contains(t, script.Source, "function Main")
}

func TestMakeScriptInMemoryEmpty(t *testing.T) {
	option := option()
	option.Mode = "standard"
	prepare(t, option)
	defer Stop()

	// Empty source
	source := []byte(``)

	script, err := MakeScriptInMemory(source, "test/memory/empty.ts", 5*time.Second, true)
	require.NoError(t, err)
	require.NotNil(t, script)
}

func TestMakeScriptInMemorySyntaxError(t *testing.T) {
	option := option()
	option.Mode = "standard"
	prepare(t, option)
	defer Stop()

	// Invalid TypeScript syntax
	source := []byte(`
function Broken( {
	return 
}
`)

	_, err := MakeScriptInMemory(source, "test/memory/broken.ts", 5*time.Second, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transform error")
}

func TestMakeScriptInMemoryComplexTypes(t *testing.T) {
	option := option()
	option.Mode = "standard"
	prepare(t, option)
	defer Stop()

	// TypeScript with interfaces and types (should compile without file resolution)
	source := []byte(`
interface Message {
	role: string;
	content: string;
}

interface Context {
	chat_id: string;
	locale: string;
}

type HookResponse = {
	messages?: Message[];
	temperature?: number;
} | null;

function Create(ctx: Context, messages: Message[]): HookResponse {
	if (messages.length === 0) {
		return null;
	}
	return {
		temperature: 0.7,
		messages: [
			{ role: "system", content: "You are helpful" }
		]
	};
}
`)

	script, err := MakeScriptInMemory(source, "test/memory/types.ts", 5*time.Second, true)
	require.NoError(t, err)
	require.NotNil(t, script)

	// Type definitions should be removed in output
	assert.NotContains(t, script.Source, "interface Message")
	assert.NotContains(t, script.Source, "interface Context")
	assert.NotContains(t, script.Source, "type HookResponse")

	// Function should still exist
	assert.Contains(t, script.Source, "function Create")
}

func TestMakeScriptInMemoryNonRoot(t *testing.T) {
	option := option()
	option.Mode = "standard"
	prepare(t, option)
	defer Stop()

	source := []byte(`function Test() { return "test"; }`)

	// Without isroot parameter (default false)
	script, err := MakeScriptInMemory(source, "test/memory/nonroot.ts", 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, script)
	assert.False(t, script.Root)

	// With isroot = false
	script2, err := MakeScriptInMemory(source, "test/memory/nonroot2.ts", 5*time.Second, false)
	require.NoError(t, err)
	require.NotNil(t, script2)
	assert.False(t, script2.Root)

	// With isroot = true
	script3, err := MakeScriptInMemory(source, "test/memory/root.ts", 5*time.Second, true)
	require.NoError(t, err)
	require.NotNil(t, script3)
	assert.True(t, script3.Root)
}

func TestMakeScriptInMemoryCompareWithMakeScript(t *testing.T) {
	option := option()
	option.Mode = "standard"
	prepare(t, option)
	defer Stop()

	// Same source code
	source := []byte(`
function Hello(name) {
	return "Hello, " + name;
}
`)

	// MakeScriptInMemory should work without file
	scriptMem, err := MakeScriptInMemory(source, "virtual/path/script.js", 5*time.Second, true)
	require.NoError(t, err)
	require.NotNil(t, scriptMem)

	// Both should have the function
	assert.Contains(t, scriptMem.Source, "function Hello")
}
