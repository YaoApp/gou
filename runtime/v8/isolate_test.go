package v8

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	prepare(t)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 2, isolates.Len)
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
	assert.Equal(t, 10, isolates.Len)

	var res error
	for i := 0; i < 5; i++ {
		if _, err := SelectIso(time.Millisecond * 100); err != nil {
			res = err
		}
	}
	assert.NotNil(t, res)
}

func TestResize(t *testing.T) {
	prepare(t)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 2, isolates.Len)
	assert.Equal(t, 2, len(chIsoReady))

	isolates.Resize(10, 20)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 10, isolates.Len)
	assert.Equal(t, 10, len(chIsoReady))
	assert.Equal(t, 20, runtimeOption.MaxSize)
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
