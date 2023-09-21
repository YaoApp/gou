package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/exception"
)

func TestRegister(t *testing.T) {
	prepare(t)
	keys := map[string]bool{}
	for key := range Handlers {
		keys[key] = true
	}
	checkHandlers(t)
}

func TestAlias(t *testing.T) {
	prepare(t)
	Alias("unit.test.prepare", "unit.test.alias")
	_, has := Handlers["unit.test.alias"]
	assert.True(t, has)
}

func TestNew(t *testing.T) {
	prepare(t)

	var p *Process = nil

	// unit.test.prepare
	assert.NotPanics(t, func() {
		p = New("unit.test.prepare", "foo", "bar")
	})
	assert.Equal(t, "unit.test.prepare", p.Name)
	assert.Equal(t, "unit", p.Group)
	assert.Equal(t, "", p.Method)
	assert.Equal(t, "", p.ID)
	assert.Equal(t, []interface{}{"foo", "bar"}, p.Args)

	// models.widget.Test
	assert.NotPanics(t, func() {
		p = New("models.widget.Test", "foo", "bar")
	})
	assert.Equal(t, "models.widget.Test", p.Name)
	assert.Equal(t, "models", p.Group)
	assert.Equal(t, "Test", p.Method)
	assert.Equal(t, "widget", p.ID)
	assert.Equal(t, []interface{}{"foo", "bar"}, p.Args)

	// models.widget.Id.Test
	assert.NotPanics(t, func() {
		p = New("models.widget.Id.Test", "foo", "bar")
	})
	assert.Equal(t, "models.widget.Id.Test", p.Name)
	assert.Equal(t, "models", p.Group)
	assert.Equal(t, "Test", p.Method)
	assert.Equal(t, "widget.id", p.ID)
	assert.Equal(t, []interface{}{"foo", "bar"}, p.Args)

	// flows.widget
	assert.NotPanics(t, func() {
		p = New("flows.widget", "foo", "bar")
	})
	assert.Equal(t, "flows.widget", p.Name)
	assert.Equal(t, "flows", p.Group)
	assert.Equal(t, "", p.Method)
	assert.Equal(t, "widget", p.ID)
	assert.Equal(t, []interface{}{"foo", "bar"}, p.Args)

	// flows.widget.Id
	assert.NotPanics(t, func() {
		p = New("flows.widget.Id", "foo", "bar")
	})
	assert.Equal(t, "flows.widget.Id", p.Name)
	assert.Equal(t, "flows", p.Group)
	assert.Equal(t, "", p.Method)
	assert.Equal(t, "widget.id", p.ID)
	assert.Equal(t, []interface{}{"foo", "bar"}, p.Args)

	// session.Get
	assert.NotPanics(t, func() {
		p = New("session.Get", "foo", "bar")
	})
	assert.Equal(t, "session.Get", p.Name)
	assert.Equal(t, "session", p.Group)
	assert.Equal(t, "Get", p.Method)
	assert.Equal(t, "", p.ID)
	assert.Equal(t, []interface{}{"foo", "bar"}, p.Args)

	// not_found
	assert.PanicsWithValue(t, *exception.New("not_found not found", 404), func() {
		p = New("not_found", "foo", "bar")
	})
}

