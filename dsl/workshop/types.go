package workshop

import "github.com/blang/semver/v4"

// Workshop the required packages
type Workshop struct {
	Require []*Package        `json:"require,omitempty"`
	Replace map[string]string `json:"replace,omitempty"` // for multi projects development
	Mapping map[string]*Package
	root    string // the root path
	file    string // the workshop.yao file path
	cfg     Config
}

// Package the YAO package info
type Package struct {
	URL        string         // github.com/yaoapp/demo-wms/cloud@v0.0.0-20220223010332-e86eab4c8490
	Name       string         // github.com/yaoapp/demo-wms/cloud
	Alias      string         // github.com/yaoapp/demo-wms/cloud
	Addr       string         // github.com/yaoapp/demo-wms
	Domain     string         // github.com
	Owner      string         // trheyi
	Repo       string         // demo-wms
	Path       string         // /cloud
	Version    semver.Version // 0.0.0-e86eab4c8490
	Rel        string         // e86eab4c8490 ( 0.9.2 / v0.9.1 / master )
	LocalPath  string         //
	Downloaded bool           // true
	Replaced   bool           // false
	Unique     string         // github.com/yaoapp/demo-wms@e86eab4c8490
	Indirect   bool           // true
	Parents    []string       // parent
}

// Config the workshop config file
type Config map[string]map[string]interface{}

const (
	// RootEnvName the environment variable name of the workshop root path in the local disk
	RootEnvName = "YAO_PATH"
)
