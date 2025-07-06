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
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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
	if len(data) > 0 {
		err = jsoniter.Unmarshal(data, &payload)
		if err != nil {
			c.JSON(400, gin.H{"message": err.Error(), "code": 400})
			return
		}
	}
	c.JSON(200, gin.H{"payload": payload})
	c.Done()
}

// =============================================================================
// Extended Unit Tests for Better Coverage
// =============================================================================

func TestRequestBuilderMethods(t *testing.T) {
	req := New("http://example.com")

	// Test header methods
	req.AddHeader("X-Test", "value1")
	assert.Equal(t, "value1", req.GetHeader("X-Test"))
	assert.True(t, req.HasHeader("X-Test"))

	req.SetHeader("X-Test", "value2")
	assert.Equal(t, "value2", req.GetHeader("X-Test"))

	req.DelHeader("X-Test")
	assert.False(t, req.HasHeader("X-Test"))
	assert.Equal(t, "", req.GetHeader("X-Test"))

	// Test WithHeader
	headers := http.Header{}
	headers.Set("Authorization", "Bearer token")
	req.WithHeader(headers)
	assert.Equal(t, "Bearer token", req.GetHeader("Authorization"))

	// Test WithQuery
	params := url.Values{}
	params.Add("key", "value")
	req.WithQuery(params)

	// Test WithContext
	ctx := context.Background()
	req.WithContext(ctx)
}

func TestFileUploadMethods(t *testing.T) {
	req := New("http://example.com")

	// Test AddFile
	tmpFile := tmpfile(t, "test content")
	defer os.Remove(tmpFile)

	req.AddFile("testfile", tmpFile)
	assert.Equal(t, "multipart/form-data", req.GetHeader("Content-Type"))

	// Test AddFileBytes
	req.AddFileBytes("bytefile", "test.txt", []byte("byte content"))
	assert.Equal(t, "multipart/form-data", req.GetHeader("Content-Type"))
}

func TestResponseError(t *testing.T) {
	resp := ResponseError(500, "Internal Server Error")
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, 0, resp.Status)
	assert.Equal(t, "Internal Server Error", resp.Message)
	assert.NotNil(t, resp.Headers)
	assert.Nil(t, resp.Data)
}

