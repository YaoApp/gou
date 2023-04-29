package http

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
)

func TestGet(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	req := New(fmt.Sprintf("%s/get?foo=bar", host))
	res := req.Get()
	data := any.Of(res.Data).MapStr().Dot()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "bar", data.Get("query.foo[0]"))

	req = New(fmt.Sprintf("%s/get?error=1", host))
	res = req.Get()
	assert.Equal(t, 400, res.Code)
	assert.Equal(t, "Error Test", res.Message)

	req = New(fmt.Sprintf("%s/get?null=1", host))
	res = req.Get()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, nil, res.Data)

	params := url.Values{}
	params.Add("foo", "bar")
	req = New(fmt.Sprintf("%s/get", host)).WithQuery(params)
	res = req.Get()
	data = any.Of(res.Data).MapStr().Dot()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "bar", data.Get("query.foo[0]"))

	req = New(fmt.Sprintf("%s/get", host)).WithQuery(params).AddHeader("Auth", "Hello")
	res = req.Get()
	data = any.Of(res.Data).MapStr().Dot()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "bar", data.Get("query.foo[0]"))
	assert.Equal(t, "Hello", data.Get("headers.Auth[0]"))
	assert.Equal(t, "Hello", res.Headers.Get("Auth-Resp"))

	req = New(fmt.Sprintf("%s/get", host)).WithQuery(params).AddHeader("Content-Type", "text/plain")
	res = req.Get()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "It works", fmt.Sprintf("%s", res.Data))
}

func TestPost(t *testing.T) {
	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	req := New(fmt.Sprintf("%s/post?null=1", host))
	res := req.Post(nil)
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, nil, res.Data)

	req = New(fmt.Sprintf("%s/post", host))
	res = req.Post(nil)
	data := any.Of(res.Data).MapStr().Dot()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, nil, data.Get("payload"))

	req = New(fmt.Sprintf("%s/post", host))
	res = req.Post(map[string]interface{}{"foo": "bar"})
	data = any.Of(res.Data).MapStr().Dot()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "bar", data.Get("payload.foo"))

	req = New(fmt.Sprintf("%s/post?urlencoded=1", host)).AddHeader("Content-Type", "application/x-www-form-urlencoded")
	res = req.Post(map[string]interface{}{"foo": "bar"})
	data = any.Of(res.Data).MapStr().Dot()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "bar", data.Get("form"))

	req = New(fmt.Sprintf("%s/post?formdata=1", host)).AddHeader("Content-Type", "multipart/form-data")
	res = req.Post(map[string]interface{}{"foo": "bar", "hello": "world"})
	data = any.Of(res.Data).MapStr().Dot()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "bar", data.Get("form"))

	req = New(fmt.Sprintf("%s/post?file=1", host)).AddHeader("Content-Type", "multipart/form-data")
	res = req.Post(tmpfile(t, "Hello World"))
	data = any.Of(res.Data).MapStr().Dot()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "Hello World", data.Get("file"))

	req = New(fmt.Sprintf("%s/post?files=1", host)).
		AddFile("f1", tmpfile(t, "T1")).
		AddFile("f2", tmpfile(t, "T2"))
	res = req.Post(nil)
	data = any.Of(res.Data).MapStr().Dot()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "T1", data.Get("f1"))
	assert.Equal(t, "T2", data.Get("f2"))

	req = New(fmt.Sprintf("%s/post?files=1", host)).
		AddFile("f1", tmpfile(t, "T1")).
		AddFile("f2", tmpfile(t, "T2"))
	res = req.Post(map[string]interface{}{"foo": "bar"})
	data = any.Of(res.Data).MapStr().Dot()
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "T1", data.Get("f1"))
	assert.Equal(t, "T2", data.Get("f2"))
	assert.Equal(t, "bar", data.Get("foo"))
}

func TestOthers(t *testing.T) {
	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	req := New(fmt.Sprintf("%s/put", host))
	res := req.Put(nil)
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, nil, res.Data)

	req = New(fmt.Sprintf("%s/patch", host))
	res = req.Patch(nil)
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, nil, res.Data)

	req = New(fmt.Sprintf("%s/delete", host))
	res = req.Delete(nil)
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, nil, res.Data)

	req = New(fmt.Sprintf("%s/head", host))
	res = req.Head(nil)
	assert.Equal(t, 302, res.Code)
	assert.Equal(t, nil, res.Data)
}

