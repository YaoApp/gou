package http

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
)

func TestHTTPGet(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	v := process.New("Get", fmt.Sprintf("%s/get?foo=bar", host),
		map[string]string{"hello": "world"},
		map[string]string{"Auth": "Test"},
	).Run()

	resp, ok := v.(*Response)
	if !ok {
		t.Fatal(fmt.Errorf("response error %#v", v))
	}
	assert.Equal(t, 200, resp.Status)
	res := any.Of(resp.Data).MapStr().Dot()
	assert.Equal(t, "bar", res.Get("query.foo[0]"))
	assert.Equal(t, "world", res.Get("query.hello[0]"))
	assert.Equal(t, "Test", res.Get("headers.Auth[0]"))

	v = process.New("Get", fmt.Sprintf("%s/get?foo=bar", host),
		map[int]int{1: 2},
		map[string]string{"Auth": "Test"},
	).Run()
	resp, ok = v.(*Response)
	if !ok {
		t.Fatal(fmt.Errorf("response error %#v", v))
	}
	assert.Equal(t, 400, resp.Status)
	assert.NotNil(t, resp.Message)
}

func TestHTTPHead(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	v := process.New("Head", fmt.Sprintf("%s/head?foo=bar", host),
		map[string]string{"name": "Lucy"},
		map[string]string{"hello": "world"},
		map[string]string{"Auth": "Test"},
	).Run()

	resp, ok := v.(*Response)
	if !ok {
		t.Fatal(fmt.Errorf("response error %#v", v))
	}
	assert.Equal(t, 200, resp.Status)

	v = process.New("Head", fmt.Sprintf("%s/head?foo=bar", host),
		map[string]string{"name": "Lucy"},
		map[int]int{1: 2},
		map[string]string{"Auth": "Test"},
	).Run()

	resp, ok = v.(*Response)
	if !ok {
		t.Fatal(fmt.Errorf("response error %#v", v))
	}
	assert.Equal(t, 400, resp.Status)
	assert.NotNil(t, resp.Message)
}

func TestHTTPPost(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready
	v := process.New("Post", fmt.Sprintf("%s/path?foo=bar", host),
		map[string]string{"name": "Lucy"},
		nil,
		map[string]string{"hello": "world"},
		map[string]string{"Auth": "Test"},
	).Run()

	resp, ok := v.(*Response)
	if !ok {
		t.Fatal(fmt.Errorf("response error %#v", v))
	}
	assert.Equal(t, 200, resp.Status)
	res := any.Of(resp.Data).MapStr().Dot()
	assert.Equal(t, "bar", res.Get("query.foo[0]"))
	assert.Equal(t, "world", res.Get("query.hello[0]"))
	assert.Equal(t, "Test", res.Get("headers.Auth[0]"))
	assert.Equal(t, `{"name":"Lucy"}`, res.Get("payload"))

	// Post File via payload
	root, file := processTmpfile(t, "Hello World via payload")
	err := SetFileRoot(root)
	if err != nil {
		t.Fatal(err)
	}

	v = process.New("Post", fmt.Sprintf("%s/path?foo=bar", host),
		file,
		nil,
		map[string]string{"hello": "world"},
		map[string]string{"Auth": "Test", "Content-Type": "multipart/form-data"},
	).Run()

	resp, ok = v.(*Response)
	if !ok {
		t.Fatal(fmt.Errorf("response error %#v", v))
	}

	assert.Equal(t, 200, resp.Status)
	res = any.Of(resp.Data).MapStr().Dot()
	assert.Equal(t, "bar", res.Get("query.foo[0]"))
	assert.Equal(t, "world", res.Get("query.hello[0]"))
	assert.Equal(t, "Test", res.Get("headers.Auth[0]"))
	assert.Contains(t, res.Get("payload"), "Hello World via payload")

	// Post File via files
	root, file = processTmpfile(t, "Hello World via files")
	err = SetFileRoot(root)
	if err != nil {
		t.Fatal(err)
	}

	v = process.New("Post", fmt.Sprintf("%s/path?foo=bar", host),
		map[string]string{"name": "Lucy"},
		map[string]interface{}{"file": file},
		map[string]string{"hello": "world"},
		map[string]string{"Auth": "Test", "Content-Type": "multipart/form-data"},
	).Run()

	resp, ok = v.(*Response)
	if !ok {
		t.Fatal(fmt.Errorf("response error %#v", v))
	}

	assert.Equal(t, 200, resp.Status)
	res = any.Of(resp.Data).MapStr().Dot()
	assert.Equal(t, "bar", res.Get("query.foo[0]"))
	assert.Equal(t, "world", res.Get("query.hello[0]"))
	assert.Equal(t, "Test", res.Get("headers.Auth[0]"))
	assert.Contains(t, res.Get("payload"), "Hello World via files")
	assert.Contains(t, res.Get("payload"), "Lucy")

	// Post Error
	v = process.New("Post", fmt.Sprintf("%s/path?foo=bar", host),
		map[string]string{"name": "Lucy"},
		nil,
		map[int]int{1: 2},
		map[string]string{"Auth": "Test"},
	).Run()

	resp, ok = v.(*Response)
	if !ok {
		t.Fatal(fmt.Errorf("response error %#v", v))
	}
	assert.Equal(t, 400, resp.Status)
	assert.NotNil(t, resp.Message)
}

