package service

import "github.com/takama/daemon"

// Service embedded daemon
//
//	{
//		"name": "Server for receiving RFID",
//		"description": "Server for receiving RFID",
//		"version": "0.9.2",
//		"restart": "on-failure",
//		"requires": ["servers.rfid_server"],
//		"after": ["servers.rfid_server"],
//		"error": "/var/log/test.err"
//		"output": "/var/log/test.log"
//		"process": "servers.rfid_client",
//		"args": ["192.168.1.192", 6000],
//	 "user": "root",
//	 "group": "root"
//	}
type Service struct {
	name        string
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	Process     string   `json:"process,omitempty"`
	Command     string   `json:"command,omitempty"`
	Requires    []string `json:"requires,omitempty"`
	After       []string `json:"after,omitempty"`
	Restart     string   `json:"restart,omitempty"`
	WorkDir     string   `json:"workdir,omitempty"`
	Args        []string `json:"args,omitempty"`
	LogError    string   `json:"error,omitempty"`
	LogOutput   string   `json:"output,omitempty"`
	User        string   `json:"user,omitempty"`
	Group       string   `json:"group,omitempty"`
	daemon.Daemon
}
