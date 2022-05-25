package query

import (
	"os"
	"testing"

	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
)

// TestAPIRoot
var TestAPIRoot = "/data/apis"
var TestFLWRoot = "/data/flows"
var TestPLGRoot = "/data/plugins"
var TestModRoot = "/data/models"
var TestQueryRoot = "/data/querys"
var TestDriver = "mysql"
var TestDSN = "root:123456@tcp(127.0.0.1:3306)/gou?charset=utf8mb4&parseTime=True&loc=Local"
var TestAESKey = "123456"

var qb query.Query

func TestMain(m *testing.M) {

	TestAPIRoot = os.Getenv("GOU_TEST_API_ROOT")
	TestFLWRoot = os.Getenv("GOU_TEST_FLW_ROOT")
	TestModRoot = os.Getenv("GOU_TEST_MOD_ROOT")
	TestPLGRoot = os.Getenv("GOU_TEST_PLG_ROOT")
	TestQueryRoot = os.Getenv("GOU_TEST_QUERY_ROOT")
	TestDriver = os.Getenv("GOU_TEST_DB_DRIVER")
	TestDSN = os.Getenv("GOU_TEST_DSN")
	TestAESKey = os.Getenv("GOU_TEST_AES_KEY")

	// 数据库连接
	switch TestDriver {
	case "sqlite3":
		capsule.AddConn("primary", "sqlite3", TestDSN).SetAsGlobal()
		break
	default:
		capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
		break
	}

	qb = capsule.Query()

	// Run test suites
	exitVal := m.Run()

	// we can do clean up code here
	os.Exit(exitVal)

}