func TestContentTypeDetection(t *testing.T) {
	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	// Test JSON content type (default behavior)
	req := New(fmt.Sprintf("%s/post", host))
	res := req.Post(map[string]interface{}{"test": "json"})
	assert.Equal(t, 200, res.Code)

	// Test with explicit JSON content type
	req = New(fmt.Sprintf("%s/post", host))
	req.SetHeader("Content-Type", "application/json; charset=utf-8")
	res = req.Post(map[string]interface{}{"test": "json"})
	assert.Equal(t, 200, res.Code)

	// Test URL encoded form data
	req = New(fmt.Sprintf("%s/post?urlencoded=1", host))
	req.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	res = req.Post(map[string]interface{}{"foo": "bar"})
	assert.Equal(t, 200, res.Code)

	// Test multipart form data
	req = New(fmt.Sprintf("%s/post?formdata=1", host))
	req.SetHeader("Content-Type", "multipart/form-data")
	res = req.Post(map[string]interface{}{"foo": "bar"})
	assert.Equal(t, 200, res.Code)
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestConcurrentRequests(t *testing.T) {
	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	numWorkers := 20
	requestsPerWorker := 10
	var wg sync.WaitGroup
	var successCount, errorCount int64

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < requestsPerWorker; j++ {
				// Mix different types of requests
				switch j % 4 {
				case 0:
					req := New(fmt.Sprintf("%s/get?worker=%d&req=%d", host, workerID, j))
					res := req.Get()
					if res.Code == 200 {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				case 1:
					req := New(fmt.Sprintf("%s/post", host))
					res := req.Post(map[string]interface{}{
						"worker": workerID,
						"req":    j,
					})
					if res.Code == 200 {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				case 2:
					req := New(fmt.Sprintf("%s/put", host))
					res := req.Put(nil)
					if res.Code == 200 {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				case 3:
					req := New(fmt.Sprintf("%s/delete", host))
					res := req.Delete(nil)
					if res.Code == 200 {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	totalRequests := int64(numWorkers * requestsPerWorker)
	t.Logf("Concurrent test completed: %d successful, %d failed out of %d total requests",
		successCount, errorCount, totalRequests)

	// Allow some tolerance for network issues
	successRate := float64(successCount) / float64(totalRequests)
	assert.Greater(t, successRate, 0.95, "Success rate should be > 95%%")
}

func TestConcurrentStreamRequests(t *testing.T) {
	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	numWorkers := 5
	var wg sync.WaitGroup
	var successCount, errorCount int64

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := New(fmt.Sprintf("%s/stream", host))
			receivedData := make([]byte, 0)

			err := req.Stream(ctx, "GET", nil, func(data []byte) int {
				receivedData = append(receivedData, data...)
				return 1
			})

			if err == nil && len(receivedData) > 0 {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&errorCount, 1)
				if err != nil {
					t.Logf("Worker %d stream error: %v", workerID, err)
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Concurrent stream test completed: %d successful, %d failed", successCount, errorCount)
	assert.Greater(t, successCount, int64(0), "At least one stream should succeed")
}

func TestConcurrentFileUploads(t *testing.T) {
	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	numWorkers := 10
	var wg sync.WaitGroup
	var successCount, errorCount int64

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			content := fmt.Sprintf("File content from worker %d", workerID)
			tmpFile := tmpfile(t, content)
			defer os.Remove(tmpFile)

			req := New(fmt.Sprintf("%s/post?file=1", host)).
				AddHeader("Content-Type", "multipart/form-data")
			res := req.Post(tmpFile)

			if res.Code == 200 {
				data := any.Of(res.Data).MapStr().Dot()
				if data.Get("file") == content {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			} else {
				atomic.AddInt64(&errorCount, 1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Concurrent file upload test completed: %d successful, %d failed", successCount, errorCount)
	assert.Greater(t, successCount, int64(numWorkers/2), "At least half of uploads should succeed")
}

// =============================================================================
// Stress Tests
// =============================================================================

func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	// High load test
	numWorkers := 50
	requestsPerWorker := 20
	duration := 10 * time.Second

	var wg sync.WaitGroup
	var requestCount, successCount, errorCount int64
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < requestsPerWorker; j++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				atomic.AddInt64(&requestCount, 1)

				req := New(fmt.Sprintf("%s/get?stress=1&worker=%d&req=%d", host, workerID, j))
				res := req.Get()

				if res.Code == 200 {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}

				// Small delay to avoid overwhelming the server
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	rps := float64(requestCount) / elapsed.Seconds()
	successRate := float64(successCount) / float64(requestCount)

	t.Logf("Stress test results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Total requests: %d", requestCount)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Failed: %d", errorCount)
	t.Logf("  Success rate: %.2f%%", successRate*100)
	t.Logf("  Requests per second: %.2f", rps)

	assert.Greater(t, successRate, 0.90, "Success rate should be > 90%% under stress")
	assert.Greater(t, rps, 10.0, "Should handle at least 10 requests per second")
}

// =============================================================================
// Precise Goroutine Leak Detection Tests
// =============================================================================

// GoroutineInfo represents information about a goroutine
type GoroutineInfo struct {
	ID       int
	State    string
	Function string
	Stack    string
	IsSystem bool
}

// parseGoroutineStack parses goroutine stack trace and extracts information
func parseGoroutineStack(stackTrace string) []GoroutineInfo {
	lines := strings.Split(stackTrace, "\n")
	var goroutines []GoroutineInfo
	var current *GoroutineInfo

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse goroutine header: "goroutine 123 [running]:"
		if strings.HasPrefix(line, "goroutine ") && strings.HasSuffix(line, ":") {
			if current != nil {
				goroutines = append(goroutines, *current)
			}

			// Extract goroutine ID and state
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				idStr := parts[1]
				stateStr := strings.Trim(parts[2], "[]:")

				current = &GoroutineInfo{
					State: stateStr,
					Stack: line,
				}

				// Parse ID
				if id := parseInt(idStr); id > 0 {
					current.ID = id
				}
			}
			continue
		}

		// Parse function call
		if current != nil && strings.Contains(line, "(") {
			if current.Function == "" {
				current.Function = line
				// Determine if it's a system goroutine
				current.IsSystem = isSystemGoroutine(line)
			}
			current.Stack += "\n" + line
		}

		// Add context lines
		if current != nil && i < len(lines)-1 {
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine != "" && !strings.HasPrefix(nextLine, "goroutine ") {
				current.Stack += "\n" + line
			}
		}
	}

	if current != nil {
		goroutines = append(goroutines, *current)
	}

	return goroutines
}

// parseInt safely parses integer from string
func parseInt(s string) int {
	result := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		} else {
			break
		}
	}
	return result
}

// isSystemGoroutine determines if a goroutine is system-provided
func isSystemGoroutine(function string) bool {
	systemPatterns := []string{
		"runtime.",
		"testing.",
		"os/signal.",
		"net/http.(*Server).", // HTTP server goroutines
		"net/http.(*conn).",   // HTTP server connection handling
		"net.(*netFD).",       // Network file descriptor operations
		"internal/poll.",      // Network polling operations
		"crypto/tls.",         // TLS operations
	}

	// HTTP client persistent connection goroutines are NOT system goroutines
	// They should be cleaned up properly by the HTTP client
	clientPatterns := []string{
		"net/http.(*persistConn).", // HTTP client persistent connections - NOT system
		"net/http.(*Transport).",   // HTTP client transport - NOT system
	}

	// Check if it's a client goroutine first (these are NOT system)
	for _, pattern := range clientPatterns {
		if strings.Contains(function, pattern) {
			return false // Explicitly mark as application goroutine
		}
	}

	// Check system patterns
	for _, pattern := range systemPatterns {
		if strings.Contains(function, pattern) {
			return true
		}
	}
	return false
}

// analyzeGoroutineLeaks provides detailed analysis of goroutine changes
func analyzeGoroutineLeaks(before, after []GoroutineInfo) (leaked, cleaned []GoroutineInfo) {
	beforeMap := make(map[int]GoroutineInfo)
	for _, g := range before {
		beforeMap[g.ID] = g
	}

	afterMap := make(map[int]GoroutineInfo)
	for _, g := range after {
		afterMap[g.ID] = g
	}

	// Find new goroutines (potential leaks)
	for id, g := range afterMap {
		if _, exists := beforeMap[id]; !exists {
			leaked = append(leaked, g)
		}
	}

	// Find cleaned up goroutines
	for id, g := range beforeMap {
		if _, exists := afterMap[id]; !exists {
			cleaned = append(cleaned, g)
		}
	}

	return leaked, cleaned
}

func TestPreciseGoroutineLeakDetection(t *testing.T) {
	// Get initial goroutine state
	initialStack := make([]byte, 64*1024)
	n := runtime.Stack(initialStack, true)
	initialGoroutines := parseGoroutineStack(string(initialStack[:n]))

	t.Logf("Initial goroutines: %d", len(initialGoroutines))
	for _, g := range initialGoroutines {
		t.Logf("  [%d] %s - %s (system: %v)", g.ID, g.State, g.Function, g.IsSystem)
	}

	// Start test server
	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	// Get state after server start
	afterServerStack := make([]byte, 64*1024)
	n = runtime.Stack(afterServerStack, true)
	afterServerGoroutines := parseGoroutineStack(string(afterServerStack[:n]))

	t.Logf("After server start: %d goroutines", len(afterServerGoroutines))
	leaked, _ := analyzeGoroutineLeaks(initialGoroutines, afterServerGoroutines)
	for _, g := range leaked {
		t.Logf("  NEW [%d] %s - %s (system: %v)", g.ID, g.State, g.Function, g.IsSystem)
	}

	// Perform HTTP operations that might leak goroutines
	for i := 0; i < 10; i++ {
		// GET requests
		req := New(fmt.Sprintf("%s/get?test=%d", host, i))
		res := req.Get()
		assert.Equal(t, 200, res.Code)

		// POST requests
		req = New(fmt.Sprintf("%s/post", host))
		res = req.Post(map[string]interface{}{"iteration": i})
		assert.Equal(t, 200, res.Code)
	}

	// Force close all HTTP client connections
	CloseAllTransports()

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Get final state
	finalStack := make([]byte, 64*1024)
	n = runtime.Stack(finalStack, true)
	finalGoroutines := parseGoroutineStack(string(finalStack[:n]))

	t.Logf("Final goroutines: %d", len(finalGoroutines))

	// Analyze leaks from after server start to final state
	leaked, cleaned := analyzeGoroutineLeaks(afterServerGoroutines, finalGoroutines)

	t.Logf("Analysis results:")
	t.Logf("  Leaked goroutines: %d", len(leaked))
	t.Logf("  Cleaned goroutines: %d", len(cleaned))

	// Report leaked goroutines
	applicationLeaks := 0
	for _, g := range leaked {
		t.Logf("  LEAKED [%d] %s - %s (system: %v)", g.ID, g.State, g.Function, g.IsSystem)
		if !g.IsSystem {
			applicationLeaks++
			t.Errorf("Application goroutine leak detected: [%d] %s", g.ID, g.Function)
		}
	}

	// Report cleaned goroutines
	for _, g := range cleaned {
		t.Logf("  CLEANED [%d] %s - %s (system: %v)", g.ID, g.State, g.Function, g.IsSystem)
	}

	// Fail test if there are application-level leaks
	if applicationLeaks > 0 {
		t.Errorf("Detected %d application-level goroutine leaks", applicationLeaks)

		// Print full stack trace for debugging
		debugStack := make([]byte, 64*1024)
		n = runtime.Stack(debugStack, true)
		t.Logf("Full stack trace:\n%s", string(debugStack[:n]))
	}
}

func TestHTTPClientGoroutineLeaks(t *testing.T) {
	// Start test server for client testing
	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	initialStack := make([]byte, 32*1024)
	n := runtime.Stack(initialStack, true)
	initialGoroutines := parseGoroutineStack(string(initialStack[:n]))

	t.Logf("Initial goroutines: %d", len(initialGoroutines))

	// Perform HTTP operations that should not leak
	for i := 0; i < 20; i++ {
		req := New(fmt.Sprintf("%s/get?client_test=%d", host, i))
		req.AddHeader("User-Agent", "gou-http-test")

		res := req.Get()
		assert.Equal(t, 200, res.Code)
	}

	// Force close all HTTP client connections
	CloseAllTransports()

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Get final state
	finalStack := make([]byte, 32*1024)
	n = runtime.Stack(finalStack, true)
	finalGoroutines := parseGoroutineStack(string(finalStack[:n]))

	t.Logf("Final goroutines: %d", len(finalGoroutines))

	// Analyze leaks
	leaked, cleaned := analyzeGoroutineLeaks(initialGoroutines, finalGoroutines)

	t.Logf("HTTP client test results:")
	t.Logf("  Leaked goroutines: %d", len(leaked))
	t.Logf("  Cleaned goroutines: %d", len(cleaned))

	// Check for application leaks
	applicationLeaks := 0
	for _, g := range leaked {
		t.Logf("  LEAKED [%d] %s - %s (system: %v)", g.ID, g.State, g.Function, g.IsSystem)
		if !g.IsSystem {
			applicationLeaks++
			t.Errorf("HTTP client goroutine leak: [%d] %s", g.ID, g.Function)
		}
	}

	if applicationLeaks > 0 {
		t.Errorf("HTTP client operations leaked %d goroutines", applicationLeaks)
	}
}

func TestTransportPoolPreciseLeaks(t *testing.T) {
	// Clear existing transport pool
	poolMutex.Lock()
	initialPoolSize := len(transportPool)
	poolMutex.Unlock()

	initialStack := make([]byte, 32*1024)
	n := runtime.Stack(initialStack, true)
	initialGoroutines := parseGoroutineStack(string(initialStack[:n]))

	t.Logf("Initial state: %d goroutines, %d transports", len(initialGoroutines), initialPoolSize)

	// Create many different transport configurations
	configs := []struct {
		isHTTPS bool
		proxy   string
	}{
		{false, ""},
		{true, ""},
		{false, "http://proxy1.example.com:8080"},
		{true, "http://proxy2.example.com:8080"},
		{false, "http://proxy3.example.com:3128"},
		{true, "http://proxy4.example.com:3128"},
	}

	for i := 0; i < 50; i++ {
		config := configs[i%len(configs)]
		transport := GetTransport(config.isHTTPS, config.proxy)
		assert.NotNil(t, transport)

		// Simulate using the transport
		if config.isHTTPS {
			assert.NotNil(t, transport.TLSClientConfig)
		}
		if config.proxy != "" {
			assert.NotNil(t, transport.Proxy)
		}
	}

	// Force close all HTTP client connections
	CloseAllTransports()

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Get final state
	finalStack := make([]byte, 32*1024)
	n = runtime.Stack(finalStack, true)
	finalGoroutines := parseGoroutineStack(string(finalStack[:n]))

	poolMutex.RLock()
	finalPoolSize := len(transportPool)
	poolMutex.RUnlock()

	t.Logf("Final state: %d goroutines, %d transports", len(finalGoroutines), finalPoolSize)

	// Analyze leaks
	leaked, cleaned := analyzeGoroutineLeaks(initialGoroutines, finalGoroutines)

	t.Logf("Transport pool test results:")
	t.Logf("  Pool growth: %d -> %d (+%d)", initialPoolSize, finalPoolSize, finalPoolSize-initialPoolSize)
	t.Logf("  Leaked goroutines: %d", len(leaked))
	t.Logf("  Cleaned goroutines: %d", len(cleaned))

	// Check for application leaks
	applicationLeaks := 0
	for _, g := range leaked {
		t.Logf("  LEAKED [%d] %s - %s (system: %v)", g.ID, g.State, g.Function, g.IsSystem)
		if !g.IsSystem {
			applicationLeaks++
			t.Errorf("Transport pool goroutine leak: [%d] %s", g.ID, g.Function)
		}
	}

	if applicationLeaks > 0 {
		t.Errorf("Transport pool operations leaked %d goroutines", applicationLeaks)
	}

	// Transport pool should be reasonable size
	poolGrowth := finalPoolSize - initialPoolSize
	if poolGrowth > len(configs) {
		t.Errorf("Transport pool grew excessively: +%d entries (expected: <= %d)", poolGrowth, len(configs))
	}
}

// =============================================================================
// Memory Leak Detection Tests
// =============================================================================

func TestMemoryLeakDetection(t *testing.T) {
	shutdown, ready, host := setup()
	go start(t, &host, shutdown, ready)
	defer stop(shutdown, ready)
	<-ready

	// Force garbage collection before starting
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform many HTTP operations to test for memory leaks
	for i := 0; i < 200; i++ {
		// Various request types
		req := New(fmt.Sprintf("%s/get?large=%d", host, i))
		req.AddHeader("X-Test", fmt.Sprintf("iteration-%d", i))
		res := req.Get()
		assert.Equal(t, 200, res.Code)

		// POST with data
		req = New(fmt.Sprintf("%s/post", host))
		largeData := make(map[string]interface{})
		for j := 0; j < 10; j++ {
			largeData[fmt.Sprintf("key_%d", j)] = strings.Repeat("data", 100)
		}
		res = req.Post(largeData)
		assert.Equal(t, 200, res.Code)

		// File uploads
		if i%10 == 0 {
			content := strings.Repeat(fmt.Sprintf("test data %d ", i), 100)
			tmpFile := tmpfile(t, content)
			req = New(fmt.Sprintf("%s/post?file=1", host)).
				AddHeader("Content-Type", "multipart/form-data")
			res = req.Post(tmpFile)
			os.Remove(tmpFile)
			assert.Equal(t, 200, res.Code)
		}

		// Force GC periodically
		if i%50 == 0 {
			runtime.GC()
		}
	}

	// Force final garbage collection
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check memory growth
	var memGrowth, heapGrowth int64
	if m2.Alloc >= m1.Alloc {
		memGrowth = int64(m2.Alloc - m1.Alloc)
	} else {
		memGrowth = -int64(m1.Alloc - m2.Alloc)
	}
	if m2.HeapAlloc >= m1.HeapAlloc {
		heapGrowth = int64(m2.HeapAlloc - m1.HeapAlloc)
	} else {
		heapGrowth = -int64(m1.HeapAlloc - m2.HeapAlloc)
	}

	t.Logf("Memory stats for HTTP operations:")
	t.Logf("  Alloc growth: %d bytes (%.2f MB)", memGrowth, float64(memGrowth)/(1024*1024))
	t.Logf("  Heap growth: %d bytes (%.2f MB)", heapGrowth, float64(heapGrowth)/(1024*1024))
	t.Logf("  Sys growth: %d bytes", int64(m2.Sys)-int64(m1.Sys))
	t.Logf("  NumGC: %d", m2.NumGC-m1.NumGC)
	t.Logf("  Transport pool size: %d", len(transportPool))

	// Check transport pool growth
	if len(transportPool) > 10 {
		t.Errorf("Transport pool grew excessively: %d entries", len(transportPool))
	}

	// Allow reasonable memory growth for HTTP operations
	maxAllowedGrowth := int64(5 * 1024 * 1024) // 5MB threshold
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: alloc grew by %d bytes (%.2f MB), threshold: %d bytes (%.2f MB)",
			memGrowth, float64(memGrowth)/(1024*1024), maxAllowedGrowth, float64(maxAllowedGrowth)/(1024*1024))
	}
}

func TestTransportPoolMemoryUsage(t *testing.T) {
	// Clear existing transport pool
	poolMutex.Lock()
	initialPoolSize := len(transportPool)
	poolMutex.Unlock()

	// Force garbage collection
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create many different transport configurations
	for i := 0; i < 100; i++ {
		isHTTPS := i%2 == 0
		proxy := ""
		if i%3 == 0 {
			proxy = fmt.Sprintf("http://proxy%d.example.com:8080", i%10)
		}

		transport := GetTransport(isHTTPS, proxy)
		assert.NotNil(t, transport)
	}

	// Force garbage collection
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	poolMutex.RLock()
	finalPoolSize := len(transportPool)
	poolMutex.RUnlock()

	poolGrowth := finalPoolSize - initialPoolSize
	t.Logf("Transport pool growth: %d -> %d (+%d entries)", initialPoolSize, finalPoolSize, poolGrowth)

	// Memory growth check
	var memGrowth int64
	if m2.Alloc >= m1.Alloc {
		memGrowth = int64(m2.Alloc - m1.Alloc)
	} else {
		memGrowth = -int64(m1.Alloc - m2.Alloc)
	}

	t.Logf("Memory growth for transport pool: %d bytes (%.2f KB)", memGrowth, float64(memGrowth)/1024)

	// Transport pool should be reasonable size
	if poolGrowth > 50 {
		t.Errorf("Transport pool grew too much: +%d entries (threshold: 50)", poolGrowth)
	}

	// Memory growth should be reasonable
	maxAllowedGrowth := int64(1024 * 1024) // 1MB
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Transport pool memory usage too high: %d bytes (%.2f MB)", memGrowth, float64(memGrowth)/(1024*1024))
	}
}

// =============================================================================
// Race Condition Tests
// =============================================================================

func TestTransportPoolRaceConditions(t *testing.T) {
	numWorkers := 50
	var wg sync.WaitGroup

	// Test concurrent access to transport pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < 20; j++ {
				isHTTPS := (workerID+j)%2 == 0
				proxy := ""
				if (workerID+j)%3 == 0 {
					proxy = fmt.Sprintf("http://proxy%d.example.com:8080", (workerID+j)%5)
				}

				transport := GetTransport(isHTTPS, proxy)
				assert.NotNil(t, transport)

				// Simulate using the transport
				if isHTTPS {
					assert.NotNil(t, transport.TLSClientConfig)
				}
			}
		}(i)
	}

	wg.Wait()

	// Check that pool is in consistent state
	poolMutex.RLock()
	poolSize := len(transportPool)
	poolMutex.RUnlock()

	t.Logf("Transport pool size after race test: %d", poolSize)
	assert.Greater(t, poolSize, 0, "Transport pool should have entries")
	assert.LessOrEqual(t, poolSize, 20, "Transport pool should not have excessive entries")
}

