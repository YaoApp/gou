package gou

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
)

func init() {
	SetHTTPGuards(map[string]gin.HandlerFunc{"bearer-jwt": func(ctx *gin.Context) {}})
}
func TestLoadAPI(t *testing.T) {
	user := LoadAPI("file://"+path.Join(TestAPIRoot, "user.http.json"), "user")
	assert.Equal(t, user.Name, "user")
}

func TestSelectAPI(t *testing.T) {
	user := SelectAPI("user")
	user.Reload()
	assert.Equal(t, user.Name, "user")
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

func TestAPIUserHello(t *testing.T) {
	router := GetTestRouter()
	response := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/user/hello", nil)
	router.ServeHTTP(response, req)
	assert.Equal(t, `"hello:world"`, response.Body.String())
}

func TestAPIUserAuth(t *testing.T) {
	router := GetTestRouter()
	response := httptest.NewRecorder()
	body := []byte(`{"response":"success"}`)
	req, _ := http.NewRequest("POST", "/user/auth/hi?foo=bar&hello=world", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer Token:123456")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, req)
	assert.Equal(t, `"hello:world"`, response.Body.String())
}

func TestAPIUserAuthSid(t *testing.T) {
	router := GetTestRouter(func(c *gin.Context) {
		c.Set("__sid", c.Query("sid"))
		c.Set("__global", map[string]interface{}{"hello": "world"})
	})
	response := httptest.NewRecorder()
	id := session.ID()
	ss := session.Global().ID(id)
	ss.Set("id", 1)

	body := []byte(`{"response":"success"}`)
	req, _ := http.NewRequest("POST", "/user/auth/hi?foo=bar&hello=world&sid="+id, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer Token:123456")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, req)
	assert.Equal(t, `"hello:world"`, response.Body.String())
}

func TestAPIUserAuthFail(t *testing.T) {
	router := GetTestRouter()
	response := httptest.NewRecorder()
	body := []byte(`{"response":"failure"}`)
	req, _ := http.NewRequest("POST", "/user/auth/hi?foo=bar&hello=world", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer Token:123456")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, req)
	assert.Equal(t, `{"code":403,"message":"failure"}`, response.Body.String())
}

func TestAPIUserSessionFlow(t *testing.T) {
	router := GetTestRouter(func(c *gin.Context) {
		c.Set("__sid", c.Query("sid"))
		c.Set("__global", map[string]interface{}{"hello": "world"})
	})
	response := httptest.NewRecorder()
	id := session.ID()
	ss := session.Global().ID(id)
	ss.Set("id", 1)
	req, _ := http.NewRequest("GET", "/user/session/flow?sid="+id, nil)
	router.ServeHTTP(response, req)
	res := GetResponseMap(response).Dot()
	assert.Equal(t, float64(1), res.Get("ID"))
	assert.Equal(t, float64(1), res.Get("会话信息.id"))
	assert.Equal(t, "admin", res.Get("会话信息.type"))
	assert.Equal(t, "world", res.Get("全局信息.hello"))
	assert.Equal(t, float64(1), res.Get("用户数据.id"))
	assert.Equal(t, "管理员", res.Get("用户数据.name"))
	assert.Equal(t, "admin", res.Get("用户数据.type"))
	assert.Equal(t, "world", res.Get("脚本数据.global.hello"))
	assert.Equal(t, float64(1), res.Get("脚本数据.session.id"))
	assert.Equal(t, "admin", res.Get("脚本数据.session.type"))
	assert.Equal(t, "application/json", response.Header()["Content-Type"][0])
	assert.Equal(t, "1", response.Header()["User-Agent"][0])
}

func TestAPIUserSessionIn(t *testing.T) {
	router := GetTestRouter(func(c *gin.Context) {
		c.Set("__sid", c.Query("sid"))
		c.Set("__global", map[string]interface{}{"hello": "world"})
	})
	response := httptest.NewRecorder()
	id := session.ID()
	ss := session.Global().ID(id)
	ss.Set("id", 1)
	req, _ := http.NewRequest("GET", "/user/session/in?sid="+id, nil)
	router.ServeHTTP(response, req)
	res := GetResponseMap(response).Dot()
	assert.Equal(t, float64(1), res.Get("id"))
}

func GetTestRouter(middlewares ...gin.HandlerFunc) *gin.Engine {
	srv := Server{
		Debug:  true,
		Host:   "127.0.0.1",
		Port:   5001,
		Allows: []string{"a.com", "b.com"},
	}
	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	SetHTTPRoutes(router, srv, middlewares...)
	return router
}

func GetResponseMap(resp *httptest.ResponseRecorder) maps.MapStrAny {
	body := resp.Body.Bytes()
	res := map[string]interface{}{}
	jsoniter.Unmarshal(body, &res)
	return maps.Of(res)
}
