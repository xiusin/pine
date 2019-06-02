package http

import "net/http"

type Response struct {
	http.ResponseWriter
}

func NewResponse(res http.ResponseWriter) *Response {
	return &Response{res}
}

func (c *Response) GetHttpResponse() http.ResponseWriter {
	return c.ResponseWriter
}

func (c *Response) Flush(content string) {
	_, _ = c.Write([]byte(content + "\n"))
	c.ResponseWriter.(http.Flusher).Flush()
}