// =============================================================================
// Coverage Tests for Medium Risk Areas
// =============================================================================

func TestUploadMethod(t *testing.T) {
	// Test Upload method (currently returns nil)
	req := New("http://example.com/upload")

	// Test with various parameters
	res := req.Upload("", 0)
	assert.Nil(t, res, "Upload should return nil for empty file")

	res = req.Upload("nonexistent.txt", 1024)
	assert.Nil(t, res, "Upload should return nil for nonexistent file")

	res = req.Upload("/tmp/test.txt", -1)
	assert.Nil(t, res, "Upload should return nil for negative chunk size")

	// Test with default chunk size
	res = req.Upload("test.dat", 0)
	assert.Nil(t, res, "Upload should return nil (method not implemented)")

	// Test with custom chunk size
	res = req.Upload("large.bin", 2048)
	assert.Nil(t, res, "Upload should return nil (method not implemented)")
}

func TestXMLHandling(t *testing.T) {
	// Test XML content type detection without server
	req := New("http://example.com/test")

	// Test xml() method
	req.SetHeader("Content-Type", "application/xml")
	assert.True(t, req.xml(), "Should detect application/xml content type")

	req.SetHeader("Content-Type", "application/xml; charset=utf-8")
	assert.True(t, req.xml(), "Should detect application/xml with charset")

	req.SetHeader("Content-Type", "text/xml")
	assert.False(t, req.xml(), "text/xml is not application/xml")

	req.SetHeader("Content-Type", "application/json")
	assert.False(t, req.xml(), "Should not detect JSON as XML")

	// Test XML body handling
	req = New("http://example.com/test")
	req.SetHeader("Content-Type", "application/xml")

	// Test with XML string
	xmlData := `<?xml version="1.0" encoding="UTF-8"?><root><item>test</item></root>`
	req.data = xmlData
	body, err := req.body()
	assert.Nil(t, err, "XML string should not cause error")
	assert.Equal(t, []byte(xmlData), body, "XML string should be converted to bytes")

	// Test with XML bytes
	req.data = []byte(xmlData)
	body, err = req.body()
	assert.Nil(t, err, "XML bytes should not cause error")
	assert.Equal(t, []byte(xmlData), body, "XML bytes should be returned as-is")

	// Test XML with struct data (should fallback to JSON)
	testStruct := struct {
		Name  string `xml:"name"`
		Value int    `xml:"value"`
	}{
		Name:  "test",
		Value: 123,
	}
	req.data = testStruct
	body, err = req.body()
	assert.Nil(t, err, "XML with struct should fallback to JSON without error")
	assert.NotNil(t, body, "Should return JSON body for struct")

	// Verify it's valid JSON
	var result map[string]interface{}
	jsonErr := jsoniter.Unmarshal(body, &result)
	assert.Nil(t, jsonErr, "Should be valid JSON")
	assert.Equal(t, "test", result["Name"], "Should contain struct data")
	assert.Equal(t, float64(123), result["Value"], "Should contain struct data")
}