func TestHTTPOthers(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	methods := []string{"Put", "Patch", "Delete"}
	for _, method := range methods {
		v := process.New(method, fmt.Sprintf("%s/path?foo=bar", host),
			map[string]string{"name": "Lucy"},
			map[string]string{"hello": "world"},
			map[string]string{"Auth": "Test"},
		).Run()

		resp, ok := v.(*Response)
		if !ok {
			t.Fatal(fmt.Errorf("response error %#v", v))
		}
		assert.Equal(t, 200, resp.Status)
		res := any.Of(resp.Data).MapStr().Dot()
		assert.Equal(t, "bar", res.Get("query.foo[0]"))
		assert.Equal(t, "world", res.Get("query.hello[0]"))
		assert.Equal(t, "Test", res.Get("headers.Auth[0]"))
		assert.Equal(t, `{"name":"Lucy"}`, res.Get("payload"))

		v = process.New(method, fmt.Sprintf("%s/path?foo=bar", host),
			map[string]string{"name": "Lucy"},
			map[int]int{1: 2},
			map[string]string{"Auth": "Test"},
		).Run()

		resp, ok = v.(*Response)
		if !ok {
			t.Fatal(fmt.Errorf("response error %#v", v))
		}
		assert.Equal(t, 400, resp.Status)
		assert.NotNil(t, resp.Message)
	}
}

func TestHTTPSend(t *testing.T) {

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	v := process.New("Send", "POST", fmt.Sprintf("%s/path?foo=bar", host),
		map[string]string{"name": "Lucy"},
		map[string]string{"hello": "world"},
		map[string]string{"Auth": "Test"},
	).Run()

	resp, ok := v.(*Response)
	if !ok {
		t.Fatal(fmt.Errorf("response error %#v", v))
	}
	if resp.Status == 0 {
		fmt.Println(resp.Message)
	}

	assert.Equal(t, 200, resp.Status)
	res := any.Of(resp.Data).MapStr().Dot()
	assert.Equal(t, "bar", res.Get("query.foo[0]"))
	assert.Equal(t, "world", res.Get("query.hello[0]"))
	assert.Equal(t, "Test", res.Get("headers.Auth[0]"))
	assert.Equal(t, `{"name":"Lucy"}`, res.Get("payload"))

	v = process.New("Send", "POST", fmt.Sprintf("%s/path?foo=bar", host),
		map[string]string{"name": "Lucy"},
		map[int]int{1: 2},
		map[string]string{"Auth": "Test"},
	).Run()

	resp, ok = v.(*Response)
	if !ok {
		t.Fatal(fmt.Errorf("response error %#v", v))
	}
	assert.Equal(t, 400, resp.Status)
	assert.NotNil(t, resp.Message)
}

func processTmpfile(t *testing.T, content string) (string, string) {
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
	c.JSON(200, gin.H{
		"payload": string(payload),
		"query":   c.Request.URL.Query(),
		"headers": c.Request.Header,
	})
}
