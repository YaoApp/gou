package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMake(t *testing.T) {
	s := Use("memory").Make().Expire(3600 * time.Second)
	assert.NotNil(t, s.GetID())
}

func TestID(t *testing.T) {
	id := ID()
	s := Use("memory").ID(id)
	assert.Equal(t, id, s.GetID())
}

func TestMustSetGet(t *testing.T) {
	id := ID()
	s := Use("memory").ID(id).Expire(5000 * time.Microsecond)
	s.MustSet("foo", "bar")
	v := s.MustGet("foo")
	assert.Equal(t, "bar", v)

	time.Sleep(5001 * time.Microsecond)
	v = s.MustGet("foo")
	assert.Nil(t, v)
}

func TestMustSetWithEx(t *testing.T) {
	id := ID()
	ss := Use("memory").ID(id)
	ss.MustSetWithEx("foo", "bar", 5000*time.Microsecond)
	assert.Equal(t, "bar", ss.MustGet("foo"))

	time.Sleep(5001 * time.Microsecond)
	assert.Nil(t, ss.MustGet("foo"))
}

func TestMustDump(t *testing.T) {
	id := ID()
	ss := Use("memory").ID(id).Expire(5000 * time.Microsecond)
	ss.MustSet("foo", "bar")
	ss.MustSet("hello", "world")

	data := ss.MustDump()
	assert.Equal(t, "bar", data["foo"])
	assert.Equal(t, "world", data["hello"])

	time.Sleep(5001 * time.Microsecond)
	data = ss.MustDump()
	assert.Equal(t, map[string]interface{}{}, data)
}
