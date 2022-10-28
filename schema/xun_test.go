package schema

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/gou/schema/xun"
	"github.com/yaoapp/xun/capsule"
)

func TestXunCreateDrop(t *testing.T) {
	sch := newXunSchema(t)
	err := sch.Drop("test_db")
	if err != nil {
		t.Fatal(err)
	}

	err = sch.Create("test_db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		sch.Drop("test_db")
		sch.Close()
	}()
}

func TestXunTables(t *testing.T) {
	sch := newXunSchema(t)
	defer sch.Close()

	createTable(t, "schema_tests_user")
	defer sch.TableDrop("schema_tests_user")

	tables, err := sch.Tables("schema_tests_")
	if err != nil {
		t.Fatal(err)
	}

	if len(tables) != 1 {
		t.Fatal(fmt.Errorf("not found schema_tests_user"))
	}
	assert.Contains(t, tables, "schema_tests_user")
}

func TestXunTableGet(t *testing.T) {
	sch := newXunSchema(t)
	defer sch.Close()

	createTable(t, "schema_tests_user")
	defer sch.TableDrop("schema_tests_user")

	table, err := sch.TableGet("schema_tests_user")
	if err != nil {
		t.Fatal(err)
	}

	_, err = sch.TableGet("schema_tests_not_found")
	assert.Contains(t, err.Error(), "does not exists")

	assert.Equal(t, 17, len(table.Columns))
	assert.Equal(t, 3, len(table.Indexes))
	assert.Equal(t, true, table.Option.SoftDeletes)
	assert.Equal(t, true, table.Option.Timestamps)
}

func TestXunTableCreateDrop(t *testing.T) {
	sch := newXunSchema(t)
	defer sch.Close()

	sch.TableDrop("schema_tests_user")
	user := newUserBlueprint(t)
	err := sch.TableCreate("schema_tests_user", user)
	if err != nil {
		t.Fatal(err)
	}
	defer sch.TableDrop("schema_tests_user")
}

func TestXunTableRename(t *testing.T) {
	sch := newXunSchema(t)
	defer sch.Close()

	createTable(t, "schema_tests_user")
	defer sch.TableDrop("schema_tests_user")

	err := sch.TableRename("schema_tests_user", "schema_tests_user_re")
	if err != nil {
		t.Fatal(err)
	}
	defer sch.TableDrop("schema_tests_user_re")

	tables, err := sch.Tables("schema_tests_")
	assert.Contains(t, tables, "schema_tests_user_re")
}

func TestXunTableDiff(t *testing.T) {
	sch := newXunSchema(t)
	defer sch.Close()
	user := newUserBlueprint(t)
	user2 := newUser2Blueprint(t)
	diff, err := sch.TableDiff(user, user2)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(diff.Option))
	assert.Equal(t, 2, len(diff.Columns.Add))
	assert.Equal(t, 1, len(diff.Columns.Alt))
	assert.Equal(t, 2, len(diff.Columns.Del))
	assert.Equal(t, 1, len(diff.Indexes.Add))
	assert.Equal(t, 0, len(diff.Indexes.Alt))
	assert.Equal(t, 1, len(diff.Indexes.Del))
}

func TestXunTableSave(t *testing.T) {
	sch := newXunSchema(t)
	defer sch.Close()

	// Create
	sch.TableDrop("schema_tests_user")
	user := newUserBlueprint(t)
	err := sch.TableSave("schema_tests_user", user)
	if err != nil {
		t.Fatal(err)
	}
	defer sch.TableDrop("schema_tests_user")

	// Update
	user2 := newUser2Blueprint(t)
	err = sch.TableSave("schema_tests_user", user2)
	if err != nil {
		t.Fatal(err)
	}

	// Checking
	table, err := sch.TableGet("schema_tests_user")
	if err != nil {
		t.Fatal(err)
	}

	mapping := table.ColumnsMapping()
	if _, has := mapping["typec"]; !has {
		t.Fatal("TableSave should have a typec column after the change")
	}

	if _, has := mapping["newfield"]; !has {
		t.Fatal("TableSave should have a newfield column after the change")
	}

	// if _, has := mapping["__DEL__resume"]; !has {
	// 	t.Fatal("TableSave should have a __DEL__resume column after the change")
	// }

	// if _, has := mapping["__DEL__type"]; !has {
	// 	t.Fatal("TableSave should have a __DEL__type column after the change")
	// }
}

