package gou

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/grpc"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun"
)

// Guards 支持的中间件
var Guards = map[string]gin.HandlerFunc{}

// SetGuards 加载中间件
func SetGuards(guards map[string]gin.HandlerFunc) {
	Guards = guards
}

// AddGuard 添加中间件
func AddGuard(name string, guard gin.HandlerFunc) {
	Guards[name] = guard
}

// Routes 配置转换为路由
func (http HTTP) Routes(router *gin.Engine, root string, allows ...string) {
	var group gin.IRoutes = router
	if http.Group != "" {
		root = path.Join(root, "/", http.Group)
	}
	group = router.Group(root)
	for _, path := range http.Paths {
		http.Route(group, path, allows...)
	}
}

// Route 路径配置转换为路由
func (http HTTP) Route(router gin.IRoutes, path Path, allows ...string) {
	getArgs := http.parseIn(path.In, path.Type)
	handlers := []gin.HandlerFunc{}

	// 跨域访问
	if len(allows) > 0 {
		allowsMap := map[string]bool{}
		for _, allow := range allows {
			allowsMap[allow] = true
		}
		http.crossDomain(path.Path, allowsMap, router)
	}

	// 中间件
	http.guard(&handlers, path.Guard, http.Guard)

	// API响应逻辑
	handlers = append(handlers, func(c *gin.Context) {

		if c.GetHeader("content-type") == "application/json" {
			bytes, err := ioutil.ReadAll(c.Request.Body)
			if err != nil {
				panic(err)
			}

			if bytes == nil || len(bytes) == 0 {
				c.Set("__payloads", map[string]interface{}{})
			} else {
				payloads := map[string]interface{}{}
				err = jsoniter.Unmarshal(bytes, &payloads)
				if err != nil {
					panic(err)
				}
				c.Set("__payloads", payloads)
			}
		}

		// 运行 Process
		var args []interface{} = getArgs(c)
		var resp interface{} = Run(path.Process, args...)
		var status int = path.Out.Status
		var contentType string = path.Out.Type

		if contentType != "" {
			c.Writer.Header().Set("Content-Type", contentType)
		}

		if resp == nil {
			c.Done()
			return
		} else if res, ok := resp.(maps.MapStrAny); ok {
			c.JSON(status, res)
			c.Done()
			return
		} else if res, ok := resp.(*grpc.Response); ok {
			c.String(status, string(res.Bytes))
			c.Done()
			return

		} else if res, ok := resp.(string); ok {
			c.String(status, res)
			c.Done()
			return
		}

	})

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
			if handler, has := Guards[name]; has {
				*handlers = append(*handlers, handler)
			}
		}
	}
}

// crossDomain 跨域许可
func (http HTTP) crossDomain(path string, allows map[string]bool, router gin.IRoutes) {
	http.method("OPTIONS", path, router, func(c *gin.Context) {
		if _, has := allows[c.Request.Host]; !has {
			c.AbortWithStatus(403)
			return
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", c.Request.Host)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
		c.AbortWithStatus(204)
	})
}

// parseIn 接口传参解析
func (http HTTP) parseIn(in []string, typ string) func(c *gin.Context) []interface{} {
	getValues := []func(c *gin.Context) interface{}{}

	for _, v := range in {

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
		} else if v == ":payload" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return xun.MakeRow(c.Get("__payloads"))
			})
			continue
		} else if v == ":query" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.Request.URL.Query()
			})
			continue
		} else if v == ":params" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				values := c.Request.URL.Query()
				return URLToQueryParam(values)
			})
			continue
		} else if v == ":context" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c
			})
			continue
		}

		arg := strings.Split(v, ".")
		if len(arg) == 1 {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return v
			})
			continue
		} else if len(arg) != 2 {
			continue
		}

		if arg[0] == "$form" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.PostForm(arg[1])
			})
		} else if arg[0] == "$param" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.Param(arg[1])
			})

		} else if arg[0] == "$query" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				return c.Query(arg[1])
			})

		} else if arg[0] == "$payload" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				if payloads, has := c.Get("__payloads"); has {
					if value, has := payloads.(map[string]interface{})[arg[1]]; has {
						return value
					}
				}
				return ""
			})

		} else if arg[0] == "$session" {
			getValues = append(getValues, func(c *gin.Context) interface{} {
				val, _ := c.Get(arg[1])
				if val == nil {
					return ""
				}
				return val
			})

		} else if arg[0] == "$file" {
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
				return xun.UploadFile{
					Name:     file.Filename,
					TempFile: tmpfile.Name(),
					Size:     file.Size,
					Header:   file.Header,
				}
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
	case "Any":
		router.Any(path, handlers...)
		return
	case "OPTIONS":
		router.OPTIONS(path, handlers...)
		return
	}
}
