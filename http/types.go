package http

import (
	"net/http"
	"net/url"
)

// Request HTTP Request
type Request struct {
	url     string
	query   url.Values
	headers http.Header
	files   map[string]string
	data    interface{}
}

// Response HTTP Response
type Response struct {
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
	Headers http.Header `json:"headers"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
}
