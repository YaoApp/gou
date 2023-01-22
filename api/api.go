package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yaoapp/kun/log"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun"
)

// APIs 已加载API列表
var APIs = map[string]*API{}

// Load load the api
func Load(file, id string, guard ...string) (*API, error) {

	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	http := HTTP{}
	err = application.Parse(file, data, &http)
	if err != nil {
		log.Error("[API] Load %s Error: %s", id, err.Error())
		return nil, err
	}

	// Filesystem Router
	if http.Group == "" {
		http.Group = strings.ReplaceAll(strings.ToLower(id), ".", "/")
	}

	// Validate API
	uniquePathCheck := map[string]bool{}
	for _, path := range http.Paths {
		unique := fmt.Sprintf("%s.%s", path.Method, path.Path)
		if _, has := uniquePathCheck[unique]; has {
			log.Error("[API] Load %s is already registered", id)
			return nil, fmt.Errorf("[API] Load %s Error: is already registered", id)
		}
		uniquePathCheck[unique] = true
	}

	// Default Guard
	if http.Guard == "" && len(guard) > 0 {
		http.Guard = guard[0]
	}

	APIs[id] = &API{
		ID:   id,
		File: file,
		HTTP: http,
		Type: "http",
	}

	return APIs[id], nil
}

// SelectAPI 读取已加载API
func SelectAPI(id string) *API {
	api, has := APIs[id]
	if !has {
		exception.New("[API] %s not loaded", 500, id).Throw()
	}
	return api
}

// ServeHTTP  Start the http server
func ServeHTTP(server Server, shutdown chan bool, onShutdown func(Server), middlewares ...gin.HandlerFunc) {
	router := gin.Default()
	ServeHTTPCustomRouter(router, server, shutdown, onShutdown, middlewares...)
}

// ServeHTTPCustomRouter Start the cumtom http server
func ServeHTTPCustomRouter(router *gin.Engine, server Server, shutdown chan bool, onShutdown func(Server), middlewares ...gin.HandlerFunc) {

	// recive interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// ctx
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// kill plugins
	// defer KillPlugins()

	// Set the routes
	SetHTTPRoutes(router, server, middlewares...)

	// server setting
	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// start WebSocket hub
	for name, upgrader := range websocket.Upgraders {
		upgrader.SetRouter(router)
		go upgrader.Start()
		log.Trace("Websocket %s start", name)
	}

	// stop WebSocket hub
	defer func() {
		for name, upgrader := range websocket.Upgraders {
			upgrader.Stop()
			log.Trace("Websocket %s quit", name)
		}
	}()

	// // start tasks
	// for name, t := range task.Tasks {
	// 	go t.Start()
	// 	log.Trace("Task %s start", name)
	// }

	// // stop tasks
	// defer func() {
	// 	for name, t := range task.Tasks {
	// 		t.Stop()
	// 		log.Trace("Task %s quit", name)
	// 	}
	// }()

	// // start Schedules
	// for name, sch := range Schedules {
	// 	sch.Start()
	// 	log.Trace("Schedule %s start", name)
	// }

	// // stop Schedules
	// defer func() {
	// 	for name, sch := range Schedules {
	// 		sch.Stop()
	// 		log.Trace("Schedule %s quit", name)
	// 	}
	// }()

	// start Http server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen: %s", err)
		}
	}()

	for {
		select {
		case <-shutdown:
			srv.Shutdown(ctx)
			onShutdown(server)
			return
		case <-interrupt:
			srv.Shutdown(ctx)
			onShutdown(server)
			return
		}
	}
}

// SetHTTPRoutes 设定路由
func SetHTTPRoutes(router *gin.Engine, server Server, middlewares ...gin.HandlerFunc) {
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

	//Load WebSocket
}

// SetHTTPGuards 加载中间件
func SetHTTPGuards(guards map[string]gin.HandlerFunc) {
	HTTPGuards = guards
}

// AddHTTPGuard 添加中间件
func AddHTTPGuard(name string, guard gin.HandlerFunc) {
	HTTPGuards[name] = guard
}

// Reload API
func (api *API) Reload() (*API, error) {
	return Load(api.File, api.ID)
}
