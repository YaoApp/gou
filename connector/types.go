package connector

import (
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
)

const (
	// DATABASE the database connector (mysql, pgsql, oracle, sqlite ... )
	DATABASE = iota + 1

	// REDIS the redis connector
	REDIS

	// MONGO the mongodb connector
	MONGO

	// ELASTICSEARCH the elasticsearch connector
	ELASTICSEARCH

	// KAFKA the kafka connector
	KAFKA

	// OPENAI the openai connector
	OPENAI

	// WEAVIATE the weaviate connector
	WEAVIATE

	// SCRIPT ? the script connector ( difference with widget ?)
	SCRIPT
)

var types = map[string]int{
	"mysql":         DATABASE,
	"sqlite":        DATABASE,
	"sqlite3":       DATABASE,
	"postgres":      DATABASE,
	"oracle":        DATABASE,
	"redis":         REDIS,
	"mongo":         MONGO,
	"elasticsearch": ELASTICSEARCH,
	"es":            ELASTICSEARCH,
	"kafka":         KAFKA,
	"openai":        OPENAI,
	"weaviate":      WEAVIATE,
	"script":        SCRIPT, // ?
}

// Connector the connector interface
type Connector interface {
	Register(file string, id string, dsl []byte) error
	Query() (query.Query, error)
	Schema() (schema.Schema, error)
	Close() error
	ID() string
	Is(int) bool
	Setting() map[string]interface{}
}

// DSL the connector DSL
type DSL struct {
	ID      string                 `json:"-"`
	Type    string                 `json:"type"`
	Name    string                 `json:"name,omitempty"`
	Label   string                 `json:"label,omitempty"`
	Version string                 `json:"version,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}
