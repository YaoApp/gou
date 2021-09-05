package gou

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun"
)

// APIs 已加载API列表
var APIs = map[string]*API{}

// LoadAPI 加载API
func LoadAPI(source string, name string) *API {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	http := HTTP{}
	err := helper.UnmarshalFile(input, &http)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	APIs[name] = &API{
		Name:   name,
		Source: source,
		HTTP:   http,
		Type:   "http",
	}
	return APIs[name]
}

// SelectAPI 读取已加载API
func SelectAPI(name string) *API {
	api, has := APIs[name]
	if !has {
		exception.New(
			fmt.Sprintf("API:%s; 尚未加载", name),
			500,
		).Throw()
	}
	return api
}

// ServeHTTP  启动HTTP服务
func ServeHTTP(server Server, middlewares ...gin.HandlerFunc) {

	if server.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	ServeHTTPCustomRouter(router, server, middlewares...)
}

// ServeHTTPCustomRouter 启动HTTP服务, 自定义路由器
func ServeHTTPCustomRouter(router *gin.Engine, server Server, middlewares ...gin.HandlerFunc) {

	// 添加中间件
	for _, handler := range middlewares {
		router.Use(handler)
	}

	// 错误处理
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {

		var code = http.StatusInternalServerError

		if err, ok := recovered.(string); ok {
			c.JSON(code, xun.R{
				"code":    code,
				"message": fmt.Sprintf("%s", err),
			})
		} else if err, ok := recovered.(exception.Exception); ok {
			code = err.Code
			c.JSON(code, xun.R{
				"code":    code,
				"message": err.Message,
			})
		} else if err, ok := recovered.(*exception.Exception); ok {
			code = err.Code
			c.JSON(code, xun.R{
				"code":    code,
				"message": err.Message,
			})
		} else {
			c.JSON(code, xun.R{
				"code":    code,
				"message": fmt.Sprintf("%v", recovered),
			})
		}

		c.AbortWithStatus(code)
	}))

	// 加载API
	for _, api := range APIs {
		api.HTTP.Routes(router, server.Root, server.Allows...)
	}

	// 服务终止时 关闭插件进程
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-c
		KillPlugins()
		os.Exit(1)
	}()

	hosting := fmt.Sprintf("%s:%d", server.Host, server.Port)
	router.Run(hosting)
}

// SetHTTPGuards 加载中间件
func SetHTTPGuards(guards map[string]gin.HandlerFunc) {
	HTTPGuards = guards
}

// AddHTTPGuard 添加中间件
func AddHTTPGuard(name string, guard gin.HandlerFunc) {
	HTTPGuards[name] = guard
}

// Reload 重新载入API
func (api *API) Reload() *API {
	api = LoadAPI(api.Source, api.Name)
	return api
}
