package schema

import (
	"fmt"

	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/gou/schema/xun"
)

/**
 * Schema helpers and processes
 */

// Use pick a schema driver
func Use(name string) types.Schema {
	switch name {
	case "xun":
		return &xun.Xun{}
	default:
		panic(fmt.Errorf("%s does not support yet", name))
	}
}
