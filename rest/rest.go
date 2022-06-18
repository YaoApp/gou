package rest

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/exception"
)

// APIs the loaded RESTful APIs
var APIs = map[string]*API{}

// Guards The Guard list
var Guards = map[string]gin.HandlerFunc{}

// Bind the http server
func Bind(srv *http.Server, option Option, middlewares ...gin.HandlerFunc) error {

	if option.Mode == "production" {
		gin.SetMode("release")
	}

	router := gin.Default()

	// handle error
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		var code = http.StatusInternalServerError
		switch recovered.(type) {
		case exception.Exception:
			err := recovered.(exception.Exception)
			code = err.Code
			c.JSON(code, gin.H{
				"code":    code,
				"message": err.Message,
			})
			break
		case *exception.Exception:
			err := recovered.(*exception.Exception)
			code = err.Code
			c.JSON(code, gin.H{
				"code":    code,
				"message": err.Message,
			})
			break
		default:
			c.JSON(code, gin.H{
				"code":    code,
				"message": fmt.Sprintf("%v", recovered),
			})
		}
		c.AbortWithStatus(code)
	}))

	// Use the middlewares
	router.Use(middlewares...)

	// API Hander
	for name, api := range APIs {
		err := api.bindHandlers(router, option)
		if err != nil {
			return fmt.Errorf("Bind RESTFul API %s %s ", name, err.Error())
		}
	}

	// bind to server
	srv.Handler = router
	return nil
}
