package debug

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"path"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/xiusin/pine"
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

var defaultHandler = &errHandler{}

func DebugBar() pine.Handler {
	return func(ctx *pine.Context) {

	}
}

func Recover(r *pine.Application) pine.Handler {
	once.Do(func() {
		_, f, _, _ := runtime.Caller(0)
		p := path.Dir(f)
		debugTemplate, _ = template.ParseFiles(path.Join(p, "assets/debug.html"))
		r.Static("/debug_static", path.Join(p, "assets"), 1)
	})
	return func(c *pine.Context) {
		defaultHandler.init()
		stack := string(debug.Stack())
		c.ResetBody()
		c.Logger().Printf("msg: %s  Method: %s  Path: %s", c.Msg, c.Method(), c.Path())
		if c.IsAjax() {
			c.Response.Header.Add("Content-Type", "application/json")
			c.Write(defaultHandler.showTraceInfo(c.Msg, stack, true))
		} else {
			c.Response.Header.Add("Content-Type", pine.ContentTypeHTML)
			defaultHandler.errors(c, c.Msg, defaultHandler.showTraceInfo(c.Msg, stack, false))
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
	jsData, _ := json.Marshal(e.fileContent)
	var buf bytes.Buffer
	if err := debugTemplate.Execute(&buf, map[string]interface{}{
		"stack":     template.HTML(trace),
		"error":     errmsg,
		"fileMap":   string(jsData),
		"firstLine": strconv.Itoa(e.firstLine),
		"firstCode": e.firstFileCode,
		"fistFile":  e.firstFile,
		"line":      e.line,
	}); err != nil {
		panic(err.Error())
	}
	c.Write(buf.Bytes())
}

func (e *errHandler) showTraceInfo(errMsg, traceMsg string, isAjax bool) []byte {
	msgs := strings.Split(strings.Trim(traceMsg, "\n"), "\n")[1:]
	var trace []map[string]string
	var fileContentMap []string

	l, idx, jsonRet, buf := len(msgs), 1, map[string]interface{}{}, bytes.NewBuffer([]byte{})
	for i := 0; i < l; i += 2 {
		paths := strings.Split(msgs[i+1], ":")
		paths[0] = strings.Trim(paths[0], "\t")

		if strings.Contains(msgs[i], "debug.Stack()") ||
			strings.Contains(msgs[i], "endRequest") ||
			strings.Contains(paths[0], "panic.go") ||
			strings.Contains(paths[0], "valyala/fasthttp") ||
			strings.Contains(paths[0], "debug.go") {
			continue
		}

		// 读取文件内容
		codeContent, _ := ioutil.ReadFile(paths[0])
		line := strings.Split(paths[1], " ")
		lineNum, _ := strconv.Atoi(line[0])
		codes := strings.Split(string(codeContent), "\n")
		ln, _ := strconv.Atoi(line[0])
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
	} else {
		e.fileContent = fileContentMap
		return buf.Bytes()
	}

}
