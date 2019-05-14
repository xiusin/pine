package renderer

import (
	"io"
	"net/http"

	"github.com/unrolled/render"
)

type
(
	Renderer struct {
		engine *render.Render
	}
	Data map[string]interface{}		// 提供给前端使用
)

func New(options Options) *Renderer {
	return &Renderer{engine: render.New(warpOpt(&options))}
}

func warpOpt(opt *Options) render.Options {
	option := render.Options{}
	if opt.Directory != "" {
		option.Directory = opt.Directory
	}
	if opt.Asset != nil {
		option.Asset = opt.Asset
	}
	if opt.AssetNames != nil {
		option.AssetNames = opt.AssetNames
	}
	if opt.Layout != "" {
		option.Layout = opt.Layout
	}
	if opt.Extensions != nil {
		option.Extensions = opt.Extensions
	}
	if opt.Funcs != nil {
		option.Funcs = opt.Funcs
	}
	option.Delims.Left = opt.Delims.Left
	option.Delims.Right = opt.Delims.Right
	if opt.Charset == "" {
		option.Charset = opt.Charset
	}
	option.DisableCharset = opt.DisableCharset
	option.IndentJSON = opt.IndentJSON
	option.IndentXML = opt.IndentXML
	option.IsDevelopment = opt.IsDevelopment
	option.UnEscapeHTML = opt.UnEscapeHTML
	option.StreamingJSON = opt.StreamingJSON
	option.RequirePartials = opt.RequirePartials
	option.DisableHTTPErrorRendering = opt.DisableHTTPErrorRendering
	option.RenderPartialsWithoutPrefix = opt.RenderPartialsWithoutPrefix
	if opt.PrefixJSON != nil {
		option.PrefixJSON = opt.PrefixJSON
	}
	if opt.PrefixXML != nil {
		option.PrefixXML = opt.PrefixXML
	}
	return option
}

func (r *Renderer) Data(writer io.Writer, v string) error {
	return r.engine.Data(writer, http.StatusOK, []byte(v))
}

func (r *Renderer) HTML(writer io.Writer, name string, v interface{}) error {
	return r.engine.HTML(writer, http.StatusOK, name, v)
}

func (r *Renderer) JSON(writer io.Writer, v interface{}) error {
	return r.engine.JSON(writer, http.StatusOK, v)
}

func (r *Renderer) JSONP(writer io.Writer, callback string, v interface{}) error {
	return r.engine.JSONP(writer, http.StatusOK, callback, v)
}

func (r *Renderer) Text(writer io.Writer, v string) error {
	return r.engine.Text(writer, http.StatusOK, v)
}

func (r *Renderer) XML(writer io.Writer, v interface{}) error {
	return r.engine.XML(writer, http.StatusOK, v)
}
