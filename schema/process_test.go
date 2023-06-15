package schema

import (
	"os"
	"path/filepath"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
)

func TestSchemaProcesses(t *testing.T) {

	dbconnect(t)
	defer clean()

	// Create
	_, err := process.New("schemas.default.Create", "unit_tests_schema_processes").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Drop
	_, err = process.New("schemas.default.Drop", "unit_tests_schema_processes").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// TableCreate
	user := prepareData(t)
	_, err = process.New("schemas.default.TableCreate", "unit_tests_schema_table_user", user).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// TableDrop
	_, err = process.New("schemas.default.TableDrop", "unit_tests_schema_table_user").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// TableSave
	_, err = process.New("schemas.default.TableSave", "unit_tests_schema_table_user", user).Exec()
	if err != nil {
		t.Fatal(err)
	}
	defer process.New("schemas.default.TableDrop", "unit_tests_schema_table_user").Exec()

	// TableDiff
	_, err = process.New("schemas.default.TableDiff", user, user).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// TableRename
	_, err = process.New("schemas.default.TableRename", "unit_tests_schema_table_user", "unit_tests_schema_table_user_re").Exec()
	if err != nil {
		t.Fatal(err)
	}
	defer process.New("schemas.default.TableDrop", "unit_tests_schema_table_user_re").Exec()

	// TableGet
	_, err = process.New("schemas.default.TableGet", "unit_tests_schema_table_user_re").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// TableExists
	exists, err := process.New("schemas.default.TableExists", "unit_tests_schema_table_user_re").Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, exists.(bool))

	exists, err = process.New("schemas.default.TableExists", "unit_tests_schema_table_user_not_exists").Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, exists.(bool))

	// Tables
	_, err = process.New("schemas.default.Tables").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// ColumnAdd
	_, err = process.New("schemas.default.ColumnAdd", "unit_tests_schema_table_user_re", map[string]interface{}{
		"name": "new_column",
		"type": "string",
	}).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// ColumnAlt
	_, err = process.New("schemas.default.ColumnAlt", "unit_tests_schema_table_user_re", map[string]interface{}{
		"name":   "new_column",
		"type":   "string",
		"length": 20,
		"index":  true,
	}).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// IndexAdd
	_, err = process.New("schemas.default.IndexAdd", "unit_tests_schema_table_user_re", map[string]interface{}{
		"name":    "new_index",
		"type":    "index",
		"columns": []string{"mobile", "new_column"},
	}).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// IndexDel
	_, err = process.New("schemas.default.IndexDel", "unit_tests_schema_table_user_re", "new_index").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// ColumnDel
	_, err = process.New("schemas.default.ColumnDel", "unit_tests_schema_table_user_re", "new_column").Exec()
	if err != nil {
		t.Fatal(err)
	}
}

func prepareData(t *testing.T) map[string]interface{} {
	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := application.OpenFromDisk(root) // Load app
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	bytes, err := application.App.Read(filepath.Join("models", "tests", "user.mod.yao"))
	if err != nil {
		t.Fatal(err)
	}

	data := map[string]interface{}{}
	err = jsoniter.Unmarshal(bytes, &data)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
