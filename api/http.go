package api

import (
	"bytes"
	"fmt"
	"io"
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
var registeredOptions = map[string]bool{}

// ProcessGuard guard process
func ProcessGuard(name string, cors ...gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body interface{}
		if c.Request.Body != nil {

			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				if strings.HasPrefix(strings.ToLower(c.Request.Header.Get("Content-Type")), "application/json") {
					jsoniter.Unmarshal(bodyBytes, &body)
				} else {
					body = string(bodyBytes)
				}
			}

			// Reset body
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

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
			if len(cors) > 0 {
				cors[0](c)
			}
			ex := exception.New(err.Error(), 500)
			c.JSON(ex.Code, gin.H{"code": ex.Code, "message": ex.Message})
			c.Abort()
			return
		}

		if sid, has := c.Get("__sid"); has { // Set session id
			if sid, ok := sid.(string); ok {
				process.WithSID(sid)
			}
		}

		if global, has := c.Get("__global"); has { // Set global variables
			if global, ok := global.(map[string]interface{}); ok {
				process.WithGlobal(global)
			}
		}

		err = process.Execute()
		if err != nil {
			ex := exception.New(err.Error(), 500)
			if len(cors) > 0 {
				cors[0](c)
			}
			c.JSON(ex.Code, gin.H{"code": ex.Code, "message": ex.Message})
			c.Abort()
			return
		}
		defer process.Release()

		v := process.Value()
		if data, ok := v.(map[string]interface{}); ok {
			if sid, ok := data["sid"].(string); ok {
				c.Set("__sid", sid)
			}

			if global, ok := data["__global"].(map[string]interface{}); ok {
				c.Set("__global", global)
			}
		}
	}
}

// get the origin
func getOrigin(c *gin.Context) string {
	referer := c.Request.Referer()
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		origin = referer
	}
	return origin
}

// IsAllowed check if the referer is in allow list
func IsAllowed(c *gin.Context, allowsMap map[string]bool) bool {
	origin := getOrigin(c)
	if origin != "" {
		url, err := url.Parse(origin)
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
	registeredOptions = map[string]bool{}
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
		http.setCorsOption(path.Path, allowsMap, router)
		handlers = append(handlers, func(c *gin.Context) {
			origin := getOrigin(c)
			if origin != "" {

				if !IsAllowed(c, allowsMap) {
					c.JSON(403, gin.H{"code": 403, "message": "referer is not allowed. allows: " + strings.Join(allows, ",")})
					c.Abort()
					return
				}

				// url parse
				url, _ := url.Parse(origin)
				origin = fmt.Sprintf("%s://%s", url.Scheme, url.Host)
				// fmt.Println("referer is:", referer)
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Access-Control-Allow-Headers", allowHeaders)
				c.Writer.Header().Set("Access-Control-Allow-Methods", allowMethods)
			}
		})
	}

	// set middlewares
	http.guard(&handlers, path.Guard, http.Guard)

	// set http handler
	if path.Out.Redirect != nil {
		handlers = append(handlers, path.redirectHandler(getArgs))

	} else if path.ProcessHandler {
		handlers = append(handlers, path.processHandler())

	} else if strings.HasPrefix(path.Out.Type, "text/event-stream") {
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

// setCorsOption 跨域许可
func (http HTTP) setCorsOption(path string, allows map[string]bool, router gin.IRoutes) {
	if _, has := registeredOptions[fmt.Sprintf("%s.%s", http.Name, path)]; has {
		return
	}
	registeredOptions[fmt.Sprintf("%s.%s", http.Name, path)] = true
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
			c.Writer.Header().Set("Access-Control-Allow-Headers", allowHeaders)
			c.Writer.Header().Set("Access-Control-Allow-Methods", allowMethods)
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
				bytes, err := io.ReadAll(c.Request.Body)
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

		} else if arg[0] == "$header" && length == 2 {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.GetHeader(arg[1])
			})

		} else if arg[0] == "$file" && length == 2 {
			getValues = append(getValues, func(c *gin.Context) interface{} {

				file, err := c.FormFile(arg[1])
				if err != nil {
					return types.UploadFile{Error: fmt.Sprintf("%s %s", arg[1], err.Error())}
				}

				ext := filepath.Ext(file.Filename)
				dir, err := os.MkdirTemp("", "upload")
				if err != nil {
					return types.UploadFile{Error: fmt.Sprintf("%s %s", arg[1], err.Error())}
				}

				tmpfile, err := os.CreateTemp(dir, fmt.Sprintf("file-*%s", ext))
				if err != nil {
					return types.UploadFile{Error: fmt.Sprintf("%s %s", arg[1], err.Error())}
				}
				defer tmpfile.Close()

				if err := c.SaveUploadedFile(file, tmpfile.Name()); err != nil {
					return types.UploadFile{Error: fmt.Sprintf("%s %s", arg[1], err.Error())}
				}

				uploadFile := types.UploadFile{
					UID:      c.GetHeader("Content-Uid"),
					Range:    c.GetHeader("Content-Range"),
					Sync:     c.GetHeader("Content-Sync") == "true", // sync upload or not
					Name:     file.Filename,
					TempFile: tmpfile.Name(),
					Size:     file.Size,
					Header:   file.Header,
				}
				file = nil
				tmpfile = nil
				return uploadFile
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
