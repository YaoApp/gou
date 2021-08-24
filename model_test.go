package gou

import (
	"os"
	"path"
	"testing"

	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xun/capsule"
)

// TestModRoot
var TestModRoot = "/data/models"
var TestDSN = "root:123456@tcp(192.168.31.119:3306)/xiang?charset=utf8mb4&parseTime=True&loc=Local"

func init() {
	TestModRoot = os.Getenv("GOU_TEST_MOD_ROOT")
	TestDSN = os.Getenv("GOU_TEST_DSN")
	capsule.AddConn("primary", "mysql", TestDSN)
}

func TestLoadModel(t *testing.T) {
	file, err := os.Open(path.Join(TestModRoot, "user.json"))
	if err != nil {
		panic(err)
	}
	LoadModel(file, "user")
	user := Select("user")
	user.Migrate()
	utils.Dump(user)
}
