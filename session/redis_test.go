package session

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {

	host := os.Getenv("GOU_TEST_REDIS_HOST")
	port := os.Getenv("GOU_TEST_REDIS_PORT")
	db := os.Getenv("GOU_TEST_REDIS_DB")
	pass := os.Getenv("GOU_TEST_REDIS_PASSWORD")

	args := []string{}
	if port != "" {
		args = append(args, port)
	}

	if db != "" {
		args = append(args, db)
	}

	if pass != "" {
		args = append(args, pass)
	}

	rdb, err := NewRedis(host, args...)
	if err != nil {
		panic(err)
	}
	Register("redis", rdb)
}

func TestRedisMake(t *testing.T) {
	s := Use("redis").Make().Expire(3600 * time.Second).AsGlobal()
	assert.NotNil(t, s.GetID())
}

func TestRedisID(t *testing.T) {
	id := ID()
	s := Use("redis").ID(id)
	assert.Equal(t, id, s.GetID())
}

func TestRedisMustSetGetDel(t *testing.T) {
	id := ID()
	s := Use("redis").ID(id).Expire(200 * time.Millisecond)
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

func TestRedisMustSetWithEx(t *testing.T) {
	id := ID()
	ss := Use("redis").ID(id)
	ss.MustSetWithEx("foo", "bar", 200*time.Millisecond)
	assert.Equal(t, "bar", ss.MustGet("foo"))

	ss.MustSetManyWithEx(map[string]interface{}{"hello": "world", "hi": "gou"}, 200*time.Millisecond)
	assert.Equal(t, "world", ss.MustGet("hello"))
	assert.Equal(t, "gou", ss.MustGet("hi"))

	time.Sleep(210 * time.Millisecond)
	assert.Nil(t, ss.MustGet("foo"))
	assert.Nil(t, ss.MustGet("hello"))
	assert.Nil(t, ss.MustGet("hi"))
}

func TestRedisMustDump(t *testing.T) {
	id := ID()
	ss := Use("redis").ID(id).Expire(200 * time.Millisecond)
	ss.MustSet("foo", "bar")
	ss.MustSet("hello", "world")

	data := ss.MustDump()
	assert.Equal(t, "bar", data["foo"])
	assert.Equal(t, "world", data["hello"])

	time.Sleep(201 * time.Millisecond)
	data = ss.MustDump()
	assert.Equal(t, map[string]interface{}{}, data)
}
