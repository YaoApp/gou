package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadSocket(t *testing.T) {
	sock, err := LoadSocket("file://"+path.Join(TestServerRoot, "rfid.sock.json"), "rfid")
	assert.Nil(t, err)
	assert.Equal(t, sock.Name, "RFID Receiver (Server Mode)")
	assert.Equal(t, sock.Mode, "server")
	assert.Equal(t, sock.Process, "flows.rfid.read")
	assert.Equal(t, sock.Port, "3019")
	assert.Equal(t, sock.Host, "0.0.0.0")
}

func TestSocketStart(t *testing.T) {
	LoadSocket("file://"+path.Join(TestServerRoot, "rfid.sock.json"), "rfid")
	sock := SelectSocket("rfid")
	assert.Equal(t, sock.Name, "RFID Receiver (Server Mode)")
	assert.Equal(t, sock.Mode, "server")
	assert.Equal(t, sock.Process, "flows.rfid.read")
	assert.Equal(t, sock.Port, "3019")
	assert.Equal(t, sock.Host, "0.0.0.0")

	// sock.Start()
}

func TestSocketConnect(t *testing.T) {
	LoadSocket("file://"+path.Join(TestServerRoot, "rfid_client.sock.json"), "rfid_client")
	sock := SelectSocket("rfid_client")
	assert.Equal(t, sock.Name, "RFID Receiver (Client Mode)")
	assert.Equal(t, sock.Mode, "client")
	assert.Equal(t, sock.Process, "flows.rfid.read")
	assert.Equal(t, sock.Port, "6000")
	assert.Equal(t, sock.Host, "192.168.1.192")

	// sock.Connect()
}
