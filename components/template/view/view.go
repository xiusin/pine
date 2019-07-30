package view

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"html/template"
	"io"
	"reflect"
	"sync"
)

var funcMap = template.FuncMap{}

type Template struct {
	debug bool
	dir   string
	cache map[string]*template.Template
	l     sync.RWMutex
}

func New(viewDir string, debug bool) *Template {
	return &Template{
		debug: debug, // 不缓存模板渲染， 直接生效
		cache: map[string]*template.Template{},
		dir:   viewDir,
	}
}

func (c *Template) AddFunc(funcName string, funcEntry interface{}) {
	// 只接受函数参数 .kind大类型  .type 具体类型
	if reflect.ValueOf(funcEntry).Kind() == reflect.Func {
		c.l.Lock()
		funcMap[funcName] = funcEntry
		c.l.Unlock()
	}
}

func (c *Template) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	var (
		tpl *template.Template
		ok  bool
		err error
	)
	c.l.RLock()
	tpl, ok = c.cache[name]
	c.l.RUnlock()
	if !ok || c.debug {
		c.l.Lock()
		defer c.l.Unlock()
		tpl, err = template.ParseFiles(c.dir + "/" + name)
		if err != nil {
			return err
		}
		tpl.Funcs(funcMap)
		c.cache[name] = tpl
	}
	return tpl.ExecuteTemplate(writer, name, binding)
}

func (_ *Template) JSON(writer io.Writer, v map[string]interface{}) error {
	b, err := json.Marshal(v)
	if err == nil {
		_, err = writer.Write(b)
	}
	return err
}

func (_ *Template) JSONP(writer io.Writer, callback string, v map[string]interface{}) error {
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

func (_ *Template) Text(writer io.Writer, v []byte) error {
	_, err := writer.Write(v)
	return err
}

func (_ *Template) XML(writer io.Writer, v map[string]interface{}) error {
	b, err := xml.Marshal(v)
	if err == nil {
		_, err = writer.Write(b)
	}
	return err
}
