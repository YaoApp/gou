package v8

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestProcessScripts(t *testing.T) {
	prepare(t)
	assert.Equal(t, 2, isolates.Len)
	assert.Equal(t, 2, len(chIsoReady))

	p, err := process.Of("scripts.runtime.basic.Hello", "world")
	if err != nil {
		t.Fatal(err)
	}

	value, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "world", value)
}

func TestProcessScriptsRoot(t *testing.T) {
	prepare(t)
	assert.Equal(t, 2, isolates.Len)
	assert.Equal(t, 2, len(chIsoReady))

	p, err := process.Of("studio.runtime.basic.Hello", "world")
	if err != nil {
		t.Fatal(err)
	}

	value, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "world", value)
}
