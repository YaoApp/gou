package api

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"rogchap.com/v8go"
)

// defaultHandler default handler
func (path Path) defaultHandler(getArgs func(c *gin.Context) []interface{}) func(c *gin.Context) {
	return func(c *gin.Context) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// defer debug.FreeOSMemory()

		path.setPayload(c)
		var status int = path.Out.Status
		var contentType = path.reqContentType(c)

		chRes := make(chan interface{}, 1)
		go path.execProcess(ctx, chRes, c, getArgs)

		select {
		case resp := <-chRes:
			close(chRes)
			if resp == nil {
				c.Done()
				return
			}

			// Set ContentType
			contentType = path.setResponseHeaders(c, resp, contentType)

			// Format Body
			body := resp
			if path.Out.Body != nil {
				res := any.Of(resp)
				if res.IsMap() {
					data := res.Map().MapStrAny.Dot()
					body = helper.Bind(path.Out.Body, data)
				}
			}

			switch data := body.(type) {
			case maps.Map, map[string]interface{}, []interface{}, []maps.Map, []map[string]interface{}:
				c.JSON(status, data)
				c.Done()
				return

			case []byte:
				c.Data(status, contentType, data)
				c.Done()
				return

			case error:
				ex := exception.Err(data, 500)
				c.JSON(ex.Code, gin.H{"message": ex.Message, "code": ex.Code})

			default:
				if strings.HasPrefix(contentType, "application/json") {
					c.JSON(status, body)
					c.Done()
					return
				}

				c.String(status, "%v", body)
				c.Done()
				return
			}

		case <-c.Request.Context().Done():
			c.Abort()
			return
		}
	}
}

func (path Path) processHandler(getArgs func(c *gin.Context) []interface{}) func(c *gin.Context) {
	process := process.New(path.Process)
	res := process.Run()
	handler, ok := res.(func(c *gin.Context))
	if !ok {
		handler = func(c *gin.Context) {
			c.Done()
		}
	}
	return handler
}

// redirectHandler default handler
func (path Path) redirectHandler(getArgs func(c *gin.Context) []interface{}) func(c *gin.Context) {
	return func(c *gin.Context) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		path.setPayload(c)
		contentType := path.reqContentType(c)

		// run process
		resp := path.runProcess(ctx, c, getArgs)

		// Response Headers
		path.setResponseHeaders(c, resp, contentType)

		code := path.Out.Redirect.Code
		if code == 0 {
			code = 301
		}

		c.Redirect(code, path.Out.Redirect.Location)
		c.Done()
	}
}

func (path Path) streamHandler(getArgs func(c *gin.Context) []interface{}) func(c *gin.Context) {
	return func(c *gin.Context) {

		path.setPayload(c)
		path.reqContentType(c)

		chanStream := make(chan ssEventData, 1)
		chanError := make(chan error, 1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			defer func() {
				close(chanStream)
				close(chanError)
			}()

			path.runStreamScript(ctx, c, getArgs,
				// event
				func(name string, message interface{}) {
					chanStream <- ssEventData{Name: name, Message: message}
				},

				// cancel
				func() { cancel() },

				// error
				func(err error) {
					chanError <- err
				},
			)
		}()

		c.Stream(func(w io.Writer) bool {
			select {
			case err := <-chanError:
				if err != nil {
					log.Error("[Stream] %s Error: %v", path.Path, err)
				}
				return false

			case msg := <-chanStream:
				log.Trace("[Stream] %s %s %s %v", path.Path, path.Process, msg.Name, msg.Message)
				c.SSEvent(msg.Name, msg.Message)
				return true

			case <-ctx.Done():
				// log.Info("[DONE] %s", path.Path)
				return false
			}
		})
	}
}

