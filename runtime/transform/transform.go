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

// MinifyJS minify the javascript code
func MinifyJS(jsCode string) (string, error) {
	result := api.Transform(jsCode, api.TransformOptions{
		Loader:            api.LoaderJS,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
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
