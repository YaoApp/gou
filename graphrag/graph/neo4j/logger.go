package neo4j

import (
	neo4jlog "github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
	"github.com/yaoapp/kun/log"
)

// kunLogger implements neo4j log.Logger interface and bridges to kun/log
type kunLogger struct{}

// newKunLogger creates a new kunLogger instance
func newKunLogger() neo4jlog.Logger {
	return &kunLogger{}
}

// Error implements neo4j log.Logger
func (l *kunLogger) Error(name, id string, err error) {
	log.With(log.F{
		"component": name,
		"id":        id,
	}).Error("%s", err.Error())
}

// Warnf implements neo4j log.Logger
func (l *kunLogger) Warnf(name, id string, msg string, args ...any) {
	log.With(log.F{
		"component": name,
		"id":        id,
	}).Warn(msg, args...)
}

// Infof implements neo4j log.Logger
func (l *kunLogger) Infof(name, id string, msg string, args ...any) {
	log.With(log.F{
		"component": name,
		"id":        id,
	}).Info(msg, args...)
}

// Debugf implements neo4j log.Logger
func (l *kunLogger) Debugf(name, id string, msg string, args ...any) {
	log.With(log.F{
		"component": name,
		"id":        id,
	}).Debug(msg, args...)
}

