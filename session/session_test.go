package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	MemoryLocalServer()
}

func TestMake(t *testing.T) {
	s := Use("memory").Make().Expire(3600 * time.Second).AsGlobal()
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

	s.MustSetMany(map[string]interface{}{"hello": "world", "hi": "gou"})
	assert.Equal(t, "world", s.MustGet("hello"))
	assert.Equal(t, "gou", s.MustGet("hi"))

	time.Sleep(5001 * time.Microsecond)
	assert.Nil(t, s.MustGet("foo"))
	assert.Nil(t, s.MustGet("hello"))
	assert.Nil(t, s.MustGet("hi"))
}

func TestMustSetWithEx(t *testing.T) {
	id := ID()
	ss := Use("memory").ID(id)
	ss.MustSetWithEx("foo", "bar", 5000*time.Microsecond)
	assert.Equal(t, "bar", ss.MustGet("foo"))

	ss.MustSetManyWithEx(map[string]interface{}{"hello": "world", "hi": "gou"}, 5000*time.Microsecond)
	assert.Equal(t, "world", ss.MustGet("hello"))
	assert.Equal(t, "gou", ss.MustGet("hi"))

	time.Sleep(5001 * time.Microsecond)
	assert.Nil(t, ss.MustGet("foo"))
	assert.Nil(t, ss.MustGet("hello"))
	assert.Nil(t, ss.MustGet("hi"))
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