func TestXunColumnAddDelAlt(t *testing.T) {
	sch := newXunSchema(t)
	defer sch.Close()

	createTable(t, "schema_tests_user")
	defer sch.TableDrop("schema_tests_user")

	// Add
	err := sch.ColumnAdd("schema_tests_user", types.Column{
		Name:   "new_column",
		Type:   "string",
		Length: 20,
		Index:  true,
	})

	if err != nil {
		t.Fatal(err)
	}

	// Delete
	err = sch.ColumnDel("schema_tests_user", "resume")
	if err != nil {
		t.Fatal(err)
	}

	// Alter
	err = sch.ColumnAlt("schema_tests_user", types.Column{
		Name:    "type",
		Type:    "enum",
		Default: "admin",
		Option:  []string{"admin", "uroot", "staff"},
	})

	if err != nil {
		t.Fatal(err)
	}

	// Checking
	table, err := sch.TableGet("schema_tests_user")
	if err != nil {
		t.Fatal(err)
	}

	mapping := table.ColumnsMapping()
	newColumn, has := mapping["new_column"]
	if !has {
		t.Fatal(has)
	}

	if _, has := mapping["resume"]; has {
		t.Fatal(fmt.Errorf("ColumnDel not work"))
	}

	// if _, has := mapping["__DEL__resume"]; !has {
	// 	t.Fatal(fmt.Errorf("ColumnDel not correct"))
	// }

	altColumn, has := mapping["type"]
	if !has {
		t.Fatal(has)
	}

	driver := os.Getenv("GOU_TEST_DB_DRIVER")
	if driver != "sqlite3" {
		assert.Equal(t, "type", altColumn.Name)
		assert.Equal(t, "admin", altColumn.Default)
		assert.Equal(t, []string{"admin", "uroot", "staff"}, altColumn.Option)
	}

	assert.Equal(t, true, newColumn.Index)
	assert.Equal(t, "new_column", newColumn.Name)
	assert.Equal(t, "string", newColumn.Type)
	assert.Equal(t, 20, newColumn.Length)
}

func TestXunIndexAdd(t *testing.T) {
	sch := newXunSchema(t)
	defer sch.Close()

	createTable(t, "schema_tests_user")
	defer sch.TableDrop("schema_tests_user")

	// Add
	err := sch.IndexAdd("schema_tests_user", types.Index{
		Name:    "new_index",
		Type:    "index",
		Columns: []string{"mobile", "type"},
	})

	if err != nil {
		t.Fatal(err)
	}

	// Del
	err = sch.IndexDel("schema_tests_user", "name_idcard_index")
	if err != nil {
		t.Fatal(err)
	}

	// Checking
	table, err := sch.TableGet("schema_tests_user")
	if err != nil {
		t.Fatal(err)
	}

	mapping := table.IndexesMapping()
	newIndex, has := mapping["new_index"]
	if !has {
		t.Fatal(has)
	}

	if _, has := mapping["name_idcard_index"]; has {
		t.Fatal(fmt.Errorf("IndexDel not work"))
	}

	assert.Equal(t, "new_index", newIndex.Name)
	assert.Equal(t, "index", newIndex.Type)
	assert.Equal(t, []string{"mobile", "type"}, newIndex.Columns)
}

func newXunSchema(t *testing.T) types.Schema {
	dsn := os.Getenv("GOU_TEST_DSN")
	driver := os.Getenv("GOU_TEST_DB_DRIVER")
	manager, err := capsule.Add("primary", driver, dsn)
	if err != nil {
		t.Fatal(err)
	}
	sch := Use("tests")
	err = sch.SetOption(xun.Option{Manager: manager})
	if err != nil {
		t.Fatal(err)
	}
	return sch
}

func newUserBlueprint(t *testing.T) types.Blueprint {
	root := os.Getenv("GOU_TEST_MOD_ROOT")
	file := path.Join(root, "user.json")

	blueprint, err := types.NewFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return blueprint
}

func newUser2Blueprint(t *testing.T) types.Blueprint {
	root := os.Getenv("GOU_TEST_MOD_ROOT")
	file := path.Join(root, "user.json")
	user, err := types.NewFile(file)
	if err != nil {
		t.Fatal(err)
	}
	user2 := user
	user2.Columns = append([]types.Column{}, user.Columns[0:10]...)
	user2.Columns = append(user2.Columns, user.Columns[11:]...)
	user2.Columns = append(user2.Columns, types.Column{Name: "newfield", Type: "string", Index: true})
	user2.Columns[1].Index = false
	user2.Columns[2].Name = "typec"
	user2.Columns[2].Length = 36
	user2.Indexes = append([]types.Index{}, user.Indexes[1:]...)
	user2.Indexes = append(user2.Indexes, types.Index{Name: "add_index", Columns: []string{"mobile", "password"}, Type: "index"})
	user2.Indexes[1].Columns = append(user2.Indexes[1].Columns, "manu_id")
	user2.Option.Timestamps = false
	return user2
}

func createTable(t *testing.T, name string) {
	sch := newXunSchema(t)
	defer sch.Close()
	sch.TableDrop(name)
	user := newUserBlueprint(t)
	err := sch.TableCreate("schema_tests_user", user)
	if err != nil {
		t.Fatal(err)
	}
}
