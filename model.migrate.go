package gou

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/day"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/schema"
)

// SchemaTableUpgrade 旧表数据结构差别对比后升级
func (mod *Model) SchemaTableUpgrade() {
}

// SchemaTableDiff 旧表数据结构差别对比
func (mod *Model) SchemaTableDiff() {
}

// SchemaTableCreate 创建新的数据表
func (mod *Model) SchemaTableCreate() {

	sch := capsule.Schema()
	err := sch.CreateTable(mod.MetaData.Table.Name, func(table schema.Blueprint) {

		// 创建字段
		for _, column := range mod.MetaData.Columns {
			col := column.SetType(table)
			column.SetOption(col)
		}

		// 创建索引
		for _, index := range mod.MetaData.Indexes {
			index.SetIndex(table)
		}

		// 创建时间, 更新时间
		if mod.MetaData.Option.Timestamps {
			table.Timestamps()
		}

		// 软删除
		if mod.MetaData.Option.SoftDeletes {
			table.SoftDeletes()
			table.JSON("__restore_data").Null()
		}

		// 追溯ID
		if mod.MetaData.Option.Trackings || mod.MetaData.Option.Logging {
			table.BigInteger("__tracking_id").Index().Null()
		}

	})

	if err != nil {
		exception.Err(err, 500).Throw()
	}

	// 添加默认值
	for _, row := range mod.MetaData.Values {
		mod.MustCreate(row)
	}
}

// ForceCreateSchema create table
func (mod *Model) ForceCreateSchema() error {

	sch := capsule.Schema()
	err := sch.DropTableIfExists(mod.MetaData.Table.Name)
	if err != nil {
		return err
	}

	err = sch.CreateTable(mod.MetaData.Table.Name, func(table schema.Blueprint) {

		// 创建字段
		for _, column := range mod.MetaData.Columns {
			col := column.SetType(table)
			column.SetOption(col)
		}

		// 创建索引
		for _, index := range mod.MetaData.Indexes {
			index.SetIndex(table)
		}

		// 创建时间, 更新时间
		if mod.MetaData.Option.Timestamps {
			table.Timestamps()
		}

		// 软删除
		if mod.MetaData.Option.SoftDeletes {
			table.SoftDeletes()
			table.JSON("__restore_data").Null()
		}

		// 追溯ID
		if mod.MetaData.Option.Trackings || mod.MetaData.Option.Logging {
			table.BigInteger("__tracking_id").Index().Null()
		}

	})

	if err != nil {
		return err
	}

	return nil
}

// Export the model
func (mod *Model) Export(chunkSize int, process func(curr, total int)) ([]string, error) {

	tmpdir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s", mod.Name, time.Now().Format("20060102150405")))
	os.MkdirAll(tmpdir, 0755)

	qb := capsule.Query().Table(mod.MetaData.Table.Name).OrderBy(mod.PrimaryKey)
	total, err := qb.Count()
	if err != nil {
		return nil, err
	}

	if total == 0 {
		return []string{}, nil
	}

	completed := 0
	files := []string{}
	err = qb.Chunk(chunkSize, func(items []interface{}, page int) error {

		if len(items) < 1 {
			return fmt.Errorf("items is null")
		}

		columns := any.Of(items[0]).MapStr().Keys()
		ctypes := map[int]string{}

		// Filter date
		for index, name := range columns {
			if column, has := mod.Columns[name]; has {
				switch strings.ToLower(column.Type) {
				case "date":
					ctypes[index] = "date"
					break
				case "time", "timetz":
					ctypes[index] = "time"
					break
				case "datetime", "datetimetz", "timestamp", "timestamptz":
					ctypes[index] = "datetime"
					break
				}
			}
		}

		values := [][]interface{}{}
		for _, item := range items {
			row := any.Of(item).MapStr().Values()
			for index, value := range row {
				if typ, has := ctypes[index]; has && value != nil {
					switch typ {
					case "date":
						row[index] = day.Of(value).Format("2006-01-02")
						break
					case "time":
						row[index] = day.Of(value).Format("15:04:05")
						break
					case "datetime":
						row[index] = day.Of(value).Format("2006-01-02T15:04:05")
						break
					}
				}
			}
			values = append(values, row)
		}

		name := filepath.Join(tmpdir, fmt.Sprintf("%s.%d.json", mod.Name, page))
		bytes, err := jsoniter.Marshal(ExportData{
			Columns: columns,
			Values:  values,
			Model:   mod.Name,
		})
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(name, bytes, 0644)
		if err != nil {
			return err
		}

		files = append(files, name)
		completed = completed + len(items)
		if process != nil {
			process(completed, int(total))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// Import the data
func (mod *Model) Import(file string) error {
	_, err := os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s not exists", file)
	}

	bytes, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	data := ExportData{}
	err = jsoniter.Unmarshal(bytes, &data)
	if err != nil {
		return err
	}

	qb := capsule.Query().Table(mod.MetaData.Table.Name)
	return qb.Insert(data.Values, data.Columns)
}
