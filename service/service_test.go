package service

import (
	"testing"
)

const testDaemon = `
{
    "name":"Service for receiving RFID",
    "description":"Service for receiving RFID",
    "version":"0.9.2",
    "restart":"on-failure",
    "requires":["syslog"],
    "after":["syslog"],
    "error": "/var/log/test.err",
    "output":"/var/log/test.log",
	"workdir":"~",
    "command":"tail",
    "args":["-f", "/dev/null"],
	"user": "root",
 	"group": "root"
}`

func TestDaemon(t *testing.T) {

	// service, err := Load("test", []byte(testDaemon))
	// if err != nil {
	// 	t.Fatalf(err.Error())
	// }

	// // prepare
	// service.Stop()
	// service.Remove()

	// status, err := service.Install()
	// if err != nil {
	// 	t.Fatalf(err.Error())
	// }

	// assert.Contains(t, status, "OK")

	// status, err = service.Start()
	// if err != nil {
	// 	t.Fatalf(err.Error())
	// }
	// assert.Contains(t, status, "OK")

	// status, err = service.Status()
	// if err != nil {
	// 	t.Fatalf(err.Error())
	// }
	// assert.Contains(t, status, "running")

	// status, err = service.Stop()
	// if err != nil {
	// 	t.Fatalf(err.Error())
	// }
	// assert.Contains(t, status, "OK")

	// status, err = service.Status()
	// if err != nil {
	// 	t.Fatalf(err.Error())
	// }
	// assert.Contains(t, status, "stopped")

	// service.Remove()

}
