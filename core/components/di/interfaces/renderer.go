package interfaces

import (
	"io"
)

type RendererInf interface {

	// 渲染html
	HTML(writer io.Writer, name string, binding map[string]interface{}) error

	// 渲染json
	JSON(writer io.Writer, v map[string]interface{}) error

	// 渲染jsonp
	JSONP(writer io.Writer, callback string, v map[string]interface{}) error

	// 渲染text
	Text(writer io.Writer, v []byte) error

	// 渲染xml
	XML(writer io.Writer, v map[string]interface{}) error
}
