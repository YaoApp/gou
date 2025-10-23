package model

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/capsule"
)

// Models 已载入模型
var Models = map[string]*Model{}
var rwlock sync.RWMutex // Use RWMutex for better concurrency

// LoadSync load model sync
func LoadSync(file string, id string) (*Model, error) {
	rwlock.Lock()
	defer rwlock.Unlock()
	return Load(file, id)
}

// LoadSourceSync load model sync
func LoadSourceSync(source []byte, id string, file string) (*Model, error) {
	rwlock.Lock()
	defer rwlock.Unlock()
	return LoadSource(source, id, "")
}

// Load load model
func Load(file string, id string) (*Model, error) {
	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}
	return LoadSource(data, id, file)
}

// LoadSource load model from source
func LoadSource(source []byte, id string, file string) (*Model, error) {
	if file == "" {
		file = fmt.Sprintf("__source.%s.mod.yao", id)
	}

	metadata := MetaData{}
	err := application.Parse(file, source, &metadata)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	mod := &Model{
		ID:       id,
		Name:     id,
		File:     file,
		MetaData: metadata,
		source:   source,
	}

	// Parse common values
	columns := map[string]*Column{}
	columnNames := []interface{}{}
	PrimaryKey := "id"
	uniqueColumns := []*Column{}

	// Add timestamp columns if enabled
	if mod.MetaData.Option.Timestamps {
		mod.MetaData.Columns = append(mod.MetaData.Columns,
			Column{
				Label:    "::Created At",
				Name:     "created_at",
				Type:     "timestamp",
				Comment:  "::Created At",
				Nullable: true,
			},
			Column{
				Label:    "::Updated At",
				Name:     "updated_at",
				Type:     "timestamp",
				Comment:  "::Updated At",
				Nullable: true,
			},
		)
	}

	// Add soft delete column if enabled
	if mod.MetaData.Option.SoftDeletes {
		mod.MetaData.Columns = append(mod.MetaData.Columns, Column{
			Label:    "::Delete At",
			Name:     "deleted_at",
			Type:     "timestamp",
			Comment:  "::Delete At",
			Nullable: true,
		})
	}

	// Add permission columns if enabled
	if mod.MetaData.Option.Permission {
		mod.MetaData.Columns = append(mod.MetaData.Columns,
			Column{
				Label:    "::Created By",
				Name:     "__yao_created_by",
				Type:     "string",
				Comment:  "::Created By User ID",
				Nullable: true,
				Length:   128,
				Index:    true,
			},
			Column{
				Label:    "::Updated By",
				Name:     "__yao_updated_by",
				Type:     "string",
				Comment:  "::Updated By User ID",
				Nullable: true,
				Length:   128,
			},
			Column{
				Label:    "::Team ID",
				Name:     "__yao_team_id",
				Type:     "string",
				Comment:  "::Team ID",
				Nullable: true,
				Length:   128,
				Index:    true,
			},
			Column{
				Label:    "::Tenant ID",
				Name:     "__yao_tenant_id",
				Type:     "string",
				Comment:  "::Tenant ID",
				Nullable: true,
				Length:   128,
				Index:    true,
			},
		)
	}

	for i, column := range mod.MetaData.Columns {
		mod.MetaData.Columns[i].model = mod
		columns[column.Name] = &mod.MetaData.Columns[i]
		columnNames = append(columnNames, column.Name)
		if strings.ToLower(column.Type) == "id" || column.Primary == true {
			PrimaryKey = column.Name
		}

		if column.Unique || column.Primary {
			uniqueColumns = append(uniqueColumns, columns[column.Name])
		}
	}

	// Process unique indexes (avoid duplicates)
	uniqueColumnMap := map[string]*Column{}
	for _, col := range uniqueColumns {
		uniqueColumnMap[col.Name] = col
	}

	for _, index := range mod.MetaData.Indexes {
		if strings.ToLower(index.Type) == "unique" {
			for _, name := range index.Columns {
				col, has := columns[name]
				if has {
					uniqueColumnMap[col.Name] = col
				}
			}
		} else if strings.ToLower(index.Type) == "primary" {
			for _, name := range index.Columns {
				col, has := columns[name]
				if has {
					PrimaryKey = col.Name
					uniqueColumnMap[col.Name] = col
				}
			}
		}
	}

	// Convert map back to slice
	uniqueColumns = []*Column{}
	for _, col := range uniqueColumnMap {
		uniqueColumns = append(uniqueColumns, col)
	}

	mod.Columns = columns
	mod.ColumnNames = columnNames
	mod.PrimaryKey = PrimaryKey
	mod.UniqueColumns = uniqueColumns

	if capsule.Global != nil {
		mod.Driver = capsule.Schema().MustGetConnection().Config.Driver
	}

	Models[id] = mod
	return mod, nil
}

// Reload updates the model
func (mod *Model) Reload() (*Model, error) {
	// Load new model first
	new, err := Load(mod.File, mod.ID)
	if err != nil {
		return nil, err
	}

	// Update model under lock
	rwlock.Lock()
	defer rwlock.Unlock()
	*mod = *new
	Models[mod.ID] = mod
	return mod, nil
}

// Migrate 数据迁移
func (mod *Model) Migrate(force bool, opts ...MigrateOption) error {
	options := &MigrateOptions{}
	for _, opt := range opts {
		opt(options)
	}
	if force {
		err := mod.DropTable()
		if err != nil {
			return err
		}
	}

	has, err := mod.HasTable()
	if err != nil {
		return err
	}

	if !has {
		err := mod.CreateTable()
		if err != nil {
			return err
		}

		if !options.DonotInsertValues {
			_, errs := mod.InsertValues()
			if len(errs) > 0 {
				for _, err := range errs {
					log.Error("[Migrate] %s", err.Error())
				}
				return fmt.Errorf("%d values error, please check the logs", len(errs))
			}
		}
		return nil
	}

	return mod.SaveTable()
}

// MigrateOptions Migrate options
type MigrateOptions struct {
	DonotInsertValues bool `json:"donot_insert_values"`
}

// MigrateOption Migrate option
type MigrateOption func(*MigrateOptions)

// WithDonotInsertValues with donot insert values
func WithDonotInsertValues(v bool) MigrateOption {
	return func(mo *MigrateOptions) {
		mo.DonotInsertValues = v
	}
}

// Select selects a model
func Select(id string) *Model {
	rwlock.RLock()
	defer rwlock.RUnlock()
	mod, has := Models[id]
	if !has {
		exception.New(
			fmt.Sprintf("Model:%s; not found", id),
			400,
		).Throw()
	}
	return mod
}

// Exists checks if model exists
func Exists(id string) bool {
	rwlock.RLock()
	defer rwlock.RUnlock()
	_, has := Models[id]
	return has
}

// GetMetaData gets model metadata
func GetMetaData(id string) MetaData {
	return Select(id).MetaData
}

// Read reads model source
func Read(id string) []byte {
	return Select(id).source
}

// Validate 数值校验
func (mod *Model) Validate(row maps.MapStrAny) []ValidateResponse {
	res := []ValidateResponse{}
	for name, value := range row {
		column, has := mod.Columns[name]
		if !has {
			continue
		}

		// 如果允许为 null
		if value == nil && column.Nullable {
			continue
		}

		success, messages := column.Validate(value, row)
		if !success {
			res = append(res, ValidateResponse{
				Column:   column.Name,
				Messages: messages,
			})
		}
	}
	return res
}

// GetMetaInfo returns the meta information of the model
func (mod *Model) GetMetaInfo() types.MetaInfo {
	return mod.MetaData.MetaInfo
}
