package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

// HTTPGuards 支持的中间件
var HTTPGuards = map[string]gin.HandlerFunc{}

// ProcessGuard guard process
func ProcessGuard(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body interface{}
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)
		if err == nil {
			if strings.HasPrefix(strings.ToLower(c.Request.Header.Get("Content-Type")), "application/json") {
				jsoniter.Unmarshal(bodyBytes, &body)
			} else {
				body = string(bodyBytes)
			}
		}

		// Reset body
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		params := map[string]string{}
		for _, param := range c.Params {
			params[param.Key] = param.Value
		}

		args := []interface{}{
			c.FullPath(),          // api path
			params,                // api params
			c.Request.URL.Query(), // query string
			body,                  // payload
			c.Request.Header,      // Request headers
		}

		process, err := process.Of(name, args...)
		if err != nil {
			c.JSON(400, gin.H{"code": 400, "message": fmt.Sprintf("Guard: %s %s", name, err.Error())})
			c.Abort()
			return
		}

		if sid, has := c.Get("__sid"); has { // 设定会话ID
			if sid, ok := sid.(string); ok {
				process.WithSID(sid)
			}
		}

		if global, has := c.Get("__global"); has { // 设定全局变量
			if global, ok := global.(map[string]interface{}); ok {
				process.WithGlobal(global)
			}
		}

		process.Run()
	}
}

// IsAllowed check if the referer is in allow list
func IsAllowed(c *gin.Context, allowsMap map[string]bool) bool {
	referer := c.Request.Referer()
	if referer != "" {
		url, err := url.Parse(referer)
		if err != nil {
			return true
		}

		port := fmt.Sprintf(":%s", url.Port())
		if port == ":" || port == ":80" || port == ":443" {
			port = ""
		}
		host := fmt.Sprintf("%s%s", url.Hostname(), port)
		// fmt.Println(url, host, c.Request.Host)
		// fmt.Println(allowsMap)
		if host == c.Request.Host {
			return true
		}
		if _, has := allowsMap[host]; !has {
			return false
		}
	}
	return true

}

// Routes 配置转换为路由
func (http HTTP) Routes(router *gin.Engine, path string, allows ...string) {
	var group gin.IRoutes = router
	if http.Group != "" {
		path = filepath.Join(path, "/", http.Group)
	}
	group = router.Group(path)
	for _, path := range http.Paths {
		path.Method = strings.ToUpper(path.Method)
		http.Route(group, path, allows...)
	}
}

// Route 路径配置转换为路由
func (http HTTP) Route(router gin.IRoutes, path Path, allows ...string) {
	getArgs := http.parseIn(path.In)
	handlers := []gin.HandlerFunc{}

	// 跨域访问
	if allows != nil && len(allows) > 0 {
		allowsMap := map[string]bool{}
		for _, allow := range allows {
			allowsMap[allow] = true
		}

		// Cross domain
		http.crossDomain(path.Path, allowsMap, router)
		handlers = append(handlers, func(c *gin.Context) {
			referer := c.Request.Referer()
			if referer != "" {
				if !IsAllowed(c, allowsMap) {
					c.AbortWithStatus(403)
					return
				}

				// url parse
				url, _ := url.Parse(referer)
				referer = fmt.Sprintf("%s://%s", url.Scheme, url.Host)
				// fmt.Println("referer is:", referer)
				c.Writer.Header().Set("Access-Control-Allow-Origin", referer)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
				c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
			}
		})
	}

	// set middlewares
	http.guard(&handlers, path.Guard, http.Guard)

	// set http handler
	if path.Out.Redirect != nil {
		handlers = append(handlers, path.redirectHandler(getArgs))

	} else if path.Out.Stream != "" {
		handlers = append(handlers, path.streamHandler(getArgs))

	} else {
		handlers = append(handlers, path.defaultHandler(getArgs))
	}

	http.method(path.Method, path.Path, router, handlers...)
}

// 加载特定中间件
func (http HTTP) guard(handlers *[]gin.HandlerFunc, guard string, defaults string) {

	if guard == "" {
		guard = defaults
	}

	if guard != "-" {
		guards := strings.Split(guard, ",")
		for _, name := range guards {
			name = strings.TrimSpace(name)
			if handler, has := HTTPGuards[name]; has {
				*handlers = append(*handlers, handler)
			} else { // run process process
				*handlers = append(*handlers, ProcessGuard(name))
			}
		}
	}
}

