package pongo

import (
	"encoding/json"
	"encoding/xml"
	"html/template"
	"io"
	"sync"

	"github.com/github.com/flosch/pongo2"
)

type Pongo struct {
	ts *pongo2.TemplateSet
	// 需要确定使用用一个单例渲染是否会出现问题
	cache map[string]*pongo2.Template
	l     sync.RWMutex
}

var funcMap = template.FuncMap{}

func New(tplName, dir string, debug bool) *Pongo {
	t := &Pongo{ts: pongo2.NewSet(tplName, pongo2.MustNewLocalFileSystemLoader(dir))}
	t.ts.Debug = debug
	t.cache = map[string]*pongo2.Template{}
	return t
}

func (c *Pongo) GetTs() *pongo2.TemplateSet {
	return c.ts
}

func (c *Pongo) AddFunc(funcName string, funcEntry interface{}) {
	//todo 提交函数 pongo内容是怎么
	//pongo2.RegisterFilter(funcName,)
}

func (c *Pongo) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	var (
		tpl *pongo2.Template
		ok  bool
		err error
	)
	c.l.RLock()
	tpl, ok = c.cache[name]
	c.l.RUnlock()
	if !ok {
		tpl, err = c.ts.FromCache(name)
		if err != nil {
			return err
		}
		c.l.Lock()
		c.cache[name] = tpl
		c.l.Unlock()
	}
	return tpl.ExecuteWriter(pongo2.Context(binding), writer)
}

func (_ *Pongo) JSON(writer io.Writer, v map[string]interface{}) error {
	b, err := json.Marshal(v)
	if err == nil {
		_, err = writer.Write(b)
	}
	return err
}

func (_ *Pongo) JSONP(writer io.Writer, callback string, v map[string]interface{}) error {
	return nil
}

func (_ *Pongo) Text(writer io.Writer, v []byte) error {
	_, err := writer.Write(v)
	return err
}

func (_ *Pongo) XML(writer io.Writer, v map[string]interface{}) error {
	b, err := xml.Marshal(v)
	if err == nil {
		_, err = writer.Write(b)
	}
	return err
}
