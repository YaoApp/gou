package flow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
)

func TestProcess(t *testing.T) {
	prepare(t)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	p, err := process.Of("flows.basic", yesterday, tomorrow)
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	r := any.Of(res).MapStr().Dot()
	assert.Equal(t, yesterday, r.Get("params[0]"))
	assert.Equal(t, tomorrow, r.Get("params[1]"))
	assert.Equal(t, "U1", r.Get("data.query[0].name"))
	assert.Equal(t, "Duck", r.Get("data.categories[2].name"))
	assert.Equal(t, "U3", r.Get("data.users[1].name"))
}
