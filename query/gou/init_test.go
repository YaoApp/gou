package gou

import (
	"bytes"
	"fmt"
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
var TestQueryRoot = "/data/querys"
var TestDriver = "mysql"
var TestDSN = "root:123456@tcp(127.0.0.1:3306)/gou?charset=utf8mb4&parseTime=True&loc=Local"
var TestAESKey = "123456"

func should(t assert.TestingT, actual, expected interface{}, msgAndArgs ...interface{}) bool {
	return assert.Equal(t, expected, actual, msgAndArgs...)
}

// Q adapts expected SQL strings to the active driver's identifier quoting
// and placeholder format. Test cases are authored with backticks and ? placeholders
// (MySQL canonical form).
func Q(sql string) string {
	if TestDriver == "postgres" {
		sql = strings.ReplaceAll(sql, "`", `"`)
		n := 1
		for strings.Contains(sql, "?") {
			sql = strings.Replace(sql, "?", fmt.Sprintf("$%d", n), 1)
			n++
		}
	}
	return sql
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

	TestQueryRoot = os.Getenv("GOU_TEST_QUERY_ROOT")
	TestDriver = os.Getenv("GOU_TEST_DB_DRIVER")
	TestDSN = os.Getenv("GOU_TEST_DSN")
	TestAESKey = os.Getenv("GOU_TEST_AES_KEY")

	// 数据库连接
	switch TestDriver {
	case "sqlite3":
		capsule.AddConn("primary", "sqlite3", TestDSN).SetAsGlobal()
	case "postgres":
		capsule.AddConn("primary", "postgres", TestDSN).SetAsGlobal()
	default:
		capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
	}

	qb = capsule.Query()

	// Setup integration test table
	setupTestTable()

	// Run test suites
	exitVal := m.Run()

	// Cleanup
	teardownTestTable()
	os.Exit(exitVal)

}

func setupTestTable() {
	var ddl string
	switch TestDriver {
	case "postgres":
		ddl = `CREATE TABLE IF NOT EXISTS gou_test_user (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			email VARCHAR(100)
		)`
	case "sqlite3":
		ddl = `CREATE TABLE IF NOT EXISTS gou_test_user (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100),
			email VARCHAR(100)
		)`
	default:
		ddl = `CREATE TABLE IF NOT EXISTS gou_test_user (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100),
			email VARCHAR(100)
		)`
	}
	_, err := qb.DB().Exec(ddl)
	if err != nil {
		panic("failed to create test table: " + err.Error())
	}
	_, _ = qb.DB().Exec("DELETE FROM gou_test_user")
	inserts := []string{
		"INSERT INTO gou_test_user (name, email) VALUES ('Alice', 'alice@test.com')",
		"INSERT INTO gou_test_user (name, email) VALUES ('Bob', 'bob@test.com')",
		"INSERT INTO gou_test_user (name, email) VALUES ('Charlie', 'charlie@test.com')",
	}
	for _, sql := range inserts {
		_, err := qb.DB().Exec(sql)
		if err != nil {
			panic("failed to insert test data: " + err.Error())
		}
	}
}

func teardownTestTable() {
	_, _ = qb.DB().Exec("DROP TABLE IF EXISTS gou_test_user")
}
