package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadServer(t *testing.T) {
	srv, err := LoadServer("file://"+path.Join(TestServerRoot, "rfid.sock.json"), "rfid")
	assert.Nil(t, err)
	assert.Equal(t, srv.Name, "RFID接收器")
	assert.Equal(t, srv.Process, "flows.rfid.read")
	assert.Equal(t, srv.Port, "3019")
	assert.Equal(t, srv.Host, "0.0.0.0")
}

func TestServerStart(t *testing.T) {
	LoadServer("file://"+path.Join(TestServerRoot, "rfid.sock.json"), "rfid")
	srv := SelectServer("rfid")
	assert.Equal(t, srv.Name, "RFID接收器")
	assert.Equal(t, srv.Process, "flows.rfid.read")
	assert.Equal(t, srv.Port, "3019")
	assert.Equal(t, srv.Host, "0.0.0.0")

	// srv.Start()
}
