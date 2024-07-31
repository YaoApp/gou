package v8

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTSConfigGetFileName(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.Import = true
	option.HeapSizeLimit = 4294967296

	// add tsconfig
	tsconfig := &TSConfig{
		CompilerOptions: &TSConfigCompilerOptions{
			Paths: map[string][]string{
				"@yao/*": {"./scripts/.types/*"},
				"@lib/*": {"./scripts/runtime/ts/lib/*"},
			},
		},
	}
	option.TSConfig = tsconfig
	prepare(t, option)
	defer Stop()

	file, match, err := tsconfig.GetFileName("@lib/foo")
	if err != nil {
		t.Fatal(err)
	}

	if !match {
		t.Fatal("not match")
	}

	assert.Equal(t, "scripts/runtime/ts/lib/foo.ts", file)

}
