package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	nethttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/any"
	"rogchap.com/v8go"
)

func TestHTTPObjectGet(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	http := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("http", http.ExportObject(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// Get
	v, err := ctx.RunScript(fmt.Sprintf(`
		http.Get("%s/get?foo=bar", {"hello":"world"}, {"Auth": "Test"})
	`, host), "")

	if err != nil {
		t.Fatal(err)
	}
	resp, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	mapv := any.Of(resp).MapStr()
	res := any.Of(mapv.Get("data")).MapStr().Dot()
	assert.Equal(t, float64(200), mapv.Get("status"))
	assert.Equal(t, "bar", res.Get("query.foo[0]"))
	assert.Equal(t, "world", res.Get("query.hello[0]"))
	assert.Equal(t, "Test", res.Get("headers.Auth[0]"))

	// Get Error
	v, err = ctx.RunScript(fmt.Sprintf(`
		http.Get("%s/get?foo=bar", 123, {"Auth": "Test"})
	`, host), "")

	if err != nil {
		t.Fatal(err)
	}
	resp, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	mapv = any.Of(resp).MapStr()
	res = any.Of(mapv.Get("data")).MapStr().Dot()
	assert.Equal(t, float64(400), mapv.Get("status"))
	assert.NotNil(t, mapv.Get("message"))
}

func TestHTTPObjectPost(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	http := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("http", http.ExportObject(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// Post
	v, err := ctx.RunScript(fmt.Sprintf(`
		http.Post("%s/path?foo=bar", {"name": "Lucy"}, null, {"hello":"world"}, {"Auth": "Test"})
	`, host), "")

	if err != nil {
		t.Fatal(err)
	}
	resp, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	mapv := any.Of(resp).MapStr()
	res := any.Of(mapv.Get("data")).MapStr().Dot()
	assert.Equal(t, float64(200), mapv.Get("status"))
	assert.Equal(t, "bar", res.Get("query.foo[0]"))
	assert.Equal(t, "world", res.Get("query.hello[0]"))
	assert.Equal(t, "Test", res.Get("headers.Auth[0]"))
	assert.Equal(t, `{"name":"Lucy"}`, res.Get("payload"))

	// Post File via payload
	root, file := tmpfile(t, "Hello World via payload")
	err = http.SetFileRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	global.Set("http", http.ExportObject(iso))

	v, err = ctx.RunScript(fmt.Sprintf(`
		http.Post("%s/path?foo=bar", 
			"%s", null, 
			{"hello":"world"}, 
			{"Auth": "Test", "Content-Type": "multipart/form-data"}
		)
	`, host, file), "")

	if err != nil {
		t.Fatal(err)
	}
	resp, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	mapv = any.Of(resp).MapStr()
	res = any.Of(mapv.Get("data")).MapStr().Dot()
	assert.Equal(t, float64(200), mapv.Get("status"))
	assert.Equal(t, "bar", res.Get("query.foo[0]"))
	assert.Equal(t, "world", res.Get("query.hello[0]"))
	assert.Equal(t, "Test", res.Get("headers.Auth[0]"))
	assert.Contains(t, res.Get("payload"), "Hello World via payload")

	// Post File via files
	root, file = tmpfile(t, "Hello World via files")
	err = http.SetFileRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	global.Set("http", http.ExportObject(iso))
	v, err = ctx.RunScript(fmt.Sprintf(`
		http.Post("%s/path?foo=bar", 
			{"name": "Lucy"}, 
			{"file": "%s"}, 
			{"hello":"world"}, 
			{"Auth": "Test", "Content-Type": "multipart/form-data"}
		)
	`, host, file), "")

	if err != nil {
		t.Fatal(err)
	}
	resp, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	mapv = any.Of(resp).MapStr()
	res = any.Of(mapv.Get("data")).MapStr().Dot()
	assert.Equal(t, float64(200), mapv.Get("status"))
	assert.Equal(t, "bar", res.Get("query.foo[0]"))
	assert.Equal(t, "world", res.Get("query.hello[0]"))
	assert.Equal(t, "Test", res.Get("headers.Auth[0]"))
	assert.Contains(t, res.Get("payload"), "Hello World via files")
	assert.Contains(t, res.Get("payload"), "Lucy")

	// Post Error
	v, err = ctx.RunScript(fmt.Sprintf(`
		http.Post("%s/path?foo=bar", null, 123, {"Auth": "Test"})
	`, host), "")

	if err != nil {
		t.Fatal(err)
	}
	resp, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	mapv = any.Of(resp).MapStr()
	res = any.Of(mapv.Get("data")).MapStr().Dot()
	assert.Equal(t, float64(400), mapv.Get("status"))
	assert.NotNil(t, mapv.Get("message"))
}

func TestHTTPObjectOthers(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	http := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("http", http.ExportObject(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	methods := []string{"http.Put", "http.Patch", "http.Delete"}
	for _, method := range methods {

		// Success
		v, err := ctx.RunScript(fmt.Sprintf(`
			%s("%s/path?foo=bar", {"name": "Lucy"}, {"hello":"world"}, {"Auth": "Test"})
		`, method, host), "")

		if err != nil {
			t.Fatal(err)
		}
		resp, err := bridge.GoValue(v, ctx)
		if err != nil {
			t.Fatal(err)
		}
		mapv := any.Of(resp).MapStr()
		res := any.Of(mapv.Get("data")).MapStr().Dot()
		assert.Equal(t, float64(200), mapv.Get("status"))
		assert.Equal(t, "bar", res.Get("query.foo[0]"))
		assert.Equal(t, "world", res.Get("query.hello[0]"))
		assert.Equal(t, "Test", res.Get("headers.Auth[0]"))
		assert.Equal(t, `{"name":"Lucy"}`, res.Get("payload"))

		// Error
		v, err = ctx.RunScript(fmt.Sprintf(`
			%s("%s/path?foo=bar", {"name": "Lucy"}, 123, {"Auth": "Test"})
		`, method, host), "")

		if err != nil {
			t.Fatal(err)
		}
		resp, err = bridge.GoValue(v, ctx)
		if err != nil {
			t.Fatal(err)
		}
		mapv = any.Of(resp).MapStr()
		res = any.Of(mapv.Get("data")).MapStr().Dot()
		assert.Equal(t, float64(400), mapv.Get("status"))
		assert.NotNil(t, mapv.Get("message"))
	}
}

func TestHTTPObjectSend(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	http := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("http", http.ExportObject(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// Send
	v, err := ctx.RunScript(fmt.Sprintf(`
		http.Send("POST", "%s/path?foo=bar", {"name": "Lucy"}, {"hello":"world"}, {"Auth": "Test"})
	`, host), "")

	if err != nil {
		t.Fatal(err)
	}
	resp, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	mapv := any.Of(resp).MapStr()
	res := any.Of(mapv.Get("data")).MapStr().Dot()
	assert.Equal(t, float64(200), mapv.Get("status"))
	assert.Equal(t, "bar", res.Get("query.foo[0]"))
	assert.Equal(t, "world", res.Get("query.hello[0]"))
	assert.Equal(t, "Test", res.Get("headers.Auth[0]"))
	assert.Equal(t, `{"name":"Lucy"}`, res.Get("payload"))

	// Send Error
	v, err = ctx.RunScript(fmt.Sprintf(`
		http.Send("POST", "%s/path?foo=bar", {"name": "Lucy"}, 123, {"Auth": "Test"})
	`, host), "")

	if err != nil {
		t.Fatal(err)
	}
	resp, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	mapv = any.Of(resp).MapStr()
	res = any.Of(mapv.Get("data")).MapStr().Dot()
	assert.Equal(t, float64(400), mapv.Get("status"))
	assert.NotNil(t, mapv.Get("message"))
}

func TestHTTPObjectStream(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	http := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("http", http.ExportObject(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// Stream
	v, err := ctx.RunScript(fmt.Sprintf(`
	function call() {
		var res = ""
		http.Stream("GET", "%s/stream", ( data )=>{
			res = res + data
			return 1
		})
		return res
	}
	call()
	`, host), "")

	if err != nil {
		t.Fatal(err)
	}
	resp, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "event:messagedata:0event:messagedata:1event:messagedata:2event:messagedata:3event:messagedata:4", resp)

	// Break
	v, err = ctx.RunScript(fmt.Sprintf(`
	function call() {
		var res = ""
		http.Stream("GET", "%s/stream", ( data )=>{
			res = res + data
			return 0
		})
		return res
	}
	call()
	`, host), "")

	if err != nil {
		t.Fatal(err)
	}
	resp, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "event:message", resp)

	// Error
	v, err = ctx.RunScript(fmt.Sprintf(`
	function call() {
		var res = ""
		return http.Stream("GET", "%s/stream","xxx")
	}
	call()
	`, host), "")

	resp, err = bridge.GoValue(v, ctx)
	mapv := any.Of(resp).MapStr()
	assert.Equal(t, "v8go: value is not a Function", mapv.Get("message"))

}

func setup() (chan bool, chan bool, string) {
	return make(chan bool, 1), make(chan bool, 1), ""
}

func start(t *testing.T, host *string, shutdown, ready chan bool) {

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	errCh := make(chan error, 1)

	// Set router
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard
	router := gin.New()

	router.GET("/get", testHanlder)
	router.HEAD("/head", testHanlder)
	router.POST("/path", testHanlder)
	router.PUT("/path", testHanlder)
	router.PATCH("/path", testHanlder)
	router.DELETE("/path", testHanlder)

	router.GET("/stream", func(c *gin.Context) {
		chanStream := make(chan int, 10)
		go func() {
			defer close(chanStream)
			for i := 0; i < 5; i++ {
				chanStream <- i
				time.Sleep(time.Millisecond * 200)
			}
		}()
		c.Stream(func(w io.Writer) bool {
			if msg, ok := <-chanStream; ok {
				c.SSEvent("message", msg)
				return true
			}
			return false
		})
	})

	// Listen
	l, err := net.Listen("tcp4", ":0")
	if err != nil {
		errCh <- fmt.Errorf("Error: can't get port")
	}

	srv := &nethttp.Server{Addr: ":0", Handler: router}
	defer func() {
		srv.Close()
		l.Close()
	}()

	// start serve
	go func() {
		fmt.Println("[TestServer] Starting")
		if err := srv.Serve(l); err != nil && err != nethttp.ErrServerClosed {
			fmt.Println("[TestServer] Error:", err)
			errCh <- err
		}
	}()

	addr := strings.Split(l.Addr().String(), ":")
	if len(addr) != 2 {
		errCh <- fmt.Errorf("Error: can't get port")
	}

	*host = fmt.Sprintf("http://127.0.0.1:%s", addr[1])
	time.Sleep(50 * time.Millisecond)
	ready <- true
	fmt.Printf("[TestServer] %s", *host)

	select {

	case <-shutdown:
		fmt.Println("[TestServer] Stop")
		break

	case <-interrupt:
		fmt.Println("[TestServer] Interrupt")
		break

	case err := <-errCh:
		fmt.Println("[TestServer] Error:", err.Error())
		break
	}
}

func stop(shutdown, ready chan bool) {
	ready <- false
	shutdown <- true
	time.Sleep(50 * time.Millisecond)
}

func tmpfile(t *testing.T, content string) (string, string) {
	file, err := os.CreateTemp("", "-data")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(file.Name(), []byte(content), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(file.Name()), filepath.Base(file.Name())
}

func testHanlder(c *gin.Context) {
	payload, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error(), "code": 400})
		return
	}
	data := string(payload)
	c.JSON(200, gin.H{
		"payload": data,
		"query":   c.Request.URL.Query(),
		"headers": c.Request.Header,
	})
}