func TestRun(t *testing.T) {

	prepare(t)
	var p *Process = nil

	// unit.test.prepare
	p = New("unit.test.prepare", "foo", "bar")
	assert.NotPanics(t, func() {
		res := p.Run()
		data, ok := res.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "unit", data["group"])
		assert.Equal(t, "", data["method"])
		assert.Equal(t, "", data["id"])
		assert.Equal(t, []interface{}{"foo", "bar"}, data["args"])
	})

	// models.widget.Test
	p = New("models.widget.Test", "foo", "bar")
	assert.NotPanics(t, func() {
		res := p.Run()
		data, ok := res.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "models", data["group"])
		assert.Equal(t, "Test", data["method"])
		assert.Equal(t, "widget", data["id"])
		assert.Equal(t, []interface{}{"foo", "bar"}, data["args"])
	})

	// flows.widget
	p = New("flows.widget", "foo", "bar")
	assert.NotPanics(t, func() {
		res := p.Run()
		data, ok := res.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "flows", data["group"])
		assert.Equal(t, "", data["method"])
		assert.Equal(t, "widget", data["id"])
		assert.Equal(t, []interface{}{"foo", "bar"}, data["args"])
	})

	// session.Get
	p = New("session.Get", "foo", "bar")
	assert.NotPanics(t, func() {
		res := p.Run()
		data, ok := res.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "session", data["group"])
		assert.Equal(t, "Get", data["method"])
		assert.Equal(t, "", data["id"])
		assert.Equal(t, []interface{}{"foo", "bar"}, data["args"])
	})

	// models.widget.Notfound
	p = New("models.widget.Notfound", "foo", "bar")
	assert.PanicsWithValue(t, *exception.New("models.widget.Notfound Handler -> models.notfound not found", 404), func() {
		p.Run()
	})
}

func TestExec(t *testing.T) {

	prepare(t)
	var p *Process = nil

	// unit.test.prepare
	p = New("unit.test.prepare", "foo", "bar")
	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	data, ok := res.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "unit", data["group"])
	assert.Equal(t, "", data["method"])
	assert.Equal(t, "", data["id"])
	assert.Equal(t, []interface{}{"foo", "bar"}, data["args"])

	// models.widget.Test
	p = New("models.widget.Test", "foo", "bar")
	res, err = p.Exec()
	data, ok = res.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "models", data["group"])
	assert.Equal(t, "Test", data["method"])
	assert.Equal(t, "widget", data["id"])
	assert.Equal(t, []interface{}{"foo", "bar"}, data["args"])

	// flows.widget
	p = New("flows.widget", "foo", "bar")
	res, err = p.Exec()
	data, ok = res.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "flows", data["group"])
	assert.Equal(t, "", data["method"])
	assert.Equal(t, "widget", data["id"])
	assert.Equal(t, []interface{}{"foo", "bar"}, data["args"])

	// session.Get
	p = New("session.Get", "foo", "bar")
	res, err = p.Exec()
	data, ok = res.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "session", data["group"])
	assert.Equal(t, "Get", data["method"])
	assert.Equal(t, "", data["id"])
	assert.Equal(t, []interface{}{"foo", "bar"}, data["args"])

	// models.widget.Notfound
	p = New("models.widget.Notfound", "foo", "bar")
	res, err = p.Exec()
	assert.Equal(t, nil, res)
	assert.Equal(t, "Exception|404:models.widget.Notfound Handler -> models.notfound not found", err.Error())
}

func TestWithSID(t *testing.T) {

	prepare(t)
	var p *Process = nil

	// unit.test.prepare
	p = New("unit.test.prepare", "foo", "bar").WithSID("101")
	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	data, ok := res.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "101", data["sid"])
}

func TestWithGlobal(t *testing.T) {
	prepare(t)
	var p *Process = nil

	// unit.test.prepare
	p = New("unit.test.prepare", "foo", "bar").WithGlobal(map[string]interface{}{"hello": "world"})
	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	data, ok := res.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, map[string]interface{}{"hello": "world"}, data["global"])
}

func prepare(t *testing.T) {
	Register("unit.test.prepare", processTest)
	Register("flows", processTest)
	RegisterGroup("models", map[string]Handler{"Test": processTest})
	RegisterGroup("session", map[string]Handler{"Get": processTest})
}

func processTest(process *Process) interface{} {
	return map[string]interface{}{
		"group":  process.Group,
		"method": process.Method,
		"id":     process.ID,
		"args":   process.Args,
		"sid":    process.Sid,
		"global": process.Global,
	}
}

func checkHandlers(t *testing.T) {
	keys := map[string]bool{}
	for key := range Handlers {
		keys[key] = true
	}
	assert.True(t, keys["flows"])
	assert.True(t, keys["models.test"])
	assert.True(t, keys["session.get"])
	assert.True(t, keys["unit.test.prepare"])
}
