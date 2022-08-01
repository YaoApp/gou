package gou

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	jsoniter "github.com/json-iterator/go"
)

func TestSchemaProcesses(t *testing.T) {

	// Create
	_, err := NewProcess("schemas.default.Create", "unit_tests_schema_processes").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Drop
	_, err = NewProcess("schemas.default.Drop", "unit_tests_schema_processes").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// TableCreate
	user := schemaTestData(t)
	_, err = NewProcess("schemas.default.TableCreate", "unit_tests_schema_table_user", user).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// TableDrop
	_, err = NewProcess("schemas.default.TableDrop", "unit_tests_schema_table_user").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// TableSave
	_, err = NewProcess("schemas.default.TableSave", "unit_tests_schema_table_user", user).Exec()
	if err != nil {
		t.Fatal(err)
	}
	defer NewProcess("schemas.default.TableDrop", "unit_tests_schema_table_user").Exec()

	// TableDiff
	_, err = NewProcess("schemas.default.TableDiff", user, user).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// TableRename
	_, err = NewProcess("schemas.default.TableRename", "unit_tests_schema_table_user", "unit_tests_schema_table_user_re").Exec()
	if err != nil {
		t.Fatal(err)
	}
	defer NewProcess("schemas.default.TableDrop", "unit_tests_schema_table_user_re").Exec()

	// TableGet
	_, err = NewProcess("schemas.default.TableGet", "unit_tests_schema_table_user_re").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Tables
	_, err = NewProcess("schemas.default.Tables").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// ColumnAdd
	_, err = NewProcess("schemas.default.ColumnAdd", "unit_tests_schema_table_user_re", map[string]interface{}{
		"name": "new_column",
		"type": "string",
	}).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// ColumnAlt
	_, err = NewProcess("schemas.default.ColumnAlt", "unit_tests_schema_table_user_re", map[string]interface{}{
		"name":   "new_column",
		"type":   "string",
		"length": 20,
		"index":  true,
	}).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// IndexAdd
	_, err = NewProcess("schemas.default.IndexAdd", "unit_tests_schema_table_user_re", map[string]interface{}{
		"name":    "new_index",
		"type":    "index",
		"columns": []string{"mobile", "new_column"},
	}).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// IndexDel
	_, err = NewProcess("schemas.default.IndexDel", "unit_tests_schema_table_user_re", "new_index").Exec()
	if err != nil {
		t.Fatal(err)
	}

	// ColumnDel
	_, err = NewProcess("schemas.default.ColumnDel", "unit_tests_schema_table_user_re", "new_column").Exec()
	if err != nil {
		t.Fatal(err)
	}
}

func schemaTestData(t *testing.T) map[string]interface{} {
	root := os.Getenv("GOU_TEST_MOD_ROOT")
	file := path.Join(root, "user.json")
	bytes, err := ioutil.ReadFile(file)
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
