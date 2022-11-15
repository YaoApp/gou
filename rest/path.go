package rest

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

var getHandlers = map[string]func(c *gin.Context, name string) interface{}{
	":body": func(c *gin.Context, name string) interface{} {
		bytes, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			exception.New("can't read :body data. %s", 500, err).Throw()
		}
		return string(bytes)
	},
	":payload": func(c *gin.Context, name string) interface{} {
		value, has := c.Get("__payloads")
		if !has {
			return map[string]interface{}{}
		}
		return value
	},
	":param": func(c *gin.Context, name string) interface{} {
		params := map[string]string{}
		for _, param := range c.Params {
			params[param.Key] = param.Value
		}
		return params
	},
	":fullpath":    func(c *gin.Context, name string) interface{} { return c.FullPath() },
	":header":      func(c *gin.Context, name string) interface{} { return c.Request.Header },
	":query":       func(c *gin.Context, name string) interface{} { return c.Request.URL.Query() },
	":form":        func(c *gin.Context, name string) interface{} { return c.Request.PostForm },
	":query-param": func(c *gin.Context, name string) interface{} { return gou.URLToQueryParam(c.Request.URL.Query()) },
	":context":     func(c *gin.Context, name string) interface{} { return c },
	"$query":       func(c *gin.Context, name string) interface{} { return c.Query(name) },
	"$payload": func(c *gin.Context, name string) interface{} {
		payloads, has := c.Get("__payloads")
		if !has {
			return nil
		}
		if value, has := payloads.(map[string]interface{})[name]; has {
			return value
		}
		return nil
	},
	"$form":  func(c *gin.Context, name string) interface{} { return c.PostForm(name) },
	"$param": func(c *gin.Context, name string) interface{} { return c.Param(name) },
	"$session": func(c *gin.Context, name string) interface{} {
		if sid := c.GetString("__sid"); sid != "" {
			return session.Global().ID(sid).MustGet(name)
		}
		return nil
	},
	"$file": func(c *gin.Context, name string) interface{} {
		file, err := c.FormFile(name)
		ext := filepath.Ext(file.Filename)
		if err != nil {
			exception.New("Can't read upload file %s", 500, err).Throw()
		}

		dir, err := ioutil.TempDir(os.TempDir(), "upload")
		if err != nil {
			exception.New("Can't create temp dir %s", 500, err).Throw()
		}

		tmpfile, err := ioutil.TempFile(dir, fmt.Sprintf("file-*%s", ext))
		if err != nil {
			exception.New("Can't create temp file %s", 500, err).Throw()
		}

		err = c.SaveUploadedFile(file, tmpfile.Name())
		if err != nil {
			exception.New("Can't save temp file %s", 500, err).Throw()
		}
		return gou.UploadFile{
			Name:     file.Filename,
			TempFile: tmpfile.Name(),
			Size:     file.Size,
			Header:   file.Header,
		}
	},
}

// Handlers set the path handler
func (path Path) setHandlers(router gin.IRoutes, guard string) error {
	handlers, err := path.handlers(router, guard)
	if err != nil {
		return err
	}
	return path.set(router, handlers...)
}

// handlers get the path handlers
func (path Path) handlers(router gin.IRoutes, guard string) ([]gin.HandlerFunc, error) {
	get := path.input()
	handlers := []gin.HandlerFunc{}

	// Set Guard

	// API Content
	handlers = append(handlers, func(c *gin.Context) {

		defer c.Done()

		// parse payload
		if strings.HasPrefix(strings.ToLower(c.GetHeader("content-type")), "application/json") {
			bytes, err := ioutil.ReadAll(c.Request.Body)
			if err != nil {
				exception.New("can't read :body data. %s", 500, err).Throw()
			}
			payloads := map[string]interface{}{}
			if bytes != nil && len(bytes) > 0 {
				err = jsoniter.Unmarshal(bytes, &payloads)
				if err != nil {
					exception.New("can't parse :body data. %s", 500, err).Throw()
				}
			}
			c.Set("__payloads", payloads)
		}

		// Exec Process
		args := get(c)
		var process = gou.NewProcess(path.Process, args...)
		if sid, has := c.Get("__sid"); has { // set sid
			if sid, ok := sid.(string); ok {
				process.WithSID(sid)
			}
		}
		if global, has := c.Get("__global"); has { // set __global vars
			if global, ok := global.(map[string]interface{}); ok {
				process.WithGlobal(global)
			}
		}

		// response
		var resp interface{} = process.Run()
		var status int = path.Out.Status
		var contentType string = path.Out.Type

		if contentType != "" {
			c.Writer.Header().Set("Content-Type", contentType)
		}

		// Response Headers
		if len(path.Out.Headers) > 0 {
			res := any.Of(resp)
			if res.IsMap() { // 处理变量
				data := res.Map().MapStrAny.Dot()
				for name, value := range path.Out.Headers {
					v := share.Bind(value, data)
					if v != nil {
						c.Writer.Header().Set(name, fmt.Sprintf("%v", v))
					}
				}
			} else {
				for name, value := range path.Out.Headers {
					c.Writer.Header().Set(name, value)
				}
			}
		}

		if resp == nil {
			return
		}

		switch resp.(type) {
		case maps.Map, map[string]interface{}, []interface{}, []maps.Map, []map[string]interface{}:
			c.JSON(status, resp)
			return
		default:
			if contentType == "application/json" {
				c.JSON(status, resp)
			} else {
				c.String(status, "%v", resp)
			}
			return
		}
	})

	return handlers, nil
}

func (path Path) input() func(c *gin.Context) []interface{} {
	var gets = []In{}
	for _, in := range path.In {
		inStr, ok := in.(string)
		if ok {
			args := strings.Split(inStr, ".") // [":query"] ["$query", "name"]
			vartype := args[0]                // :query, $query
			varname := args[0][1:]            // query, name
			if len(args) > 1 {
				varname = strings.Join(args[1:], ".")
			}

			if get, has := getHandlers[vartype]; has {
				gets = append(gets, In{handler: get, varname: varname})
				continue
			}
		}
		// Default var
		gets = append(gets, In{handler: func(c *gin.Context, name string) interface{} { return in }, varname: ""})
	}

	return func(c *gin.Context) []interface{} {
		values := []interface{}{}
		for _, in := range gets {
			values = append(values, in.handler(c, in.varname))
		}
		return values
	}
}

// set handlers
func (path Path) set(router gin.IRoutes, handlers ...gin.HandlerFunc) error {
	method := strings.TrimSpace(strings.ToUpper(path.Method))
	switch method {
	case "POST":
		router.POST(path.Path, handlers...)
		return nil
	case "GET":
		router.GET(path.Path, handlers...)
		return nil
	case "PUT":
		router.PUT(path.Path, handlers...)
		return nil
	case "DELETE":
		router.DELETE(path.Path, handlers...)
		return nil
	case "HEAD":
		router.HEAD(path.Path, handlers...)
		return nil
	case "ANY":
		router.Any(path.Path, handlers...)
		return nil
	case "PATCH":
		router.PATCH(path.Path, handlers...)
		return nil
	case "OPTIONS":
		router.OPTIONS(path.Path, handlers...)
		return nil
	}

	return fmt.Errorf("Method %s does not found", path.Method)
}