func TestProxyConfiguration(t *testing.T) {
	// Test proxy URL parsing and configuration
	testCases := []struct {
		name      string
		isHTTPS   bool
		proxy     string
		shouldErr bool
	}{
		{
			name:      "HTTP with valid proxy",
			isHTTPS:   false,
			proxy:     "http://proxy.example.com:8080",
			shouldErr: false,
		},
		{
			name:      "HTTPS with valid proxy",
			isHTTPS:   true,
			proxy:     "http://proxy.example.com:8080",
			shouldErr: false,
		},
		{
			name:      "HTTPS with HTTPS proxy",
			isHTTPS:   true,
			proxy:     "https://secure-proxy.example.com:8443",
			shouldErr: false,
		},
		{
			name:      "Proxy with authentication",
			isHTTPS:   false,
			proxy:     "http://user:pass@proxy.example.com:3128",
			shouldErr: false,
		},
		{
			name:      "SOCKS5 proxy",
			isHTTPS:   false,
			proxy:     "socks5://proxy.example.com:1080",
			shouldErr: false,
		},
		{
			name:      "Invalid proxy URL",
			isHTTPS:   false,
			proxy:     "invalid://proxy/url",
			shouldErr: false, // GetTransport handles invalid URLs gracefully
		},
		{
			name:      "Empty proxy",
			isHTTPS:   false,
			proxy:     "",
			shouldErr: false,
		},
		{
			name:      "Malformed proxy URL",
			isHTTPS:   true,
			proxy:     "://malformed",
			shouldErr: false, // Should handle gracefully
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transport := GetTransport(tc.isHTTPS, tc.proxy)
			assert.NotNil(t, transport, "Transport should not be nil")

			// Verify HTTPS configuration
			if tc.isHTTPS {
				assert.NotNil(t, transport.TLSClientConfig, "HTTPS transport should have TLS config")
			}

			// Verify proxy configuration
			if tc.proxy != "" && !tc.shouldErr {
				// We can't easily test the proxy function directly, but we can verify
				// that the transport was created successfully
				assert.NotNil(t, transport, "Transport with proxy should be created")
			}

			// Verify production settings
			assert.Equal(t, 100, transport.MaxIdleConns, "MaxIdleConns should be 100")
			assert.Equal(t, 10, transport.MaxIdleConnsPerHost, "MaxIdleConnsPerHost should be 10")
			assert.Equal(t, 30*time.Second, transport.IdleConnTimeout, "IdleConnTimeout should be 30s")
			assert.Equal(t, 30*time.Second, transport.TLSHandshakeTimeout, "TLSHandshakeTimeout should be 30s")
			assert.Equal(t, 300*time.Second, transport.ResponseHeaderTimeout, "ResponseHeaderTimeout should be 300s")
			assert.Equal(t, 30*time.Second, transport.ExpectContinueTimeout, "ExpectContinueTimeout should be 30s")
		})
	}
}