func (path Path) runStreamScript(ctx context.Context, c *gin.Context, getArgs func(c *gin.Context) []interface{}, onEvent func(name string, message interface{}), onCancel func(), onError func(error)) {

	if !strings.HasPrefix(path.Process, "scripts") {
		onError(fmt.Errorf("process must be a script"))
		return
	}

	namer := strings.Split(path.Process, ".")
	method := namer[len(namer)-1]
	scriptID := strings.Join(namer[1:len(namer)-1], ".")
	script, err := v8.Select(scriptID)
	if err != nil {
		onError(err)
		return
	}

	// bind session and global data
	sid := ""
	global := map[string]interface{}{}
	if v, has := c.Get("__sid"); has { // set session id
		if v, ok := v.(string); ok {
			sid = v
		}
	}
	if v, has := c.Get("__global"); has { // set global
		if v, ok := v.(map[string]interface{}); ok {
			global = v
		}
	}

	// make a new script context
	v8ctx, err := script.NewContext(sid, global)
	if err != nil {
		onError(err)
		return
	}
	defer v8ctx.Close()

	v8ctx.WithFunction("ssEvent", func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) != 2 {
			return v8go.Null(info.Context().Isolate())
		}

		name := args[0].String()
		message, err := bridge.GoValue(args[1], info.Context())
		if err != nil {
			return v8go.Null(info.Context().Isolate())
		}

		onEvent(name, message)
		return v8go.Null(info.Context().Isolate())
	})

	v8ctx.WithFunction("cancel", func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		onCancel()
		return v8go.Null(info.Context().Isolate())
	})

	args := getArgs(c)
	_, err = v8ctx.CallWith(ctx, method, args...)
	if err != nil {
		onError(err)
		return
	}
}

func (path Path) execProcess(ctx context.Context, chRes chan<- interface{}, c *gin.Context, getArgs func(c *gin.Context) []interface{}) {

	var args []interface{} = getArgs(c)
	var process, err = process.Of(path.Process, args...)
	if err != nil {
		log.Error("[Path] %s %s", path.Path, err.Error())
		chRes <- err
	}
	defer process.Dispose()

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

	process.WithContext(ctx)
	res, err := process.Exec()
	if err != nil {
		log.Error("[Path] %s %s", path.Path, err.Error())
		chRes <- err
		return
	}
	chRes <- res
}

func (path Path) runProcess(ctx context.Context, c *gin.Context, getArgs func(c *gin.Context) []interface{}) interface{} {
	var args []interface{} = getArgs(c)
	var process = process.New(path.Process, args...)
	defer process.Dispose()

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

	process.WithContext(ctx)
	return process.Run()
}

func (path Path) reqContentType(c *gin.Context) string {
	var contentType string = path.Out.Type
	if path.Out.Type != "" {
		c.Writer.Header().Set("Content-Type", path.Out.Type)
	}
	return contentType
}

func (path Path) setResponseHeaders(c *gin.Context, resp interface{}, contentType string) string {
	if len(path.Out.Headers) > 0 {
		res := any.Of(resp)
		if res.IsMap() { // 处理变量
			data := res.Map().MapStrAny.Dot()
			for name, value := range path.Out.Headers {
				v := helper.Bind(value, data)
				if v != nil {
					c.Writer.Header().Set(name, fmt.Sprintf("%v", v))
				}
			}
		} else {
			for name, value := range path.Out.Headers {
				c.Writer.Header().Set(name, value)
				if name == "Content-Type" {
					contentType = value
				}
			}
		}
	}
	return contentType
}

func (path Path) setPayload(c *gin.Context) {

	if strings.HasPrefix(strings.ToLower(c.GetHeader("content-type")), "application/json") {

		if c.Request.Body == nil {
			c.Set("__payloads", map[string]interface{}{})
			return
		}

		bytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Set("__payloads", map[string]interface{}{})
			log.Error("[Path] %s %s", path.Path, err.Error())
			return
		}

		if bytes == nil || len(bytes) == 0 {
			c.Set("__payloads", map[string]interface{}{})
			return

		}

		payloads := map[string]interface{}{}
		err = jsoniter.Unmarshal(bytes, &payloads)
		if err != nil {
			c.Set("__payloads", map[string]interface{}{})
			log.Error("[Path] %s %s", path.Path, err.Error())
		}
		c.Set("__payloads", payloads)
		c.Request.Body = io.NopCloser(strings.NewReader(string(bytes)))

	}
}
