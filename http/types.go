package http

import (
	"net/http"
	"net/url"
)

const (

	// HandlerReturnOk handler return ok
	HandlerReturnOk = 1

	// HandlerReturnBreak handler return break
	HandlerReturnBreak = 0

	// HandlerReturnError handler return error
	HandlerReturnError = -1
)

// Request HTTP Request
type Request struct {
	url       string
	query     url.Values
	headers   http.Header
	files     []File
	fileBytes []File
	data      interface{}
}

// Response HTTP Response
type Response struct {
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
	Headers http.Header `json:"headers"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
}

// File file
type File struct {
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
	Data string `json:"data,omitempty"`
	data []byte
}