func TestEnvironmentProxyConfiguration(t *testing.T) {
	// Save original environment variables
	originalHTTPProxy := os.Getenv("HTTP_PROXY")
	originalHTTPSProxy := os.Getenv("HTTPS_PROXY")
	originalHttpProxy := os.Getenv("http_proxy")
	originalHttpsProxy := os.Getenv("https_proxy")

	// Restore environment variables after test
	defer func() {
		os.Setenv("HTTP_PROXY", originalHTTPProxy)
		os.Setenv("HTTPS_PROXY", originalHTTPSProxy)
		os.Setenv("http_proxy", originalHttpProxy)
		os.Setenv("https_proxy", originalHttpsProxy)
	}()

	testCases := []struct {
		name           string
		envVars        map[string]string
		isHTTPS        bool
		expectedResult string
	}{
		{
			name: "HTTP_PROXY for HTTP",
			envVars: map[string]string{
				"HTTP_PROXY": "http://proxy1.example.com:8080",
			},
			isHTTPS:        false,
			expectedResult: "http://proxy1.example.com:8080",
		},
		{
			name: "HTTPS_PROXY for HTTPS",
			envVars: map[string]string{
				"HTTPS_PROXY": "http://proxy2.example.com:8080",
			},
			isHTTPS:        true,
			expectedResult: "http://proxy2.example.com:8080",
		},
		{
			name: "http_proxy fallback for HTTP",
			envVars: map[string]string{
				"http_proxy": "http://proxy3.example.com:8080",
			},
			isHTTPS:        false,
			expectedResult: "http://proxy3.example.com:8080",
		},
		{
			name: "https_proxy fallback for HTTPS",
			envVars: map[string]string{
				"https_proxy": "http://proxy4.example.com:8080",
			},
			isHTTPS:        true,
			expectedResult: "http://proxy4.example.com:8080",
		},
		{
			name: "HTTP_PROXY takes precedence over http_proxy",
			envVars: map[string]string{
				"HTTP_PROXY": "http://proxy5.example.com:8080",
				"http_proxy": "http://proxy6.example.com:8080",
			},
			isHTTPS:        false,
			expectedResult: "http://proxy5.example.com:8080",
		},
		{
			name: "HTTPS_PROXY takes precedence over https_proxy",
			envVars: map[string]string{
				"HTTPS_PROXY": "http://proxy7.example.com:8080",
				"https_proxy": "http://proxy8.example.com:8080",
			},
			isHTTPS:        true,
			expectedResult: "http://proxy7.example.com:8080",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear all proxy environment variables
			os.Unsetenv("HTTP_PROXY")
			os.Unsetenv("HTTPS_PROXY")
			os.Unsetenv("http_proxy")
			os.Unsetenv("https_proxy")

			// Set test environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}

			// Test GetProxy function
			result := GetProxy(tc.isHTTPS)
			assert.Equal(t, tc.expectedResult, result, "GetProxy should return expected proxy URL")
		})
	}
}

