package lang

import (
	"fmt"
)

// Change on file change
func (yao *YAO) Change(file string, event int) error {
	return yao.DSL.DSLChange(file, event)
}

// Register the DSL
func (yao *YAO) Register() error {
	return yao.DSL.DSLRegister()
}

// Refresh DSL
func (yao *YAO) Refresh() error {
	return yao.DSL.DSLRefresh()
}

// Dependencies check the Dependencies list
func (yao *YAO) Dependencies() ([]string, error) {
	return yao.DSL.DSLDependencies()
}

// Compile compile the content
func (yao *YAO) Compile() error {
	return yao.DSL.DSLCompile()
}

// NewDSL create DSL with type
func NewDSL(kind int) (DSL, error) {
	switch kind {
	case HTTP:
		return nil, nil
	case Model:
		return nil, nil
	case Flow:
		return nil, nil
	case MySQL, PgSQL, Oracle, TiDB, ClickHouse, Redis, MongoDB, Elastic:
		return nil, nil
	case Socket, WebSocket, Store, Queue:
		return nil, nil
	case Schedule:
		return nil, nil
	case Module:
		return nil, nil
	case Component:
		return nil, nil
	}
	return nil, fmt.Errorf("the given Type is not defined or not supported yet (%d)", kind)
}
