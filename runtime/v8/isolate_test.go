package v8

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	prepare(t)
	assert.Equal(t, 2, len(isolates))
	assert.Equal(t, 2, len(chIsoReady))
}

func TestSelectIso(t *testing.T) {
	prepare(t)
	for i := 0; i < 10; i++ {
		_, err := SelectIso(time.Millisecond * 100)
		if err != nil {
			t.Fatal(fmt.Errorf("%d %s", i, err.Error()))
		}
	}
	assert.Equal(t, 10, len(isolates))

	var res error
	for i := 0; i < 5; i++ {
		if _, err := SelectIso(time.Millisecond * 100); err != nil {
			res = err
		}
	}
	assert.NotNil(t, res)
}

func TestUnlock(t *testing.T) {
	prepare(t)
	iso, err := SelectIso(time.Millisecond * 100)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, iso.Locked())

	size := len(chIsoReady)
	iso.Unlock()
	assert.False(t, iso.Locked())
	assert.Equal(t, size+1, len(chIsoReady))
}
