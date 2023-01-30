package v8

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	prepare(t)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 1, len(Scripts))
	assert.Equal(t, 1, len(RootScripts))
	assert.Equal(t, 1, len(WidgetScripts))
	assert.Equal(t, 2, len(chIsoReady))
}
