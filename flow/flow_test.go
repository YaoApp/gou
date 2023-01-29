package flow

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/capsule"
)

func TestLoad(t *testing.T) {
	prepare(t)
	defer clean()
	check(t)
}

func TestSelect(t *testing.T) {

	prepare(t)
	defer clean()

	basic, err := Select("basic")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Basic", basic.Name)
	assert.Equal(t, "basic", basic.ID)
	assert.Equal(t, 3, len(basic.Nodes))
}

func prepare(t *testing.T) {

	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := application.OpenFromDisk(root) // Load app
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	dbconnect(t)
	loadModels(t)
	loadQuery(t)
	loadFlows(t)
	prepareData(t)
}

func check(t *testing.T) {
	keys := map[string]bool{}
	for id := range Flows {
		keys[id] = true
	}
	flows := []string{"basic"}
	for _, id := range flows {
		_, has := keys[id]
		assert.True(t, has)
	}
}

func clean() {
	dbclose()
}

func dbclose() {
	if capsule.Global != nil {
		capsule.Global.Connections.Range(func(key, value any) bool {
			if conn, ok := value.(*capsule.Connection); ok {
				conn.Close()
			}
			return true
		})
	}
}

func dbconnect(t *testing.T) {

	TestDriver := os.Getenv("GOU_TEST_DB_DRIVER")
	TestDSN := os.Getenv("GOU_TEST_DSN")

	// connect db
	switch TestDriver {
	case "sqlite3":
		capsule.AddConn("primary", "sqlite3", TestDSN).SetAsGlobal()
		break
	default:
		capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
		break
	}

}

func prepareData(t *testing.T) {
	for id := range model.Models {
		mod := model.Select(id)
		err := mod.Migrate(true)
		if err != nil {
			t.Fatal(err)
		}
	}

	user := model.Select("user")
	user.Insert(
		[]string{"name", "mobile", "status"},
		[][]interface{}{
			{"U1", "13911101001", "enabled"},
			{"U2", "13911101002", "enabled"},
			{"U3", "13911101003", "enabled"},
			{"U4", "13911101004", "enabled"},
		})

	category := model.Select("category")
	category.Insert(
		[]string{"name"},
		[][]interface{}{{"Cat"}, {"Dog"}, {"Duck"}, {"Tiger"}, {"Lion"}},
	)

}

func loadFlows(t *testing.T) {

	flows := map[string]string{
		"basic": filepath.Join("flows", "tests", "basic.flow.yao"),
	}

	for id, file := range flows {
		flow, err := Load(file, id)
		if err != nil {
			t.Fatal(err)
		}

		_, err = flow.Reload()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func loadModels(t *testing.T) {

	TestAESKey := os.Getenv("GOU_TEST_AES_KEY")
	_, err := model.WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, TestAESKey)), "AES")
	if err != nil {
		t.Fatal(err)
	}

	mods := map[string]string{
		"user":     filepath.Join("models", "user.mod.yao"),
		"pet":      filepath.Join("models", "pet.mod.yao"),
		"tag":      filepath.Join("models", "tag.mod.yao"),
		"category": filepath.Join("models", "category.mod.yao"),
		"user.pet": filepath.Join("models", "user", "pet.mod.yao"),
		"pet.tag":  filepath.Join("models", "pet", "tag.mod.yao"),
	}

	// load mods
	for id, file := range mods {
		_, err := model.Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func loadQuery(t *testing.T) {

	TestAESKey := os.Getenv("GOU_TEST_AES_KEY")

	// query engine
	query.Register("query-test", &gou.Query{
		Query: capsule.Query(),
		GetTableName: func(s string) string {
			if mod, has := model.Models[s]; has {
				return mod.MetaData.Table.Name
			}
			exception.New("[query] %s not found", 404).Throw()
			return s
		},
		AESKey: TestAESKey,
	})
}
