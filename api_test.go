package gou

import (
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
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
	shutdown := make(chan bool)
	go ServeHTTP(Server{
		Debug:  true,
		Host:   "127.0.0.1",
		Port:   5001,
		Allows: []string{"a.com", "b.com"},
	}, &shutdown, func(s Server) {
		log.Println("服务已关闭")
	})
	defer func() { shutdown <- true }()

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

func TestServeHTTPShutDown(t *testing.T) {
	shutdown := make(chan bool)
	go ServeHTTP(Server{
		Debug:  true,
		Host:   "127.0.0.1",
		Port:   5001,
		Allows: []string{"a.com", "b.com"},
	}, &shutdown, func(s Server) {
		log.Println("服务已关闭")
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

		// 测试关闭
		shutdown <- true
		time.Sleep(time.Second * 1)
		_, err = request()
		assert.NotNil(t, err)
		return
	}

	assert.True(t, false)
}
