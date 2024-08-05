package transform

import (
	"testing"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestTypeScript(t *testing.T) {
	inputCode := `
		const message: string = "Hello, TypeScript!";
		console.log(message);
		add(a, b)
		hello( "World" )

		function add(a: number, b: number) {
			return a + b;
		}

		const hello = (name: string) => {
			console.log(hello, name)
		}
	`
	jsCode, err := TypeScript(inputCode, api.TransformOptions{
		Target:            api.ES2015,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
	})

	if err != nil {
		t.Errorf("transform ts code error: %v", err)
	}

	assert.Equal(t, `const message="Hello, TypeScript!";console.log(message),add(a,b),hello("World");function add(o,n){return o+n}const hello=o=>{console.log(hello,o)};`+"\n", jsCode)
}

func TestTypeScriptWithSourceMap(t *testing.T) {
	inputCode := `
		const message: string = "Hello, TypeScript!";
		console.log(message);
		add(a, b)
		hello( "World" )

		function add(a: number, b: number) {
			return a + b;
		}

		const hello = (name: string) => {
			console.log(hello, name)
		}
	`
	jsCode, sourcemap, err := TypeScriptWithSourceMap(inputCode, api.TransformOptions{
		Target:            api.ES2015,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
	})

	if err != nil {
		t.Errorf("transform ts code error: %v", err)
	}

	assert.Equal(t, `const message="Hello, TypeScript!";console.log(message),add(a,b),hello("World");function add(o,n){return o+n}const hello=o=>{console.log(hello,o)};`+"\n", string(jsCode))
	assert.Contains(t, string(sourcemap), "mappings")
}

func TestJavaScript(t *testing.T) {
	inputCode := `
		const message = "Hello, JavaScript!";
		console.log(message);
		add(a, b)
		hello( "World" )

		function add(a, b) {
			return a + b;
		}

		const hello = (name) => {
			console.log(hello, name)
		}
	`
	jsCode, err := JavaScript(inputCode, api.TransformOptions{
		Target:            api.ES2015,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
	})

	if err != nil {
		t.Errorf("transform js code error: %v", err)
	}
	assert.Equal(t, `const message="Hello, JavaScript!";console.log(message),add(a,b),hello("World");function add(o,l){return o+l}const hello=o=>{console.log(hello,o)};`+"\n", jsCode)
}

func TestJavaScriptWithSourceMap(t *testing.T) {
	inputCode := `
		const message = "Hello, JavaScript!";
		console.log(message);
		add(a, b)
		hello( "World" )

		function add(a, b) {
			return a + b;
		}

		const hello = (name) => {
			console.log(hello, name)
		}
	`
	jsCode, sourcemap, err := JavaScriptWithSourceMap(inputCode, api.TransformOptions{
		Target:            api.ES2015,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
	})

	if err != nil {
		t.Errorf("transform js code error: %v", err)
	}
	assert.Equal(t, `const message="Hello, JavaScript!";console.log(message),add(a,b),hello("World");function add(o,l){return o+l}const hello=o=>{console.log(hello,o)};`+"\n", string(jsCode))
	assert.Contains(t, string(sourcemap), "mappings")
}

func TestMinifyCSS(t *testing.T) {
	inputCode := `
		.bordered {
		border-top: dotted 1px black;
		border-bottom: solid 2px black;
	  }
	`
	cssCode, err := MinifyCSS(inputCode)

	if err != nil {
		t.Errorf("transform ts code error: %v", err)
	}
	assert.Equal(t, `.bordered{border-top:dotted 1px black;border-bottom:solid 2px black}`+"\n", cssCode)
}

func TestMinifyJS(t *testing.T) {
	inputCode := `
		function hello( a ) {
			return a
		}
		hello()
	`
	jsCode, err := MinifyJS(inputCode)
	if err != nil {
		t.Errorf("transform ts code error: %v", err)
	}

	assert.Equal(t, "function hello(a){return a}hello();\n", jsCode)
}
