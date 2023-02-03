package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/yaoapp/kun/log"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/application"
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

	return LoadSource(file, data, id, guard...)
}

// LoadSource load api by source
func LoadSource(file string, data []byte, id string, guard ...string) (*API, error) {

	http := HTTP{}
	err := application.Parse(file, data, &http)
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

// Select select api
func Select(id string) *API {
	api, has := APIs[id]
	if !has {
		exception.New("[API] %s not loaded", 500, id).Throw()
	}
	return api
}

// SetRoutes set the api routes
func SetRoutes(router *gin.Engine, path string, allows ...string) {

	// Error handler
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

	// Load apis
	for _, api := range APIs {
		api.HTTP.Routes(router, path, allows...)
	}
}

// SetGuards set guards
func SetGuards(guards map[string]gin.HandlerFunc) {
	HTTPGuards = guards
}

// AddGuard add guard
func AddGuard(name string, guard gin.HandlerFunc) {
	HTTPGuards[name] = guard
}

// Reload API
func (api *API) Reload() (*API, error) {
	return Load(api.File, api.ID)
}