func TestStream(t *testing.T) {
	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready
	res := []byte{}
	req := New(fmt.Sprintf("%s/stream", host))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req.Stream(ctx, "GET", nil, func(data []byte) int {
		res = append(res, data...)
		return 1
	})
	assert.Equal(t, "event:messagedata:0event:messagedata:1event:messagedata:2event:messagedata:3event:messagedata:4", string(res))

	// test break
	res = []byte{}
	req.Stream(ctx, "GET", nil, func(data []byte) int {
		res = append(res, data...)
		return 0
	})
	assert.Equal(t, "event:message", string(res))

	// test cancel
	res = []byte{}
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := req.Stream(ctx, "GET", nil, func(data []byte) int {
		res = append(res, data...)
		return 1
	})
	assert.Equal(t, "context canceled", err.Error())
}

func tmpfile(t *testing.T, content string) string {
	file, err := os.CreateTemp("", "-data")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(file.Name(), []byte(content), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	return file.Name()
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

	router.GET("/get", testGet)
	router.POST("/post", testPost)
	router.PUT("/put", func(c *gin.Context) { c.Status(200) })
	router.PATCH("/patch", func(c *gin.Context) { c.Status(200) })
	router.DELETE("/delete", func(c *gin.Context) { c.Status(200) })
	router.HEAD("/head", func(c *gin.Context) { c.Status(302) })
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

	srv := &http.Server{Addr: ":0", Handler: router}
	defer func() {
		srv.Close()
		l.Close()
	}()

	// start serve
	go func() {
		fmt.Println("[TestServer] Starting")
		if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
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

func testGet(c *gin.Context) {

	if c.Query("error") == "1" {
		c.JSON(400, gin.H{"code": 400, "message": "Error Test"})
		c.Abort()
		return
	}

	if c.Query("null") == "1" {
		c.Status(200)
		c.Done()
		return
	}

	if len(c.Request.Header["Auth"]) > 0 {
		c.Header("Auth-Resp", c.Request.Header["Auth"][0])
	}

	if len(c.Request.Header["Content-Type"]) > 0 && c.Request.Header["Content-Type"][0] == "text/plain" {
		c.Writer.Write([]byte("It works"))
		c.Done()
		return
	}

	c.JSON(200, gin.H{
		"query":   c.Request.URL.Query(),
		"headers": c.Request.Header,
	})
}

func testPost(c *gin.Context) {

	if c.Query("null") == "1" {
		c.Status(200)
		c.Done()
		return
	}

	if c.Query("urlencoded") == "1" {

		var f struct {
			Foo string `form:"foo" binding:"required"`
		}
		c.Bind(&f)
		c.JSON(200, gin.H{
			"form": f.Foo,
		})
		c.Done()
		return
	}

	if c.Query("formdata") == "1" {
		c.JSON(200, gin.H{"form": c.PostForm("foo")})
		c.Done()
		return
	}

	if c.Query("file") == "1" {

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(402, gin.H{"message": err.Error(), "code": 400})
			return
		}

		fd, err := file.Open()
		if err != nil {
			fmt.Println(err)
			c.JSON(402, gin.H{"message": err.Error(), "code": 400})
			return
		}
		defer fd.Close()

		data, err := io.ReadAll(fd)
		if err != nil {
			fmt.Println(err)
			c.JSON(402, gin.H{"message": err.Error(), "code": 400})
			return
		}

		c.JSON(200, gin.H{"file": string(data)})
		c.Done()
		return
	}

	if c.Query("files") == "1" {

		res := gin.H{
			"foo": c.PostForm("foo"),
		}

		for _, name := range []string{"f1", "f2"} {

			file, err := c.FormFile(name)
			if err != nil {
				fmt.Println(err)
				c.JSON(402, gin.H{"message": err.Error(), "code": 400})
				return
			}

			fd, err := file.Open()
			if err != nil {
				fmt.Println(err)
				c.JSON(402, gin.H{"message": err.Error(), "code": 400})
				return
			}
			defer fd.Close()

			data, err := io.ReadAll(fd)
			if err != nil {
				fmt.Println(err)
				c.JSON(402, gin.H{"message": err.Error(), "code": 400})
				return
			}

			res[name] = string(data)
		}

		c.JSON(200, res)
		c.Done()
		return
	}

	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error(), "code": 400})
		return
	}

	var payload interface{}
	if data != nil && len(data) > 0 {
		err = jsoniter.Unmarshal(data, &payload)
		if err != nil {
			c.JSON(400, gin.H{"message": err.Error(), "code": 400})
			return
		}
	}
	c.JSON(200, gin.H{"payload": payload})
	c.Done()
}
