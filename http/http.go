package http

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/cast"
	"github.com/yaoapp/gou/dns"
)

// New make a new  http Request
func New(url string) *Request {
	return &Request{
		url:       url,
		headers:   http.Header{},
		query:     neturl.Values{},
		files:     []File{},
		fileBytes: []File{},
	}
}

// ResponseError return new  error response
func ResponseError(code int, message string) *Response {
	return &Response{
		Code:    0,
		Status:  0,
		Message: message,
		Headers: http.Header{},
		Data:    nil,
	}
}

// AddHeader set the request header
func (r *Request) AddHeader(name, value string) *Request {
	r.headers.Add(name, value)
	return r
}

// AddFile set the file
func (r *Request) AddFile(name, file string) *Request {
	r.files = append(r.files, File{Name: name, Path: file})
	r.SetHeader("Content-Type", "multipart/form-data")
	return r
}

// AddFileBytes set the file
func (r *Request) AddFileBytes(name, pathname string, data []byte) *Request {
	r.fileBytes = append(r.fileBytes, File{Name: name, Path: pathname, data: data})
	r.SetHeader("Content-Type", "multipart/form-data")
	return r
}

// DelHeader unset the request header
func (r *Request) DelHeader(name string) *Request {
	r.headers.Del(name)
	return r
}

// GetHeader get the request header
func (r *Request) GetHeader(name string) string {
	return r.headers.Get(name)
}

// SetHeader set the request header
func (r *Request) SetHeader(name string, value string) *Request {
	r.headers.Set(name, value)
	return r
}

// HasHeader check if the header name is exists
func (r *Request) HasHeader(name string) bool {
	return r.headers.Get(name) != ""
}

// WithHeader set the request headers
func (r *Request) WithHeader(headers http.Header) *Request {
	r.headers = headers
	return r
}

// WithQuery set the request query params
func (r *Request) WithQuery(values neturl.Values) *Request {
	r.query = values
	return r
}

// Get send the GET request
func (r *Request) Get() *Response {
	if !r.HasHeader("Content-Type") {
		r.AddHeader("Content-Type", "application/json; charset=utf-8")
	}
	return r.Send("GET", nil)
}

// Post send the POST request
func (r *Request) Post(data interface{}) *Response {

	if !r.HasHeader("Content-Type") {
		r.AddHeader("Content-Type", "application/json; charset=utf-8")
	}
	return r.Send("POST", data)
}

// Put send the PUT request
func (r *Request) Put(data interface{}) *Response {
	if !r.HasHeader("Content-Type") {
		r.AddHeader("Content-Type", "application/json; charset=utf-8")
	}
	return r.Send("PUT", data)
}

// Patch send the PATCH request
func (r *Request) Patch(data interface{}) *Response {
	if !r.HasHeader("Content-Type") {
		r.AddHeader("Content-Type", "application/json; charset=utf-8")
	}
	return r.Send("PATCH", data)
}

// Delete send the DELETE request
func (r *Request) Delete(data interface{}) *Response {
	if !r.HasHeader("Content-Type") {
		r.AddHeader("Content-Type", "application/json; charset=utf-8")
	}
	return r.Send("DELETE", data)
}

// Head send the Head request
func (r *Request) Head(data interface{}) *Response {
	return r.Send("HEAD", data)
}

