package interfaces

import (
	"io"
)

type IRenderer interface {
	AddFunc(string, interface{})
	HTML(writer io.Writer, name string, binding map[string]interface{}) error
	JSON(writer io.Writer, v map[string]interface{}) error
	JSONP(writer io.Writer, callback string, v map[string]interface{}) error
	Text(writer io.Writer, v []byte) error
	XML(writer io.Writer, v map[string]interface{}) error
}
