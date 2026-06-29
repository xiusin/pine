// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package debug

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/xiusin/pine"
	"github.com/xiusin/pine/middlewares/debug/collector"
)

var (
	once          sync.Once
	codeLineNum   = 60
	codeMiddle    = codeLineNum / 2
	debugTemplate *template.Template
)

type errHandler struct {
	fileContent   []string
	firstFileCode string
	firstFile     string
	firstLine     int
	line          int
}

// DebugBar 调试栏中间件.
// 流程: 注册 collector -> 广播 ctx -> Next() -> 采集 -> 构建 HTML -> 注入响应 -> 销毁.
func DebugBar(enable bool) pine.Handler {
	return func(ctx *pine.Context) {
		collectorMgr := NewCollectorMgr(ctx, enable)
		ctx.Set("collectorMgr", collectorMgr)
		// 显式注册 collector (NewCollectorMgr 不再预注册, 避免重复)
		collectorMgr.RegisterCollector(
			collector.NewServerDataCollector(),
			collector.NewRequestDataCollector(),
		)
		// 将 ctx 广播给所有 collector, Collect 才能读取请求数据
		collectorMgr.SetContext(ctx)
		ctx.Next()
		// 在 Next() 之后采集, 可捕获 handler 中对 session 等的修改
		collectorMgr.Collect()

		if ctx.Response.StatusCode() == http.StatusOK {
			// 构建 debug 栏 HTML 并注入响应; 仅对 HTML 响应追加, 避免破坏 JSON / 文件流等
			if html, err := collectorMgr.BuildHtmlTag(); err == nil && html != "" {
				ct := ctx.Response.Header().Get(pine.HeaderContentType)
				if strings.HasPrefix(ct, pine.ContentTypeHTML) {
					ctx.Response.SetBodyString(string(ctx.Response.Body()) + html)
				}
			}
		}
		collectorMgr.Destroy()
	}
}

// Recover 返回 panic 恢复处理器, 输出调试页面.
// 每次请求创建独立的 errHandler 实例, 避免并发请求数据竞争.
//
// 注意: 本 Handler 通常通过 app.SetRecoverHandler 注册, 由 endRequest 在
// recover() 之后调用. 此时原始 panic 堆栈已被 endRequest 消费, debug.Stack()
// 只能拿到当前 (recoverHandler) 调用栈, 并非触发 panic 的原始栈.
// 这里至少记录 "panic 已发生" (c.Msg) 与当前调用栈, 便于定位问题.
// 若需捕获原始 panic 栈, 应改用 app.Use() 注册的中间件模式 (defer recover()),
// 但会改变与 SetRecoverHandler 的兼容用法, 暂未采用.
func Recover(r *pine.Application) pine.Handler {
	once.Do(func() {
		_, f, _, _ := runtime.Caller(0)
		p := path.Dir(f)
		debugTemplate, _ = template.ParseFiles(path.Join(p, "assets/debug.html"))
		r.Static("/debug_static", path.Join(p, "assets"))
	})
	return func(c *pine.Context) {
		handler := &errHandler{}
		handler.init()
		// 此时 panic 已被 endRequest recover, 这里获取的是 recoverHandler 调用栈
		stack := string(debug.Stack())
		c.Response.ResetBody()
		c.Logger().Info("msg: %s  Method: %s  Path: %s", c.Msg, c.Method(), c.Path())
		if c.IsAjax() {
			c.Response.Header().Add("Content-Type", pine.ContentTypeJSON)
			_ = c.Write(handler.showTraceInfo(c.Msg, stack, true))
		} else {
			c.Response.Header().Add("Content-Type", pine.ContentTypeHTML)
			handler.errors(c, c.Msg, handler.showTraceInfo(c.Msg, stack, false))
		}
	}
}

func (e *errHandler) init() {
	e.firstLine = 0
	e.fileContent = []string{}
	e.firstFile = ""
	e.firstFileCode = ""
}

