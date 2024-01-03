package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
)

func TestID(t *testing.T) {
	prepare(t)
	res := execOf(t, "session.ID")
	assert.Equal(t, "SID-UNIT-TEST", res)
}

func TestSetGetDel(t *testing.T) {
	prepare(t)
	assert.NotPanics(t, func() {
		execOf(t, "session.Set", "user_id", "UID-01")
	})
	res := execOf(t, "session.Get", "user_id")
	assert.Equal(t, "UID-01", res)

	assert.NotPanics(t, func() {
		execOf(t, "session.Set", "user_id", "UID-02", 1)
	})
	res = execOf(t, "session.Get", "user_id")
	assert.Equal(t, "UID-02", res)

	time.Sleep(2 * time.Second)
	res = execOf(t, "session.Get", "user_id")
	assert.Equal(t, nil, res)

	assert.NotPanics(t, func() {
		execOf(t, "session.Set", "user_id", "UID-03", 1, "SID-UNIT-TEST-2")
	})
	res = execOfSID(t, "SID-UNIT-TEST-2", "session.ID")
	assert.Equal(t, "SID-UNIT-TEST-2", res)

	res = execOfSID(t, "SID-UNIT-TEST-2", "session.Get", "user_id")
	assert.Equal(t, "UID-03", res)

	assert.NotPanics(t, func() {
		execOf(t, "session.SetMany", map[string]interface{}{"user_id": "UID-06", "user_data": "UID-06-DATA"})
	})
	assert.Equal(t, "UID-06", execOf(t, "session.Get", "user_id"))
	assert.Equal(t, "UID-06-DATA", execOf(t, "session.Get", "user_data"))

	assert.NotPanics(t, func() {
		execOf(t, "session.SetMany", map[string]interface{}{"user_id": "UID-07", "user_data": "UID-07-DATA"}, 1)
	})
	assert.Equal(t, "UID-07", execOf(t, "session.Get", "user_id"))
	assert.Equal(t, "UID-07-DATA", execOf(t, "session.Get", "user_data"))

	time.Sleep(2 * time.Second)
	assert.Equal(t, nil, execOf(t, "session.Get", "user_id"))
	assert.Equal(t, nil, execOf(t, "session.Get", "user_data"))

	assert.NotPanics(t, func() {
		execOf(t, "session.SetMany", map[string]interface{}{"user_id": "UID-08", "user_data": "UID-08-DATA"}, 1, "SID-UNIT-TEST-2")
	})

	assert.Equal(t, "UID-08", execOf(t, "session.Get", "user_id", "SID-UNIT-TEST-2"))
	assert.Equal(t, "UID-08-DATA", execOf(t, "session.Get", "user_data", "SID-UNIT-TEST-2"))

	// Test del
	assert.NotPanics(t, func() {
		execOf(t, "session.Set", "user_id", "UID-09", 1)
	})
	res = execOf(t, "session.Get", "user_id")
	assert.Equal(t, "UID-09", res)
	assert.NotPanics(t, func() {
		execOf(t, "session.Del", "user_id")
	})
	assert.Equal(t, nil, execOf(t, "session.Get", "user_id"))

	// Delete Many
	assert.NotPanics(t, func() {
		execOf(t, "session.SetMany", map[string]interface{}{"user_id": "UID-10", "user_data": "UID-10-DATA"})
	})
	res = execOf(t, "session.GetMany", []string{"user_id", "user_data"})
	v, ok := res.(map[string]interface{})
	if !ok {
		t.Fatal("session.GetMany return not map")
	}

	assert.Equal(t, "UID-10", v["user_id"])
	assert.Equal(t, "UID-10-DATA", v["user_data"])
	assert.NotPanics(t, func() {
		execOf(t, "session.DelMany", []string{"user_id", "user_data"})
	})
	assert.Equal(t, nil, execOf(t, "session.Get", "user_id"))
	assert.Equal(t, nil, execOf(t, "session.Get", "user_data"))
}

func TestDump(t *testing.T) {
	prepare(t)
	execOf(t, "session.Set", "user_id", "UID-04")
	execOf(t, "session.Set", "user_data", "UID-04-DATA")
	res := execOf(t, "session.Dump")
	r := any.Of(res).MapStr().Dot()
	assert.Equal(t, "UID-04", r.Get("user_id"))
	assert.Equal(t, "UID-04-DATA", r.Get("user_data"))

	execOfSID(t, "SID-UNIT-TEST-2", "session.Set", "user_id", "UID-05")
	execOfSID(t, "SID-UNIT-TEST-2", "session.Set", "user_data", "UID-05-DATA")

	res = execOf(t, "session.Dump", "SID-UNIT-TEST-2")
	r = any.Of(res).MapStr().Dot()
	assert.Equal(t, "UID-05", r.Get("user_id"))
	assert.Equal(t, "UID-05-DATA", r.Get("user_data"))
}

func TestLang(t *testing.T) {
	prepare(t)
	p := makeP(t, "session.Get")
	lang := Lang(p, "en-us")
	assert.Equal(t, "en-us", lang)

	execOf(t, "session.Set", "__yao_lang", "zh-cn")
	lang = Lang(p, "en-us")
	assert.Equal(t, "zh-cn", lang)
}

func execOf(t *testing.T, name string, args ...interface{}) interface{} {
	p := makeP(t, name, args...)
	return exec(t, p)
}

func execOfSID(t *testing.T, sid string, name string, args ...interface{}) interface{} {
	p := makeP(t, name, args...).WithSID(sid)
	return exec(t, p)
}

func makeP(t *testing.T, name string, args ...interface{}) *process.Process {
	p, err := process.Of(name, args...)
	if err != nil {
		t.Fatal(err)
	}
	return p.WithSID("SID-UNIT-TEST")
}

func exec(t *testing.T, p *process.Process) interface{} {
	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func prepare(t *testing.T) {
	bunt, err := NewBuntDB("")
	if err != nil {
		t.Fatal(err)
	}

	Register("buntdb", bunt)
	Use("buntdb").Make().Expire(3600 * time.Second).AsGlobal()
}
