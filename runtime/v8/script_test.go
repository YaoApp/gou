package v8

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	prepare(t)
	assert.Equal(t, 1, len(Scripts))
	assert.Equal(t, 1, len(RootScripts))
	assert.Equal(t, 2, len(chIsoReady))
}
