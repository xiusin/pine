package plush

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"sync"

	"github.com/gobuffalo/plush"
)

type Plush struct {
	cache map[string]string
	l     sync.RWMutex
	dir   string
	debug bool
}

func New(dir string, debug bool) *Plush {
	t := &Plush{debug: debug, dir: dir}
	t.cache = make(map[string]string)
	return t
}

func (c *Plush) AddFunc(funcName string, funcEntry interface{}) {
	c.l.Lock()
	_ = plush.Helpers.Add(funcName, funcEntry)
	c.l.Unlock()
}

func (c *Plush) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	c.l.RLock()
	html, ok := c.cache[name]
	c.l.RUnlock()
	if !ok || c.debug {
		c.l.Lock()
		defer c.l.Unlock()
		s, err := ioutil.ReadFile(c.dir + "/" + name) // 读取模板内容
		if err != nil {
			return err
		}
		c.cache[name] = string(s)
		html = c.cache[name]
	}
	html, err := plush.BuffaloRenderer(html, binding, nil)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(html))
	return err
}

func (_ *Plush) JSON(writer io.Writer, v map[string]interface{}) error {
	b, err := json.Marshal(v)
	if err == nil {
		_, err = writer.Write(b)
	}
	return err
}

func (_ *Plush) JSONP(writer io.Writer, callback string, v map[string]interface{}) error {
	var ret bytes.Buffer
	b, err := json.Marshal(v)
	if err == nil {
		ret.Write([]byte(callback))
		ret.Write([]byte("("))
		ret.Write(b)
		ret.Write([]byte(")"))
		_, err = writer.Write(ret.Bytes())
	}
	return err
}

func (_ *Plush) Text(writer io.Writer, v []byte) error {
	_, err := writer.Write(v)
	return err
}

func (_ *Plush) XML(writer io.Writer, v map[string]interface{}) error {
	b, err := xml.Marshal(v)
	if err == nil {
		_, err = writer.Write(b)
	}
	return err
}
