package m

import (
	"context"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/xun/dbal/query"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connector the ConnectorDB struct
type Connector struct {
	id       string
	file     string
	Name     string          `json:"name,omitempty"`
	Version  string          `json:"version,omitempty"`
	Options  Options         `json:"options"`
	Client   *mongo.Client   `json:"-"`
	Database *mongo.Database `json:"-"`
}

// Options the connetion options
type Options struct {
	DB      string                 `json:"db"`
	Timeout int                    `json:"timeout,omitempty"`
	Hosts   []Host                 `json:"hosts"`
	Params  map[string]interface{} `json:"params"`
	dsn     string
}

// Host the connection host
type Host struct {
	Host string `json:"host,omitempty"`
	Port string `json:"port,omitempty"`
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
}

// Register the connections from dsl
func (m *Connector) Register(file string, id string, dsl []byte) error {

	m.id = id
	m.file = file

	err := application.Parse(file, dsl, m)
	if err != nil {
		return err
	}

	err = m.setDefaults()
	if err != nil {
		return err
	}

	return m.makeConnection()
}

// ID get connector id
func (m *Connector) ID() string {
	return m.id
}

// Query get connector query interface
func (m *Connector) Query() (query.Query, error) {
	return nil, nil
}

// Close connections
func (m *Connector) Close() error {
	return m.Client.Disconnect(context.Background())
}

// Is the connections from dsl
func (m *Connector) Is(typ int) bool {
	return 3 == typ
}

func (m *Connector) makeConnection() error {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(m.Options.dsn))
	if err != nil {
		return err
	}

	m.Client = client
	m.Database = client.Database(m.Options.DB)
	return nil
}

func (m *Connector) setDefaults() error {
	m.Options.DB = helper.EnvString(m.Options.DB)
	m.Options.Timeout = helper.EnvInt(m.Options.Timeout, 5)
	if m.Options.Timeout == 0 {
		m.Options.Timeout = 5
	}

	for i := range m.Options.Hosts {
		m.Options.Hosts[i].Host = helper.EnvString(m.Options.Hosts[i].Host)
		m.Options.Hosts[i].Pass = helper.EnvString(m.Options.Hosts[i].Pass)
		m.Options.Hosts[i].User = helper.EnvString(m.Options.Hosts[i].User)
		m.Options.Hosts[i].Port = helper.EnvString(m.Options.Hosts[i].Port)

		dsn, err := m.getDSN()
		if err != nil {
			return err
		}
		m.Options.dsn = dsn
	}
	return nil
}

// getDSN get the DSN
func (m *Connector) getDSN() (string, error) {

	if m.Options.DB == "" {
		return "", fmt.Errorf("%s options.db is required", m.id)
	}

	if len(m.Options.Hosts) == 0 {
		return "", fmt.Errorf("%s options.hosts is required", m.id)
	}

	hosts := []string{}
	for i := range m.Options.Hosts {
		host := m.Options.Hosts[i]
		if host.Host == "" {
			return "", fmt.Errorf("%s hosts.%d.host is required", m.id, i)
		}

		if host.Port == "" {
			host.Port = "27017"
		}

		if host.User == "" {
			return "", fmt.Errorf("%s hosts.%d.user is required", m.id, i)
		}

		if host.Pass == "" {
			return "", fmt.Errorf("%s hosts.%d.pass is required", m.id, i)
		}

		hosts = append(hosts, fmt.Sprintf("%s:%s@%s:%s", host.User, host.Pass, host.Host, host.Port))
	}

	params := []string{}
	if m.Options.Params != nil {
		for name, value := range m.Options.Params {
			params = append(params, fmt.Sprintf("%s=%v", name, value))
		}
	}

	dsn := fmt.Sprintf("mongodb://%s/", strings.Join(hosts, ","))
	if len(params) > 0 {
		dsn = dsn + "?" + strings.Join(params, "&")
	}

	return dsn, nil
}

// Setting get the connection setting
func (m *Connector) Setting() map[string]interface{} {

	return map[string]interface{}{
		"db":      m.Options.DB,
		"params":  m.Options.Params,
		"timeout": m.Options.Timeout,
		"hosts":   m.Options.Hosts,
		"dsn":     m.Options.dsn,
	}
}
