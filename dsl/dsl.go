package dsl

import (
	"fmt"

	"github.com/yaoapp/gou"
)

// Change on file change
func (yao *YAO) Change(file string, event int) error {
	return yao.DSL.DSLChange(file, event)
}

// Refresh DSL
func (yao *YAO) Refresh() error {
	file := yao.Head.File
	*yao = *New(yao.Workshop) // RENEW
	err := yao.Open(file)
	if err != nil {
		return fmt.Errorf("%s %s", yao.Head.File, err.Error())
	}
	return yao.DSL.DSLRefresh(yao.Workshop.Root(), yao.Head.File, yao.Compiled)
}

// Check DSL
func (yao *YAO) Check() error {
	err := yao.DSL.DSLCheck(yao.Compiled)
	if err != nil {
		return fmt.Errorf("%s %s", yao.Head.File, err.Error())
	}
	return nil
}

// Compile compile the content
func (yao *YAO) Compile() error {
	err := yao.DSL.DSLCheck(yao.Compiled)
	if err != nil {
		return err
	}
	return yao.DSL.DSLCompile(yao.Workshop.Root(), yao.Head.File, yao.Compiled)
}

// NewDSL create DSL with type
func NewDSL(kind int) (DSL, error) {
	switch kind {
	case HTTP:
		return nil, nil
	case Model:
		return gou.MakeModel(), nil
	case Template:
		return gou.MakeTemplate(), nil
	case Flow:
		return nil, nil
	case MySQL, PgSQL, Oracle, TiDB, ClickHouse, Redis, MongoDB, Elastic, SQLite:
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
