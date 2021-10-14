package query

import (
	"io"

	"github.com/yaoapp/gou/query/gou"
)

// Gou 创建 Gou Query share.DSL
func Gou(input []byte) *gou.Query {
	return gou.Make(input)
}

// GouRead 创建 Gou Query share.DSL (输入接口)
func GouRead(reader io.Reader) *gou.Query {
	return gou.Read(reader)
}

// GouOpen 创建 Gou Query share.DSL (文件)
func GouOpen(filename string) *gou.Query {
	return gou.Open(filename)
}