func (e *errHandler) errors(c *pine.Context, errmsg string, trace []byte) {
	if debugTemplate == nil {
		// 模板解析失败, 降级输出纯文本错误
		c.Response.Header().Set("Content-Type", pine.ContentTypeText)
		_ = c.Write([]byte(errmsg + "\n\n" + string(trace)))
		return
	}
	jsData, _ := json.Marshal(e.fileContent)
	var buf bytes.Buffer
	if err := debugTemplate.Execute(&buf, map[string]any{
		"stack":     template.HTML(trace),
		"error":     errmsg,
		"fileMap":   string(jsData),
		"firstLine": strconv.Itoa(e.firstLine),
		"firstCode": e.firstFileCode,
		"fistFile":  e.firstFile,
		"line":      e.line,
	}); err != nil {
		c.Logger().Error("debug template execute: " + err.Error())
		return
	}
	c.Write(buf.Bytes())
}

func (e *errHandler) showTraceInfo(errMsg, traceMsg string, isAjax bool) []byte {
	msgs := strings.Split(strings.Trim(traceMsg, "\n"), "\n")[1:]
	var trace []map[string]string
	var fileContentMap []string

	// 确保 msgs 长度为偶数, 避免下方 i+1 越界
	if len(msgs)%2 != 0 {
		msgs = msgs[:len(msgs)-1]
	}
	l, idx, jsonRet, buf := len(msgs), 1, map[string]any{}, bytes.NewBuffer([]byte{})
	for i := 0; i < l; i += 2 {
		paths := strings.SplitN(msgs[i+1], ":", 2)
		if len(paths) < 2 {
			continue
		}
		paths[0] = strings.Trim(paths[0], "\t")

		if strings.Contains(msgs[i], "debug.Stack()") ||
			strings.Contains(msgs[i], "endRequest") ||
			strings.Contains(paths[0], "panic.go") ||
			strings.Contains(paths[0], "net/http") ||
			strings.Contains(paths[0], "debug.go") {
			continue
		}

		// 读取文件内容
		codeContent, _ := os.ReadFile(paths[0])
		line := strings.Split(paths[1], " ")
		lineNum, _ := strconv.Atoi(line[0])
		codes := strings.Split(string(codeContent), "\n")
		ln, _ := strconv.Atoi(line[0])
		// 边界检查: 行号必须在有效范围内
		if ln < 1 || ln > len(codes) {
			continue
		}
		codes[ln-1] = codes[ln-1] + "	  			//	 <-----   Here"
		count := len(codes)
		var firstLine int

		if count-lineNum < codeMiddle && count-codeLineNum > 0 {
			firstLine = count - codeLineNum
			codes = codes[count-codeLineNum:]
		} else if lineNum < codeMiddle && count > codeLineNum {
			codes = codes[:]
			firstLine = 0
		} else {
			var start int
			var end int
			if lineNum > codeMiddle {
				start = lineNum - codeMiddle
			}
			if lineNum+codeMiddle > count {
				end = count
			} else {
				end = lineNum + codeMiddle
			}
			firstLine = start
			codes = codes[start:end]
		}
		s := strings.Join(codes, "\n")
		fileContentMap = append(fileContentMap, s)
		if isAjax {
			trace = append(trace, map[string]string{
				"file": paths[0],
				"line": line[0],
				"func": msgs[i],
			})
		} else {
			buf.WriteString(`<div class="__BtrD__loop-tog __BtrD__l-parent" data-id="proc-`)
			buf.WriteString(strconv.Itoa(idx) + `" title="_GLOBAL" data-file="` + paths[0])
			buf.WriteString(`" data-class="trigger_error" data-fline="` + strconv.Itoa(firstLine) + `" data-line="`)
			buf.WriteString(line[0] + `"><div class="__BtrD__id __BtrD__loop-tog __BtrD__code">`)
			buf.WriteString(strconv.Itoa(idx) + `</div><div class="__BtrD__holder"><span class="__BtrD__name">`)
			buf.WriteString(msgs[i] + `</b><i class="__BtrD__line">` + line[0] + `</i></span><span class="__BtrD__path">`)
			buf.WriteString(paths[0] + `</span></div></div>`)
		}
		idx++
		if e.firstFileCode == "" {
			jsonRet["file"] = paths[0]
			jsonRet["line"] = firstLine + 1
			e.firstFileCode = s
			e.firstFile = paths[0]
			e.firstLine = firstLine + 1
			e.line = ln
		}
	}
	if isAjax {
		jsonRet["trace"] = trace
		jsonRet["message"] = errMsg
		s, _ := json.Marshal(jsonRet)
		return s
	}
	e.fileContent = fileContentMap
	return buf.Bytes()
}
