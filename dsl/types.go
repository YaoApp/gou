package dsl

import "github.com/blang/semver/v4"

// YAO the YAO DSL
type YAO struct {
	Head    *Head
	Content *Content
	DSL     DSL
	Mode    string // development | production
}

// DSL the YAO domain specific language interface
type DSL interface {
	DSLCompile() error
	DSLCheck() error
	DSLRefresh() error
	DSLRegister() error
	DSLChange(file string, event int) error
	DSLDependencies() ([]string, error)
}

// Content the content of YAO DSL
type Content map[string]interface{}

// Head the YAO domain specific language Head
type Head struct {
	File    string
	Name    string
	Bindata bool
	Type    int
	Lang    semver.Version
	Version semver.Version
	From    Package
	Alias   LocalPackage
	Delete  []string
}

// Workshop the required packages
type Workshop struct {
	Require []Package         `json:"require,omitempty"`
	Replace map[string]string `json:"replace,omitempty"` // for multi projects development
	Mapping map[string]Package
	file    string // the workshop.yao file path
	cfg     WorkshopConfig
}

// WorkshopConfig the workshop config file
type WorkshopConfig map[string]map[string]interface{}

// Package the YAO package info
type Package struct {
	URL        string         // github.com/yaoapp/demo-wms/cloud@v0.0.0-20220223010332-e86eab4c8490
	Name       string         // demo-wms.yaoapp.cloud
	Alias      string         // demo-wms.yaoapp.cloud
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
}

// LocalPackage the YAO local package info
type LocalPackage struct{}

const (
	// Model the Model
	Model = iota
	// Flow the Data Flow
	Flow
	// HTTP RESTFul API
	HTTP
	// MQTT MQTT API
	MQTT
	// MySQL the MySQL connector
	MySQL
	// PgSQL the PostgreSQL connector
	PgSQL
	// TiDB the TiDB connector
	TiDB
	// Oracle the Oracle connector
	Oracle
	// ClickHouse the ClickHouse connector
	ClickHouse
	// Elastic the Elastic connector
	Elastic
	// Redis the Redis connector
	Redis
	// MongoDB the MongoDB connector
	MongoDB
	// Kafka the Kafka connector
	Kafka
	// WebSocket the WebSocket service
	WebSocket
	// Socket the Socket service
	Socket
	// Store the Store service
	Store
	// Queue the Queue service
	Queue
	// Schedule the schedule programs
	Schedule
	// Brain the behaviors of Brain (?a group of processes)
	Brain
	// Module the Cloud Module
	Module
	// Component the Cloud Component
	Component
	// Template the DSL template
	Template
)

const (
	// CREATE the DSL file create event
	CREATE = iota
	// CHANGE the DSL file change event
	CHANGE
	// REMOVE the DSL file remove event
	REMOVE
)

const (
	// RootEnvName the environment variable name of the workshop root path in the local disk
	RootEnvName = "YAO_PATH"
)

// TypeExtensions the DSL file extensions
var TypeExtensions = map[int]string{
	HTTP:       "http",
	MQTT:       "mqtt",
	Model:      "mod",
	Flow:       "flow",
	MySQL:      "mysql",
	PgSQL:      "pgsql",
	Oracle:     "oracle",
	TiDB:       "tidb",
	ClickHouse: "click",
	Redis:      "redis",
	MongoDB:    "mongo",
	Socket:     "sock",
	WebSocket:  "webs",
	Store:      "store",
	Queue:      "que",
	Schedule:   "sch",
	Component:  "com",
	Template:   "tpl",
	Elastic:    "es",
}

// ExtensionTypes the extension types
var ExtensionTypes = map[string]int{
	"model":      Model,
	"mod":        Model,
	"flow":       Flow,
	"flw":        Flow,
	"http":       HTTP,
	"mqtt":       MQTT,
	"mysql":      MySQL,
	"my":         MySQL,
	"pgsql":      PgSQL,
	"pg":         PgSQL,
	"tidb":       TiDB,
	"oracle":     Oracle,
	"click":      ClickHouse,
	"clickhouse": ClickHouse,
	"redis":      Redis,
	"mongo":      MongoDB,
	"es":         Elastic,
	"kafka":      Kafka,
	"ws":         WebSocket,
	"webs":       WebSocket,
	"sock":       Socket,
	"socket":     Socket,
	"store":      Store,
	"queue":      Queue,
	"que":        Queue,
	"module":     Module,
	"m":          Module,
	"com":        Component,
	"c":          Component,
	"sch":        Schedule,
	"schedule":   Schedule,
	"tpl":        Template,
	"tmpl":       Template,
}

// DirTypes the directories for different types of DSL
var DirTypes = map[string][]int{
	"/apis":       {HTTP, MQTT},
	"/models":     {Model},
	"/flows":      {Flow},
	"/connectors": {MySQL, PgSQL, Oracle, TiDB, ClickHouse, Redis, MongoDB, Elastic},
	"/services":   {Socket, WebSocket, Store, Queue},
	"/schedules":  {Schedule},
	"/components": {Component},
	"/templates":  {Template},
}

// TypeDirs the root of different types of DSL
var TypeDirs = map[int]string{
	HTTP:       "/apis",
	MQTT:       "/apis",
	Model:      "/models",
	Flow:       "/flows",
	MySQL:      "/connectors",
	PgSQL:      "/connectors",
	Oracle:     "/connectors",
	TiDB:       "/connectors",
	ClickHouse: "/connectors",
	Elastic:    "/connectors",
	Redis:      "/connectors",
	MongoDB:    "/connectors",
	Socket:     "/services",
	WebSocket:  "/services",
	Store:      "/services",
	Queue:      "/services",
	Schedule:   "/schedules",
	Component:  "/components",
	Template:   "/templates",
}
