package gou

import (
	"io/fs"

	"github.com/yaoapp/gou/helper"
)

// LoadAPI 载入数据接口
func LoadAPI(file fs.File) *API {
	defer file.Close()
	http := HTTP{}
	err := helper.UnmarshalFile(file, &http)
	if err != nil {
		panic(err)
	}

	return &API{
		File: file,
		Type: "http",
		HTTP: http,
	}
}

// Reload 重新载入API
func (api *API) Reload() {}

// Run 执行API并返回结果
func (api *API) Run() {}
