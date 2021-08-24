package gou

import (
	"os"
	"path"
	"testing"

	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xun/capsule"
)

// TestAPIRoot
var TestAPIRoot = "/data/apis"

func init() {
	TestAPIRoot = os.Getenv("GOU_TEST_API_ROOT")
	TestModRoot = os.Getenv("GOU_TEST_MOD_ROOT")
	TestDSN = os.Getenv("GOU_TEST_DSN")
	capsule.AddConn("primary", "mysql", TestDSN)

	userfile, err := os.Open(path.Join(TestModRoot, "user.json"))
	if err != nil {
		panic(err)
	}

	manufile, err := os.Open(path.Join(TestModRoot, "user.json"))
	if err != nil {
		panic(err)
	}

	LoadModel(userfile, "user")
	LoadModel(manufile, "manu")
}

func TestLoadAPI(t *testing.T) {
	file, err := os.Open(path.Join(TestAPIRoot, "user.http.json"))
	if err != nil {
		panic(err)
	}
	user := LoadAPI(file)
	user.Reload()
}

func TestRunModel(t *testing.T) {
	res := Run("models.user.Find", 1)
	id := res.(maps.MapStr).Get("id")
	utils.Dump(id)
}

func TestRunPlugin(t *testing.T) {
	res := Run("plugins.user.Login", 1)
	utils.Dump(res)
}