func TestTransportPoolWithProxyVariations(t *testing.T) {
	// Clear transport pool
	poolMutex.Lock()
	initialPoolSize := len(transportPool)
	poolMutex.Unlock()

	// Test various proxy configurations to ensure proper pooling
	proxyConfigs := []struct {
		isHTTPS bool
		proxy   string
	}{
		{false, ""},
		{true, ""},
		{false, "http://proxy1.example.com:8080"},
		{true, "http://proxy1.example.com:8080"},
		{false, "http://proxy2.example.com:3128"},
		{true, "http://proxy2.example.com:3128"},
		{false, "https://secure-proxy.example.com:8443"},
		{true, "https://secure-proxy.example.com:8443"},
		{false, "socks5://socks-proxy.example.com:1080"},
		{true, "socks5://socks-proxy.example.com:1080"},
	}

	transports := make([]*http.Transport, len(proxyConfigs))

	// Create transports for each configuration
	for i, config := range proxyConfigs {
		transport := GetTransport(config.isHTTPS, config.proxy)
		transports[i] = transport
		assert.NotNil(t, transport, "Transport should not be nil")

		// Verify production settings
		assert.Equal(t, 100, transport.MaxIdleConns, "MaxIdleConns should be production value")
		assert.Equal(t, 10, transport.MaxIdleConnsPerHost, "MaxIdleConnsPerHost should be production value")
	}

	// Verify transport reuse - requesting same config should return same transport
	for i, config := range proxyConfigs {
		transport := GetTransport(config.isHTTPS, config.proxy)
		assert.Same(t, transports[i], transport, "Same configuration should return same transport instance")
	}

	// Check pool growth
	poolMutex.RLock()
	finalPoolSize := len(transportPool)
	poolMutex.RUnlock()

	expectedGrowth := len(proxyConfigs)
	actualGrowth := finalPoolSize - initialPoolSize

	t.Logf("Transport pool growth: %d -> %d (+%d), expected: +%d",
		initialPoolSize, finalPoolSize, actualGrowth, expectedGrowth)

	// Some configurations might be duplicates or already exist, so allow some tolerance
	assert.Greater(t, actualGrowth, 0, "Pool should grow with new configurations")
	assert.LessOrEqual(t, actualGrowth, expectedGrowth, "Pool should not grow more than expected")
}

// =============================================================================
// Performance Benchmarks
// =============================================================================

func BenchmarkHTTPRequestCreation(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := New("http://example.com/api/test")
			req.AddHeader("Authorization", "Bearer token")
			req.AddHeader("Content-Type", "application/json")
			// Just benchmark request creation, not actual HTTP calls
		}
	})
}

func BenchmarkHTTPRequestWithData(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data := map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
				"key3": 123,
				"nested": map[string]interface{}{
					"inner": "value",
				},
			}
			req := New("http://example.com/api/test")
			req.AddHeader("Content-Type", "application/json")
			// Benchmark JSON marshaling - this will use the data
			jsoniter.Marshal(data)
		}
	})
}

func BenchmarkTransportPool(b *testing.B) {
	configs := []struct {
		isHTTPS bool
		proxy   string
	}{
		{false, ""},
		{true, ""},
		{false, "http://proxy.example.com:8080"},
		{true, "http://proxy.example.com:8080"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			config := configs[b.N%len(configs)]
			transport := GetTransport(config.isHTTPS, config.proxy)
			if transport == nil {
				b.Error("Transport should not be nil")
			}
		}
	})
}
