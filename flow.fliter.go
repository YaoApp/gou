package gou

import "github.com/yaoapp/gou/query/share"

// Helper function
type Helper = share.Helper

// RegisterHelper 注册 helper
func RegisterHelper(name string, helper Helper) {
	share.Filters[name] = helper
}
