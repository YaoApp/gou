package gou

// API 数据接口
type API struct{}

// LoadAPI 载入数据接口
func LoadAPI(name string) *API {
	return &API{}
}

// Reload 重新载入API
func (api *API) Reload() {}

// Run 执行API并返回结果
func (api *API) Run() {}
