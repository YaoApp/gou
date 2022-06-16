package gou

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// MakeModel make a model instance
func MakeModel() *Model {
	return &Model{}
}

// DSLCompile compile the DSL
func (mod *Model) DSLCompile(root string, file string, source map[string]interface{}) error {

	// SHOULD BE CACHED

	err := mod.compile(source)
	if err != nil {
		return err
	}

	fullname, _, _ := mod.nameRouter(root, file)
	mod.Name = fullname

	// Generate table name. model: admin.dashboard.user prefix: yao_ -> yao_admin_dashboard_user
	if mod.MetaData.Table.Name == "" {
		mod.MetaData.Table.Name = fmt.Sprintf(
			"%s%s",
			mod.MetaData.Table.Prefix,
			strings.ReplaceAll(fullname, ".", "_"),
		)

		// Attach prefix to the table
	} else if mod.MetaData.Table.Prefix != "" {
		mod.MetaData.Table.Name = fmt.Sprintf(
			"%s%s",
			mod.MetaData.Table.Prefix,
			mod.MetaData.Table.Name,
		)
	}

	Models[fullname] = mod
	return nil
}

// DSLCheck check the DSL
func (mod *Model) DSLCheck(source map[string]interface{}) error {

	// Check Columns
	err := mod.checkColumns(source["columns"])
	if err != nil {
		return err
	}

	// Check Indexes

	// Check Relations

	// Check Options

	return nil
}

// DSLRefresh refresh the DSL
func (mod *Model) DSLRefresh(root string, file string, source map[string]interface{}) error {
	fullname, _, _ := mod.nameRouter(root, file)
	delete(Models, fullname)
	return mod.DSLCompile(root, file, source)
}

// DSLRemove the DSL
func (mod *Model) DSLRemove(root string, file string) error {
	fullname, _, _ := mod.nameRouter(root, file)
	delete(Models, fullname)
	return nil
}

func (mod *Model) checkColumns(input interface{}) error {
	columns, ok := input.([]interface{})
	if !ok {
		return fmt.Errorf("columns should be a array, bug got:%#v", input)
	}

	loaded := map[string]int{}
	for i, column := range columns {
		col, ok := column.(map[string]interface{})
		if !ok {
			return fmt.Errorf("columns[%d] should be a map, bug got:%#v", i, column)
		}

		name, has := col["name"].(string)
		if !has {
			return fmt.Errorf("columns[%d].name is required and should be a string, bug got:%#v", i, column)
		}

		if _, has := loaded[name]; has {
			return fmt.Errorf("columns[%d].name %s is existed, check columns[%d]", i, name, loaded[name])
		}

		// t, has := col["type"].(string)
		// if !has {
		// 	return fmt.Errorf("columns[%d].name is required and should be a string, bug got:%#v", i, column)
		// }

		loaded[name] = i
	}

	return nil
}

// nameRouter get the model name from router
func (mod *Model) nameRouter(root string, file string) (fullname string, namespace string, name string) {
	dir, filename := filepath.Split(strings.TrimPrefix(file, filepath.Join(root, "models")))
	name = strings.TrimRight(filename, ".mod.yao")
	namespace = strings.ReplaceAll(strings.Trim(filepath.ToSlash(dir), "/"), "/", ".")
	fullname = name
	if namespace != "" {
		fullname = fmt.Sprintf("%s.%s", namespace, name)
	}
	return fullname, namespace, name
}

// compile the model
func (mod *Model) compile(source map[string]interface{}) error {
	data, err := jsoniter.Marshal(source)
	if err != nil {
		return err
	}

	metadata := MetaData{}
	err = jsoniter.Unmarshal(data, &metadata)
	if err != nil {
		return err
	}

	new := &Model{MetaData: metadata}
	columns := map[string]*Column{} // the fields
	columnNames := []interface{}{}  // the columnNames
	PrimaryKey := "id"              // the primary key
	uniqueColumns := []*Column{}    // the unique columns

	// SoftDeletes
	if new.MetaData.Option.SoftDeletes {
		new.MetaData.Columns = append(new.MetaData.Columns, Column{
			Label:    "$.Delete Marker",
			Name:     "deleted_at",
			Type:     "timestamp",
			Comment:  "$.Delete Marker",
			Nullable: true,
		})
	}

	// Timestamps
	if new.MetaData.Option.Timestamps {
		new.MetaData.Columns = append(new.MetaData.Columns,
			Column{
				Label:    "$.Created At",
				Name:     "created_at",
				Type:     "timestamp",
				Comment:  "$.Created At",
				Nullable: true,
			},
			Column{
				Label:    "$.Updated At",
				Name:     "updated_at",
				Type:     "timestamp",
				Comment:  "$.Updated At",
				Nullable: true,
			},
		)
	}

	for i, column := range new.MetaData.Columns {
		new.MetaData.Columns[i].model = mod // link attr
		columns[column.Name] = &new.MetaData.Columns[i]
		columnNames = append(columnNames, column.Name)
		if strings.ToLower(column.Type) == "id" {
			PrimaryKey = column.Name
		}

		// unique field
		if column.Unique {
			uniqueColumns = append(uniqueColumns, columns[column.Name])
		}
	}

	// unique indexes
	for _, index := range new.MetaData.Indexes {
		if strings.ToLower(index.Type) == "unique" {
			for _, name := range index.Columns {
				col, has := columns[name]
				if has {
					uniqueColumns = append(uniqueColumns, col)
				}
			}
		}
	}

	new.Columns = columns
	new.ColumnNames = columnNames
	new.PrimaryKey = PrimaryKey
	new.UniqueColumns = uniqueColumns

	// COPY
	*mod = *new
	return nil
}
