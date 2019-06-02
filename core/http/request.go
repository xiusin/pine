package http

import (
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type Request struct {
	*http.Request
}

func NewRequest(handler *http.Request) *Request {
	return &Request{Request: handler}
}

// 判断是不是ajax请求
func (c *Request) GetHttpRequest() *http.Request {
	return c.Request
}

// 判断是不是ajax请求
func (c *Request) IsAjax() bool {
	return c.Header("X-Requested-With") == "XMLHttpRequest"
}

// 判断是不是Get请求
func (c *Request) IsGet() bool {
	return c.Method == http.MethodGet
}

// 判断是不是Post请求
func (c *Request) IsPost() bool {
	return c.Method == http.MethodPost
}

// 获取cookie
func (c *Request) GetCookie(name string) (cookie string, err error) {
	cok, err := c.Cookie(name)
	if err == nil {
		cookie = cok.Value
	}
	return
}

// 获取客户端IP
func (c *Request) ClientIP() string {
	clientIP := c.Header("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(c.Header("X-Real-Ip"))
	}
	if clientIP != "" {
		return clientIP
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.GetHttpRequest().RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

func (c *Request) GetInt(key string, defaultVal ...int) (val int, res bool) {
	val, err := strconv.Atoi(c.URL.Query().Get(key))
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Request) GetInt64(key string, defaultVal ...int64) (val int64, res bool) {
	val, err := strconv.ParseInt(c.URL.Query().Get(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Request) GetFloat64(key string, defaultVal ...float64) (val float64, res bool) {
	val, err := strconv.ParseFloat(c.URL.Query().Get(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Request) GetBool(key string) (val bool, err error) {
	val, err = strconv.ParseBool(c.URL.Query().Get(key))
	return
}

func (c *Request) GetStrings(key string) (val []string, ok bool) {
	val, ok = c.URL.Query()[key]
	return
}

func (c *Request) Header(key string) string {
	return c.Request.Header.Get(key)
}

func (c *Request) PostInt(key string, defaultVal ...int) (val int, res bool) {
	val, err := strconv.Atoi(c.PostFormValue(key))
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Request) PostInt64(key string, defaultVal ...int64) (val int64, res bool) {
	val, err := strconv.ParseInt(c.PostFormValue(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Request) PostFloat64(key string, defaultVal ...float64) (val float64, res bool) {
	val, err := strconv.ParseFloat(c.PostFormValue(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Request) PostBool(key string) (val bool, err error) {
	val, err = strconv.ParseBool(c.PostFormValue(key))
	return
}

func (c *Request) PostStrings(key string) (val []string, ok bool) {
	val, ok = c.PostForm[key]
	return
}

func (c *Request) Files(key string) (val []*multipart.FileHeader) {
	val = c.MultipartForm.File[key]
	return
}

func (c *Request) File(key string) multipart.File {
	val, _, err := c.FormFile(key)
	if err != nil {
		val = nil
	}
	return val
}
