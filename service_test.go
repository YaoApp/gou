package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadService(t *testing.T) {
	service, err := LoadService("file://"+path.Join(TestServiceRoot, "cmd.srv.json"), "cmd")
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	assert.Equal(t, "tail", service.Command)

	service, err = LoadService("file://"+path.Join(TestServiceRoot, "process.srv.json"), "process")
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	assert.Equal(t, "xiang.sys.ping", service.Process)
}
