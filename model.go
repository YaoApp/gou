package gou

// Model 数据模型
type Model struct{}

// LoadModel 载入数据模型
func LoadModel(name string) *Model {
	return &Model{}
}

// Find 查询单条记录
func (mod *Model) Find() {}

// Save 保存单条数据
func (mod *Model) Save() {}

// Delete 删除单条记录
func (mod *Model) Delete() {}

// Search 按条件检索
func (mod *Model) Search() {}

// Import 批量导入模型
func (mod *Model) Import() {}

// Export 导出数据模型
func (mod *Model) Export() {}

// Setting 数据模型配置
func (mod *Model) Setting() {}

// List 列表界面配置
func (mod *Model) List() {}

// View 详情界面配置
func (mod *Model) View() {}

// Migrate 数据迁移
func (mod *Model) Migrate() {}

// Reload 重新载入
func (mod *Model) Reload() {}

// Query xun.Query
func (mod *Model) Query() {}

// Schema xun.Schema
func (mod *Model) Schema() {}
