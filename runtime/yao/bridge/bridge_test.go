package bridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"rogchap.com/v8go"
)

func TestMustAnyToValue(t *testing.T) {
	ctx := v8go.NewContext()
	v := MustAnyToValue(ctx, 0.618)
	assert.True(t, v.IsNumber())
}
