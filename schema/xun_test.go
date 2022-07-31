package schema

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

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

	assert.Equal(t, 17, len(table.Columns))
	assert.Equal(t, 2, len(table.Indexes))
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
	user2 := user
	user2.Columns = append([]types.Column{}, user.Columns[0:10]...)
	user2.Columns = append(user2.Columns, user.Columns[11:]...)
	user2.Columns = append(user2.Columns, types.Column{Name: "newfield", Type: "string", Index: true})
	user2.Columns[1].Index = false
	user2.Columns[2].Name = "typec"
	user2.Columns[2].Length = 36
	user2.Indexes = append([]types.Index{}, user.Indexes[1:]...)
	user2.Indexes = append(user2.Indexes, types.Index{Name: "add_index", Columns: []string{"mobile", "password"}, Type: "index"})
	user2.Option.Timestamps = false
	sch.TableDiff(user, user2)

}

func TestXunTableSave(t *testing.T) {}

func TestXunColumnAdd(t *testing.T) {}

func TestXunColumnDel(t *testing.T) {}

func TestXunColumnAlt(t *testing.T) {}

func TestXunIndexAdd(t *testing.T) {}

func TestXunIndexDel(t *testing.T) {}

func TestXunIndexAlt(t *testing.T) {}

func newXunSchema(t *testing.T) types.Schema {
	dsn := os.Getenv("GOU_TEST_DSN")
	driver := os.Getenv("GOU_TEST_DB_DRIVER")
	manager := capsule.AddConn("primary", driver, dsn, 5*time.Second)
	sch := Use("xun")
	err := sch.SetOption(xun.Option{Manager: manager})
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