// Send  send the request
func (r *Request) Send(method string, data interface{}) *Response {

	var res *Response
	var body []byte

	if data != nil {
		r.data = data
	}

	if method != "GET" && method != "HEAD" {
		if r.headers.Get("Content-Type") == "" {
			r.headers.Set("Content-Type", "text/plain")
		}

		body, res = r.body()
		if res != nil {
			return res
		}
	}

	requestURL := r.url

	// URL Parse
	if strings.Contains(requestURL, "?") {
		uri := strings.Split(requestURL, "?")
		requestURL = uri[0]
		query, err := neturl.ParseQuery(uri[1])
		if err != nil {
			return ResponseError(0, err.Error())
		}
		cast.MergeURLValues(r.query, query)
	}

	if r.query != nil && len(r.query) > 0 {
		requestURL = fmt.Sprintf("%s?%s", requestURL, r.query.Encode())
	}

	req, err := http.NewRequest(method, requestURL, bytes.NewBuffer(body))
	if err != nil {
		return ResponseError(0, fmt.Sprintf("http.NewRequest: %s", err.Error()))
	}

	// Request Header
	req.Header = r.headers

	// Force using system DSN resolver
	// var dialer = &net.Dialer{Resolver: &net.Resolver{PreferGo: false}}
	var dialContext = dns.DialContext()
	var tr = &http.Transport{DialContext: dialContext}
	var client *http.Client = &http.Client{Transport: tr}

	// check if the proxy is set
	proxy := getProxy(false)
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return ResponseError(0, err.Error())
		}
		tr := &http.Transport{
			Proxy:       http.ProxyURL(proxyURL),
			DialContext: dialContext,
		}

		client = &http.Client{Transport: tr}
	}

	// Https SkipVerify false
	if strings.HasPrefix(r.url, "https://") {

		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext:     dialContext,
		}

		// check if the proxy is set
		proxy := getProxy(true)
		if proxy != "" {
			proxyURL, err := url.Parse(proxy)
			if err != nil {
				return ResponseError(0, err.Error())
			}
			tr.Proxy = http.ProxyURL(proxyURL)
		}

		client = &http.Client{Transport: tr}
	}
	defer tr.CloseIdleConnections()

	resp, err := client.Do(req)
	if err != nil {
		return ResponseError(0, err.Error())
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	res = &Response{
		Status:  resp.StatusCode,
		Data:    nil,
		Code:    resp.StatusCode,
		Headers: resp.Header,
	}

	if method == "HEAD" {
		return res
	}

	rBody, err := io.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		return ResponseError(resp.StatusCode, err.Error())
	}

	res.Data = rBody
	if len(rBody) == 0 {
		res.Data = nil
		return res
	}

	var rData interface{}
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		err = jsoniter.Unmarshal(rBody, &rData)
		if err != nil {
			return ResponseError(resp.StatusCode, err.Error())
		}
	}

	if rData != nil {

		res.Data = rData

		switch value := rData.(type) {

		case map[string]string:
			res.Message = value["message"]
			break

		case map[string]interface{}:
			if v, ok := value["message"].(string); ok {
				res.Message = v
			}
			break
		}
	}

	return res
}

// Stream stream the request
func (r *Request) Stream(ctx context.Context, method string, data interface{}, handler func(data []byte) int) error {

	var res *Response
	var body []byte

	if data != nil {
		r.data = data
	}

	if method != "GET" && method != "HEAD" {
		if r.headers.Get("Content-Type") == "" {
			r.headers.Set("Content-Type", "text/plain")
		}

		body, res = r.body()
		if res != nil {
			return nil
		}
	}

	requestURL := r.url

	// URL Parse
	if strings.Contains(requestURL, "?") {
		uri := strings.Split(requestURL, "?")
		requestURL = uri[0]
		query, err := neturl.ParseQuery(uri[1])
		if err != nil {
			return err
		}
		cast.MergeURLValues(r.query, query)
	}

	if r.query != nil && len(r.query) > 0 {
		requestURL = fmt.Sprintf("%s?%s", requestURL, r.query.Encode())
	}

	req, err := http.NewRequest(method, requestURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	// Request Header
	req.Header = r.headers

	// Force using system DSN resolver
	// var dialer = &net.Dialer{Resolver: &net.Resolver{PreferGo: false}}
	var dialContext = dns.DialContext()
	var tr = &http.Transport{DialContext: dialContext}
	var client *http.Client = &http.Client{Transport: tr}

	// check if the proxy is set
	proxy := getProxy(false)
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return err
		}
		tr := &http.Transport{
			Proxy:       http.ProxyURL(proxyURL),
			DialContext: dialContext,
		}

		client = &http.Client{Transport: tr}
	}

	// Https SkipVerify false
	if strings.HasPrefix(r.url, "https://") {

		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext:     dialContext,
		}

		// check if the proxy is set
		proxy := getProxy(true)
		if proxy != "" {
			proxyURL, err := url.Parse(proxy)
			if err != nil {
				return err
			}
			tr.Proxy = http.ProxyURL(proxyURL)
		}

		client = &http.Client{Transport: tr}
	}
	defer tr.CloseIdleConnections()

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		res := handler(scanner.Bytes())
		switch res {
		case HandlerReturnOk:
			break

		case HandlerReturnBreak:
			return nil

		case HandlerReturnError:
			return fmt.Errorf("handler return error %d", res)
		}
	}

	return scanner.Err()
}

// Upload upload a big file
func (r *Request) Upload(file string, chunksize int) *Response {
	return nil
}

