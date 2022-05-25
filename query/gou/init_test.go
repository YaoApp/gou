package gou

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/exception"
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

func should(t assert.TestingT, actual, expected interface{}, msgAndArgs ...interface{}) bool {
	return assert.Equal(t, expected, actual, msgAndArgs...)
}

var qb query.Query

// GetFileName
func GetFileName(name string) string {
	return path.Join(TestQueryRoot, name)
}

// TableName
func TableName(name string) string {
	return strings.TrimPrefix(name, "$")
}

// ReadFile
func ReadFile(name string) []byte {
	fullname := path.Join(TestQueryRoot, name)

	file, err := os.Open(fullname)
	if err != nil {
		exception.New("读取文件失败 %s", 500, err.Error()).Throw()
	}
	defer file.Close()

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, file)
	if err != nil {
		exception.New("读取数据失败 %s", 500, err.Error()).Throw()
	}
	return buf.Bytes()
}

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
