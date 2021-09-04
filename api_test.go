package gou

import (
	"io/ioutil"
	"net/http"
	"path"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
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

func TestServeHTTP(t *testing.T) {
	go ServeHTTP(Server{
		Host:   "127.0.0.1",
		Port:   5001,
		Allows: []string{"a.com", "b.com"},
	})

	// 等待服务启动
	time.Sleep(time.Microsecond * 200)
	resp, err := http.Get("http://127.0.0.1:5001/user/info/1?select=id,name")
	if err != nil {
		assert.Nil(t, err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	res := maps.MakeMapStr()
	err = jsoniter.Unmarshal(body, &res)
	if err != nil {
		assert.Nil(t, err)
		return
	}
	assert.Equal(t, 1, any.Of(res.Get("id")).CInt())
	assert.Equal(t, "管理员", res.Get("name"))
}

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