// body
func (r *Request) body() ([]byte, *Response) {

	if r.data == nil && len(r.files) == 0 && len(r.fileBytes) == 0 {
		return nil, nil
	}

	if r.json() {
		return r.jsonBody()
	}

	if r.urlencoded() {
		return r.urlencodedBody()
	}

	if r.formdata() {
		body, header, err := r.formBody()
		if err != nil {
			return nil, err
		}
		r.SetHeader("Content-Type", header)
		return body, nil
	}

	if r.text() || r.xml() {
		switch data := r.data.(type) {
		case []byte:
			return data, nil
		case string:
			return []byte(data), nil
		default:
			return r.jsonBody()
		}
	}

	return nil, ResponseError(0, fmt.Sprintf("Content-Type Error: %#v", r.headers.Get("Content-Type")))
}

// json check if the content-type is application/json
func (r *Request) json() bool {
	return strings.HasPrefix(r.headers.Get("Content-Type"), "application/json")
}

// xml check if the content-type is application/xml
func (r *Request) xml() bool {
	return strings.HasPrefix(r.headers.Get("Content-Type"), "application/xml")
}

// urlencoded check if the content-type is application/x-www-form-urlencoded
func (r *Request) urlencoded() bool {
	return strings.HasPrefix(r.headers.Get("Content-Type"), "application/x-www-form-urlencoded")
}

// formdata check if the content-type is multipart/form-data
func (r *Request) formdata() bool {
	return strings.HasPrefix(r.headers.Get("Content-Type"), "multipart/form-data")
}

// text check if the content-type is text/plain
func (r *Request) text() bool {
	return strings.HasPrefix(r.headers.Get("Content-Type"), "text/plain")
}

func (r *Request) jsonBody() ([]byte, *Response) {
	body, err := jsoniter.Marshal(r.data)
	if err != nil {
		return nil, ResponseError(0, err.Error())
	}
	return body, nil
}

func (r *Request) urlencodedBody() ([]byte, *Response) {
	switch value := r.data.(type) {
	case string:
		return []byte(value), nil

	case map[string]string:
		data := url.Values{}
		for k, v := range value {
			data.Add(k, v)
		}
		return []byte(data.Encode()), nil

	case map[string]interface{}:
		data := url.Values{}
		for k, v := range value {
			data.Add(k, fmt.Sprintf("%v", v))
		}
		return []byte(data.Encode()), nil
	}

	return nil, nil
}

func (r *Request) formBody() ([]byte, string, *Response) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// upload files
	for _, f := range r.files {
		file, err := os.Open(f.Path)
		if err != nil {
			return nil, "", ResponseError(0, err.Error())
		}
		defer file.Close()
		part, _ := writer.CreateFormFile(f.Name, filepath.Base(file.Name()))
		io.Copy(part, file)
	}

	for _, f := range r.fileBytes {
		part, _ := writer.CreateFormFile(f.Name, filepath.Base(f.Path))
		part.Write(f.data)
	}

	switch value := r.data.(type) {

	case []byte:
		part, _ := writer.CreateFormField("data")
		part.Write(value)
		break

	case string: // file upload
		file, err := os.Open(value)
		if err != nil {
			return nil, "", ResponseError(0, err.Error())
		}
		defer file.Close()
		part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
		io.Copy(part, file)
		break

	case map[string]string:
		for name, v := range value {
			_ = writer.WriteField(name, v)
		}
		break

	case map[string]interface{}:
		for name, v := range value {
			_ = writer.WriteField(name, fmt.Sprintf("%v", v))
		}
		break

	default:
		var data map[string]interface{}
		raw, err := jsoniter.Marshal(value)
		if err != nil {
			return nil, "", ResponseError(0, err.Error())
		}

		err = jsoniter.Unmarshal(raw, &data)
		if err != nil {
			return nil, "", ResponseError(0, err.Error())
		}
		for name, v := range data {
			_ = writer.WriteField(name, fmt.Sprintf("%v", v))
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, "", ResponseError(0, err.Error())
	}
	return body.Bytes(), writer.FormDataContentType(), nil
}

func getProxy(https bool) string {
	if https {
		proxy := os.Getenv("HTTPS_PROXY")
		if proxy != "" {
			return proxy
		}
		return os.Getenv("https_proxy")
	}

	proxy := os.Getenv("HTTP_PROXY")
	if proxy != "" {
		return proxy
	}
	return os.Getenv("http_proxy")
}
