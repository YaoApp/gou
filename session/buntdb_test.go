package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	bunt, err := NewBuntDB("")
	if err != nil {
		panic(err)
	}
	Register("buntdb", bunt)
}

func TestBuntDBMake(t *testing.T) {
	s := Use("buntdb").Make().Expire(3600 * time.Second).AsGlobal()
	assert.NotNil(t, s.GetID())
}

func TestBuntDBID(t *testing.T) {
	id := ID()
	s := Use("buntdb").ID(id)
	assert.Equal(t, id, s.GetID())
}

func TestBuntDBMustSetGetDel(t *testing.T) {
	id := ID()
	s := Use("buntdb").ID(id).Expire(200 * time.Millisecond)
	s.MustSet("foo", "bar")
	v := s.MustGet("foo")
	assert.Equal(t, "bar", v)

	s.MustSetMany(map[string]interface{}{"hello": "world", "hi": "gou"})
	assert.Equal(t, "world", s.MustGet("hello"))
	assert.Equal(t, "gou", s.MustGet("hi"))

	s.MustDel("hi")
	assert.Nil(t, s.MustGet("hi"))

	time.Sleep(201 * time.Millisecond)
	assert.Nil(t, s.MustGet("foo"))
	assert.Nil(t, s.MustGet("hello"))
	assert.Nil(t, s.MustGet("hi"))
}

func TestBuntDBMustSetWithEx(t *testing.T) {
	id := ID()
	ss := Use("buntdb").ID(id)
	ss.MustSetWithEx("foo", "bar", 200*time.Millisecond)
	assert.Equal(t, "bar", ss.MustGet("foo"))

	ss.MustSetManyWithEx(map[string]interface{}{"hello": "world", "hi": "gou"}, 200*time.Millisecond)
	assert.Equal(t, "world", ss.MustGet("hello"))
	assert.Equal(t, "gou", ss.MustGet("hi"))

	time.Sleep(201 * time.Millisecond)
	assert.Nil(t, ss.MustGet("foo"))
	assert.Nil(t, ss.MustGet("hello"))
	assert.Nil(t, ss.MustGet("hi"))
}

func TestBuntDBMustDump(t *testing.T) {
	id := ID()
	ss := Use("buntdb").ID(id).Expire(200 * time.Millisecond)
	ss.MustSet("foo", "bar")
	ss.MustSet("hello", "world")

	data := ss.MustDump()
	assert.Equal(t, "bar", data["foo"])
	assert.Equal(t, "world", data["hello"])

	time.Sleep(1000 * time.Millisecond)
	data = ss.MustDump()
	assert.Equal(t, map[string]interface{}{}, data)
}
