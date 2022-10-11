package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal"
)

// Xun the xun database ORM
type Xun struct {
	id      string
	Manager *capsule.Manager `json:"-"`
	Name    string           `json:"name,omitempty"`
	Driver  string           `json:"type"`
	Version string           `json:"version,omitempty"`
	Options XunOptions       `json:"options"`
}

// XunOptions the connetion options
type XunOptions struct {
	DB          string    `json:"db"`
	TablePrefix string    `json:"prefix"`
	Collation   string    `json:"collation,omitempty"`
	Charset     string    `json:"charset,omitempty"`
	ParseTime   bool      `json:"parseTime,omitempty"`
	Timeout     int       `json:"timeout,omitempty"`
	File        string    `json:"file,omitempty"`
	Hosts       []XunHost `json:"hosts"`
}

// XunHost the connection host
type XunHost struct {
	File    string `json:"file,omitempty"`
	Host    string `json:"host,omitempty"`
	Port    string `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	Pass    string `json:"pass,omitempty"`
	Primary bool   `json:"primary,omitempty"`
	dsn     string
}

// Register the connections from dsl
func (x *Xun) Register(id string, dsl []byte) error {
	err := jsoniter.Unmarshal(dsl, x)
	if err != nil {
		return err
	}

	err = x.setDefaults()
	if err != nil {
		return err
	}
	x.id = id
	return x.makeConnections()
}

// Is the connections from dsl
func (x *Xun) Is(typ int) bool {
	return 1 == typ
}

// ID get connector id
func (x *Xun) ID() string {
	return x.id
}

func (x *Xun) makeConnections() (err error) {

	defer func() { err = exception.Catch(recover()) }()
	manager := capsule.NewWithOption(dbal.Option{
		Prefix:    x.Options.TablePrefix,
		Charset:   x.Options.Collation,
		Collation: x.Options.Collation,
	})

	for i, host := range x.Options.Hosts {
		name := fmt.Sprintf("%s_%d", x.Name, i)
		if host.Primary {
			manager.AddConn(name, x.Driver, host.dsn, time.Duration(x.Options.Timeout)*time.Second)
			continue
		}

		manager.AddReadConn(name, x.Driver, host.dsn, time.Duration(x.Options.Timeout)*time.Second)
	}

	x.Manager = manager
	return err
}

func (x *Xun) setDefaults() error {
	x.Options.DB = helper.EnvString(x.Options.DB)
	x.Options.Timeout = helper.EnvInt(x.Options.Timeout, 5)
	if x.Options.Timeout == 0 {
		x.Options.Timeout = 5
	}

	// for sqlite3
	if x.Options.File != "" {
		x.Options.Hosts = append(x.Options.Hosts, XunHost{File: x.Options.File})
	}

	for i := range x.Options.Hosts {
		x.Options.Hosts[i].Host = helper.EnvString(x.Options.Hosts[i].Host)
		x.Options.Hosts[i].Pass = helper.EnvString(x.Options.Hosts[i].Pass)
		x.Options.Hosts[i].User = helper.EnvString(x.Options.Hosts[i].User)
		x.Options.Hosts[i].Port = helper.EnvString(x.Options.Hosts[i].Port)
		x.Options.Hosts[i].File = helper.EnvString(x.Options.Hosts[i].File)

		dsn, err := x.getDSN(i)
		if err != nil {
			return err
		}
		x.Options.Hosts[i].dsn = dsn
	}
	return nil
}

// getDSN get the DSN
func (x *Xun) getDSN(i int) (string, error) {
	switch x.Driver {
	case "mysql":
		return x.mysqlDSN(i)
	case "sqlite3":
		return x.sqlite3DSN(i)
	}

	return "", fmt.Errorf("the driver %s does not support", x.Driver)
}

func (x *Xun) sqlite3DSN(i int) (string, error) {
	host := x.Options.Hosts[i]
	if host.File == "" {
		return "", fmt.Errorf("options.file is required")
	}

	file, err := filepath.Abs(host.File)
	if err != err {
		return "", fmt.Errorf("options.file %s", err.Error())
	}

	root := filepath.Dir(file)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		err = os.MkdirAll(root, os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("options.file %s", err.Error())
		}
	}

	return file, nil
}

func (x *Xun) mysqlDSN(i int) (string, error) {

	if x.Options.DB == "" {
		return "", fmt.Errorf("options.db is required")
	}

	if len(x.Options.Hosts) == 0 {
		return "", fmt.Errorf("options.hosts is required")
	}

	host := x.Options.Hosts[i]
	if host.Host == "" {
		return "", fmt.Errorf("hosts.%d.host is required", i)
	}

	if host.Port == "" {
		host.Port = "3306"
	}

	if host.User == "" {
		return "", fmt.Errorf("hosts.%d.user is required", i)
	}

	if host.Pass == "" {
		return "", fmt.Errorf("hosts.%d.pass is required", i)
	}

	params := []string{}
	if x.Options.Charset != "" {
		params = append(params, fmt.Sprintf("charset=%s", x.Options.Charset))
	}

	if x.Options.ParseTime {
		params = append(params, fmt.Sprintf("parseTime=True"))
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", host.User, host.Pass, host.Host, host.Port, x.Options.DB)
	if len(params) > 0 {
		dsn = dsn + "?" + strings.Join(params, "&")
	}

	return dsn, nil
}
