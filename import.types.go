package gou

// Import 数据导入配置
type Import struct {
	Mapping map[string]string             `json:"mapping,omitempty"` // 源数据与数据模型字段映射表
	Process string                        `json:"process,omitempty"` // 源数据与数据模型转换处理器 (process & mapping 任选其一)
	Chunk   int                           `json:"chunk,omitempty"`   // 每次处理数据数量
	Notify  func(line int, record string) `json:"-"`                 // 进度通报函数
	Option  map[string]interface{}        `json:"option,omitempty"`  // CSV 分割符等参数

}

// ResponseImport 导出结果
type ResponseImport struct {
	Import  *Import       `json:"import"`  // 导入数据
	Errors  *ErrorsImport `json:"errors"`  // 导入错误集合
	Success int           `json:"success"` // 成功导入记录数量
	Failure int           `json:"failure"` // 失败导入记录数量
	Total   int           `json:"total"`   // 总记录记录数量
}

// ErrorsImport 导入错误集合
type ErrorsImport []*ErrorOfImport

// ErrorOfImport 导入错误结构体
type ErrorOfImport struct {
	Code    int         `json:"code"`            // 错误码
	Message string      `json:"message"`         // 错误描述
	File    string      `json:"file"`            // 出错文件
	Line    int         `json:"line,omitempty"`  // 出错行
	Field   string      `json:"field,omitempty"` // 源数据出错字段名称
	Value   interface{} `json:"value,omitempty"` // 源数据出错数值
}
