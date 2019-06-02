package http

import (
	"net"
	"net/http"
	"strings"
)

type Request struct {
	*http.Request
}

func NewRequest(handler *http.Request) *Request {
	return &Request{Request: handler}
}

// 判断是不是ajax请求
func (r *Request) GetHttpRequest() *http.Request {
	return r.Request
}

// 判断是不是ajax请求
func (r *Request) IsAjax() bool {
	return r.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

// 判断是不是Get请求
func (r *Request) IsGet() bool {
	return r.Method == http.MethodGet
}

// 判断是不是Post请求
func (r *Request) IsPost() bool {
	return r.Method == http.MethodPost
}

// 获取cookie
func (r *Request) GetCookie(name string) (cookie string, err error) {
	cok, err := r.Cookie(name)
	if err == nil {
		cookie = cok.Value
	}
	return
}

// 获取客户端IP
func (c *Request) ClientIP() string {
	clientIP := c.Header.Get("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(c.Header.Get("X-Real-Ip"))
	}
	if clientIP != "" {
		return clientIP
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.GetHttpRequest().RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

//todo 其他各种getParams之类的方法
