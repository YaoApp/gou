package dsl

import (
	"github.com/blang/semver/v4"
	"github.com/yaoapp/gou/dsl/workshop"
)

// YAO the YAO DSL
type YAO struct {
	Head      *Head
	Content   map[string]interface{}
	Compiled  map[string]interface{}
	DSL       DSL
	Workshop  *workshop.Workshop // DSL workshop
	Trace     []string           // FROM Trace
	templates map[string]*YAO    // The templates cache
	Mode      string             // ? development | production
}

// DSL the YAO domain specific language interface
type DSL interface {
	DSLCompile(root string, file string, source map[string]interface{}) error
	DSLCheck(source map[string]interface{}) error
	DSLRefresh(root string, file string, source map[string]interface{}) error
	DSLRemove(root string, file string) error
}

// Head the YAO domain specific language Head
type Head struct {
	File    string          // the DSL file path
	Name    string          // the name of the DSL
	Bindata bool            // is saved in bindata
	Type    int             // which type of the DSL
	Lang    *semver.Version // the DSL LANG version
	Version *semver.Version // the DSL version
	From    string          // inherited from
	Run     *Command        // MERGE COMMAND
}

// Command the DSL command
type Command struct {
	DELETE  []string                   // remove the given fields
	MERGE   []map[string]interface{}   // merge fields with the given values (not deep merge)
	REPLACE []map[string]interface{}   // replace the fields with the new definition
	APPEND  []map[string][]interface{} // append to the array fields with the given values
}

const (
	// Model the Model
	Model = iota + 1
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
	// SQLite the SQLite3 connector
	SQLite
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
	SQLite:     "db",
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
	"db":         SQLite,
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
	"/connectors": {MySQL, PgSQL, Oracle, TiDB, SQLite, ClickHouse, Redis, MongoDB, Elastic},
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
	SQLite:     "/connectors",
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
