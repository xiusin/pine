package plush

import (
	"github.com/spf13/viper"
	"github.com/xiusin/router/components/option"
	"github.com/xiusin/router/components/template"
	"io"
	"io/ioutil"
	"sync"

	"github.com/gobuffalo/plush"
)

type Plush struct {
	template.Template
	cache map[string]string
	l     sync.RWMutex
	dir   string
	debug bool
}

func New(dir string) *Plush {
	t := &Plush{dir: dir}
	t.debug = viper.GetInt32("ENV") == option.DevMode
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
