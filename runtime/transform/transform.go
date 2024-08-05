package transform

import (
	"fmt"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// TypeScript transform the typescript code to javascript code
func TypeScript(tsCode string, option api.TransformOptions) (string, error) {
	option.Loader = api.LoaderTS
	result := api.Transform(tsCode, option)
	if len(result.Errors) > 0 {
		errors := []string{}
		for _, err := range result.Errors {
			errors = append(errors, fmt.Sprintf("%s", err.Text))
		}
		return "", fmt.Errorf("transform ts code error: %v", strings.Join(errors, "\n"))
	}
	return string(result.Code), nil
}

// TypeScriptWithSourceMap transform the typescript code with source map
func TypeScriptWithSourceMap(tsCode string, option api.TransformOptions) ([]byte, []byte, error) {
	option.Loader = api.LoaderTS
	option.Sourcemap = api.SourceMapExternal
	result := api.Transform(tsCode, option)
	if len(result.Errors) > 0 {
		errors := []string{}
		for _, err := range result.Errors {
			errors = append(errors, fmt.Sprintf("%s", err.Text))
		}
		return nil, nil, fmt.Errorf("transform ts code error: %v", strings.Join(errors, "\n"))
	}
	return result.Code, result.Map, nil
}

// JavaScriptWithSourceMap transform the javascript code with source map
func JavaScriptWithSourceMap(jsCode string, option api.TransformOptions) ([]byte, []byte, error) {

	option.Sourcemap = api.SourceMapExternal
	result := api.Transform(jsCode, option)
	if len(result.Errors) > 0 {
		errors := []string{}
		for _, err := range result.Errors {
			errors = append(errors, fmt.Sprintf("%s", err.Text))
		}
		return nil, nil, fmt.Errorf("transform js code error: %v", strings.Join(errors, "\n"))
	}
	return result.Code, result.Map, nil
}

// JavaScript transform the javascript code
func JavaScript(jsCode string, option api.TransformOptions) (string, error) {
	result := api.Transform(jsCode, option)
	if len(result.Errors) > 0 {
		errors := []string{}
		for _, err := range result.Errors {
			errors = append(errors, fmt.Sprintf("%s", err.Text))
		}
		return "", fmt.Errorf("transform js code error: %v", strings.Join(errors, "\n"))
	}
	return string(result.Code), nil
}

// MinifyJS minify the javascript code
func MinifyJS(jsCode string, target ...api.Target) (string, error) {

	t := api.ES2015
	if len(target) > 0 {
		t = target[0]
	}
	result := api.Transform(jsCode, api.TransformOptions{
		Loader:            api.LoaderJS,
		MinifyWhitespace:  true,
		MinifyIdentifiers: false,
		MinifySyntax:      true,
		Target:            t,
	})

	if len(result.Errors) > 0 {
		errors := []string{}
		for _, err := range result.Errors {
			errors = append(errors, fmt.Sprintf("%s", err.Text))
		}
		return "", fmt.Errorf("transform js code error: %v", strings.Join(errors, "\n"))
	}
	return string(result.Code), nil
}

// MinifyCSS  minify the css code
func MinifyCSS(cssCode string) (string, error) {

	result := api.Transform(cssCode, api.TransformOptions{
		Loader:            api.LoaderCSS,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
	})
	if len(result.Errors) > 0 {
		errors := []string{}
		for _, err := range result.Errors {
			errors = append(errors, fmt.Sprintf("%s", err.Text))
		}
		return "", fmt.Errorf("transform less code error: %v", strings.Join(errors, "\n"))
	}
	return string(result.Code), nil
}
