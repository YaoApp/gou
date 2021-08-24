package gou

import (
	"os"
	"path"
	"testing"

	"github.com/yaoapp/kun/utils"
)

// APIROOT
var APIROOT = "/data/apis"

func init() {
	APIROOT = os.Getenv("API_ROOT")
}

func TestLoadAPI(t *testing.T) {
	file, err := os.Open(path.Join(APIROOT, "user.http.json"))
	if err != nil {
		panic(err)
	}
	user := LoadAPI(file)
	user.Reload()
	utils.Dump(user)
}
