package model

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/schema"
	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/day"
	"github.com/yaoapp/xun/capsule"
)

// CreateTable create the table of the model
func (mod *Model) CreateTable() error {
	connector := mod.MetaData.Connector
	if connector == "" {
		connector = "default"
	}

	table := mod.MetaData.Table.Name
	if table == "" {
		return fmt.Errorf("missing table name")
	}

	blueprint, err := mod.Blueprint()
	if err != nil {
		return err
	}

	sch := schema.Use(connector)
	return sch.TableCreate(table, blueprint)

}

// SaveTable update or create the table of the model
func (mod *Model) SaveTable() error {
	connector := mod.MetaData.Connector
	if connector == "" {
		connector = "default"
	}

	table := mod.MetaData.Table.Name
	if table == "" {
		return fmt.Errorf("missing table name")
	}

	blueprint, err := mod.Blueprint()
	if err != nil {
		return err
	}

	sch := schema.Use(connector)
	err = sch.TableSave(table, blueprint)
	if err != nil {
		return err
	}
	return nil
}

// DropTable drop the table of the model
func (mod *Model) DropTable() error {
	connector := mod.MetaData.Connector
	if connector == "" {
		connector = "default"
	}

	table := mod.MetaData.Table.Name
	if table == "" {
		return fmt.Errorf("missing table name")
	}

	sch := schema.Use(connector)
	return sch.TableDrop(table)
}

// HasTable check if the table of the model is exists
func (mod *Model) HasTable() (bool, error) {
	connector := mod.MetaData.Connector
	if connector == "" {
		connector = "default"
	}

	table := mod.MetaData.Table.Name
	if table == "" {
		return false, fmt.Errorf("missing table name")
	}

	sch := schema.Use(connector)
	_, err := sch.TableGet(table)
	if err != nil && strings.Contains(err.Error(), "does not exists") {
		return false, nil
	}

	return true, nil
}

// InsertValues insert the default values of the model
func (mod *Model) InsertValues() ([]int, []error) {

	ids := []int{}
	errs := []error{}

	// Add the default values
	for _, row := range mod.MetaData.Values {
		id, err := mod.Create(row)
		if err != nil {
			errs = append(errs, err)
		}
		ids = append(ids, id)
	}

	return ids, errs
}

// Blueprint cast to the blueprint struct
func (mod *Model) Blueprint() (types.Blueprint, error) {
	return types.NewAny(mod.MetaData)
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