// crossDomain 跨域许可
func (http HTTP) crossDomain(path string, allows map[string]bool, router gin.IRoutes) {
	http.method("OPTIONS", path, router, func(c *gin.Context) {
		referer := c.Request.Referer()
		if referer != "" {
			if !IsAllowed(c, allows) {
				c.AbortWithStatus(403)
				return
			}

			// url parse
			url, _ := url.Parse(referer)
			referer = fmt.Sprintf("%s://%s", url.Scheme, url.Host)
			c.Writer.Header().Set("Access-Control-Allow-Origin", referer)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
			c.AbortWithStatus(204)
		}
	})
}

// parseIn 接口传参解析 (这个函数应该重构)
func (http HTTP) parseIn(in []interface{}) func(c *gin.Context) []interface{} {

	getValues := []func(c *gin.Context) interface{}{}
	for _, value := range in {

		v, ok := value.(string)
		if !ok {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return v
			})
			continue
		}

		if v == ":body" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				bytes, err := ioutil.ReadAll(c.Request.Body)
				if err != nil {
					panic(err)
				}
				return string(bytes)
			})
			continue
		} else if v == ":fullpath" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.FullPath()
			})
			continue
		} else if v == ":headers" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.Request.Header
			})
			continue
		} else if v == ":payload" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				value, has := c.Get("__payloads")
				if !has {
					return maps.MapStr{}
				}
				valueMap, ok := value.(map[string]interface{})
				if !ok {
					return maps.MapStr{}
				}
				return valueMap
			})
			continue
		} else if v == ":query" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.Request.URL.Query()
			})
			continue
		} else if v == ":form" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				values := c.Request.PostForm
				return values
			})
			continue
		} else if v == ":params" || v == ":query-param" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				values := c.Request.URL.Query()
				return types.URLToQueryParam(values)
			})
			continue
		} else if v == ":context" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c
			})
			continue
		}

		arg := strings.Split(v, ".")
		length := len(arg)
		if arg[0] == "$form" && length == 2 {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.PostForm(arg[1])
			})
		} else if arg[0] == "$param" && length == 2 {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.Param(arg[1])
			})

		} else if arg[0] == "$query" && length == 2 {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.Query(arg[1])
			})

		} else if arg[0] == "$payload" && length == 2 {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				if payloads, has := c.Get("__payloads"); has {
					if value, has := payloads.(map[string]interface{})[arg[1]]; has {
						return value
					}
				}
				return ""
			})

		} else if arg[0] == "$session" && length == 2 {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				if sid := c.GetString("__sid"); sid != "" {
					name := arg[1]
					return session.Global().ID(sid).MustGet(name)
				}
				return ""
			})

		} else if arg[0] == "$file" && length == 2 {
			getValues = append(getValues, func(c *gin.Context) interface{} {

				file, err := c.FormFile(arg[1])

				ext := filepath.Ext(file.Filename)

				if err != nil {
					exception.New("读取上传文件出错 %s", 500, err).Throw()
				}

				dir, err := ioutil.TempDir(os.TempDir(), "upload")
				if err != nil {
					exception.New("创建临时文件夹 %s", 500, err).Throw()
				}

				tmpfile, err := ioutil.TempFile(dir, fmt.Sprintf("file-*%s", ext))
				if err != nil {
					exception.New("创建临时文件出错 %s", 500, err).Throw()
				}

				err = c.SaveUploadedFile(file, tmpfile.Name())
				if err != nil {
					exception.New("保存文件出错 %s", 500, err).Throw()
				}
				return types.UploadFile{
					Name:     file.Filename,
					TempFile: tmpfile.Name(),
					Size:     file.Size,
					Header:   file.Header,
				}
			})
		} else { // 原始数值
			new := v
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return new
			})
		}
	}

	return func(c *gin.Context) []interface{} {
		values := []interface{}{}
		for _, get := range getValues {
			values = append(values, get(c))
		}
		return values
	}
}

// router 方法设定
func (http HTTP) method(name string, path string, router gin.IRoutes, handlers ...gin.HandlerFunc) {
	switch name {
	case "POST":
		router.POST(path, handlers...)
		return
	case "GET":
		router.GET(path, handlers...)
		return
	case "PUT":
		router.PUT(path, handlers...)
		return
	case "DELETE":
		router.DELETE(path, handlers...)
		return
	case "HEAD":
		router.HEAD(path, handlers...)
		return
	case "ANY":
		router.Any(path, handlers...)
		return
	case "OPTIONS":
		router.OPTIONS(path, handlers...)
		return
	}
}
