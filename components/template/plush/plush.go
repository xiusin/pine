package pongo

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
	cache map[string]*plush.Template
	l     sync.RWMutex
	dir   string
	debug bool
}

func New(dir string, debug bool) *Plush {
	t := &Plush{debug: debug, dir: dir}

	t.cache = map[string]*plush.Template{}
	return t
}

func (c *Plush) AddFunc(funcName string, funcEntry interface{}) {
	c.l.Lock()
	_ = plush.Helpers.Add(funcName, funcEntry)
	c.l.Unlock()
}

func (c *Plush) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	var (
		tpl *plush.Template
		ok  bool
		err error
	)
	c.l.RLock()
	tpl, ok = c.cache[name]
	c.l.RUnlock()
	if !ok {
		c.l.Lock()
		defer c.l.Unlock()
		s, err := ioutil.ReadFile(c.dir + "/" + name) // 读取模板内容
		if err != nil {
			return err
		}
		tpl, err = plush.NewTemplate(string(s))
		if err != nil {
			return err
		}
		c.cache[name] = tpl

	}
	ctx := plush.NewContext()
	for k, v := range binding {
		ctx.Set(k, v)
	}
	html, err := tpl.Exec(ctx)
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
