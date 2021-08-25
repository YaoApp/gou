package gou

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

// Models 已载入模型
var Models = map[string]*Model{}

// LoadModel 载入数据模型
func LoadModel(source string, name string) *Model {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	metadata := MetaData{}
	err := helper.UnmarshalFile(input, &metadata)
	if err != nil {
		panic(err)
	}
	mod := &Model{
		Name:     name,
		Source:   source,
		MetaData: metadata,
	}

	Models[name] = mod
	return mod
}

// Select 读取已加载模型
func Select(name string) *Model {
	mod, has := Models[name]
	if !has {
		exception.New(
			fmt.Sprintf("Model:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return mod
}

// Reload 更新模型
func (mod *Model) Reload() *Model {
	return LoadModel(mod.Source, mod.Name)
}

// Find 查询单条记录
func (mod *Model) Find(id interface{}) maps.MapStr {
	return maps.MapStrOf(map[string]interface{}{
		"id": 1,
	})
}

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
