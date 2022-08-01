package schema

import (
	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/gou/schema/xun"
	"github.com/yaoapp/xun/capsule"
)

/**
 * Schema helpers and processes
 */

// Use pick a schema driver
func Use(name string) types.Schema {
	switch name {
	case "tests":
		return &xun.Xun{}
	default:
		return &xun.Xun{
			Option: xun.Option{Manager: capsule.Global},
		}
	}
}
