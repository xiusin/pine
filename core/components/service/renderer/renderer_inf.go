package renderer

import "io"

type RendererInf interface {
	// 渲染data
	Data(writer io.Writer, v string) error

	// 渲染html
	HTML(writer io.Writer, name string, binding interface{}) error

	// 渲染json
	JSON(writer io.Writer, v interface{}) error

	// 渲染jsonp
	JSONP(writer io.Writer, callback string, v interface{}) error

	// 渲染text
	Text(writer io.Writer, v string) error

	// 渲染xml
	XML(writer io.Writer, v interface{}) error
}
