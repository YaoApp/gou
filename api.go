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
	"github.com/yaoapp/kun/grpc"
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

// Run 执行指令并返回结果 name = "models.user.Find", name = "plugins.user.Login"
func Run(name string, args ...interface{}) interface{} {
	typ, class, method := extraProcess(name)
	switch typ {
	case "models":
		return runModel(class, method, args...)
	case "plugins":
		return runPlugin(class, method, args...)
	}
	return nil
}

// ServeHTTP 启动HTTP服务
func ServeHTTP(port int, host string, root string, allow string) {

	router := gin.Default()
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

	for _, api := range APIs {
		api.HTTP.Routes(root, allow, router)
	}

	// 释放资源
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		KillPlugins()
		os.Exit(1)
	}()

	hosting := fmt.Sprintf("%s:%d", host, port)
	router.Run(hosting)
}

// Reload 重新载入API
func (api *API) Reload() *API {
	return LoadAPI(api.Source, api.Name)
}

// runModel name = user, method = login, args = [1]
func runPlugin(name string, method string, args ...interface{}) *grpc.Response {
	mod := SelectPluginModel(name)
	res, err := mod.Exec(method, args...)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return res
}

// runModel name = user, method = find, args = [1]
func runModel(name string, method string, args ...interface{}) interface{} {
	mod := Select(name)
	switch method {
	case "find":
		validateArgs(name, method, args, 1)
		return mod.MustFind(args[0])
	}
	return nil
}

// validateArgs( args )
func validateArgs(name string, method string, args []interface{}, length int) {
	if len(args) < length {
		exception.New(
			fmt.Sprintf("Model:%s%s(args...); 参数错误", name, method),
			400,
		).Throw()
	}
}

// extraProcess 解析执行方法  name = "models.user.Find", name = "plugins.user.Login"
// return type=models, name=login, class=user
func extraProcess(name string) (typ string, class string, method string) {
	namer := strings.Split(name, ".")
	last := len(namer) - 1
	if last < 2 {
		exception.New(
			fmt.Sprintf("Process:%s 格式错误", name),
			400,
		).Throw()
	}
	typ = strings.ToLower(namer[0])
	class = strings.ToLower(strings.Join(namer[1:last], "."))
	method = strings.ToLower(namer[last])
	return typ, class, method
}
