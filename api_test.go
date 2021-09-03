package gou

import (
	"path"
	"testing"

	"github.com/yaoapp/kun/grpc"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
)

func TestLoadAPI(t *testing.T) {
	user := LoadAPI("file://"+path.Join(TestAPIRoot, "user.http.json"), "user")
	user.Reload()
}

func TestSelectAPI(t *testing.T) {
	user := SelectAPI("user")
	user.Reload()
}

// func TestServeHTTP(t *testing.T) {
// 	ServeHTTP(5011, "127.0.0.1", "/api", "*")
// }

func TestRunModel(t *testing.T) {
	res := Run("models.user.Find", 1, QueryParam{})
	id := res.(maps.MapStr).Get("id")
	utils.Dump(id)
}

func TestRunPlugin(t *testing.T) {
	defer SelectPlugin("user").Client.Kill()
	res := Run("plugins.user.Login", 1)
	res2 := Run("plugins.user.Login", 2)
	utils.Dump(res.(*grpc.Response).MustMap())
	utils.Dump(res2.(*grpc.Response).MustMap())
}
