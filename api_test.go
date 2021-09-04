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
		Debug:  true,
		Host:   "127.0.0.1",
		Port:   5001,
		Allows: []string{"a.com", "b.com"},
	})

	// 发送请求
	request := func() (maps.MapStr, error) {
		time.Sleep(time.Microsecond * 100)
		resp, err := http.Get("http://127.0.0.1:5001/user/info/1?select=id,name")
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		res := maps.MakeMapStr()
		err = jsoniter.Unmarshal(body, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// 等待服务启动
	times := 0
	for times < 20 { // 2秒超时
		times++
		res, err := request()
		if err != nil {
			continue
		}
		assert.Equal(t, 1, any.Of(res.Get("id")).CInt())
		assert.Equal(t, "管理员", res.Get("name"))
		return
	}

	assert.True(t, false)
}

func TestCallerExec(t *testing.T) {
	defer SelectPlugin("user").Client.Kill()
	res := NewCaller("plugins.user.Login", 1).Run().(*grpc.Response).MustMap()
	res2 := NewCaller("plugins.user.Login", 2).Run().(*grpc.Response).MustMap()
	assert.Equal(t, "login", res.Get("name"))
	assert.Equal(t, "login", res2.Get("name"))
	assert.Equal(t, 1, any.Of(res.Dot().Get("args.0")).CInt())
	assert.Equal(t, 2, any.Of(res2.Dot().Get("args.0")).CInt())
}

func TestCallerFind(t *testing.T) {
	res := NewCaller("models.user.Find", 1, QueryParam{}).Run().(maps.MapStr)
	assert.Equal(t, 1, any.Of(res.Dot().Get("id")).CInt())
	assert.Equal(t, "男", res.Dot().Get("extra.sex"))
}
