package gou

import (
	"os"
	"path"
	"testing"

	"github.com/yaoapp/kun/utils"
)

// TestAPIRoot
var TestAPIRoot = "/data/apis"

func init() {
	TestAPIRoot = os.Getenv("GOU_TEST_API_ROOT")
}

func TestLoadAPI(t *testing.T) {
	file, err := os.Open(path.Join(TestAPIRoot, "user.http.json"))
	if err != nil {
		panic(err)
	}
	user := LoadAPI(file)
	user.Reload()
	utils.Dump(user)
}
