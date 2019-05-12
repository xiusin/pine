package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"runtime/debug"
	//"fmt"
	//"runtime/debug"
	"strconv"
	"strings"
	"text/template"
)

// 错误信息
type Errors interface {
	Error40x(c *Context, msg string)
	Error50x(c *Context, msg string)
	Recover(c *Context) func()
}

var DefaultErrorHandler = &ErrHandler{}

type ErrHandler struct {
	errMsg        string
	fileContent   []string
	firstFileCode string
}

func (e *ErrHandler) Error40x(c *Context, msg string) {
	_, _ = c.res.Write([]byte(msg))
}

func (e *ErrHandler) Error50x(c *Context, msg string) {
	tpl := template.New("example")
	jsData, _ := json.Marshal(e.fileContent)

	tpl, err := tpl.Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Xiusin Debug</title>
	<style>` + string(e.css()) + `</style>
	<script>` + string(e.js()) + `</script>
</head>
<body>
<div style="font-family:Inconsolata !important;font-weight:bold !important;line-height:1.3 !important;font-size:14px !important;">
    <div class="__BtrD__container-fluid">
        <div class="__BtrD__row __BtrD__contents">
            <div class="__BtrD__col-md-4 __BtrD__attr __BtrD__left">
                <div class="__BtrD__content-nav" id="cont-nav">
                    <div class="__BtrD__top-tog __BtrD__active" id="__BtrD__function">Trace</div>
                </div>
                <div class="__BtrD__content-body">
                    <div class="__BtrD__function __BtrD__loops __BtrD__active">
                        {{.stack}}
                    </div>
                </div>
            </div>
            <div class="__BtrD__col-md-8 __BtrD__attr __BtrD__middle ">
                <div class="__BtrD__exception-type">
                    <span>Panic Error</span>
                </div>
                <div class="__BtrD__exception-msg"><span class="__BtrD__char-string">{{.error}}</span></div>
                <div class="__BtrD__code-view" id="proc-main">
	<textarea id="code-go" name="code" style="display:none">` + e.firstFileCode + `</textarea>
                </div>
                <div class="__BtrD__active-desc" id="repop">
                    <div class="__BtrD__keyword">Class: <span class="__BtrD__char-null">null</span></div>
                    <div class="__BtrD__namespace">Namespace: <span class="__BtrD__char-null">null</span></div>
                    <div class="__BtrD__file">File: C:\wamp64\www\Dbug\index.php:<span
                                class="__BtrD__char-integer">31</span></div>

                </div>
            </div>
        </div>
    </div>
</div>
<script>
var fileMap = ` + string(jsData) + `
var editor = CodeMirror.fromTextArea(document.getElementById('code-go'), {
    mode: "go",
    lineNumbers: true,
	lineWrapping: true,
	firstLineNumber: 10,
	readOnly: true,
	theme: 'cobalt',
    extraKeys: {"Ctrl-Q": function(cm){ cm.foldCode(cm.getCursor()); }},
    foldGutter: true,
    gutters: ["CodeMirror-linenumbers", "CodeMirror-foldgutter"]
  });
 editor.setSize('auto','660px');
</script>
</body>
</html>`)
	if err != nil {
		log.Println(err)
		return
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]string{
		"stack": msg,
		"error": e.errMsg,
	}); err != nil {
		log.Println(err.Error())
	}
	_, _ = c.res.Write(buf.Bytes())
}

func (e *ErrHandler) Recover(c *Context) func() {
	return func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			//logrus.Printf(
			//	"msg: %s  Method: %s  Path: %s\n Stack: %s",
			//	err,
			//	c.Request().Method,
			//	c.Request().URL.Path,
			//	stack,
			//)
			c.SetStatus(500)
			e.errMsg = fmt.Sprintf("%s", err)
			e.Error50x(c, e.ShowTraceInfo(string(stack)))
		}
	}
}

func (e *ErrHandler) ShowTraceInfo(msg string) string {
	msgs := strings.Split(strings.Trim(msg, "\n"), "\n")
	var str string
	msgs = msgs[1:]
	l := len(msgs)
	idx := 1
	fileContentMap := []string{}
	for i := 0; i < l; i += 2 {

		path := strings.Split(msgs[i+1], ":")
		path[0] = strings.Trim(path[0], "\t")
		// 读取文件内容
		codeContent, _ := ioutil.ReadFile(path[0])
		line := strings.Split(path[1], " ")
		lineNum, _ := strconv.Atoi(line[0])
		codes := strings.Split(string(codeContent), "\n")
		count := len(codes)
		if count-lineNum < 15 && count-30 > 0 {
			codes = codes[count-30:]
		} else if lineNum < 15 && count > 30 {
			codes = codes[:]
		} else {
			var start int
			var end int
			if lineNum > 15 {
				start = lineNum - 15
			}
			if lineNum+15 > count {
				end = count
			} else {
				end = lineNum + 15
			}
			codes = codes[start:end]
		}
		s := strings.Join(codes, "\n")
		fileContentMap = append(fileContentMap, s)

		str += `<div class="__BtrD__loop-tog __BtrD__l-parent" data-id="proc-` +
			strconv.Itoa(idx) + `" title="_GLOBAL" data-file="` +
			path[0] + `" data-class="trigger_error" data-type="asdas()" data-function="" data-line="` +
			line[0] + `"><div class="__BtrD__id __BtrD__loop-tog __BtrD__code">` +
			strconv.Itoa(idx) + `</div><div class="__BtrD__holder"><span class="__BtrD__name">` +
			msgs[i] + `</b><i class="__BtrD__line">` +
			line[0] + `</i></span><span class="__BtrD__path">` +
			path[0] + `</span></div></div>`
		idx++
		if e.firstFileCode == "" {
			e.firstFileCode = s
		}
	}
	e.fileContent = fileContentMap
	return str
}

func (e *ErrHandler) css() []byte {
	return []byte(`

	@font-face {
        font-family: Inconsolata;
        src: url(data:font/truetype;charset=utf-8;base64,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) format("woff2");
    }
	pre {
	margin: 0;
	}
    .__BtrD__middle, body {
        overflow: hidden !important
    }

    .__BtrD__contents .__BtrD__right .__BtrD__global, .__BtrD__left .__BtrD__content-body .__BtrD__loops .__BtrD__holder:hover, .__BtrD__left .__BtrD__content-nav .__BtrD__top-tog {
        cursor: pointer
    }

    .__BtrD__contents .__BtrD__listed:last-child, .__BtrD__contents .__BtrD__right .__BtrD__global:last-child {
        border-bottom: none
    }

    html {
        font-family: sans-serif;
        -webkit-text-size-adjust: 100%;
        -ms-text-size-adjust: 100%;
        font-size: 10px;
        -webkit-tap-highlight-color: transparent
    }

    body {
        margin: 0 !important
    }

    .__BtrD__container, .__BtrD__container-fluid {
        margin-right: auto;
        margin-left: auto;
        padding-right: 15px;
        padding-left: 15px
    }

    .__BtrD__caret, .__BtrD__header .__BtrD__hints .__BtrD__type, .__BtrD__left .__BtrD__content-nav .__BtrD__top-tog {
        display: inline-block
    }

    *, :after, :before {
        -webkit-box-sizing: border-box;
        -moz-box-sizing: border-box;
        box-sizing: border-box
    }

    @media (min-width: 768px) {
        .__BtrD__container {
            width: 750px
        }
    }

    @media (min-width: 992px) {
        .__BtrD__container {
            width: 970px
        }
    }

    @media (min-width: 1200px) {
        .__BtrD__container {
            width: 1170px
        }
    }

    .__BtrD__row {
        margin-right: -15px;
        margin-left: -15px
    }

    .__BtrD__col-md-4, .__BtrD__col-md-6 {
        position: relative;
        min-height: 1px;
        padding-right: 15px;
        padding-left: 15px
    }

    @media (min-width: 992px) {
        .__BtrD__col-md-4, .__BtrD__col-md-6 {
            float: left
        }

        .__BtrD__col-md-8 {
            width: 65%
        }

        .__BtrD__col-md-4 {
            width: 33.33333333%
        }
    }

    .__BtrD__caret {
        width: 0;
        height: 0;
        margin-left: 2px;
        vertical-align: middle;
        border-top: 4px dashed;
        border-top: 4px solid \9;
        border-right: 4px solid transparent;
        border-left: 4px solid transparent
    }

    .__BtrD__header {
        padding: 5px;
        height: 35px;
        width: 100%
    }

    .__BtrD__header .__BtrD__hints {
        position: absolute;
        right: 0
    }

    .__BtrD__header .__BtrD__type {
        margin-left: 10px;
        width: 70px;
        text-align: center;
        border-radius: 10px;
        font-size: 10px;
        margin-top: 5px
    }

    .__BtrD__header .__BtrD__logo span, .__BtrD__header .__BtrD__tops {
        display: inline-block;
        margin-left: 20px;
        vertical-align: middle
    }

    .__BtrD__header .__BtrD__logo-img {
        width: 80px;
        height: 30px;
        margin-top: -2px
    }

    .__BtrD__header .__BtrD__logo span button, .__BtrD__header .__BtrD__logo span button:active, .__BtrD__header .__BtrD__logo span button:active:hover {
        border: none
    }

    .__BtrD__header .__BtrD__logo span .__BtrD__dropdown-menu li a {
        padding: 8px 30px
    }

    .__BtrD__header .__BtrD__logo span .__BtrD__dropdown-menu .__BtrD__divider {
        margin: 0
    }

    .__BtrD__contents .__BtrD__attr {
        height: 96vh;
        padding: 0 !important
    }

    .__BtrD__contents .__BtrD__right {
        word-wrap: break-word;
        font-weight: 600 !important
    }

    .__BtrD__left .__BtrD__content-nav .__BtrD__top-tog {
        font-weight: 600;
        width: 34%;
        text-align: center;
        margin-left: -8px;
        padding-top: 5px;
        font-size: 12px;
        padding-bottom: 5px;
        text-transform: uppercase
    }

    .__BtrD__left .__BtrD__content-nav .__BtrD__top-tog:first-child {
        padding-left: 10px;
        border-left: none;
        margin-right: -1px
    }

    .__BtrD__left .__BtrD__content-nav .__BtrD__top-tog:last-child {
        margin-left: -7px;
        border-right: none
    }

    .__BtrD__left .__BtrD__content-body {
        height: 95%
    }

    .__BtrD__l-parent {
        margin-left: 3px;
        margin-top: 1px
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops {
        margin-top: 3px;
        width: 100%;
        display: none
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops .__BtrD__holder i {
        float: right;
        margin-top: 10px;
        font-size: 10px;
        border-radius: 10px;
        padding: 1px 10px;
        font-style: normal
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops .__BtrD__holder {
        padding: 5px 10px
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops div span {
        display: block
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops div .__BtrD__name {
        font-weight: 600;
        padding-left: 10px
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops div .__BtrD__path {
        font-size: 11px
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops .__BtrD__id {
        width: 15px;
        font-size: 10px;
        text-align: center;
        border: none;
        -webkit-box-shadow: none;
        -moz-box-shadow: none;
        box-shadow: none;
        position: absolute;
        left: 4px;
        padding-top: 5px;
        padding-bottom: 5px;
        display: inline-block;
        vertical-align: middle
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops.__BtrD__active {
        display: block
    }

    .__BtrD__exception-type {
        padding: 10px
    }

    .__BtrD__exception-type > .__BtrD__action, .__BtrD__exception-type > span {
        display: inline-block
    }

    .__BtrD__exception-type > .__BtrD__action {
        float: right
    }

    .__BtrD__exception-type > .__BtrD__action span {
        cursor: pointer;
        font-size: 12px;
        padding: 0 10px 2px;
        border-radius: 10px
    }

    .__BtrD__exception-type > span {
        border-radius: 10px;
        padding: 3px 10px
    }

    .__BtrD__exception-msg {
        padding: 10px;
        font-size: 13px
    }

    .__BtrD__active-desc {
        position: absolute;
        width: 100%;
        bottom: 0;
        padding: 10px;
        font-size: 12px;
        z-index: 4 !important
    }

    .__BtrD__active-desc .__BtrD__file b {
        font-size: 10px;
        margin-top: -5px
    }

    .__BtrD__browser-view {
        width: 103%
    }

    .__BtrD__code-view table {
        font-size: 12px;
        font-weight: 400;
        width: 100%;
        white-space: nowrap;
        table-layout: fixed;
        border-collapse: collapse
    }

    .__BtrD__code-view table .function, .__BtrD__code-view table .keyword {
        font-weight: 600
    }

    .__BtrD__code-view table tr td:first-child {
        width: 60px;
        text-align: center
    }

    .__BtrD__code-view table tr .__BtrD__line-content {
        padding-left: 5px
    }

    .__BtrD__code-view table .__BtrD__highlighted td {
        padding: 8px 0 !important
    }

    .__BtrD__code-view table .__BtrD__highlighted td:first-child {
        border: none;
        font-weight: 700
    }

    .__BtrD__contents .__BtrD__right .__BtrD__global:first-child {
        border-top: none
    }

    .__BtrD__contents .__BtrD__right .__BtrD__global .__BtrD__env-arr {
        padding-left: 20px
    }

    .__BtrD__contents .__BtrD__right .__BtrD__global .__BtrD__content {
        cursor: default
    }

    .__BtrD__contents .__BtrD__right .__BtrD__global .__BtrD__content .__BtrD__caret {
        cursor: pointer
    }

    .__BtrD__contents .__BtrD__right .__BtrD__global .__BtrD__content, .__BtrD__contents .__BtrD__right .__BtrD__global .__BtrD__labeled {
        display: block
    }

    .__BtrD__contents .__BtrD__right .__BtrD__global .__BtrD__labeled {
        font-weight: 600;
        padding: 10px
    }

    .__BtrD__contents .__BtrD__listed {
        padding: 5px 14px 5px 10px
    }

    .__BtrD__contents .__BtrD__right .__BtrD__global .__BtrD__content {
        padding: 5px 0;
        width: 104%
    }

    .__BtrD__contents .__BtrD__listed .__BtrD__index, .__BtrD__contents .__BtrD__listed .__BtrD__value {
        font-weight: 500;
        font-size: 10px
    }

    .__BtrD__ss-wrapper {
        overflow: hidden;
        width: 100%;
        height: 100%;
        position: relative;
        float: left
    }

    .__BtrD__ss-content {
        height: 100%;
        width: 104%;
        position: relative;
        overflow: auto;
        box-sizing: border-box
    }

    .__BtrD__ss-scroll {
        position: relative;
        width: 6px;
        border-radius: 4px;
        top: 0;
        z-index: 2;
        cursor: pointer;
        opacity: 0;
        transition: opacity .25s linear
    }

    .__BtrD__ss-hidden {
        display: none
    }

    .__BtrD__ss-container:hover .__BtrD__ss-scroll {
        opacity: 1
    }

    .__BtrD__ss-grabbed {
        -o-user-select: none;
        -ms-user-select: none;
        -moz-user-select: none;
        -webkit-user-select: none;
        user-select: none
    }

    .__BtrD__header, .__BtrD__left .__BtrD__content-nav {
        border-bottom: 1px solid #000
    }

    body {
        background: #343c45;
        color: #f5f7fa !important
    }

    .__BtrD__header {
        background: #252c33
    }

    .__BtrD__header .__BtrD__type {
        color: #fff
    }

    .__BtrD__header .__BtrD__logo span button, .__BtrD__header .__BtrD__logo span button:active, .__BtrD__header .__BtrD__logo span button:active:hover {
        color: #c1c9d4 !important;
        background: 0 0 !important
    }

    .__BtrD__header .__BtrD__logo span .__BtrD__dropdown-menu {
        background: #252c33
    }

    .__BtrD__header .__BtrD__logo span .__BtrD__dropdown-menu li a {
        color: #c1c9d4
    }

    .__BtrD__header .__BtrD__logo span .__BtrD__dropdown-menu li a:hover {
        background-color: #2a3138
    }

    .__BtrD__header .__BtrD__logo span .__BtrD__dropdown-menu .__BtrD__divider {
        background-color: transparent;
        border-top: 1px solid #343c45;
        border-bottom: 1px solid #000
    }

    .__BtrD__contents, .__BtrD__left .__BtrD__content-body {
        border-top: 1px solid #555e65
    }

    .__BtrD__contents .__BtrD__attr {
        color: #d8dadd
    }

    .__BtrD__contents .__BtrD__left {
        background: #343c45;
        border-right: 1px solid #555e65
    }

    .__BtrD__contents .__BtrD__middle {
        background: #131c26;
        border-left: 1px solid #000;
        border-right: 1px solid #000
    }

    .__BtrD__contents .__BtrD__right {
        background: #363e47;
        border-left: 1px solid #555e65
    }

    .__BtrD__left .__BtrD__content-nav .__BtrD__top-tog {
        color: #d8dadd;
        border-right: 1px solid #000;
        border-left: 1px solid #555e65
    }

    .__BtrD__left .__BtrD__content-nav .__BtrD__active, .__BtrD__left .__BtrD__content-nav .__BtrD__top-tog:hover {
        background: #252c33;
        border-left: 1px solid #000
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops .__BtrD__loop-tog {
        border: 1px solid #14171d;
        background: #343c45;
        -webkit-box-shadow: 0 1px 5px 0 rgba(71, 80, 88, 1);
        -moz-box-shadow: 0 1px 5px 0 rgba(71, 80, 88, 1);
        box-shadow: 0 1px 5px 0 rgba(71, 80, 88, 1)
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops .__BtrD__holder:hover {
        background: #222731
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops .__BtrD__holder i {
        background: #1f577f;
        color: #fff;
        box-shadow: 0 1px #222731
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops .__BtrD__holder b {
        color: #aee288
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops .__BtrD__holder {
        border: 1px solid #6b757e
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops div .__BtrD__path {
        color: #95a1a9
    }

    .__BtrD__left .__BtrD__content-body .__BtrD__loops .__BtrD__id {
        background: #14171d
    }

    .__BtrD__exception-type {
        background: #343c45;
        border-bottom: 1px solid #000
    }

    .__BtrD__exception-type > .__BtrD__action span {
        background: #272e37;
        border-top: 1px solid #161821;
        border-bottom: 1px solid #3f4750
    }

    .__BtrD__exception-type > span {
        background: #6789f8
    }

    .__BtrD__exception-msg {
        background: #343c45;
        border-top: 1px solid #555e65;
        border-bottom: 1px solid #555e65
    }

    .__BtrD__browser-view, .__BtrD__code-view {
        border-top: 1px solid #000
    }

    .__BtrD__code-view table tr td:first-child {
        background: #24374b;
        border-right: 1px solid #0b1016
    }

    .__BtrD__code-view table .__BtrD__highlighted td {
        background: #24374b;
        border-top: 1px solid #000;
        border-bottom: 1px solid #000
    }

    .__BtrD__code-view table .__BtrD__highlighted td:first-child {
        color: #ec4c4e
    }

    .__BtrD__active-desc {
        background: #343c45;
        border-top: 1px solid #555e65
    }

    .__BtrD__contents .__BtrD__right .__BtrD__global {
        border-top: 1px solid #555e65;
        border-bottom: 1px solid #0d1012
    }

    .__BtrD__contents .__BtrD__listed {
        border-bottom: 1px solid #313a46
    }

    .__BtrD__contents .__BtrD__right .__BtrD__global .__BtrD__content {
        background: #131c26
    }

    .__BtrD__ss-scroll {
        background: rgba(0, 0, 0, .5)
    }

    .__BtrD__contents .__BtrD__listed .__BtrD__index, .__BtrD__contents .__BtrD__right .__BtrD__global .__BtrD__env-arr .__BtrD__key {
        color: #3689b2
    }

    .__BtrD__type-object {
        background: #5ba415
    }

    .__BtrD__type-null {
        background: #6789f8
    }

    .__BtrD__type-bool {
        background: #f8b93c
    }

    .__BtrD__type-array {
        background: #6db679
    }

    .__BtrD__type-float {
        background: #9C6E25
    }

    .__BtrD__type-double {
        background: #9a6cf3
    }

    .__BtrD__type-integer {
        background: #1BAABB
    }

    .__BtrD__type-string {
        background: #f99
    }

    .__BtrD__quote-string {
        color: #ff7c88
    }

    .__BtrD__char-object {
        color: #5ba415
    }

    .__BtrD__char-null {
        color: #6789f8
    }

    .__BtrD__char-bool {
        color: #f8b93c
    }

    .__BtrD__char-array {
        color: #6db679
    }

    .__BtrD__char-float {
        color: #9C6E25
    }

    .__BtrD__char-double {
        color: #9a6cf3
    }

    .__BtrD__char-string {
        color: #f99
    }

    .__BtrD__char-integer {
        color: #1BAABB
    }

    .__BtrD__char-type {
        color: #aee288
    }

    .__BtrD__header .__BtrD__logo .__BtrD__logo-img {
        background: url(data:image/png;base64,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) no-repeat;
        background-size: contain;
    }


/*编辑器样式*/

.CodeMirror {
  /* Set height, width, borders, and global font properties here */
  font-family: monospace;
  height: 300px;
  color: black;
  direction: ltr;
}

/* PADDING */

.CodeMirror-lines {
  padding: 4px 0; /* Vertical padding around content */
}
.CodeMirror pre {
  padding: 0 4px; /* Horizontal padding of content */
}

.CodeMirror-scrollbar-filler, .CodeMirror-gutter-filler {
  background-color: white; /* The little square between H and V scrollbars */
}

/* GUTTER */

.CodeMirror-gutters {
  border-right: 1px solid #ddd;
  background-color: #f7f7f7;
  white-space: nowrap;
}
.CodeMirror-linenumbers {}
.CodeMirror-linenumber {
  padding: 0 3px 0 5px;
  min-width: 20px;
  text-align: right;
  color: #999;
  white-space: nowrap;
}

.CodeMirror-guttermarker { color: black; }
.CodeMirror-guttermarker-subtle { color: #999; }

/* CURSOR */

.CodeMirror-cursor {
  border-left: 1px solid black;
  border-right: none;
  width: 0;
}
/* Shown when moving in bi-directional text */
.CodeMirror div.CodeMirror-secondarycursor {
  border-left: 1px solid silver;
}
.cm-fat-cursor .CodeMirror-cursor {
  width: auto;
  border: 0 !important;
  background: #7e7;
}
.cm-fat-cursor div.CodeMirror-cursors {
  z-index: 1;
}
.cm-fat-cursor-mark {
  background-color: rgba(20, 255, 20, 0.5);
  -webkit-animation: blink 1.06s steps(1) infinite;
  -moz-animation: blink 1.06s steps(1) infinite;
  animation: blink 1.06s steps(1) infinite;
}
.cm-animate-fat-cursor {
  width: auto;
  border: 0;
  -webkit-animation: blink 1.06s steps(1) infinite;
  -moz-animation: blink 1.06s steps(1) infinite;
  animation: blink 1.06s steps(1) infinite;
  background-color: #7e7;
}
@-moz-keyframes blink {
  0% {}
  50% { background-color: transparent; }
  100% {}
}
@-webkit-keyframes blink {
  0% {}
  50% { background-color: transparent; }
  100% {}
}
@keyframes blink {
  0% {}
  50% { background-color: transparent; }
  100% {}
}

/* Can style cursor different in overwrite (non-insert) mode */
.CodeMirror-overwrite .CodeMirror-cursor {}

.cm-tab { display: inline-block; text-decoration: inherit; }

.CodeMirror-rulers {
  position: absolute;
  left: 0; right: 0; top: -50px; bottom: -20px;
  overflow: hidden;
}
.CodeMirror-ruler {
  border-left: 1px solid #ccc;
  top: 0; bottom: 0;
  position: absolute;
}

div.CodeMirror span.CodeMirror-matchingbracket {color: #0b0;}
div.CodeMirror span.CodeMirror-nonmatchingbracket {color: #a22;}
.CodeMirror-matchingtag { background: rgba(255, 150, 0, .3); }
.CodeMirror-activeline-background {background: #e8f2ff;}

/* STOP */

/* The rest of this file contains styles related to the mechanics of
   the editor. You probably shouldn't touch them. */

.CodeMirror {
  position: relative;
  overflow: hidden;
  background: white;
}

.CodeMirror-scroll {
  overflow: scroll !important; /* Things will break if this is overridden */
  /* 30px is the magic margin used to hide the element's real scrollbars */
  /* See overflow: hidden in .CodeMirror */
  margin-bottom: -30px; margin-right: -30px;
  padding-bottom: 30px;
  height: 100%;
  outline: none; /* Prevent dragging from highlighting the element */
  position: relative;
}
.CodeMirror-sizer {
  position: relative;
  border-right: 30px solid transparent;
}

/* The fake, visible scrollbars. Used to force redraw during scrolling
   before actual scrolling happens, thus preventing shaking and
   flickering artifacts. */
.CodeMirror-vscrollbar, .CodeMirror-hscrollbar, .CodeMirror-scrollbar-filler, .CodeMirror-gutter-filler {
  position: absolute;
  z-index: 6;
  display: none;
}
.CodeMirror-vscrollbar {
  right: 0; top: 0;
  overflow-x: hidden;
  overflow-y: scroll;
}
.CodeMirror-hscrollbar {
  bottom: 0; left: 0;
  overflow-y: hidden;
  overflow-x: scroll;
}
.CodeMirror-scrollbar-filler {
  right: 0; bottom: 0;
}
.CodeMirror-gutter-filler {
  left: 0; bottom: 0;
}

.CodeMirror-gutters {
  position: absolute; left: 0; top: 0;
  min-height: 100%;
  z-index: 3;
}
.CodeMirror-gutter {
  white-space: normal;
  height: 100%;
  display: inline-block;
  vertical-align: top;
  margin-bottom: -30px;
}
.CodeMirror-gutter-wrapper {
  position: absolute;
  z-index: 4;
  background: none !important;
  border: none !important;
}
.CodeMirror-gutter-background {
  position: absolute;
  top: 0; bottom: 0;
  z-index: 4;
}
.CodeMirror-gutter-elt {
  position: absolute;
  cursor: default;
  z-index: 4;
}
.CodeMirror-gutter-wrapper ::selection { background-color: transparent }
.CodeMirror-gutter-wrapper ::-moz-selection { background-color: transparent }

.CodeMirror-lines {
  cursor: text;
  min-height: 1px; /* prevents collapsing before first draw */
}
.CodeMirror pre {
  /* Reset some styles that the rest of the page might have set */
  -moz-border-radius: 0; -webkit-border-radius: 0; border-radius: 0;
  border-width: 0;
  background: transparent;
  font-family: inherit;
  font-size: inherit;
  margin: 0;
  white-space: pre;
  word-wrap: normal;
  line-height: inherit;
  color: inherit;
  z-index: 2;
  position: relative;
  overflow: visible;
  -webkit-tap-highlight-color: transparent;
  -webkit-font-variant-ligatures: contextual;
  font-variant-ligatures: contextual;
}
.CodeMirror-wrap pre {
  word-wrap: break-word;
  white-space: pre-wrap;
  word-break: normal;
}

.CodeMirror-linebackground {
  position: absolute;
  left: 0; right: 0; top: 0; bottom: 0;
  z-index: 0;
}

.CodeMirror-linewidget {
  position: relative;
  z-index: 2;
  padding: 0.1px; /* Force widget margins to stay inside of the container */
}

.CodeMirror-widget {}

.CodeMirror-rtl pre { direction: rtl; }

.CodeMirror-code {
  outline: none;
}

/* Force content-box sizing for the elements where we expect it */
.CodeMirror-scroll,
.CodeMirror-sizer,
.CodeMirror-gutter,
.CodeMirror-gutters,
.CodeMirror-linenumber {
  -moz-box-sizing: content-box;
  box-sizing: content-box;
}

.CodeMirror-measure {
  position: absolute;
  width: 100%;
  height: 0;
  overflow: hidden;
  visibility: hidden;
}

.CodeMirror-cursor {
  position: absolute;
  pointer-events: none;
}
.CodeMirror-measure pre { position: static; }

div.CodeMirror-cursors {
  visibility: hidden;
  position: relative;
  z-index: 3;
}
div.CodeMirror-dragcursors {
  visibility: visible;
}

.CodeMirror-focused div.CodeMirror-cursors {
  visibility: visible;
}

.CodeMirror-selected { background: #d9d9d9; }
.CodeMirror-focused .CodeMirror-selected { background: #d7d4f0; }
.CodeMirror-crosshair { cursor: crosshair; }
.CodeMirror-line::selection, .CodeMirror-line > span::selection, .CodeMirror-line > span > span::selection { background: #d7d4f0; }
.CodeMirror-line::-moz-selection, .CodeMirror-line > span::-moz-selection, .CodeMirror-line > span > span::-moz-selection { background: #d7d4f0; }

.cm-searching {
  background-color: #ffa;
  background-color: rgba(255, 255, 0, .4);
}

/* Used to force a border model for a node */
.cm-force-border { padding-right: .1px; }

@media print {
  /* Hide the cursor when printing */
  .CodeMirror div.CodeMirror-cursors {
    visibility: hidden;
  }
}

/* See issue #2901 */
.cm-tab-wrap-hack:after { content: ''; }

/* Help users use markselection to safely style text background */
span.CodeMirror-selectedtext { background: none; }











/*主题*/
.cm-s-cobalt.CodeMirror { background: #131c26; color: white; }
.cm-s-cobalt div.CodeMirror-selected { background: #b36539; }
.cm-s-cobalt .CodeMirror-line::selection, .cm-s-cobalt .CodeMirror-line > span::selection, .cm-s-cobalt .CodeMirror-line > span > span::selection { background: rgba(179, 101, 57, .99); }
.cm-s-cobalt .CodeMirror-line::-moz-selection, .cm-s-cobalt .CodeMirror-line > span::-moz-selection, .cm-s-cobalt .CodeMirror-line > span > span::-moz-selection { background: rgba(179, 101, 57, .99); }
.cm-s-cobalt .CodeMirror-gutters { background: #131c26; border-right: 1px solid #aaa; }
.cm-s-cobalt .CodeMirror-guttermarker { color: #ffee80; }
.cm-s-cobalt .CodeMirror-guttermarker-subtle { color: #d0d0d0; }
.cm-s-cobalt .CodeMirror-linenumber { color: #d0d0d0; }
.cm-s-cobalt .CodeMirror-cursor { border-left: 1px solid white; }

.cm-s-cobalt span.cm-comment { color: #08f; }
.cm-s-cobalt span.cm-atom { color: #845dc4; }
.cm-s-cobalt span.cm-number, .cm-s-cobalt span.cm-attribute { color: #ff80e1; }
.cm-s-cobalt span.cm-keyword { color: #ffee80; }
.cm-s-cobalt span.cm-string { color: #3ad900; }
.cm-s-cobalt span.cm-meta { color: #ff9d00; }
.cm-s-cobalt span.cm-variable-2, .cm-s-cobalt span.cm-tag { color: #9effff; }
.cm-s-cobalt span.cm-variable-3, .cm-s-cobalt span.cm-def, .cm-s-cobalt .cm-type { color: white; }
.cm-s-cobalt span.cm-bracket { color: #d8d8d8; }
.cm-s-cobalt span.cm-builtin, .cm-s-cobalt span.cm-special { color: #ff9e59; }
.cm-s-cobalt span.cm-link { color: #845dc4; }
.cm-s-cobalt span.cm-error { color: #9d1e15; }

.cm-s-cobalt .CodeMirror-activeline-background { background: #002D57; }
.cm-s-cobalt .CodeMirror-matchingbracket { outline:1px solid grey;color:white !important; }

`)
}

func (e *ErrHandler) js() []byte {
	return []byte(`
// CodeMirror, copyright (c) by Marijn Haverbeke and others
// Distributed under an MIT license: https://codemirror.net/LICENSE

// This is CodeMirror (https://codemirror.net), a code editor
// implemented in JavaScript on top of the browser's DOM.
//
// You can find some technical background for some of the code below
// at http://marijnhaverbeke.nl/blog/#cm-internals .

(function (global, factory) {
  typeof exports === 'object' && typeof module !== 'undefined' ? module.exports = factory() :
  typeof define === 'function' && define.amd ? define(factory) :
  (global.CodeMirror = factory());
}(this, (function () { 'use strict';

  // Kludges for bugs and behavior differences that can't be feature
  // detected are enabled based on userAgent etc sniffing.
  var userAgent = navigator.userAgent;
  var platform = navigator.platform;

  var gecko = /gecko\/\d/i.test(userAgent);
  var ie_upto10 = /MSIE \d/.test(userAgent);
  var ie_11up = /Trident\/(?:[7-9]|\d{2,})\..*rv:(\d+)/.exec(userAgent);
  var edge = /Edge\/(\d+)/.exec(userAgent);
  var ie = ie_upto10 || ie_11up || edge;
  var ie_version = ie && (ie_upto10 ? document.documentMode || 6 : +(edge || ie_11up)[1]);
  var webkit = !edge && /WebKit\//.test(userAgent);
  var qtwebkit = webkit && /Qt\/\d+\.\d+/.test(userAgent);
  var chrome = !edge && /Chrome\//.test(userAgent);
  var presto = /Opera\//.test(userAgent);
  var safari = /Apple Computer/.test(navigator.vendor);
  var mac_geMountainLion = /Mac OS X 1\d\D([8-9]|\d\d)\D/.test(userAgent);
  var phantom = /PhantomJS/.test(userAgent);

  var ios = !edge && /AppleWebKit/.test(userAgent) && /Mobile\/\w+/.test(userAgent);
  var android = /Android/.test(userAgent);
  // This is woefully incomplete. Suggestions for alternative methods welcome.
  var mobile = ios || android || /webOS|BlackBerry|Opera Mini|Opera Mobi|IEMobile/i.test(userAgent);
  var mac = ios || /Mac/.test(platform);
  var chromeOS = /\bCrOS\b/.test(userAgent);
  var windows = /win/i.test(platform);

  var presto_version = presto && userAgent.match(/Version\/(\d*\.\d*)/);
  if (presto_version) { presto_version = Number(presto_version[1]); }
  if (presto_version && presto_version >= 15) { presto = false; webkit = true; }
  // Some browsers use the wrong event properties to signal cmd/ctrl on OS X
  var flipCtrlCmd = mac && (qtwebkit || presto && (presto_version == null || presto_version < 12.11));
  var captureRightClick = gecko || (ie && ie_version >= 9);

  function classTest(cls) { return new RegExp("(^|\\s)" + cls + "(?:$|\\s)\\s*") }

  var rmClass = function(node, cls) {
    var current = node.className;
    var match = classTest(cls).exec(current);
    if (match) {
      var after = current.slice(match.index + match[0].length);
      node.className = current.slice(0, match.index) + (after ? match[1] + after : "");
    }
  };

  function removeChildren(e) {
    for (var count = e.childNodes.length; count > 0; --count)
      { e.removeChild(e.firstChild); }
    return e
  }

  function removeChildrenAndAdd(parent, e) {
    return removeChildren(parent).appendChild(e)
  }

  function elt(tag, content, className, style) {
    var e = document.createElement(tag);
    if (className) { e.className = className; }
    if (style) { e.style.cssText = style; }
    if (typeof content == "string") { e.appendChild(document.createTextNode(content)); }
    else if (content) { for (var i = 0; i < content.length; ++i) { e.appendChild(content[i]); } }
    return e
  }
  // wrapper for elt, which removes the elt from the accessibility tree
  function eltP(tag, content, className, style) {
    var e = elt(tag, content, className, style);
    e.setAttribute("role", "presentation");
    return e
  }

  var range;
  if (document.createRange) { range = function(node, start, end, endNode) {
    var r = document.createRange();
    r.setEnd(endNode || node, end);
    r.setStart(node, start);
    return r
  }; }
  else { range = function(node, start, end) {
    var r = document.body.createTextRange();
    try { r.moveToElementText(node.parentNode); }
    catch(e) { return r }
    r.collapse(true);
    r.moveEnd("character", end);
    r.moveStart("character", start);
    return r
  }; }

  function contains(parent, child) {
    if (child.nodeType == 3) // Android browser always returns false when child is a textnode
      { child = child.parentNode; }
    if (parent.contains)
      { return parent.contains(child) }
    do {
      if (child.nodeType == 11) { child = child.host; }
      if (child == parent) { return true }
    } while (child = child.parentNode)
  }

  function activeElt() {
    // IE and Edge may throw an "Unspecified Error" when accessing document.activeElement.
    // IE < 10 will throw when accessed while the page is loading or in an iframe.
    // IE > 9 and Edge will throw when accessed in an iframe if document.body is unavailable.
    var activeElement;
    try {
      activeElement = document.activeElement;
    } catch(e) {
      activeElement = document.body || null;
    }
    while (activeElement && activeElement.shadowRoot && activeElement.shadowRoot.activeElement)
      { activeElement = activeElement.shadowRoot.activeElement; }
    return activeElement
  }

  function addClass(node, cls) {
    var current = node.className;
    if (!classTest(cls).test(current)) { node.className += (current ? " " : "") + cls; }
  }
  function joinClasses(a, b) {
    var as = a.split(" ");
    for (var i = 0; i < as.length; i++)
      { if (as[i] && !classTest(as[i]).test(b)) { b += " " + as[i]; } }
    return b
  }

  var selectInput = function(node) { node.select(); };
  if (ios) // Mobile Safari apparently has a bug where select() is broken.
    { selectInput = function(node) { node.selectionStart = 0; node.selectionEnd = node.value.length; }; }
  else if (ie) // Suppress mysterious IE10 errors
    { selectInput = function(node) { try { node.select(); } catch(_e) {} }; }

  function bind(f) {
    var args = Array.prototype.slice.call(arguments, 1);
    return function(){return f.apply(null, args)}
  }

  function copyObj(obj, target, overwrite) {
    if (!target) { target = {}; }
    for (var prop in obj)
      { if (obj.hasOwnProperty(prop) && (overwrite !== false || !target.hasOwnProperty(prop)))
        { target[prop] = obj[prop]; } }
    return target
  }

  // Counts the column offset in a string, taking tabs into account.
  // Used mostly to find indentation.
  function countColumn(string, end, tabSize, startIndex, startValue) {
    if (end == null) {
      end = string.search(/[^\s\u00a0]/);
      if (end == -1) { end = string.length; }
    }
    for (var i = startIndex || 0, n = startValue || 0;;) {
      var nextTab = string.indexOf("\t", i);
      if (nextTab < 0 || nextTab >= end)
        { return n + (end - i) }
      n += nextTab - i;
      n += tabSize - (n % tabSize);
      i = nextTab + 1;
    }
  }

  var Delayed = function() {this.id = null;};
  Delayed.prototype.set = function (ms, f) {
    clearTimeout(this.id);
    this.id = setTimeout(f, ms);
  };

  function indexOf(array, elt) {
    for (var i = 0; i < array.length; ++i)
      { if (array[i] == elt) { return i } }
    return -1
  }

  // Number of pixels added to scroller and sizer to hide scrollbar
  var scrollerGap = 30;

  // Returned or thrown by various protocols to signal 'I'm not
  // handling this'.
  var Pass = {toString: function(){return "CodeMirror.Pass"}};

  // Reused option objects for setSelection & friends
  var sel_dontScroll = {scroll: false}, sel_mouse = {origin: "*mouse"}, sel_move = {origin: "+move"};

  // The inverse of countColumn -- find the offset that corresponds to
  // a particular column.
  function findColumn(string, goal, tabSize) {
    for (var pos = 0, col = 0;;) {
      var nextTab = string.indexOf("\t", pos);
      if (nextTab == -1) { nextTab = string.length; }
      var skipped = nextTab - pos;
      if (nextTab == string.length || col + skipped >= goal)
        { return pos + Math.min(skipped, goal - col) }
      col += nextTab - pos;
      col += tabSize - (col % tabSize);
      pos = nextTab + 1;
      if (col >= goal) { return pos }
    }
  }

  var spaceStrs = [""];
  function spaceStr(n) {
    while (spaceStrs.length <= n)
      { spaceStrs.push(lst(spaceStrs) + " "); }
    return spaceStrs[n]
  }

  function lst(arr) { return arr[arr.length-1] }

  function map(array, f) {
    var out = [];
    for (var i = 0; i < array.length; i++) { out[i] = f(array[i], i); }
    return out
  }

  function insertSorted(array, value, score) {
    var pos = 0, priority = score(value);
    while (pos < array.length && score(array[pos]) <= priority) { pos++; }
    array.splice(pos, 0, value);
  }

  function nothing() {}

  function createObj(base, props) {
    var inst;
    if (Object.create) {
      inst = Object.create(base);
    } else {
      nothing.prototype = base;
      inst = new nothing();
    }
    if (props) { copyObj(props, inst); }
    return inst
  }

  var nonASCIISingleCaseWordChar = /[\u00df\u0587\u0590-\u05f4\u0600-\u06ff\u3040-\u309f\u30a0-\u30ff\u3400-\u4db5\u4e00-\u9fcc\uac00-\ud7af]/;
  function isWordCharBasic(ch) {
    return /\w/.test(ch) || ch > "\x80" &&
      (ch.toUpperCase() != ch.toLowerCase() || nonASCIISingleCaseWordChar.test(ch))
  }
  function isWordChar(ch, helper) {
    if (!helper) { return isWordCharBasic(ch) }
    if (helper.source.indexOf("\\w") > -1 && isWordCharBasic(ch)) { return true }
    return helper.test(ch)
  }

  function isEmpty(obj) {
    for (var n in obj) { if (obj.hasOwnProperty(n) && obj[n]) { return false } }
    return true
  }

  // Extending unicode characters. A series of a non-extending char +
  // any number of extending chars is treated as a single unit as far
  // as editing and measuring is concerned. This is not fully correct,
  // since some scripts/fonts/browsers also treat other configurations
  // of code points as a group.
  var extendingChars = /[\u0300-\u036f\u0483-\u0489\u0591-\u05bd\u05bf\u05c1\u05c2\u05c4\u05c5\u05c7\u0610-\u061a\u064b-\u065e\u0670\u06d6-\u06dc\u06de-\u06e4\u06e7\u06e8\u06ea-\u06ed\u0711\u0730-\u074a\u07a6-\u07b0\u07eb-\u07f3\u0816-\u0819\u081b-\u0823\u0825-\u0827\u0829-\u082d\u0900-\u0902\u093c\u0941-\u0948\u094d\u0951-\u0955\u0962\u0963\u0981\u09bc\u09be\u09c1-\u09c4\u09cd\u09d7\u09e2\u09e3\u0a01\u0a02\u0a3c\u0a41\u0a42\u0a47\u0a48\u0a4b-\u0a4d\u0a51\u0a70\u0a71\u0a75\u0a81\u0a82\u0abc\u0ac1-\u0ac5\u0ac7\u0ac8\u0acd\u0ae2\u0ae3\u0b01\u0b3c\u0b3e\u0b3f\u0b41-\u0b44\u0b4d\u0b56\u0b57\u0b62\u0b63\u0b82\u0bbe\u0bc0\u0bcd\u0bd7\u0c3e-\u0c40\u0c46-\u0c48\u0c4a-\u0c4d\u0c55\u0c56\u0c62\u0c63\u0cbc\u0cbf\u0cc2\u0cc6\u0ccc\u0ccd\u0cd5\u0cd6\u0ce2\u0ce3\u0d3e\u0d41-\u0d44\u0d4d\u0d57\u0d62\u0d63\u0dca\u0dcf\u0dd2-\u0dd4\u0dd6\u0ddf\u0e31\u0e34-\u0e3a\u0e47-\u0e4e\u0eb1\u0eb4-\u0eb9\u0ebb\u0ebc\u0ec8-\u0ecd\u0f18\u0f19\u0f35\u0f37\u0f39\u0f71-\u0f7e\u0f80-\u0f84\u0f86\u0f87\u0f90-\u0f97\u0f99-\u0fbc\u0fc6\u102d-\u1030\u1032-\u1037\u1039\u103a\u103d\u103e\u1058\u1059\u105e-\u1060\u1071-\u1074\u1082\u1085\u1086\u108d\u109d\u135f\u1712-\u1714\u1732-\u1734\u1752\u1753\u1772\u1773\u17b7-\u17bd\u17c6\u17c9-\u17d3\u17dd\u180b-\u180d\u18a9\u1920-\u1922\u1927\u1928\u1932\u1939-\u193b\u1a17\u1a18\u1a56\u1a58-\u1a5e\u1a60\u1a62\u1a65-\u1a6c\u1a73-\u1a7c\u1a7f\u1b00-\u1b03\u1b34\u1b36-\u1b3a\u1b3c\u1b42\u1b6b-\u1b73\u1b80\u1b81\u1ba2-\u1ba5\u1ba8\u1ba9\u1c2c-\u1c33\u1c36\u1c37\u1cd0-\u1cd2\u1cd4-\u1ce0\u1ce2-\u1ce8\u1ced\u1dc0-\u1de6\u1dfd-\u1dff\u200c\u200d\u20d0-\u20f0\u2cef-\u2cf1\u2de0-\u2dff\u302a-\u302f\u3099\u309a\ua66f-\ua672\ua67c\ua67d\ua6f0\ua6f1\ua802\ua806\ua80b\ua825\ua826\ua8c4\ua8e0-\ua8f1\ua926-\ua92d\ua947-\ua951\ua980-\ua982\ua9b3\ua9b6-\ua9b9\ua9bc\uaa29-\uaa2e\uaa31\uaa32\uaa35\uaa36\uaa43\uaa4c\uaab0\uaab2-\uaab4\uaab7\uaab8\uaabe\uaabf\uaac1\uabe5\uabe8\uabed\udc00-\udfff\ufb1e\ufe00-\ufe0f\ufe20-\ufe26\uff9e\uff9f]/;
  function isExtendingChar(ch) { return ch.charCodeAt(0) >= 768 && extendingChars.test(ch) }

  function skipExtendingChars(str, pos, dir) {
    while ((dir < 0 ? pos > 0 : pos < str.length) && isExtendingChar(str.charAt(pos))) { pos += dir; }
    return pos
  }

  function findFirst(pred, from, to) {
    var dir = from > to ? -1 : 1;
    for (;;) {
      if (from == to) { return from }
      var midF = (from + to) / 2, mid = dir < 0 ? Math.ceil(midF) : Math.floor(midF);
      if (mid == from) { return pred(mid) ? from : to }
      if (pred(mid)) { to = mid; }
      else { from = mid + dir; }
    }
  }

  // BIDI HELPERS

  function iterateBidiSections(order, from, to, f) {
    if (!order) { return f(from, to, "ltr", 0) }
    var found = false;
    for (var i = 0; i < order.length; ++i) {
      var part = order[i];
      if (part.from < to && part.to > from || from == to && part.to == from) {
        f(Math.max(part.from, from), Math.min(part.to, to), part.level == 1 ? "rtl" : "ltr", i);
        found = true;
      }
    }
    if (!found) { f(from, to, "ltr"); }
  }

  var bidiOther = null;
  function getBidiPartAt(order, ch, sticky) {
    var found;
    bidiOther = null;
    for (var i = 0; i < order.length; ++i) {
      var cur = order[i];
      if (cur.from < ch && cur.to > ch) { return i }
      if (cur.to == ch) {
        if (cur.from != cur.to && sticky == "before") { found = i; }
        else { bidiOther = i; }
      }
      if (cur.from == ch) {
        if (cur.from != cur.to && sticky != "before") { found = i; }
        else { bidiOther = i; }
      }
    }
    return found != null ? found : bidiOther
  }

  // Bidirectional ordering algorithm
  // See http://unicode.org/reports/tr9/tr9-13.html for the algorithm
  // that this (partially) implements.

  // One-char codes used for character types:
  // L (L):   Left-to-Right
  // R (R):   Right-to-Left
  // r (AL):  Right-to-Left Arabic
  // 1 (EN):  European Number
  // + (ES):  European Number Separator
  // % (ET):  European Number Terminator
  // n (AN):  Arabic Number
  // , (CS):  Common Number Separator
  // m (NSM): Non-Spacing Mark
  // b (BN):  Boundary Neutral
  // s (B):   Paragraph Separator
  // t (S):   Segment Separator
  // w (WS):  Whitespace
  // N (ON):  Other Neutrals

  // Returns null if characters are ordered as they appear
  // (left-to-right), or an array of sections ({from, to, level}
  // objects) in the order in which they occur visually.
  var bidiOrdering = (function() {
    // Character types for codepoints 0 to 0xff
    var lowTypes = "bbbbbbbbbtstwsbbbbbbbbbbbbbbssstwNN%%%NNNNNN,N,N1111111111NNNNNNNLLLLLLLLLLLLLLLLLLLLLLLLLLNNNNNNLLLLLLLLLLLLLLLLLLLLLLLLLLNNNNbbbbbbsbbbbbbbbbbbbbbbbbbbbbbbbbb,N%%%%NNNNLNNNNN%%11NLNNN1LNNNNNLLLLLLLLLLLLLLLLLLLLLLLNLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLN";
    // Character types for codepoints 0x600 to 0x6f9
    var arabicTypes = "nnnnnnNNr%%r,rNNmmmmmmmmmmmrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrmmmmmmmmmmmmmmmmmmmmmnnnnnnnnnn%nnrrrmrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrmmmmmmmnNmmmmmmrrmmNmmmmrr1111111111";
    function charType(code) {
      if (code <= 0xf7) { return lowTypes.charAt(code) }
      else if (0x590 <= code && code <= 0x5f4) { return "R" }
      else if (0x600 <= code && code <= 0x6f9) { return arabicTypes.charAt(code - 0x600) }
      else if (0x6ee <= code && code <= 0x8ac) { return "r" }
      else if (0x2000 <= code && code <= 0x200b) { return "w" }
      else if (code == 0x200c) { return "b" }
      else { return "L" }
    }

    var bidiRE = /[\u0590-\u05f4\u0600-\u06ff\u0700-\u08ac]/;
    var isNeutral = /[stwN]/, isStrong = /[LRr]/, countsAsLeft = /[Lb1n]/, countsAsNum = /[1n]/;

    function BidiSpan(level, from, to) {
      this.level = level;
      this.from = from; this.to = to;
    }

    return function(str, direction) {
      var outerType = direction == "ltr" ? "L" : "R";

      if (str.length == 0 || direction == "ltr" && !bidiRE.test(str)) { return false }
      var len = str.length, types = [];
      for (var i = 0; i < len; ++i)
        { types.push(charType(str.charCodeAt(i))); }

      // W1. Examine each non-spacing mark (NSM) in the level run, and
      // change the type of the NSM to the type of the previous
      // character. If the NSM is at the start of the level run, it will
      // get the type of sor.
      for (var i$1 = 0, prev = outerType; i$1 < len; ++i$1) {
        var type = types[i$1];
        if (type == "m") { types[i$1] = prev; }
        else { prev = type; }
      }

      // W2. Search backwards from each instance of a European number
      // until the first strong type (R, L, AL, or sor) is found. If an
      // AL is found, change the type of the European number to Arabic
      // number.
      // W3. Change all ALs to R.
      for (var i$2 = 0, cur = outerType; i$2 < len; ++i$2) {
        var type$1 = types[i$2];
        if (type$1 == "1" && cur == "r") { types[i$2] = "n"; }
        else if (isStrong.test(type$1)) { cur = type$1; if (type$1 == "r") { types[i$2] = "R"; } }
      }

      // W4. A single European separator between two European numbers
      // changes to a European number. A single common separator between
      // two numbers of the same type changes to that type.
      for (var i$3 = 1, prev$1 = types[0]; i$3 < len - 1; ++i$3) {
        var type$2 = types[i$3];
        if (type$2 == "+" && prev$1 == "1" && types[i$3+1] == "1") { types[i$3] = "1"; }
        else if (type$2 == "," && prev$1 == types[i$3+1] &&
                 (prev$1 == "1" || prev$1 == "n")) { types[i$3] = prev$1; }
        prev$1 = type$2;
      }

      // W5. A sequence of European terminators adjacent to European
      // numbers changes to all European numbers.
      // W6. Otherwise, separators and terminators change to Other
      // Neutral.
      for (var i$4 = 0; i$4 < len; ++i$4) {
        var type$3 = types[i$4];
        if (type$3 == ",") { types[i$4] = "N"; }
        else if (type$3 == "%") {
          var end = (void 0);
          for (end = i$4 + 1; end < len && types[end] == "%"; ++end) {}
          var replace = (i$4 && types[i$4-1] == "!") || (end < len && types[end] == "1") ? "1" : "N";
          for (var j = i$4; j < end; ++j) { types[j] = replace; }
          i$4 = end - 1;
        }
      }

      // W7. Search backwards from each instance of a European number
      // until the first strong type (R, L, or sor) is found. If an L is
      // found, then change the type of the European number to L.
      for (var i$5 = 0, cur$1 = outerType; i$5 < len; ++i$5) {
        var type$4 = types[i$5];
        if (cur$1 == "L" && type$4 == "1") { types[i$5] = "L"; }
        else if (isStrong.test(type$4)) { cur$1 = type$4; }
      }

      // N1. A sequence of neutrals takes the direction of the
      // surrounding strong text if the text on both sides has the same
      // direction. European and Arabic numbers act as if they were R in
      // terms of their influence on neutrals. Start-of-level-run (sor)
      // and end-of-level-run (eor) are used at level run boundaries.
      // N2. Any remaining neutrals take the embedding direction.
      for (var i$6 = 0; i$6 < len; ++i$6) {
        if (isNeutral.test(types[i$6])) {
          var end$1 = (void 0);
          for (end$1 = i$6 + 1; end$1 < len && isNeutral.test(types[end$1]); ++end$1) {}
          var before = (i$6 ? types[i$6-1] : outerType) == "L";
          var after = (end$1 < len ? types[end$1] : outerType) == "L";
          var replace$1 = before == after ? (before ? "L" : "R") : outerType;
          for (var j$1 = i$6; j$1 < end$1; ++j$1) { types[j$1] = replace$1; }
          i$6 = end$1 - 1;
        }
      }

      // Here we depart from the documented algorithm, in order to avoid
      // building up an actual levels array. Since there are only three
      // levels (0, 1, 2) in an implementation that doesn't take
      // explicit embedding into account, we can build up the order on
      // the fly, without following the level-based algorithm.
      var order = [], m;
      for (var i$7 = 0; i$7 < len;) {
        if (countsAsLeft.test(types[i$7])) {
          var start = i$7;
          for (++i$7; i$7 < len && countsAsLeft.test(types[i$7]); ++i$7) {}
          order.push(new BidiSpan(0, start, i$7));
        } else {
          var pos = i$7, at = order.length;
          for (++i$7; i$7 < len && types[i$7] != "L"; ++i$7) {}
          for (var j$2 = pos; j$2 < i$7;) {
            if (countsAsNum.test(types[j$2])) {
              if (pos < j$2) { order.splice(at, 0, new BidiSpan(1, pos, j$2)); }
              var nstart = j$2;
              for (++j$2; j$2 < i$7 && countsAsNum.test(types[j$2]); ++j$2) {}
              order.splice(at, 0, new BidiSpan(2, nstart, j$2));
              pos = j$2;
            } else { ++j$2; }
          }
          if (pos < i$7) { order.splice(at, 0, new BidiSpan(1, pos, i$7)); }
        }
      }
      if (direction == "ltr") {
        if (order[0].level == 1 && (m = str.match(/^\s+/))) {
          order[0].from = m[0].length;
          order.unshift(new BidiSpan(0, 0, m[0].length));
        }
        if (lst(order).level == 1 && (m = str.match(/\s+$/))) {
          lst(order).to -= m[0].length;
          order.push(new BidiSpan(0, len - m[0].length, len));
        }
      }

      return direction == "rtl" ? order.reverse() : order
    }
  })();

  // Get the bidi ordering for the given line (and cache it). Returns
  // false for lines that are fully left-to-right, and an array of
  // BidiSpan objects otherwise.
  function getOrder(line, direction) {
    var order = line.order;
    if (order == null) { order = line.order = bidiOrdering(line.text, direction); }
    return order
  }

  // EVENT HANDLING

  // Lightweight event framework. on/off also work on DOM nodes,
  // registering native DOM handlers.

  var noHandlers = [];

  var on = function(emitter, type, f) {
    if (emitter.addEventListener) {
      emitter.addEventListener(type, f, false);
    } else if (emitter.attachEvent) {
      emitter.attachEvent("on" + type, f);
    } else {
      var map$$1 = emitter._handlers || (emitter._handlers = {});
      map$$1[type] = (map$$1[type] || noHandlers).concat(f);
    }
  };

  function getHandlers(emitter, type) {
    return emitter._handlers && emitter._handlers[type] || noHandlers
  }

  function off(emitter, type, f) {
    if (emitter.removeEventListener) {
      emitter.removeEventListener(type, f, false);
    } else if (emitter.detachEvent) {
      emitter.detachEvent("on" + type, f);
    } else {
      var map$$1 = emitter._handlers, arr = map$$1 && map$$1[type];
      if (arr) {
        var index = indexOf(arr, f);
        if (index > -1)
          { map$$1[type] = arr.slice(0, index).concat(arr.slice(index + 1)); }
      }
    }
  }

  function signal(emitter, type /*, values...*/) {
    var handlers = getHandlers(emitter, type);
    if (!handlers.length) { return }
    var args = Array.prototype.slice.call(arguments, 2);
    for (var i = 0; i < handlers.length; ++i) { handlers[i].apply(null, args); }
  }

  // The DOM events that CodeMirror handles can be overridden by
  // registering a (non-DOM) handler on the editor for the event name,
  // and preventDefault-ing the event in that handler.
  function signalDOMEvent(cm, e, override) {
    if (typeof e == "string")
      { e = {type: e, preventDefault: function() { this.defaultPrevented = true; }}; }
    signal(cm, override || e.type, cm, e);
    return e_defaultPrevented(e) || e.codemirrorIgnore
  }

  function signalCursorActivity(cm) {
    var arr = cm._handlers && cm._handlers.cursorActivity;
    if (!arr) { return }
    var set = cm.curOp.cursorActivityHandlers || (cm.curOp.cursorActivityHandlers = []);
    for (var i = 0; i < arr.length; ++i) { if (indexOf(set, arr[i]) == -1)
      { set.push(arr[i]); } }
  }

  function hasHandler(emitter, type) {
    return getHandlers(emitter, type).length > 0
  }

  // Add on and off methods to a constructor's prototype, to make
  // registering events on such objects more convenient.
  function eventMixin(ctor) {
    ctor.prototype.on = function(type, f) {on(this, type, f);};
    ctor.prototype.off = function(type, f) {off(this, type, f);};
  }

  // Due to the fact that we still support jurassic IE versions, some
  // compatibility wrappers are needed.

  function e_preventDefault(e) {
    if (e.preventDefault) { e.preventDefault(); }
    else { e.returnValue = false; }
  }
  function e_stopPropagation(e) {
    if (e.stopPropagation) { e.stopPropagation(); }
    else { e.cancelBubble = true; }
  }
  function e_defaultPrevented(e) {
    return e.defaultPrevented != null ? e.defaultPrevented : e.returnValue == false
  }
  function e_stop(e) {e_preventDefault(e); e_stopPropagation(e);}

  function e_target(e) {return e.target || e.srcElement}
  function e_button(e) {
    var b = e.which;
    if (b == null) {
      if (e.button & 1) { b = 1; }
      else if (e.button & 2) { b = 3; }
      else if (e.button & 4) { b = 2; }
    }
    if (mac && e.ctrlKey && b == 1) { b = 3; }
    return b
  }

  // Detect drag-and-drop
  var dragAndDrop = function() {
    // There is *some* kind of drag-and-drop support in IE6-8, but I
    // couldn't get it to work yet.
    if (ie && ie_version < 9) { return false }
    var div = elt('div');
    return "draggable" in div || "dragDrop" in div
  }();

  var zwspSupported;
  function zeroWidthElement(measure) {
    if (zwspSupported == null) {
      var test = elt("span", "\u200b");
      removeChildrenAndAdd(measure, elt("span", [test, document.createTextNode("x")]));
      if (measure.firstChild.offsetHeight != 0)
        { zwspSupported = test.offsetWidth <= 1 && test.offsetHeight > 2 && !(ie && ie_version < 8); }
    }
    var node = zwspSupported ? elt("span", "\u200b") :
      elt("span", "\u00a0", null, "display: inline-block; width: 1px; margin-right: -1px");
    node.setAttribute("cm-text", "");
    return node
  }

  // Feature-detect IE's crummy client rect reporting for bidi text
  var badBidiRects;
  function hasBadBidiRects(measure) {
    if (badBidiRects != null) { return badBidiRects }
    var txt = removeChildrenAndAdd(measure, document.createTextNode("A\u062eA"));
    var r0 = range(txt, 0, 1).getBoundingClientRect();
    var r1 = range(txt, 1, 2).getBoundingClientRect();
    removeChildren(measure);
    if (!r0 || r0.left == r0.right) { return false } // Safari returns null in some cases (#2780)
    return badBidiRects = (r1.right - r0.right < 3)
  }

  // See if "".split is the broken IE version, if so, provide an
  // alternative way to split lines.
  var splitLinesAuto = "\n\nb".split(/\n/).length != 3 ? function (string) {
    var pos = 0, result = [], l = string.length;
    while (pos <= l) {
      var nl = string.indexOf("\n", pos);
      if (nl == -1) { nl = string.length; }
      var line = string.slice(pos, string.charAt(nl - 1) == "\r" ? nl - 1 : nl);
      var rt = line.indexOf("\r");
      if (rt != -1) {
        result.push(line.slice(0, rt));
        pos += rt + 1;
      } else {
        result.push(line);
        pos = nl + 1;
      }
    }
    return result
  } : function (string) { return string.split(/\r\n?|\n/); };

  var hasSelection = window.getSelection ? function (te) {
    try { return te.selectionStart != te.selectionEnd }
    catch(e) { return false }
  } : function (te) {
    var range$$1;
    try {range$$1 = te.ownerDocument.selection.createRange();}
    catch(e) {}
    if (!range$$1 || range$$1.parentElement() != te) { return false }
    return range$$1.compareEndPoints("StartToEnd", range$$1) != 0
  };

  var hasCopyEvent = (function () {
    var e = elt("div");
    if ("oncopy" in e) { return true }
    e.setAttribute("oncopy", "return;");
    return typeof e.oncopy == "function"
  })();

  var badZoomedRects = null;
  function hasBadZoomedRects(measure) {
    if (badZoomedRects != null) { return badZoomedRects }
    var node = removeChildrenAndAdd(measure, elt("span", "x"));
    var normal = node.getBoundingClientRect();
    var fromRange = range(node, 0, 1).getBoundingClientRect();
    return badZoomedRects = Math.abs(normal.left - fromRange.left) > 1
  }

  // Known modes, by name and by MIME
  var modes = {}, mimeModes = {};

  // Extra arguments are stored as the mode's dependencies, which is
  // used by (legacy) mechanisms like loadmode.js to automatically
  // load a mode. (Preferred mechanism is the require/define calls.)
  function defineMode(name, mode) {
    if (arguments.length > 2)
      { mode.dependencies = Array.prototype.slice.call(arguments, 2); }
    modes[name] = mode;
  }

  function defineMIME(mime, spec) {
    mimeModes[mime] = spec;
  }

  // Given a MIME type, a {name, ...options} config object, or a name
  // string, return a mode config object.
  function resolveMode(spec) {
    if (typeof spec == "string" && mimeModes.hasOwnProperty(spec)) {
      spec = mimeModes[spec];
    } else if (spec && typeof spec.name == "string" && mimeModes.hasOwnProperty(spec.name)) {
      var found = mimeModes[spec.name];
      if (typeof found == "string") { found = {name: found}; }
      spec = createObj(found, spec);
      spec.name = found.name;
    } else if (typeof spec == "string" && /^[\w\-]+\/[\w\-]+\+xml$/.test(spec)) {
      return resolveMode("application/xml")
    } else if (typeof spec == "string" && /^[\w\-]+\/[\w\-]+\+json$/.test(spec)) {
      return resolveMode("application/json")
    }
    if (typeof spec == "string") { return {name: spec} }
    else { return spec || {name: "null"} }
  }

  // Given a mode spec (anything that resolveMode accepts), find and
  // initialize an actual mode object.
  function getMode(options, spec) {
    spec = resolveMode(spec);
    var mfactory = modes[spec.name];
    if (!mfactory) { return getMode(options, "text/plain") }
    var modeObj = mfactory(options, spec);
    if (modeExtensions.hasOwnProperty(spec.name)) {
      var exts = modeExtensions[spec.name];
      for (var prop in exts) {
        if (!exts.hasOwnProperty(prop)) { continue }
        if (modeObj.hasOwnProperty(prop)) { modeObj["_" + prop] = modeObj[prop]; }
        modeObj[prop] = exts[prop];
      }
    }
    modeObj.name = spec.name;
    if (spec.helperType) { modeObj.helperType = spec.helperType; }
    if (spec.modeProps) { for (var prop$1 in spec.modeProps)
      { modeObj[prop$1] = spec.modeProps[prop$1]; } }

    return modeObj
  }

  // This can be used to attach properties to mode objects from
  // outside the actual mode definition.
  var modeExtensions = {};
  function extendMode(mode, properties) {
    var exts = modeExtensions.hasOwnProperty(mode) ? modeExtensions[mode] : (modeExtensions[mode] = {});
    copyObj(properties, exts);
  }

  function copyState(mode, state) {
    if (state === true) { return state }
    if (mode.copyState) { return mode.copyState(state) }
    var nstate = {};
    for (var n in state) {
      var val = state[n];
      if (val instanceof Array) { val = val.concat([]); }
      nstate[n] = val;
    }
    return nstate
  }

  // Given a mode and a state (for that mode), find the inner mode and
  // state at the position that the state refers to.
  function innerMode(mode, state) {
    var info;
    while (mode.innerMode) {
      info = mode.innerMode(state);
      if (!info || info.mode == mode) { break }
      state = info.state;
      mode = info.mode;
    }
    return info || {mode: mode, state: state}
  }

  function startState(mode, a1, a2) {
    return mode.startState ? mode.startState(a1, a2) : true
  }

  // STRING STREAM

  // Fed to the mode parsers, provides helper functions to make
  // parsers more succinct.

  var StringStream = function(string, tabSize, lineOracle) {
    this.pos = this.start = 0;
    this.string = string;
    this.tabSize = tabSize || 8;
    this.lastColumnPos = this.lastColumnValue = 0;
    this.lineStart = 0;
    this.lineOracle = lineOracle;
  };

  StringStream.prototype.eol = function () {return this.pos >= this.string.length};
  StringStream.prototype.sol = function () {return this.pos == this.lineStart};
  StringStream.prototype.peek = function () {return this.string.charAt(this.pos) || undefined};
  StringStream.prototype.next = function () {
    if (this.pos < this.string.length)
      { return this.string.charAt(this.pos++) }
  };
  StringStream.prototype.eat = function (match) {
    var ch = this.string.charAt(this.pos);
    var ok;
    if (typeof match == "string") { ok = ch == match; }
    else { ok = ch && (match.test ? match.test(ch) : match(ch)); }
    if (ok) {++this.pos; return ch}
  };
  StringStream.prototype.eatWhile = function (match) {
    var start = this.pos;
    while (this.eat(match)){}
    return this.pos > start
  };
  StringStream.prototype.eatSpace = function () {
      var this$1 = this;

    var start = this.pos;
    while (/[\s\u00a0]/.test(this.string.charAt(this.pos))) { ++this$1.pos; }
    return this.pos > start
  };
  StringStream.prototype.skipToEnd = function () {this.pos = this.string.length;};
  StringStream.prototype.skipTo = function (ch) {
    var found = this.string.indexOf(ch, this.pos);
    if (found > -1) {this.pos = found; return true}
  };
  StringStream.prototype.backUp = function (n) {this.pos -= n;};
  StringStream.prototype.column = function () {
    if (this.lastColumnPos < this.start) {
      this.lastColumnValue = countColumn(this.string, this.start, this.tabSize, this.lastColumnPos, this.lastColumnValue);
      this.lastColumnPos = this.start;
    }
    return this.lastColumnValue - (this.lineStart ? countColumn(this.string, this.lineStart, this.tabSize) : 0)
  };
  StringStream.prototype.indentation = function () {
    return countColumn(this.string, null, this.tabSize) -
      (this.lineStart ? countColumn(this.string, this.lineStart, this.tabSize) : 0)
  };
  StringStream.prototype.match = function (pattern, consume, caseInsensitive) {
    if (typeof pattern == "string") {
      var cased = function (str) { return caseInsensitive ? str.toLowerCase() : str; };
      var substr = this.string.substr(this.pos, pattern.length);
      if (cased(substr) == cased(pattern)) {
        if (consume !== false) { this.pos += pattern.length; }
        return true
      }
    } else {
      var match = this.string.slice(this.pos).match(pattern);
      if (match && match.index > 0) { return null }
      if (match && consume !== false) { this.pos += match[0].length; }
      return match
    }
  };
  StringStream.prototype.current = function (){return this.string.slice(this.start, this.pos)};
  StringStream.prototype.hideFirstChars = function (n, inner) {
    this.lineStart += n;
    try { return inner() }
    finally { this.lineStart -= n; }
  };
  StringStream.prototype.lookAhead = function (n) {
    var oracle = this.lineOracle;
    return oracle && oracle.lookAhead(n)
  };
  StringStream.prototype.baseToken = function () {
    var oracle = this.lineOracle;
    return oracle && oracle.baseToken(this.pos)
  };

  // Find the line object corresponding to the given line number.
  function getLine(doc, n) {
    n -= doc.first;
    if (n < 0 || n >= doc.size) { throw new Error("There is no line " + (n + doc.first) + " in the document.") }
    var chunk = doc;
    while (!chunk.lines) {
      for (var i = 0;; ++i) {
        var child = chunk.children[i], sz = child.chunkSize();
        if (n < sz) { chunk = child; break }
        n -= sz;
      }
    }
    return chunk.lines[n]
  }

  // Get the part of a document between two positions, as an array of
  // strings.
  function getBetween(doc, start, end) {
    var out = [], n = start.line;
    doc.iter(start.line, end.line + 1, function (line) {
      var text = line.text;
      if (n == end.line) { text = text.slice(0, end.ch); }
      if (n == start.line) { text = text.slice(start.ch); }
      out.push(text);
      ++n;
    });
    return out
  }
  // Get the lines between from and to, as array of strings.
  function getLines(doc, from, to) {
    var out = [];
    doc.iter(from, to, function (line) { out.push(line.text); }); // iter aborts when callback returns truthy value
    return out
  }

  // Update the height of a line, propagating the height change
  // upwards to parent nodes.
  function updateLineHeight(line, height) {
    var diff = height - line.height;
    if (diff) { for (var n = line; n; n = n.parent) { n.height += diff; } }
  }

  // Given a line object, find its line number by walking up through
  // its parent links.
  function lineNo(line) {
    if (line.parent == null) { return null }
    var cur = line.parent, no = indexOf(cur.lines, line);
    for (var chunk = cur.parent; chunk; cur = chunk, chunk = chunk.parent) {
      for (var i = 0;; ++i) {
        if (chunk.children[i] == cur) { break }
        no += chunk.children[i].chunkSize();
      }
    }
    return no + cur.first
  }

  // Find the line at the given vertical position, using the height
  // information in the document tree.
  function lineAtHeight(chunk, h) {
    var n = chunk.first;
    outer: do {
      for (var i$1 = 0; i$1 < chunk.children.length; ++i$1) {
        var child = chunk.children[i$1], ch = child.height;
        if (h < ch) { chunk = child; continue outer }
        h -= ch;
        n += child.chunkSize();
      }
      return n
    } while (!chunk.lines)
    var i = 0;
    for (; i < chunk.lines.length; ++i) {
      var line = chunk.lines[i], lh = line.height;
      if (h < lh) { break }
      h -= lh;
    }
    return n + i
  }

  function isLine(doc, l) {return l >= doc.first && l < doc.first + doc.size}

  function lineNumberFor(options, i) {
    return String(options.lineNumberFormatter(i + options.firstLineNumber))
  }

  // A Pos instance represents a position within the text.
  function Pos(line, ch, sticky) {
    if ( sticky === void 0 ) sticky = null;

    if (!(this instanceof Pos)) { return new Pos(line, ch, sticky) }
    this.line = line;
    this.ch = ch;
    this.sticky = sticky;
  }

  // Compare two positions, return 0 if they are the same, a negative
  // number when a is less, and a positive number otherwise.
  function cmp(a, b) { return a.line - b.line || a.ch - b.ch }

  function equalCursorPos(a, b) { return a.sticky == b.sticky && cmp(a, b) == 0 }

  function copyPos(x) {return Pos(x.line, x.ch)}
  function maxPos(a, b) { return cmp(a, b) < 0 ? b : a }
  function minPos(a, b) { return cmp(a, b) < 0 ? a : b }

  // Most of the external API clips given positions to make sure they
  // actually exist within the document.
  function clipLine(doc, n) {return Math.max(doc.first, Math.min(n, doc.first + doc.size - 1))}
  function clipPos(doc, pos) {
    if (pos.line < doc.first) { return Pos(doc.first, 0) }
    var last = doc.first + doc.size - 1;
    if (pos.line > last) { return Pos(last, getLine(doc, last).text.length) }
    return clipToLen(pos, getLine(doc, pos.line).text.length)
  }
  function clipToLen(pos, linelen) {
    var ch = pos.ch;
    if (ch == null || ch > linelen) { return Pos(pos.line, linelen) }
    else if (ch < 0) { return Pos(pos.line, 0) }
    else { return pos }
  }
  function clipPosArray(doc, array) {
    var out = [];
    for (var i = 0; i < array.length; i++) { out[i] = clipPos(doc, array[i]); }
    return out
  }

  var SavedContext = function(state, lookAhead) {
    this.state = state;
    this.lookAhead = lookAhead;
  };

  var Context = function(doc, state, line, lookAhead) {
    this.state = state;
    this.doc = doc;
    this.line = line;
    this.maxLookAhead = lookAhead || 0;
    this.baseTokens = null;
    this.baseTokenPos = 1;
  };

  Context.prototype.lookAhead = function (n) {
    var line = this.doc.getLine(this.line + n);
    if (line != null && n > this.maxLookAhead) { this.maxLookAhead = n; }
    return line
  };

  Context.prototype.baseToken = function (n) {
      var this$1 = this;

    if (!this.baseTokens) { return null }
    while (this.baseTokens[this.baseTokenPos] <= n)
      { this$1.baseTokenPos += 2; }
    var type = this.baseTokens[this.baseTokenPos + 1];
    return {type: type && type.replace(/( |^)overlay .*/, ""),
            size: this.baseTokens[this.baseTokenPos] - n}
  };

  Context.prototype.nextLine = function () {
    this.line++;
    if (this.maxLookAhead > 0) { this.maxLookAhead--; }
  };

  Context.fromSaved = function (doc, saved, line) {
    if (saved instanceof SavedContext)
      { return new Context(doc, copyState(doc.mode, saved.state), line, saved.lookAhead) }
    else
      { return new Context(doc, copyState(doc.mode, saved), line) }
  };

  Context.prototype.save = function (copy) {
    var state = copy !== false ? copyState(this.doc.mode, this.state) : this.state;
    return this.maxLookAhead > 0 ? new SavedContext(state, this.maxLookAhead) : state
  };


  // Compute a style array (an array starting with a mode generation
  // -- for invalidation -- followed by pairs of end positions and
  // style strings), which is used to highlight the tokens on the
  // line.
  function highlightLine(cm, line, context, forceToEnd) {
    // A styles array always starts with a number identifying the
    // mode/overlays that it is based on (for easy invalidation).
    var st = [cm.state.modeGen], lineClasses = {};
    // Compute the base array of styles
    runMode(cm, line.text, cm.doc.mode, context, function (end, style) { return st.push(end, style); },
            lineClasses, forceToEnd);
    var state = context.state;

    // Run overlays, adjust style array.
    var loop = function ( o ) {
      context.baseTokens = st;
      var overlay = cm.state.overlays[o], i = 1, at = 0;
      context.state = true;
      runMode(cm, line.text, overlay.mode, context, function (end, style) {
        var start = i;
        // Ensure there's a token end at the current position, and that i points at it
        while (at < end) {
          var i_end = st[i];
          if (i_end > end)
            { st.splice(i, 1, end, st[i+1], i_end); }
          i += 2;
          at = Math.min(end, i_end);
        }
        if (!style) { return }
        if (overlay.opaque) {
          st.splice(start, i - start, end, "overlay " + style);
          i = start + 2;
        } else {
          for (; start < i; start += 2) {
            var cur = st[start+1];
            st[start+1] = (cur ? cur + " " : "") + "overlay " + style;
          }
        }
      }, lineClasses);
      context.state = state;
      context.baseTokens = null;
      context.baseTokenPos = 1;
    };

    for (var o = 0; o < cm.state.overlays.length; ++o) loop( o );

    return {styles: st, classes: lineClasses.bgClass || lineClasses.textClass ? lineClasses : null}
  }

  function getLineStyles(cm, line, updateFrontier) {
    if (!line.styles || line.styles[0] != cm.state.modeGen) {
      var context = getContextBefore(cm, lineNo(line));
      var resetState = line.text.length > cm.options.maxHighlightLength && copyState(cm.doc.mode, context.state);
      var result = highlightLine(cm, line, context);
      if (resetState) { context.state = resetState; }
      line.stateAfter = context.save(!resetState);
      line.styles = result.styles;
      if (result.classes) { line.styleClasses = result.classes; }
      else if (line.styleClasses) { line.styleClasses = null; }
      if (updateFrontier === cm.doc.highlightFrontier)
        { cm.doc.modeFrontier = Math.max(cm.doc.modeFrontier, ++cm.doc.highlightFrontier); }
    }
    return line.styles
  }

  function getContextBefore(cm, n, precise) {
    var doc = cm.doc, display = cm.display;
    if (!doc.mode.startState) { return new Context(doc, true, n) }
    var start = findStartLine(cm, n, precise);
    var saved = start > doc.first && getLine(doc, start - 1).stateAfter;
    var context = saved ? Context.fromSaved(doc, saved, start) : new Context(doc, startState(doc.mode), start);

    doc.iter(start, n, function (line) {
      processLine(cm, line.text, context);
      var pos = context.line;
      line.stateAfter = pos == n - 1 || pos % 5 == 0 || pos >= display.viewFrom && pos < display.viewTo ? context.save() : null;
      context.nextLine();
    });
    if (precise) { doc.modeFrontier = context.line; }
    return context
  }

  // Lightweight form of highlight -- proceed over this line and
  // update state, but don't save a style array. Used for lines that
  // aren't currently visible.
  function processLine(cm, text, context, startAt) {
    var mode = cm.doc.mode;
    var stream = new StringStream(text, cm.options.tabSize, context);
    stream.start = stream.pos = startAt || 0;
    if (text == "") { callBlankLine(mode, context.state); }
    while (!stream.eol()) {
      readToken(mode, stream, context.state);
      stream.start = stream.pos;
    }
  }

  function callBlankLine(mode, state) {
    if (mode.blankLine) { return mode.blankLine(state) }
    if (!mode.innerMode) { return }
    var inner = innerMode(mode, state);
    if (inner.mode.blankLine) { return inner.mode.blankLine(inner.state) }
  }

  function readToken(mode, stream, state, inner) {
    for (var i = 0; i < 10; i++) {
      if (inner) { inner[0] = innerMode(mode, state).mode; }
      var style = mode.token(stream, state);
      if (stream.pos > stream.start) { return style }
    }
    throw new Error("Mode " + mode.name + " failed to advance stream.")
  }

  var Token = function(stream, type, state) {
    this.start = stream.start; this.end = stream.pos;
    this.string = stream.current();
    this.type = type || null;
    this.state = state;
  };

  // Utility for getTokenAt and getLineTokens
  function takeToken(cm, pos, precise, asArray) {
    var doc = cm.doc, mode = doc.mode, style;
    pos = clipPos(doc, pos);
    var line = getLine(doc, pos.line), context = getContextBefore(cm, pos.line, precise);
    var stream = new StringStream(line.text, cm.options.tabSize, context), tokens;
    if (asArray) { tokens = []; }
    while ((asArray || stream.pos < pos.ch) && !stream.eol()) {
      stream.start = stream.pos;
      style = readToken(mode, stream, context.state);
      if (asArray) { tokens.push(new Token(stream, style, copyState(doc.mode, context.state))); }
    }
    return asArray ? tokens : new Token(stream, style, context.state)
  }

  function extractLineClasses(type, output) {
    if (type) { for (;;) {
      var lineClass = type.match(/(?:^|\s+)line-(background-)?(\S+)/);
      if (!lineClass) { break }
      type = type.slice(0, lineClass.index) + type.slice(lineClass.index + lineClass[0].length);
      var prop = lineClass[1] ? "bgClass" : "textClass";
      if (output[prop] == null)
        { output[prop] = lineClass[2]; }
      else if (!(new RegExp("(?:^|\s)" + lineClass[2] + "(?:$|\s)")).test(output[prop]))
        { output[prop] += " " + lineClass[2]; }
    } }
    return type
  }

  // Run the given mode's parser over a line, calling f for each token.
  function runMode(cm, text, mode, context, f, lineClasses, forceToEnd) {
    var flattenSpans = mode.flattenSpans;
    if (flattenSpans == null) { flattenSpans = cm.options.flattenSpans; }
    var curStart = 0, curStyle = null;
    var stream = new StringStream(text, cm.options.tabSize, context), style;
    var inner = cm.options.addModeClass && [null];
    if (text == "") { extractLineClasses(callBlankLine(mode, context.state), lineClasses); }
    while (!stream.eol()) {
      if (stream.pos > cm.options.maxHighlightLength) {
        flattenSpans = false;
        if (forceToEnd) { processLine(cm, text, context, stream.pos); }
        stream.pos = text.length;
        style = null;
      } else {
        style = extractLineClasses(readToken(mode, stream, context.state, inner), lineClasses);
      }
      if (inner) {
        var mName = inner[0].name;
        if (mName) { style = "m-" + (style ? mName + " " + style : mName); }
      }
      if (!flattenSpans || curStyle != style) {
        while (curStart < stream.start) {
          curStart = Math.min(stream.start, curStart + 5000);
          f(curStart, curStyle);
        }
        curStyle = style;
      }
      stream.start = stream.pos;
    }
    while (curStart < stream.pos) {
      // Webkit seems to refuse to render text nodes longer than 57444
      // characters, and returns inaccurate measurements in nodes
      // starting around 5000 chars.
      var pos = Math.min(stream.pos, curStart + 5000);
      f(pos, curStyle);
      curStart = pos;
    }
  }

  // Finds the line to start with when starting a parse. Tries to
  // find a line with a stateAfter, so that it can start with a
  // valid state. If that fails, it returns the line with the
  // smallest indentation, which tends to need the least context to
  // parse correctly.
  function findStartLine(cm, n, precise) {
    var minindent, minline, doc = cm.doc;
    var lim = precise ? -1 : n - (cm.doc.mode.innerMode ? 1000 : 100);
    for (var search = n; search > lim; --search) {
      if (search <= doc.first) { return doc.first }
      var line = getLine(doc, search - 1), after = line.stateAfter;
      if (after && (!precise || search + (after instanceof SavedContext ? after.lookAhead : 0) <= doc.modeFrontier))
        { return search }
      var indented = countColumn(line.text, null, cm.options.tabSize);
      if (minline == null || minindent > indented) {
        minline = search - 1;
        minindent = indented;
      }
    }
    return minline
  }

  function retreatFrontier(doc, n) {
    doc.modeFrontier = Math.min(doc.modeFrontier, n);
    if (doc.highlightFrontier < n - 10) { return }
    var start = doc.first;
    for (var line = n - 1; line > start; line--) {
      var saved = getLine(doc, line).stateAfter;
      // change is on 3
      // state on line 1 looked ahead 2 -- so saw 3
      // test 1 + 2 < 3 should cover this
      if (saved && (!(saved instanceof SavedContext) || line + saved.lookAhead < n)) {
        start = line + 1;
        break
      }
    }
    doc.highlightFrontier = Math.min(doc.highlightFrontier, start);
  }

  // Optimize some code when these features are not used.
  var sawReadOnlySpans = false, sawCollapsedSpans = false;

  function seeReadOnlySpans() {
    sawReadOnlySpans = true;
  }

  function seeCollapsedSpans() {
    sawCollapsedSpans = true;
  }

  // TEXTMARKER SPANS

  function MarkedSpan(marker, from, to) {
    this.marker = marker;
    this.from = from; this.to = to;
  }

  // Search an array of spans for a span matching the given marker.
  function getMarkedSpanFor(spans, marker) {
    if (spans) { for (var i = 0; i < spans.length; ++i) {
      var span = spans[i];
      if (span.marker == marker) { return span }
    } }
  }
  // Remove a span from an array, returning undefined if no spans are
  // left (we don't store arrays for lines without spans).
  function removeMarkedSpan(spans, span) {
    var r;
    for (var i = 0; i < spans.length; ++i)
      { if (spans[i] != span) { (r || (r = [])).push(spans[i]); } }
    return r
  }
  // Add a span to a line.
  function addMarkedSpan(line, span) {
    line.markedSpans = line.markedSpans ? line.markedSpans.concat([span]) : [span];
    span.marker.attachLine(line);
  }

  // Used for the algorithm that adjusts markers for a change in the
  // document. These functions cut an array of spans at a given
  // character position, returning an array of remaining chunks (or
  // undefined if nothing remains).
  function markedSpansBefore(old, startCh, isInsert) {
    var nw;
    if (old) { for (var i = 0; i < old.length; ++i) {
      var span = old[i], marker = span.marker;
      var startsBefore = span.from == null || (marker.inclusiveLeft ? span.from <= startCh : span.from < startCh);
      if (startsBefore || span.from == startCh && marker.type == "bookmark" && (!isInsert || !span.marker.insertLeft)) {
        var endsAfter = span.to == null || (marker.inclusiveRight ? span.to >= startCh : span.to > startCh)
        ;(nw || (nw = [])).push(new MarkedSpan(marker, span.from, endsAfter ? null : span.to));
      }
    } }
    return nw
  }
  function markedSpansAfter(old, endCh, isInsert) {
    var nw;
    if (old) { for (var i = 0; i < old.length; ++i) {
      var span = old[i], marker = span.marker;
      var endsAfter = span.to == null || (marker.inclusiveRight ? span.to >= endCh : span.to > endCh);
      if (endsAfter || span.from == endCh && marker.type == "bookmark" && (!isInsert || span.marker.insertLeft)) {
        var startsBefore = span.from == null || (marker.inclusiveLeft ? span.from <= endCh : span.from < endCh)
        ;(nw || (nw = [])).push(new MarkedSpan(marker, startsBefore ? null : span.from - endCh,
                                              span.to == null ? null : span.to - endCh));
      }
    } }
    return nw
  }

  // Given a change object, compute the new set of marker spans that
  // cover the line in which the change took place. Removes spans
  // entirely within the change, reconnects spans belonging to the
  // same marker that appear on both sides of the change, and cuts off
  // spans partially within the change. Returns an array of span
  // arrays with one element for each line in (after) the change.
  function stretchSpansOverChange(doc, change) {
    if (change.full) { return null }
    var oldFirst = isLine(doc, change.from.line) && getLine(doc, change.from.line).markedSpans;
    var oldLast = isLine(doc, change.to.line) && getLine(doc, change.to.line).markedSpans;
    if (!oldFirst && !oldLast) { return null }

    var startCh = change.from.ch, endCh = change.to.ch, isInsert = cmp(change.from, change.to) == 0;
    // Get the spans that 'stick out' on both sides
    var first = markedSpansBefore(oldFirst, startCh, isInsert);
    var last = markedSpansAfter(oldLast, endCh, isInsert);

    // Next, merge those two ends
    var sameLine = change.text.length == 1, offset = lst(change.text).length + (sameLine ? startCh : 0);
    if (first) {
      // Fix up .to properties of first
      for (var i = 0; i < first.length; ++i) {
        var span = first[i];
        if (span.to == null) {
          var found = getMarkedSpanFor(last, span.marker);
          if (!found) { span.to = startCh; }
          else if (sameLine) { span.to = found.to == null ? null : found.to + offset; }
        }
      }
    }
    if (last) {
      // Fix up .from in last (or move them into first in case of sameLine)
      for (var i$1 = 0; i$1 < last.length; ++i$1) {
        var span$1 = last[i$1];
        if (span$1.to != null) { span$1.to += offset; }
        if (span$1.from == null) {
          var found$1 = getMarkedSpanFor(first, span$1.marker);
          if (!found$1) {
            span$1.from = offset;
            if (sameLine) { (first || (first = [])).push(span$1); }
          }
        } else {
          span$1.from += offset;
          if (sameLine) { (first || (first = [])).push(span$1); }
        }
      }
    }
    // Make sure we didn't create any zero-length spans
    if (first) { first = clearEmptySpans(first); }
    if (last && last != first) { last = clearEmptySpans(last); }

    var newMarkers = [first];
    if (!sameLine) {
      // Fill gap with whole-line-spans
      var gap = change.text.length - 2, gapMarkers;
      if (gap > 0 && first)
        { for (var i$2 = 0; i$2 < first.length; ++i$2)
          { if (first[i$2].to == null)
            { (gapMarkers || (gapMarkers = [])).push(new MarkedSpan(first[i$2].marker, null, null)); } } }
      for (var i$3 = 0; i$3 < gap; ++i$3)
        { newMarkers.push(gapMarkers); }
      newMarkers.push(last);
    }
    return newMarkers
  }

  // Remove spans that are empty and don't have a clearWhenEmpty
  // option of false.
  function clearEmptySpans(spans) {
    for (var i = 0; i < spans.length; ++i) {
      var span = spans[i];
      if (span.from != null && span.from == span.to && span.marker.clearWhenEmpty !== false)
        { spans.splice(i--, 1); }
    }
    if (!spans.length) { return null }
    return spans
  }

  // Used to 'clip' out readOnly ranges when making a change.
  function removeReadOnlyRanges(doc, from, to) {
    var markers = null;
    doc.iter(from.line, to.line + 1, function (line) {
      if (line.markedSpans) { for (var i = 0; i < line.markedSpans.length; ++i) {
        var mark = line.markedSpans[i].marker;
        if (mark.readOnly && (!markers || indexOf(markers, mark) == -1))
          { (markers || (markers = [])).push(mark); }
      } }
    });
    if (!markers) { return null }
    var parts = [{from: from, to: to}];
    for (var i = 0; i < markers.length; ++i) {
      var mk = markers[i], m = mk.find(0);
      for (var j = 0; j < parts.length; ++j) {
        var p = parts[j];
        if (cmp(p.to, m.from) < 0 || cmp(p.from, m.to) > 0) { continue }
        var newParts = [j, 1], dfrom = cmp(p.from, m.from), dto = cmp(p.to, m.to);
        if (dfrom < 0 || !mk.inclusiveLeft && !dfrom)
          { newParts.push({from: p.from, to: m.from}); }
        if (dto > 0 || !mk.inclusiveRight && !dto)
          { newParts.push({from: m.to, to: p.to}); }
        parts.splice.apply(parts, newParts);
        j += newParts.length - 3;
      }
    }
    return parts
  }

  // Connect or disconnect spans from a line.
  function detachMarkedSpans(line) {
    var spans = line.markedSpans;
    if (!spans) { return }
    for (var i = 0; i < spans.length; ++i)
      { spans[i].marker.detachLine(line); }
    line.markedSpans = null;
  }
  function attachMarkedSpans(line, spans) {
    if (!spans) { return }
    for (var i = 0; i < spans.length; ++i)
      { spans[i].marker.attachLine(line); }
    line.markedSpans = spans;
  }

  // Helpers used when computing which overlapping collapsed span
  // counts as the larger one.
  function extraLeft(marker) { return marker.inclusiveLeft ? -1 : 0 }
  function extraRight(marker) { return marker.inclusiveRight ? 1 : 0 }

  // Returns a number indicating which of two overlapping collapsed
  // spans is larger (and thus includes the other). Falls back to
  // comparing ids when the spans cover exactly the same range.
  function compareCollapsedMarkers(a, b) {
    var lenDiff = a.lines.length - b.lines.length;
    if (lenDiff != 0) { return lenDiff }
    var aPos = a.find(), bPos = b.find();
    var fromCmp = cmp(aPos.from, bPos.from) || extraLeft(a) - extraLeft(b);
    if (fromCmp) { return -fromCmp }
    var toCmp = cmp(aPos.to, bPos.to) || extraRight(a) - extraRight(b);
    if (toCmp) { return toCmp }
    return b.id - a.id
  }

  // Find out whether a line ends or starts in a collapsed span. If
  // so, return the marker for that span.
  function collapsedSpanAtSide(line, start) {
    var sps = sawCollapsedSpans && line.markedSpans, found;
    if (sps) { for (var sp = (void 0), i = 0; i < sps.length; ++i) {
      sp = sps[i];
      if (sp.marker.collapsed && (start ? sp.from : sp.to) == null &&
          (!found || compareCollapsedMarkers(found, sp.marker) < 0))
        { found = sp.marker; }
    } }
    return found
  }
  function collapsedSpanAtStart(line) { return collapsedSpanAtSide(line, true) }
  function collapsedSpanAtEnd(line) { return collapsedSpanAtSide(line, false) }

  function collapsedSpanAround(line, ch) {
    var sps = sawCollapsedSpans && line.markedSpans, found;
    if (sps) { for (var i = 0; i < sps.length; ++i) {
      var sp = sps[i];
      if (sp.marker.collapsed && (sp.from == null || sp.from < ch) && (sp.to == null || sp.to > ch) &&
          (!found || compareCollapsedMarkers(found, sp.marker) < 0)) { found = sp.marker; }
    } }
    return found
  }

  // Test whether there exists a collapsed span that partially
  // overlaps (covers the start or end, but not both) of a new span.
  // Such overlap is not allowed.
  function conflictingCollapsedRange(doc, lineNo$$1, from, to, marker) {
    var line = getLine(doc, lineNo$$1);
    var sps = sawCollapsedSpans && line.markedSpans;
    if (sps) { for (var i = 0; i < sps.length; ++i) {
      var sp = sps[i];
      if (!sp.marker.collapsed) { continue }
      var found = sp.marker.find(0);
      var fromCmp = cmp(found.from, from) || extraLeft(sp.marker) - extraLeft(marker);
      var toCmp = cmp(found.to, to) || extraRight(sp.marker) - extraRight(marker);
      if (fromCmp >= 0 && toCmp <= 0 || fromCmp <= 0 && toCmp >= 0) { continue }
      if (fromCmp <= 0 && (sp.marker.inclusiveRight && marker.inclusiveLeft ? cmp(found.to, from) >= 0 : cmp(found.to, from) > 0) ||
          fromCmp >= 0 && (sp.marker.inclusiveRight && marker.inclusiveLeft ? cmp(found.from, to) <= 0 : cmp(found.from, to) < 0))
        { return true }
    } }
  }

  // A visual line is a line as drawn on the screen. Folding, for
  // example, can cause multiple logical lines to appear on the same
  // visual line. This finds the start of the visual line that the
  // given line is part of (usually that is the line itself).
  function visualLine(line) {
    var merged;
    while (merged = collapsedSpanAtStart(line))
      { line = merged.find(-1, true).line; }
    return line
  }

  function visualLineEnd(line) {
    var merged;
    while (merged = collapsedSpanAtEnd(line))
      { line = merged.find(1, true).line; }
    return line
  }

  // Returns an array of logical lines that continue the visual line
  // started by the argument, or undefined if there are no such lines.
  function visualLineContinued(line) {
    var merged, lines;
    while (merged = collapsedSpanAtEnd(line)) {
      line = merged.find(1, true).line
      ;(lines || (lines = [])).push(line);
    }
    return lines
  }

  // Get the line number of the start of the visual line that the
  // given line number is part of.
  function visualLineNo(doc, lineN) {
    var line = getLine(doc, lineN), vis = visualLine(line);
    if (line == vis) { return lineN }
    return lineNo(vis)
  }

  // Get the line number of the start of the next visual line after
  // the given line.
  function visualLineEndNo(doc, lineN) {
    if (lineN > doc.lastLine()) { return lineN }
    var line = getLine(doc, lineN), merged;
    if (!lineIsHidden(doc, line)) { return lineN }
    while (merged = collapsedSpanAtEnd(line))
      { line = merged.find(1, true).line; }
    return lineNo(line) + 1
  }

  // Compute whether a line is hidden. Lines count as hidden when they
  // are part of a visual line that starts with another line, or when
  // they are entirely covered by collapsed, non-widget span.
  function lineIsHidden(doc, line) {
    var sps = sawCollapsedSpans && line.markedSpans;
    if (sps) { for (var sp = (void 0), i = 0; i < sps.length; ++i) {
      sp = sps[i];
      if (!sp.marker.collapsed) { continue }
      if (sp.from == null) { return true }
      if (sp.marker.widgetNode) { continue }
      if (sp.from == 0 && sp.marker.inclusiveLeft && lineIsHiddenInner(doc, line, sp))
        { return true }
    } }
  }
  function lineIsHiddenInner(doc, line, span) {
    if (span.to == null) {
      var end = span.marker.find(1, true);
      return lineIsHiddenInner(doc, end.line, getMarkedSpanFor(end.line.markedSpans, span.marker))
    }
    if (span.marker.inclusiveRight && span.to == line.text.length)
      { return true }
    for (var sp = (void 0), i = 0; i < line.markedSpans.length; ++i) {
      sp = line.markedSpans[i];
      if (sp.marker.collapsed && !sp.marker.widgetNode && sp.from == span.to &&
          (sp.to == null || sp.to != span.from) &&
          (sp.marker.inclusiveLeft || span.marker.inclusiveRight) &&
          lineIsHiddenInner(doc, line, sp)) { return true }
    }
  }

  // Find the height above the given line.
  function heightAtLine(lineObj) {
    lineObj = visualLine(lineObj);

    var h = 0, chunk = lineObj.parent;
    for (var i = 0; i < chunk.lines.length; ++i) {
      var line = chunk.lines[i];
      if (line == lineObj) { break }
      else { h += line.height; }
    }
    for (var p = chunk.parent; p; chunk = p, p = chunk.parent) {
      for (var i$1 = 0; i$1 < p.children.length; ++i$1) {
        var cur = p.children[i$1];
        if (cur == chunk) { break }
        else { h += cur.height; }
      }
    }
    return h
  }

  // Compute the character length of a line, taking into account
  // collapsed ranges (see markText) that might hide parts, and join
  // other lines onto it.
  function lineLength(line) {
    if (line.height == 0) { return 0 }
    var len = line.text.length, merged, cur = line;
    while (merged = collapsedSpanAtStart(cur)) {
      var found = merged.find(0, true);
      cur = found.from.line;
      len += found.from.ch - found.to.ch;
    }
    cur = line;
    while (merged = collapsedSpanAtEnd(cur)) {
      var found$1 = merged.find(0, true);
      len -= cur.text.length - found$1.from.ch;
      cur = found$1.to.line;
      len += cur.text.length - found$1.to.ch;
    }
    return len
  }

  // Find the longest line in the document.
  function findMaxLine(cm) {
    var d = cm.display, doc = cm.doc;
    d.maxLine = getLine(doc, doc.first);
    d.maxLineLength = lineLength(d.maxLine);
    d.maxLineChanged = true;
    doc.iter(function (line) {
      var len = lineLength(line);
      if (len > d.maxLineLength) {
        d.maxLineLength = len;
        d.maxLine = line;
      }
    });
  }

  // LINE DATA STRUCTURE

  // Line objects. These hold state related to a line, including
  // highlighting info (the styles array).
  var Line = function(text, markedSpans, estimateHeight) {
    this.text = text;
    attachMarkedSpans(this, markedSpans);
    this.height = estimateHeight ? estimateHeight(this) : 1;
  };

  Line.prototype.lineNo = function () { return lineNo(this) };
  eventMixin(Line);

  // Change the content (text, markers) of a line. Automatically
  // invalidates cached information and tries to re-estimate the
  // line's height.
  function updateLine(line, text, markedSpans, estimateHeight) {
    line.text = text;
    if (line.stateAfter) { line.stateAfter = null; }
    if (line.styles) { line.styles = null; }
    if (line.order != null) { line.order = null; }
    detachMarkedSpans(line);
    attachMarkedSpans(line, markedSpans);
    var estHeight = estimateHeight ? estimateHeight(line) : 1;
    if (estHeight != line.height) { updateLineHeight(line, estHeight); }
  }

  // Detach a line from the document tree and its markers.
  function cleanUpLine(line) {
    line.parent = null;
    detachMarkedSpans(line);
  }

  // Convert a style as returned by a mode (either null, or a string
  // containing one or more styles) to a CSS style. This is cached,
  // and also looks for line-wide styles.
  var styleToClassCache = {}, styleToClassCacheWithMode = {};
  function interpretTokenStyle(style, options) {
    if (!style || /^\s*$/.test(style)) { return null }
    var cache = options.addModeClass ? styleToClassCacheWithMode : styleToClassCache;
    return cache[style] ||
      (cache[style] = style.replace(/\S+/g, "cm-$&"))
  }

  // Render the DOM representation of the text of a line. Also builds
  // up a 'line map', which points at the DOM nodes that represent
  // specific stretches of text, and is used by the measuring code.
  // The returned object contains the DOM node, this map, and
  // information about line-wide styles that were set by the mode.
  function buildLineContent(cm, lineView) {
    // The padding-right forces the element to have a 'border', which
    // is needed on Webkit to be able to get line-level bounding
    // rectangles for it (in measureChar).
    var content = eltP("span", null, null, webkit ? "padding-right: .1px" : null);
    var builder = {pre: eltP("pre", [content], "CodeMirror-line"), content: content,
                   col: 0, pos: 0, cm: cm,
                   trailingSpace: false,
                   splitSpaces: cm.getOption("lineWrapping")};
    lineView.measure = {};

    // Iterate over the logical lines that make up this visual line.
    for (var i = 0; i <= (lineView.rest ? lineView.rest.length : 0); i++) {
      var line = i ? lineView.rest[i - 1] : lineView.line, order = (void 0);
      builder.pos = 0;
      builder.addToken = buildToken;
      // Optionally wire in some hacks into the token-rendering
      // algorithm, to deal with browser quirks.
      if (hasBadBidiRects(cm.display.measure) && (order = getOrder(line, cm.doc.direction)))
        { builder.addToken = buildTokenBadBidi(builder.addToken, order); }
      builder.map = [];
      var allowFrontierUpdate = lineView != cm.display.externalMeasured && lineNo(line);
      insertLineContent(line, builder, getLineStyles(cm, line, allowFrontierUpdate));
      if (line.styleClasses) {
        if (line.styleClasses.bgClass)
          { builder.bgClass = joinClasses(line.styleClasses.bgClass, builder.bgClass || ""); }
        if (line.styleClasses.textClass)
          { builder.textClass = joinClasses(line.styleClasses.textClass, builder.textClass || ""); }
      }

      // Ensure at least a single node is present, for measuring.
      if (builder.map.length == 0)
        { builder.map.push(0, 0, builder.content.appendChild(zeroWidthElement(cm.display.measure))); }

      // Store the map and a cache object for the current logical line
      if (i == 0) {
        lineView.measure.map = builder.map;
        lineView.measure.cache = {};
      } else {
  (lineView.measure.maps || (lineView.measure.maps = [])).push(builder.map)
        ;(lineView.measure.caches || (lineView.measure.caches = [])).push({});
      }
    }

    // See issue #2901
    if (webkit) {
      var last = builder.content.lastChild;
      if (/\bcm-tab\b/.test(last.className) || (last.querySelector && last.querySelector(".cm-tab")))
        { builder.content.className = "cm-tab-wrap-hack"; }
    }

    signal(cm, "renderLine", cm, lineView.line, builder.pre);
    if (builder.pre.className)
      { builder.textClass = joinClasses(builder.pre.className, builder.textClass || ""); }

    return builder
  }

  function defaultSpecialCharPlaceholder(ch) {
    var token = elt("span", "\u2022", "cm-invalidchar");
    token.title = "\\u" + ch.charCodeAt(0).toString(16);
    token.setAttribute("aria-label", token.title);
    return token
  }

  // Build up the DOM representation for a single token, and add it to
  // the line map. Takes care to render special characters separately.
  function buildToken(builder, text, style, startStyle, endStyle, css, attributes) {
    if (!text) { return }
    var displayText = builder.splitSpaces ? splitSpaces(text, builder.trailingSpace) : text;
    var special = builder.cm.state.specialChars, mustWrap = false;
    var content;
    if (!special.test(text)) {
      builder.col += text.length;
      content = document.createTextNode(displayText);
      builder.map.push(builder.pos, builder.pos + text.length, content);
      if (ie && ie_version < 9) { mustWrap = true; }
      builder.pos += text.length;
    } else {
      content = document.createDocumentFragment();
      var pos = 0;
      while (true) {
        special.lastIndex = pos;
        var m = special.exec(text);
        var skipped = m ? m.index - pos : text.length - pos;
        if (skipped) {
          var txt = document.createTextNode(displayText.slice(pos, pos + skipped));
          if (ie && ie_version < 9) { content.appendChild(elt("span", [txt])); }
          else { content.appendChild(txt); }
          builder.map.push(builder.pos, builder.pos + skipped, txt);
          builder.col += skipped;
          builder.pos += skipped;
        }
        if (!m) { break }
        pos += skipped + 1;
        var txt$1 = (void 0);
        if (m[0] == "\t") {
          var tabSize = builder.cm.options.tabSize, tabWidth = tabSize - builder.col % tabSize;
          txt$1 = content.appendChild(elt("span", spaceStr(tabWidth), "cm-tab"));
          txt$1.setAttribute("role", "presentation");
          txt$1.setAttribute("cm-text", "\t");
          builder.col += tabWidth;
        } else if (m[0] == "\r" || m[0] == "\n") {
          txt$1 = content.appendChild(elt("span", m[0] == "\r" ? "\u240d" : "\u2424", "cm-invalidchar"));
          txt$1.setAttribute("cm-text", m[0]);
          builder.col += 1;
        } else {
          txt$1 = builder.cm.options.specialCharPlaceholder(m[0]);
          txt$1.setAttribute("cm-text", m[0]);
          if (ie && ie_version < 9) { content.appendChild(elt("span", [txt$1])); }
          else { content.appendChild(txt$1); }
          builder.col += 1;
        }
        builder.map.push(builder.pos, builder.pos + 1, txt$1);
        builder.pos++;
      }
    }
    builder.trailingSpace = displayText.charCodeAt(text.length - 1) == 32;
    if (style || startStyle || endStyle || mustWrap || css) {
      var fullStyle = style || "";
      if (startStyle) { fullStyle += startStyle; }
      if (endStyle) { fullStyle += endStyle; }
      var token = elt("span", [content], fullStyle, css);
      if (attributes) {
        for (var attr in attributes) { if (attributes.hasOwnProperty(attr) && attr != "style" && attr != "class")
          { token.setAttribute(attr, attributes[attr]); } }
      }
      return builder.content.appendChild(token)
    }
    builder.content.appendChild(content);
  }

  // Change some spaces to NBSP to prevent the browser from collapsing
  // trailing spaces at the end of a line when rendering text (issue #1362).
  function splitSpaces(text, trailingBefore) {
    if (text.length > 1 && !/  /.test(text)) { return text }
    var spaceBefore = trailingBefore, result = "";
    for (var i = 0; i < text.length; i++) {
      var ch = text.charAt(i);
      if (ch == " " && spaceBefore && (i == text.length - 1 || text.charCodeAt(i + 1) == 32))
        { ch = "\u00a0"; }
      result += ch;
      spaceBefore = ch == " ";
    }
    return result
  }

  // Work around nonsense dimensions being reported for stretches of
  // right-to-left text.
  function buildTokenBadBidi(inner, order) {
    return function (builder, text, style, startStyle, endStyle, css, attributes) {
      style = style ? style + " cm-force-border" : "cm-force-border";
      var start = builder.pos, end = start + text.length;
      for (;;) {
        // Find the part that overlaps with the start of this text
        var part = (void 0);
        for (var i = 0; i < order.length; i++) {
          part = order[i];
          if (part.to > start && part.from <= start) { break }
        }
        if (part.to >= end) { return inner(builder, text, style, startStyle, endStyle, css, attributes) }
        inner(builder, text.slice(0, part.to - start), style, startStyle, null, css, attributes);
        startStyle = null;
        text = text.slice(part.to - start);
        start = part.to;
      }
    }
  }

  function buildCollapsedSpan(builder, size, marker, ignoreWidget) {
    var widget = !ignoreWidget && marker.widgetNode;
    if (widget) { builder.map.push(builder.pos, builder.pos + size, widget); }
    if (!ignoreWidget && builder.cm.display.input.needsContentAttribute) {
      if (!widget)
        { widget = builder.content.appendChild(document.createElement("span")); }
      widget.setAttribute("cm-marker", marker.id);
    }
    if (widget) {
      builder.cm.display.input.setUneditable(widget);
      builder.content.appendChild(widget);
    }
    builder.pos += size;
    builder.trailingSpace = false;
  }

  // Outputs a number of spans to make up a line, taking highlighting
  // and marked text into account.
  function insertLineContent(line, builder, styles) {
    var spans = line.markedSpans, allText = line.text, at = 0;
    if (!spans) {
      for (var i$1 = 1; i$1 < styles.length; i$1+=2)
        { builder.addToken(builder, allText.slice(at, at = styles[i$1]), interpretTokenStyle(styles[i$1+1], builder.cm.options)); }
      return
    }

    var len = allText.length, pos = 0, i = 1, text = "", style, css;
    var nextChange = 0, spanStyle, spanEndStyle, spanStartStyle, collapsed, attributes;
    for (;;) {
      if (nextChange == pos) { // Update current marker set
        spanStyle = spanEndStyle = spanStartStyle = css = "";
        attributes = null;
        collapsed = null; nextChange = Infinity;
        var foundBookmarks = [], endStyles = (void 0);
        for (var j = 0; j < spans.length; ++j) {
          var sp = spans[j], m = sp.marker;
          if (m.type == "bookmark" && sp.from == pos && m.widgetNode) {
            foundBookmarks.push(m);
          } else if (sp.from <= pos && (sp.to == null || sp.to > pos || m.collapsed && sp.to == pos && sp.from == pos)) {
            if (sp.to != null && sp.to != pos && nextChange > sp.to) {
              nextChange = sp.to;
              spanEndStyle = "";
            }
            if (m.className) { spanStyle += " " + m.className; }
            if (m.css) { css = (css ? css + ";" : "") + m.css; }
            if (m.startStyle && sp.from == pos) { spanStartStyle += " " + m.startStyle; }
            if (m.endStyle && sp.to == nextChange) { (endStyles || (endStyles = [])).push(m.endStyle, sp.to); }
            // support for the old title property
            // https://github.com/codemirror/CodeMirror/pull/5673
            if (m.title) { (attributes || (attributes = {})).title = m.title; }
            if (m.attributes) {
              for (var attr in m.attributes)
                { (attributes || (attributes = {}))[attr] = m.attributes[attr]; }
            }
            if (m.collapsed && (!collapsed || compareCollapsedMarkers(collapsed.marker, m) < 0))
              { collapsed = sp; }
          } else if (sp.from > pos && nextChange > sp.from) {
            nextChange = sp.from;
          }
        }
        if (endStyles) { for (var j$1 = 0; j$1 < endStyles.length; j$1 += 2)
          { if (endStyles[j$1 + 1] == nextChange) { spanEndStyle += " " + endStyles[j$1]; } } }

        if (!collapsed || collapsed.from == pos) { for (var j$2 = 0; j$2 < foundBookmarks.length; ++j$2)
          { buildCollapsedSpan(builder, 0, foundBookmarks[j$2]); } }
        if (collapsed && (collapsed.from || 0) == pos) {
          buildCollapsedSpan(builder, (collapsed.to == null ? len + 1 : collapsed.to) - pos,
                             collapsed.marker, collapsed.from == null);
          if (collapsed.to == null) { return }
          if (collapsed.to == pos) { collapsed = false; }
        }
      }
      if (pos >= len) { break }

      var upto = Math.min(len, nextChange);
      while (true) {
        if (text) {
          var end = pos + text.length;
          if (!collapsed) {
            var tokenText = end > upto ? text.slice(0, upto - pos) : text;
            builder.addToken(builder, tokenText, style ? style + spanStyle : spanStyle,
                             spanStartStyle, pos + tokenText.length == nextChange ? spanEndStyle : "", css, attributes);
          }
          if (end >= upto) {text = text.slice(upto - pos); pos = upto; break}
          pos = end;
          spanStartStyle = "";
        }
        text = allText.slice(at, at = styles[i++]);
        style = interpretTokenStyle(styles[i++], builder.cm.options);
      }
    }
  }


  // These objects are used to represent the visible (currently drawn)
  // part of the document. A LineView may correspond to multiple
  // logical lines, if those are connected by collapsed ranges.
  function LineView(doc, line, lineN) {
    // The starting line
    this.line = line;
    // Continuing lines, if any
    this.rest = visualLineContinued(line);
    // Number of logical lines in this visual line
    this.size = this.rest ? lineNo(lst(this.rest)) - lineN + 1 : 1;
    this.node = this.text = null;
    this.hidden = lineIsHidden(doc, line);
  }

  // Create a range of LineView objects for the given lines.
  function buildViewArray(cm, from, to) {
    var array = [], nextPos;
    for (var pos = from; pos < to; pos = nextPos) {
      var view = new LineView(cm.doc, getLine(cm.doc, pos), pos);
      nextPos = pos + view.size;
      array.push(view);
    }
    return array
  }

  var operationGroup = null;

  function pushOperation(op) {
    if (operationGroup) {
      operationGroup.ops.push(op);
    } else {
      op.ownsGroup = operationGroup = {
        ops: [op],
        delayedCallbacks: []
      };
    }
  }

  function fireCallbacksForOps(group) {
    // Calls delayed callbacks and cursorActivity handlers until no
    // new ones appear
    var callbacks = group.delayedCallbacks, i = 0;
    do {
      for (; i < callbacks.length; i++)
        { callbacks[i].call(null); }
      for (var j = 0; j < group.ops.length; j++) {
        var op = group.ops[j];
        if (op.cursorActivityHandlers)
          { while (op.cursorActivityCalled < op.cursorActivityHandlers.length)
            { op.cursorActivityHandlers[op.cursorActivityCalled++].call(null, op.cm); } }
      }
    } while (i < callbacks.length)
  }

  function finishOperation(op, endCb) {
    var group = op.ownsGroup;
    if (!group) { return }

    try { fireCallbacksForOps(group); }
    finally {
      operationGroup = null;
      endCb(group);
    }
  }

  var orphanDelayedCallbacks = null;

  // Often, we want to signal events at a point where we are in the
  // middle of some work, but don't want the handler to start calling
  // other methods on the editor, which might be in an inconsistent
  // state or simply not expect any other events to happen.
  // signalLater looks whether there are any handlers, and schedules
  // them to be executed when the last operation ends, or, if no
  // operation is active, when a timeout fires.
  function signalLater(emitter, type /*, values...*/) {
    var arr = getHandlers(emitter, type);
    if (!arr.length) { return }
    var args = Array.prototype.slice.call(arguments, 2), list;
    if (operationGroup) {
      list = operationGroup.delayedCallbacks;
    } else if (orphanDelayedCallbacks) {
      list = orphanDelayedCallbacks;
    } else {
      list = orphanDelayedCallbacks = [];
      setTimeout(fireOrphanDelayed, 0);
    }
    var loop = function ( i ) {
      list.push(function () { return arr[i].apply(null, args); });
    };

    for (var i = 0; i < arr.length; ++i)
      loop( i );
  }

  function fireOrphanDelayed() {
    var delayed = orphanDelayedCallbacks;
    orphanDelayedCallbacks = null;
    for (var i = 0; i < delayed.length; ++i) { delayed[i](); }
  }

  // When an aspect of a line changes, a string is added to
  // lineView.changes. This updates the relevant part of the line's
  // DOM structure.
  function updateLineForChanges(cm, lineView, lineN, dims) {
    for (var j = 0; j < lineView.changes.length; j++) {
      var type = lineView.changes[j];
      if (type == "text") { updateLineText(cm, lineView); }
      else if (type == "gutter") { updateLineGutter(cm, lineView, lineN, dims); }
      else if (type == "class") { updateLineClasses(cm, lineView); }
      else if (type == "widget") { updateLineWidgets(cm, lineView, dims); }
    }
    lineView.changes = null;
  }

  // Lines with gutter elements, widgets or a background class need to
  // be wrapped, and have the extra elements added to the wrapper div
  function ensureLineWrapped(lineView) {
    if (lineView.node == lineView.text) {
      lineView.node = elt("div", null, null, "position: relative");
      if (lineView.text.parentNode)
        { lineView.text.parentNode.replaceChild(lineView.node, lineView.text); }
      lineView.node.appendChild(lineView.text);
      if (ie && ie_version < 8) { lineView.node.style.zIndex = 2; }
    }
    return lineView.node
  }

  function updateLineBackground(cm, lineView) {
    var cls = lineView.bgClass ? lineView.bgClass + " " + (lineView.line.bgClass || "") : lineView.line.bgClass;
    if (cls) { cls += " CodeMirror-linebackground"; }
    if (lineView.background) {
      if (cls) { lineView.background.className = cls; }
      else { lineView.background.parentNode.removeChild(lineView.background); lineView.background = null; }
    } else if (cls) {
      var wrap = ensureLineWrapped(lineView);
      lineView.background = wrap.insertBefore(elt("div", null, cls), wrap.firstChild);
      cm.display.input.setUneditable(lineView.background);
    }
  }

  // Wrapper around buildLineContent which will reuse the structure
  // in display.externalMeasured when possible.
  function getLineContent(cm, lineView) {
    var ext = cm.display.externalMeasured;
    if (ext && ext.line == lineView.line) {
      cm.display.externalMeasured = null;
      lineView.measure = ext.measure;
      return ext.built
    }
    return buildLineContent(cm, lineView)
  }

  // Redraw the line's text. Interacts with the background and text
  // classes because the mode may output tokens that influence these
  // classes.
  function updateLineText(cm, lineView) {
    var cls = lineView.text.className;
    var built = getLineContent(cm, lineView);
    if (lineView.text == lineView.node) { lineView.node = built.pre; }
    lineView.text.parentNode.replaceChild(built.pre, lineView.text);
    lineView.text = built.pre;
    if (built.bgClass != lineView.bgClass || built.textClass != lineView.textClass) {
      lineView.bgClass = built.bgClass;
      lineView.textClass = built.textClass;
      updateLineClasses(cm, lineView);
    } else if (cls) {
      lineView.text.className = cls;
    }
  }

  function updateLineClasses(cm, lineView) {
    updateLineBackground(cm, lineView);
    if (lineView.line.wrapClass)
      { ensureLineWrapped(lineView).className = lineView.line.wrapClass; }
    else if (lineView.node != lineView.text)
      { lineView.node.className = ""; }
    var textClass = lineView.textClass ? lineView.textClass + " " + (lineView.line.textClass || "") : lineView.line.textClass;
    lineView.text.className = textClass || "";
  }

  function updateLineGutter(cm, lineView, lineN, dims) {
    if (lineView.gutter) {
      lineView.node.removeChild(lineView.gutter);
      lineView.gutter = null;
    }
    if (lineView.gutterBackground) {
      lineView.node.removeChild(lineView.gutterBackground);
      lineView.gutterBackground = null;
    }
    if (lineView.line.gutterClass) {
      var wrap = ensureLineWrapped(lineView);
      lineView.gutterBackground = elt("div", null, "CodeMirror-gutter-background " + lineView.line.gutterClass,
                                      ("left: " + (cm.options.fixedGutter ? dims.fixedPos : -dims.gutterTotalWidth) + "px; width: " + (dims.gutterTotalWidth) + "px"));
      cm.display.input.setUneditable(lineView.gutterBackground);
      wrap.insertBefore(lineView.gutterBackground, lineView.text);
    }
    var markers = lineView.line.gutterMarkers;
    if (cm.options.lineNumbers || markers) {
      var wrap$1 = ensureLineWrapped(lineView);
      var gutterWrap = lineView.gutter = elt("div", null, "CodeMirror-gutter-wrapper", ("left: " + (cm.options.fixedGutter ? dims.fixedPos : -dims.gutterTotalWidth) + "px"));
      cm.display.input.setUneditable(gutterWrap);
      wrap$1.insertBefore(gutterWrap, lineView.text);
      if (lineView.line.gutterClass)
        { gutterWrap.className += " " + lineView.line.gutterClass; }
      if (cm.options.lineNumbers && (!markers || !markers["CodeMirror-linenumbers"]))
        { lineView.lineNumber = gutterWrap.appendChild(
          elt("div", lineNumberFor(cm.options, lineN),
              "CodeMirror-linenumber CodeMirror-gutter-elt",
              ("left: " + (dims.gutterLeft["CodeMirror-linenumbers"]) + "px; width: " + (cm.display.lineNumInnerWidth) + "px"))); }
      if (markers) { for (var k = 0; k < cm.display.gutterSpecs.length; ++k) {
        var id = cm.display.gutterSpecs[k].className, found = markers.hasOwnProperty(id) && markers[id];
        if (found)
          { gutterWrap.appendChild(elt("div", [found], "CodeMirror-gutter-elt",
                                     ("left: " + (dims.gutterLeft[id]) + "px; width: " + (dims.gutterWidth[id]) + "px"))); }
      } }
    }
  }

  function updateLineWidgets(cm, lineView, dims) {
    if (lineView.alignable) { lineView.alignable = null; }
    for (var node = lineView.node.firstChild, next = (void 0); node; node = next) {
      next = node.nextSibling;
      if (node.className == "CodeMirror-linewidget")
        { lineView.node.removeChild(node); }
    }
    insertLineWidgets(cm, lineView, dims);
  }

  // Build a line's DOM representation from scratch
  function buildLineElement(cm, lineView, lineN, dims) {
    var built = getLineContent(cm, lineView);
    lineView.text = lineView.node = built.pre;
    if (built.bgClass) { lineView.bgClass = built.bgClass; }
    if (built.textClass) { lineView.textClass = built.textClass; }

    updateLineClasses(cm, lineView);
    updateLineGutter(cm, lineView, lineN, dims);
    insertLineWidgets(cm, lineView, dims);
    return lineView.node
  }

  // A lineView may contain multiple logical lines (when merged by
  // collapsed spans). The widgets for all of them need to be drawn.
  function insertLineWidgets(cm, lineView, dims) {
    insertLineWidgetsFor(cm, lineView.line, lineView, dims, true);
    if (lineView.rest) { for (var i = 0; i < lineView.rest.length; i++)
      { insertLineWidgetsFor(cm, lineView.rest[i], lineView, dims, false); } }
  }

  function insertLineWidgetsFor(cm, line, lineView, dims, allowAbove) {
    if (!line.widgets) { return }
    var wrap = ensureLineWrapped(lineView);
    for (var i = 0, ws = line.widgets; i < ws.length; ++i) {
      var widget = ws[i], node = elt("div", [widget.node], "CodeMirror-linewidget");
      if (!widget.handleMouseEvents) { node.setAttribute("cm-ignore-events", "true"); }
      positionLineWidget(widget, node, lineView, dims);
      cm.display.input.setUneditable(node);
      if (allowAbove && widget.above)
        { wrap.insertBefore(node, lineView.gutter || lineView.text); }
      else
        { wrap.appendChild(node); }
      signalLater(widget, "redraw");
    }
  }

  function positionLineWidget(widget, node, lineView, dims) {
    if (widget.noHScroll) {
  (lineView.alignable || (lineView.alignable = [])).push(node);
      var width = dims.wrapperWidth;
      node.style.left = dims.fixedPos + "px";
      if (!widget.coverGutter) {
        width -= dims.gutterTotalWidth;
        node.style.paddingLeft = dims.gutterTotalWidth + "px";
      }
      node.style.width = width + "px";
    }
    if (widget.coverGutter) {
      node.style.zIndex = 5;
      node.style.position = "relative";
      if (!widget.noHScroll) { node.style.marginLeft = -dims.gutterTotalWidth + "px"; }
    }
  }

  function widgetHeight(widget) {
    if (widget.height != null) { return widget.height }
    var cm = widget.doc.cm;
    if (!cm) { return 0 }
    if (!contains(document.body, widget.node)) {
      var parentStyle = "position: relative;";
      if (widget.coverGutter)
        { parentStyle += "margin-left: -" + cm.display.gutters.offsetWidth + "px;"; }
      if (widget.noHScroll)
        { parentStyle += "width: " + cm.display.wrapper.clientWidth + "px;"; }
      removeChildrenAndAdd(cm.display.measure, elt("div", [widget.node], null, parentStyle));
    }
    return widget.height = widget.node.parentNode.offsetHeight
  }

  // Return true when the given mouse event happened in a widget
  function eventInWidget(display, e) {
    for (var n = e_target(e); n != display.wrapper; n = n.parentNode) {
      if (!n || (n.nodeType == 1 && n.getAttribute("cm-ignore-events") == "true") ||
          (n.parentNode == display.sizer && n != display.mover))
        { return true }
    }
  }

  // POSITION MEASUREMENT

  function paddingTop(display) {return display.lineSpace.offsetTop}
  function paddingVert(display) {return display.mover.offsetHeight - display.lineSpace.offsetHeight}
  function paddingH(display) {
    if (display.cachedPaddingH) { return display.cachedPaddingH }
    var e = removeChildrenAndAdd(display.measure, elt("pre", "x"));
    var style = window.getComputedStyle ? window.getComputedStyle(e) : e.currentStyle;
    var data = {left: parseInt(style.paddingLeft), right: parseInt(style.paddingRight)};
    if (!isNaN(data.left) && !isNaN(data.right)) { display.cachedPaddingH = data; }
    return data
  }

  function scrollGap(cm) { return scrollerGap - cm.display.nativeBarWidth }
  function displayWidth(cm) {
    return cm.display.scroller.clientWidth - scrollGap(cm) - cm.display.barWidth
  }
  function displayHeight(cm) {
    return cm.display.scroller.clientHeight - scrollGap(cm) - cm.display.barHeight
  }

  // Ensure the lineView.wrapping.heights array is populated. This is
  // an array of bottom offsets for the lines that make up a drawn
  // line. When lineWrapping is on, there might be more than one
  // height.
  function ensureLineHeights(cm, lineView, rect) {
    var wrapping = cm.options.lineWrapping;
    var curWidth = wrapping && displayWidth(cm);
    if (!lineView.measure.heights || wrapping && lineView.measure.width != curWidth) {
      var heights = lineView.measure.heights = [];
      if (wrapping) {
        lineView.measure.width = curWidth;
        var rects = lineView.text.firstChild.getClientRects();
        for (var i = 0; i < rects.length - 1; i++) {
          var cur = rects[i], next = rects[i + 1];
          if (Math.abs(cur.bottom - next.bottom) > 2)
            { heights.push((cur.bottom + next.top) / 2 - rect.top); }
        }
      }
      heights.push(rect.bottom - rect.top);
    }
  }

  // Find a line map (mapping character offsets to text nodes) and a
  // measurement cache for the given line number. (A line view might
  // contain multiple lines when collapsed ranges are present.)
  function mapFromLineView(lineView, line, lineN) {
    if (lineView.line == line)
      { return {map: lineView.measure.map, cache: lineView.measure.cache} }
    for (var i = 0; i < lineView.rest.length; i++)
      { if (lineView.rest[i] == line)
        { return {map: lineView.measure.maps[i], cache: lineView.measure.caches[i]} } }
    for (var i$1 = 0; i$1 < lineView.rest.length; i$1++)
      { if (lineNo(lineView.rest[i$1]) > lineN)
        { return {map: lineView.measure.maps[i$1], cache: lineView.measure.caches[i$1], before: true} } }
  }

  // Render a line into the hidden node display.externalMeasured. Used
  // when measurement is needed for a line that's not in the viewport.
  function updateExternalMeasurement(cm, line) {
    line = visualLine(line);
    var lineN = lineNo(line);
    var view = cm.display.externalMeasured = new LineView(cm.doc, line, lineN);
    view.lineN = lineN;
    var built = view.built = buildLineContent(cm, view);
    view.text = built.pre;
    removeChildrenAndAdd(cm.display.lineMeasure, built.pre);
    return view
  }

  // Get a {top, bottom, left, right} box (in line-local coordinates)
  // for a given character.
  function measureChar(cm, line, ch, bias) {
    return measureCharPrepared(cm, prepareMeasureForLine(cm, line), ch, bias)
  }

  // Find a line view that corresponds to the given line number.
  function findViewForLine(cm, lineN) {
    if (lineN >= cm.display.viewFrom && lineN < cm.display.viewTo)
      { return cm.display.view[findViewIndex(cm, lineN)] }
    var ext = cm.display.externalMeasured;
    if (ext && lineN >= ext.lineN && lineN < ext.lineN + ext.size)
      { return ext }
  }

  // Measurement can be split in two steps, the set-up work that
  // applies to the whole line, and the measurement of the actual
  // character. Functions like coordsChar, that need to do a lot of
  // measurements in a row, can thus ensure that the set-up work is
  // only done once.
  function prepareMeasureForLine(cm, line) {
    var lineN = lineNo(line);
    var view = findViewForLine(cm, lineN);
    if (view && !view.text) {
      view = null;
    } else if (view && view.changes) {
      updateLineForChanges(cm, view, lineN, getDimensions(cm));
      cm.curOp.forceUpdate = true;
    }
    if (!view)
      { view = updateExternalMeasurement(cm, line); }

    var info = mapFromLineView(view, line, lineN);
    return {
      line: line, view: view, rect: null,
      map: info.map, cache: info.cache, before: info.before,
      hasHeights: false
    }
  }

  // Given a prepared measurement object, measures the position of an
  // actual character (or fetches it from the cache).
  function measureCharPrepared(cm, prepared, ch, bias, varHeight) {
    if (prepared.before) { ch = -1; }
    var key = ch + (bias || ""), found;
    if (prepared.cache.hasOwnProperty(key)) {
      found = prepared.cache[key];
    } else {
      if (!prepared.rect)
        { prepared.rect = prepared.view.text.getBoundingClientRect(); }
      if (!prepared.hasHeights) {
        ensureLineHeights(cm, prepared.view, prepared.rect);
        prepared.hasHeights = true;
      }
      found = measureCharInner(cm, prepared, ch, bias);
      if (!found.bogus) { prepared.cache[key] = found; }
    }
    return {left: found.left, right: found.right,
            top: varHeight ? found.rtop : found.top,
            bottom: varHeight ? found.rbottom : found.bottom}
  }

  var nullRect = {left: 0, right: 0, top: 0, bottom: 0};

  function nodeAndOffsetInLineMap(map$$1, ch, bias) {
    var node, start, end, collapse, mStart, mEnd;
    // First, search the line map for the text node corresponding to,
    // or closest to, the target character.
    for (var i = 0; i < map$$1.length; i += 3) {
      mStart = map$$1[i];
      mEnd = map$$1[i + 1];
      if (ch < mStart) {
        start = 0; end = 1;
        collapse = "left";
      } else if (ch < mEnd) {
        start = ch - mStart;
        end = start + 1;
      } else if (i == map$$1.length - 3 || ch == mEnd && map$$1[i + 3] > ch) {
        end = mEnd - mStart;
        start = end - 1;
        if (ch >= mEnd) { collapse = "right"; }
      }
      if (start != null) {
        node = map$$1[i + 2];
        if (mStart == mEnd && bias == (node.insertLeft ? "left" : "right"))
          { collapse = bias; }
        if (bias == "left" && start == 0)
          { while (i && map$$1[i - 2] == map$$1[i - 3] && map$$1[i - 1].insertLeft) {
            node = map$$1[(i -= 3) + 2];
            collapse = "left";
          } }
        if (bias == "right" && start == mEnd - mStart)
          { while (i < map$$1.length - 3 && map$$1[i + 3] == map$$1[i + 4] && !map$$1[i + 5].insertLeft) {
            node = map$$1[(i += 3) + 2];
            collapse = "right";
          } }
        break
      }
    }
    return {node: node, start: start, end: end, collapse: collapse, coverStart: mStart, coverEnd: mEnd}
  }

  function getUsefulRect(rects, bias) {
    var rect = nullRect;
    if (bias == "left") { for (var i = 0; i < rects.length; i++) {
      if ((rect = rects[i]).left != rect.right) { break }
    } } else { for (var i$1 = rects.length - 1; i$1 >= 0; i$1--) {
      if ((rect = rects[i$1]).left != rect.right) { break }
    } }
    return rect
  }

  function measureCharInner(cm, prepared, ch, bias) {
    var place = nodeAndOffsetInLineMap(prepared.map, ch, bias);
    var node = place.node, start = place.start, end = place.end, collapse = place.collapse;

    var rect;
    if (node.nodeType == 3) { // If it is a text node, use a range to retrieve the coordinates.
      for (var i$1 = 0; i$1 < 4; i$1++) { // Retry a maximum of 4 times when nonsense rectangles are returned
        while (start && isExtendingChar(prepared.line.text.charAt(place.coverStart + start))) { --start; }
        while (place.coverStart + end < place.coverEnd && isExtendingChar(prepared.line.text.charAt(place.coverStart + end))) { ++end; }
        if (ie && ie_version < 9 && start == 0 && end == place.coverEnd - place.coverStart)
          { rect = node.parentNode.getBoundingClientRect(); }
        else
          { rect = getUsefulRect(range(node, start, end).getClientRects(), bias); }
        if (rect.left || rect.right || start == 0) { break }
        end = start;
        start = start - 1;
        collapse = "right";
      }
      if (ie && ie_version < 11) { rect = maybeUpdateRectForZooming(cm.display.measure, rect); }
    } else { // If it is a widget, simply get the box for the whole widget.
      if (start > 0) { collapse = bias = "right"; }
      var rects;
      if (cm.options.lineWrapping && (rects = node.getClientRects()).length > 1)
        { rect = rects[bias == "right" ? rects.length - 1 : 0]; }
      else
        { rect = node.getBoundingClientRect(); }
    }
    if (ie && ie_version < 9 && !start && (!rect || !rect.left && !rect.right)) {
      var rSpan = node.parentNode.getClientRects()[0];
      if (rSpan)
        { rect = {left: rSpan.left, right: rSpan.left + charWidth(cm.display), top: rSpan.top, bottom: rSpan.bottom}; }
      else
        { rect = nullRect; }
    }

    var rtop = rect.top - prepared.rect.top, rbot = rect.bottom - prepared.rect.top;
    var mid = (rtop + rbot) / 2;
    var heights = prepared.view.measure.heights;
    var i = 0;
    for (; i < heights.length - 1; i++)
      { if (mid < heights[i]) { break } }
    var top = i ? heights[i - 1] : 0, bot = heights[i];
    var result = {left: (collapse == "right" ? rect.right : rect.left) - prepared.rect.left,
                  right: (collapse == "left" ? rect.left : rect.right) - prepared.rect.left,
                  top: top, bottom: bot};
    if (!rect.left && !rect.right) { result.bogus = true; }
    if (!cm.options.singleCursorHeightPerLine) { result.rtop = rtop; result.rbottom = rbot; }

    return result
  }

  // Work around problem with bounding client rects on ranges being
  // returned incorrectly when zoomed on IE10 and below.
  function maybeUpdateRectForZooming(measure, rect) {
    if (!window.screen || screen.logicalXDPI == null ||
        screen.logicalXDPI == screen.deviceXDPI || !hasBadZoomedRects(measure))
      { return rect }
    var scaleX = screen.logicalXDPI / screen.deviceXDPI;
    var scaleY = screen.logicalYDPI / screen.deviceYDPI;
    return {left: rect.left * scaleX, right: rect.right * scaleX,
            top: rect.top * scaleY, bottom: rect.bottom * scaleY}
  }

  function clearLineMeasurementCacheFor(lineView) {
    if (lineView.measure) {
      lineView.measure.cache = {};
      lineView.measure.heights = null;
      if (lineView.rest) { for (var i = 0; i < lineView.rest.length; i++)
        { lineView.measure.caches[i] = {}; } }
    }
  }

  function clearLineMeasurementCache(cm) {
    cm.display.externalMeasure = null;
    removeChildren(cm.display.lineMeasure);
    for (var i = 0; i < cm.display.view.length; i++)
      { clearLineMeasurementCacheFor(cm.display.view[i]); }
  }

  function clearCaches(cm) {
    clearLineMeasurementCache(cm);
    cm.display.cachedCharWidth = cm.display.cachedTextHeight = cm.display.cachedPaddingH = null;
    if (!cm.options.lineWrapping) { cm.display.maxLineChanged = true; }
    cm.display.lineNumChars = null;
  }

  function pageScrollX() {
    // Work around https://bugs.chromium.org/p/chromium/issues/detail?id=489206
    // which causes page_Offset and bounding client rects to use
    // different reference viewports and invalidate our calculations.
    if (chrome && android) { return -(document.body.getBoundingClientRect().left - parseInt(getComputedStyle(document.body).marginLeft)) }
    return window.pageXOffset || (document.documentElement || document.body).scrollLeft
  }
  function pageScrollY() {
    if (chrome && android) { return -(document.body.getBoundingClientRect().top - parseInt(getComputedStyle(document.body).marginTop)) }
    return window.pageYOffset || (document.documentElement || document.body).scrollTop
  }

  function widgetTopHeight(lineObj) {
    var height = 0;
    if (lineObj.widgets) { for (var i = 0; i < lineObj.widgets.length; ++i) { if (lineObj.widgets[i].above)
      { height += widgetHeight(lineObj.widgets[i]); } } }
    return height
  }

  // Converts a {top, bottom, left, right} box from line-local
  // coordinates into another coordinate system. Context may be one of
  // "line", "div" (display.lineDiv), "local"./null (editor), "window",
  // or "page".
  function intoCoordSystem(cm, lineObj, rect, context, includeWidgets) {
    if (!includeWidgets) {
      var height = widgetTopHeight(lineObj);
      rect.top += height; rect.bottom += height;
    }
    if (context == "line") { return rect }
    if (!context) { context = "local"; }
    var yOff = heightAtLine(lineObj);
    if (context == "local") { yOff += paddingTop(cm.display); }
    else { yOff -= cm.display.viewOffset; }
    if (context == "page" || context == "window") {
      var lOff = cm.display.lineSpace.getBoundingClientRect();
      yOff += lOff.top + (context == "window" ? 0 : pageScrollY());
      var xOff = lOff.left + (context == "window" ? 0 : pageScrollX());
      rect.left += xOff; rect.right += xOff;
    }
    rect.top += yOff; rect.bottom += yOff;
    return rect
  }

  // Coverts a box from "div" coords to another coordinate system.
  // Context may be "window", "page", "div", or "local"./null.
  function fromCoordSystem(cm, coords, context) {
    if (context == "div") { return coords }
    var left = coords.left, top = coords.top;
    // First move into "page" coordinate system
    if (context == "page") {
      left -= pageScrollX();
      top -= pageScrollY();
    } else if (context == "local" || !context) {
      var localBox = cm.display.sizer.getBoundingClientRect();
      left += localBox.left;
      top += localBox.top;
    }

    var lineSpaceBox = cm.display.lineSpace.getBoundingClientRect();
    return {left: left - lineSpaceBox.left, top: top - lineSpaceBox.top}
  }

  function charCoords(cm, pos, context, lineObj, bias) {
    if (!lineObj) { lineObj = getLine(cm.doc, pos.line); }
    return intoCoordSystem(cm, lineObj, measureChar(cm, lineObj, pos.ch, bias), context)
  }

  function cursorCoords(cm, pos, context, lineObj, preparedMeasure, varHeight) {
    lineObj = lineObj || getLine(cm.doc, pos.line);
    if (!preparedMeasure) { preparedMeasure = prepareMeasureForLine(cm, lineObj); }
    function get(ch, right) {
      var m = measureCharPrepared(cm, preparedMeasure, ch, right ? "right" : "left", varHeight);
      if (right) { m.left = m.right; } else { m.right = m.left; }
      return intoCoordSystem(cm, lineObj, m, context)
    }
    var order = getOrder(lineObj, cm.doc.direction), ch = pos.ch, sticky = pos.sticky;
    if (ch >= lineObj.text.length) {
      ch = lineObj.text.length;
      sticky = "before";
    } else if (ch <= 0) {
      ch = 0;
      sticky = "after";
    }
    if (!order) { return get(sticky == "before" ? ch - 1 : ch, sticky == "before") }

    function getBidi(ch, partPos, invert) {
      var part = order[partPos], right = part.level == 1;
      return get(invert ? ch - 1 : ch, right != invert)
    }
    var partPos = getBidiPartAt(order, ch, sticky);
    var other = bidiOther;
    var val = getBidi(ch, partPos, sticky == "before");
    if (other != null) { val.other = getBidi(ch, other, sticky != "before"); }
    return val
  }

  // Used to cheaply estimate the coordinates for a position. Used for
  // intermediate scroll updates.
  function estimateCoords(cm, pos) {
    var left = 0;
    pos = clipPos(cm.doc, pos);
    if (!cm.options.lineWrapping) { left = charWidth(cm.display) * pos.ch; }
    var lineObj = getLine(cm.doc, pos.line);
    var top = heightAtLine(lineObj) + paddingTop(cm.display);
    return {left: left, right: left, top: top, bottom: top + lineObj.height}
  }

  // Positions returned by coordsChar contain some extra information.
  // xRel is the relative x position of the input coordinates compared
  // to the found position (so xRel > 0 means the coordinates are to
  // the right of the character position, for example). When outside
  // is true, that means the coordinates lie outside the line's
  // vertical range.
  function PosWithInfo(line, ch, sticky, outside, xRel) {
    var pos = Pos(line, ch, sticky);
    pos.xRel = xRel;
    if (outside) { pos.outside = true; }
    return pos
  }

  // Compute the character position closest to the given coordinates.
  // Input must be lineSpace-local ("div" coordinate system).
  function coordsChar(cm, x, y) {
    var doc = cm.doc;
    y += cm.display.viewOffset;
    if (y < 0) { return PosWithInfo(doc.first, 0, null, true, -1) }
    var lineN = lineAtHeight(doc, y), last = doc.first + doc.size - 1;
    if (lineN > last)
      { return PosWithInfo(doc.first + doc.size - 1, getLine(doc, last).text.length, null, true, 1) }
    if (x < 0) { x = 0; }

    var lineObj = getLine(doc, lineN);
    for (;;) {
      var found = coordsCharInner(cm, lineObj, lineN, x, y);
      var collapsed = collapsedSpanAround(lineObj, found.ch + (found.xRel > 0 ? 1 : 0));
      if (!collapsed) { return found }
      var rangeEnd = collapsed.find(1);
      if (rangeEnd.line == lineN) { return rangeEnd }
      lineObj = getLine(doc, lineN = rangeEnd.line);
    }
  }

  function wrappedLineExtent(cm, lineObj, preparedMeasure, y) {
    y -= widgetTopHeight(lineObj);
    var end = lineObj.text.length;
    var begin = findFirst(function (ch) { return measureCharPrepared(cm, preparedMeasure, ch - 1).bottom <= y; }, end, 0);
    end = findFirst(function (ch) { return measureCharPrepared(cm, preparedMeasure, ch).top > y; }, begin, end);
    return {begin: begin, end: end}
  }

  function wrappedLineExtentChar(cm, lineObj, preparedMeasure, target) {
    if (!preparedMeasure) { preparedMeasure = prepareMeasureForLine(cm, lineObj); }
    var targetTop = intoCoordSystem(cm, lineObj, measureCharPrepared(cm, preparedMeasure, target), "line").top;
    return wrappedLineExtent(cm, lineObj, preparedMeasure, targetTop)
  }

  // Returns true if the given side of a box is after the given
  // coordinates, in top-to-bottom, left-to-right order.
  function boxIsAfter(box, x, y, left) {
    return box.bottom <= y ? false : box.top > y ? true : (left ? box.left : box.right) > x
  }

  function coordsCharInner(cm, lineObj, lineNo$$1, x, y) {
    // Move y into line-local coordinate space
    y -= heightAtLine(lineObj);
    var preparedMeasure = prepareMeasureForLine(cm, lineObj);
   
    // for the widgets at this line.
    var widgetHeight$$1 = widgetTopHeight(lineObj);
    var begin = 0, end = lineObj.text.length, ltr = true;

    var order = getOrder(lineObj, cm.doc.direction);
    // If the line isn't plain left-to-right text, first figure out
    // which bidi section the coordinates fall into.
    if (order) {
      var part = (cm.options.lineWrapping ? coordsBidiPartWrapped : coordsBidiPart)
                   (cm, lineObj, lineNo$$1, preparedMeasure, order, x, y);
      ltr = part.level != 1;
      // The awkward -1 offsets are needed because findFirst (called
      // on these below) will treat its first bound as inclusive,
      // second as exclusive, but we want to actually address the
      // characters in the part's range
      begin = ltr ? part.from : part.to - 1;
      end = ltr ? part.to : part.from - 1;
    }

    // A binary search to find the first character whose bounding box
    // starts after the coordinates. If we run across any whose box wrap
    // the coordinates, store that.
    var chAround = null, boxAround = null;
    var ch = findFirst(function (ch) {
      var box = measureCharPrepared(cm, preparedMeasure, ch);
      box.top += widgetHeight$$1; box.bottom += widgetHeight$$1;
      if (!boxIsAfter(box, x, y, false)) { return false }
      if (box.top <= y && box.left <= x) {
        chAround = ch;
        boxAround = box;
      }
      return true
    }, begin, end);

    var baseX, sticky, outside = false;
    // If a box around the coordinates was found, use that
    if (boxAround) {
      // Distinguish coordinates nearer to the left or right side of the box
      var atLeft = x - boxAround.left < boxAround.right - x, atStart = atLeft == ltr;
      ch = chAround + (atStart ? 0 : 1);
      sticky = atStart ? "after" : "before";
      baseX = atLeft ? boxAround.left : boxAround.right;
    } else {
      // (Adjust for extended bound, if necessary.)
      if (!ltr && (ch == end || ch == begin)) { ch++; }
      // To determine which side to associate with, get the box to the
      // left of the character and compare it's vertical position to the
      // coordinates
      sticky = ch == 0 ? "after" : ch == lineObj.text.length ? "before" :
        (measureCharPrepared(cm, preparedMeasure, ch - (ltr ? 1 : 0)).bottom + widgetHeight$$1 <= y) == ltr ?
        "after" : "before";
      // Now get accurate coordinates for this place, in order to get a
      // base X position
      var coords = cursorCoords(cm, Pos(lineNo$$1, ch, sticky), "line", lineObj, preparedMeasure);
      baseX = coords.left;
      outside = y < coords.top || y >= coords.bottom;
    }

    ch = skipExtendingChars(lineObj.text, ch, 1);
    return PosWithInfo(lineNo$$1, ch, sticky, outside, x - baseX)
  }

  function coordsBidiPart(cm, lineObj, lineNo$$1, preparedMeasure, order, x, y) {
    // Bidi parts are sorted left-to-right, and in a non-line-wrapping
    // situation, we can take this ordering to correspond to the visual
    // ordering. This finds the first part whose end is after the given
    // coordinates.
    var index = findFirst(function (i) {
      var part = order[i], ltr = part.level != 1;
      return boxIsAfter(cursorCoords(cm, Pos(lineNo$$1, ltr ? part.to : part.from, ltr ? "before" : "after"),
                                     "line", lineObj, preparedMeasure), x, y, true)
    }, 0, order.length - 1);
    var part = order[index];
    // If this isn't the first part, the part's start is also after
    // the coordinates, and the coordinates aren't on the same line as
    // that start, move one part back.
    if (index > 0) {
      var ltr = part.level != 1;
      var start = cursorCoords(cm, Pos(lineNo$$1, ltr ? part.from : part.to, ltr ? "after" : "before"),
                               "line", lineObj, preparedMeasure);
      if (boxIsAfter(start, x, y, true) && start.top > y)
        { part = order[index - 1]; }
    }
    return part
  }

  function coordsBidiPartWrapped(cm, lineObj, _lineNo, preparedMeasure, order, x, y) {
    // In a wrapped line, rtl text on wrapping boundaries can do things
    // all, so a binary search doesn't work, and we want to return a
    // part that only spans one line so that the binary search in
    // coordsCharInner is safe. As such, we first find the extent of the
    // wrapped line, and then do a flat search in which we discard any
    // spans that aren't on the line.
    var ref = wrappedLineExtent(cm, lineObj, preparedMeasure, y);
    var begin = ref.begin;
    var end = ref.end;
    if (/\s/.test(lineObj.text.charAt(end - 1))) { end--; }
    var part = null, closestDist = null;
    for (var i = 0; i < order.length; i++) {
      var p = order[i];
      if (p.from >= end || p.to <= begin) { continue }
      var ltr = p.level != 1;
      var endX = measureCharPrepared(cm, preparedMeasure, ltr ? Math.min(end, p.to) - 1 : Math.max(begin, p.from)).right;
      // Weigh against spans ending before this, so that they are only
      // picked if nothing ends after
      var dist = endX < x ? x - endX + 1e9 : endX - x;
      if (!part || closestDist > dist) {
        part = p;
        closestDist = dist;
      }
    }
    if (!part) { part = order[order.length - 1]; }
    // Clip the part to the wrapped line.
    if (part.from < begin) { part = {from: begin, to: part.to, level: part.level}; }
    if (part.to > end) { part = {from: part.from, to: end, level: part.level}; }
    return part
  }

  var measureText;
  // Compute the default text height.
  function textHeight(display) {
    if (display.cachedTextHeight != null) { return display.cachedTextHeight }
    if (measureText == null) {
      measureText = elt("pre");
      // Measure a bunch of lines, for browsers that compute
      // fractional heights.
      for (var i = 0; i < 49; ++i) {
        measureText.appendChild(document.createTextNode("x"));
        measureText.appendChild(elt("br"));
      }
      measureText.appendChild(document.createTextNode("x"));
    }
    removeChildrenAndAdd(display.measure, measureText);
    var height = measureText.offsetHeight / 50;
    if (height > 3) { display.cachedTextHeight = height; }
    removeChildren(display.measure);
    return height || 1
  }

  // Compute the default character width.
  function charWidth(display) {
    if (display.cachedCharWidth != null) { return display.cachedCharWidth }
    var anchor = elt("span", "xxxxxxxxxx");
    var pre = elt("pre", [anchor]);
    removeChildrenAndAdd(display.measure, pre);
    var rect = anchor.getBoundingClientRect(), width = (rect.right - rect.left) / 10;
    if (width > 2) { display.cachedCharWidth = width; }
    return width || 10
  }

  // Do a bulk-read of the DOM positions and sizes needed to draw the
  // view, so that we don't interleave reading and writing to the DOM.
  function getDimensions(cm) {
    var d = cm.display, left = {}, width = {};
    var gutterLeft = d.gutters.clientLeft;
    for (var n = d.gutters.firstChild, i = 0; n; n = n.nextSibling, ++i) {
      var id = cm.display.gutterSpecs[i].className;
      left[id] = n.offsetLeft + n.clientLeft + gutterLeft;
      width[id] = n.clientWidth;
    }
    return {fixedPos: compensateForHScroll(d),
            gutterTotalWidth: d.gutters.offsetWidth,
            gutterLeft: left,
            gutterWidth: width,
            wrapperWidth: d.wrapper.clientWidth}
  }

  // Computes display.scroller.scrollLeft + display.gutters.offsetWidth,
  // but using getBoundingClientRect to get a sub-pixel-accurate
  // result.
  function compensateForHScroll(display) {
    return display.scroller.getBoundingClientRect().left - display.sizer.getBoundingClientRect().left
  }

  // Returns a function that estimates the height of a line, to use as
  // first approximation until the line becomes visible (and is thus
  // properly measurable).
  function estimateHeight(cm) {
    var th = textHeight(cm.display), wrapping = cm.options.lineWrapping;
    var perLine = wrapping && Math.max(5, cm.display.scroller.clientWidth / charWidth(cm.display) - 3);
    return function (line) {
      if (lineIsHidden(cm.doc, line)) { return 0 }

      var widgetsHeight = 0;
      if (line.widgets) { for (var i = 0; i < line.widgets.length; i++) {
        if (line.widgets[i].height) { widgetsHeight += line.widgets[i].height; }
      } }

      if (wrapping)
        { return widgetsHeight + (Math.ceil(line.text.length / perLine) || 1) * th }
      else
        { return widgetsHeight + th }
    }
  }

  function estimateLineHeights(cm) {
    var doc = cm.doc, est = estimateHeight(cm);
    doc.iter(function (line) {
      var estHeight = est(line);
      if (estHeight != line.height) { updateLineHeight(line, estHeight); }
    });
  }

  // Given a mouse event, find the corresponding position. If liberal
  // is false, it checks whether a gutter or scrollbar was clicked,
  // and returns null if it was. forRect is used by rectangular
  // selections, and tries to estimate a character position even for
  // coordinates beyond the right of the text.
  function posFromMouse(cm, e, liberal, forRect) {
    var display = cm.display;
    if (!liberal && e_target(e).getAttribute("cm-not-content") == "true") { return null }

    var x, y, space = display.lineSpace.getBoundingClientRect();
    // Fails unpredictably on IE[67] when mouse is dragged around quickly.
    try { x = e.clientX - space.left; y = e.clientY - space.top; }
    catch (e) { return null }
    var coords = coordsChar(cm, x, y), line;
    if (forRect && coords.xRel == 1 && (line = getLine(cm.doc, coords.line).text).length == coords.ch) {
      var colDiff = countColumn(line, line.length, cm.options.tabSize) - line.length;
      coords = Pos(coords.line, Math.max(0, Math.round((x - paddingH(cm.display).left) / charWidth(cm.display)) - colDiff));
    }
    return coords
  }

  // Find the view element corresponding to a given line. Return null
  // when the line isn't visible.
  function findViewIndex(cm, n) {
    if (n >= cm.display.viewTo) { return null }
    n -= cm.display.viewFrom;
    if (n < 0) { return null }
    var view = cm.display.view;
    for (var i = 0; i < view.length; i++) {
      n -= view[i].size;
      if (n < 0) { return i }
    }
  }

  // Updates the display.view data structure for a given change to the
  // document. From and to are in pre-change coordinates. Lendiff is
  // the amount of lines added or subtracted by the change. This is
  // used for changes that span multiple lines, or change the way
  // lines are divided into visual lines. regLineChange (below)
  // registers single-line changes.
  function regChange(cm, from, to, lendiff) {
    if (from == null) { from = cm.doc.first; }
    if (to == null) { to = cm.doc.first + cm.doc.size; }
    if (!lendiff) { lendiff = 0; }

    var display = cm.display;
    if (lendiff && to < display.viewTo &&
        (display.updateLineNumbers == null || display.updateLineNumbers > from))
      { display.updateLineNumbers = from; }

    cm.curOp.viewChanged = true;

    if (from >= display.viewTo) { // Change after
      if (sawCollapsedSpans && visualLineNo(cm.doc, from) < display.viewTo)
        { resetView(cm); }
    } else if (to <= display.viewFrom) { // Change before
      if (sawCollapsedSpans && visualLineEndNo(cm.doc, to + lendiff) > display.viewFrom) {
        resetView(cm);
      } else {
        display.viewFrom += lendiff;
        display.viewTo += lendiff;
      }
    } else if (from <= display.viewFrom && to >= display.viewTo) { // Full overlap
      resetView(cm);
    } else if (from <= display.viewFrom) { // Top overlap
      var cut = viewCuttingPoint(cm, to, to + lendiff, 1);
      if (cut) {
        display.view = display.view.slice(cut.index);
        display.viewFrom = cut.lineN;
        display.viewTo += lendiff;
      } else {
        resetView(cm);
      }
    } else if (to >= display.viewTo) { // Bottom overlap
      var cut$1 = viewCuttingPoint(cm, from, from, -1);
      if (cut$1) {
        display.view = display.view.slice(0, cut$1.index);
        display.viewTo = cut$1.lineN;
      } else {
        resetView(cm);
      }
    } else { // Gap in the middle
      var cutTop = viewCuttingPoint(cm, from, from, -1);
      var cutBot = viewCuttingPoint(cm, to, to + lendiff, 1);
      if (cutTop && cutBot) {
        display.view = display.view.slice(0, cutTop.index)
          .concat(buildViewArray(cm, cutTop.lineN, cutBot.lineN))
          .concat(display.view.slice(cutBot.index));
        display.viewTo += lendiff;
      } else {
        resetView(cm);
      }
    }

    var ext = display.externalMeasured;
    if (ext) {
      if (to < ext.lineN)
        { ext.lineN += lendiff; }
      else if (from < ext.lineN + ext.size)
        { display.externalMeasured = null; }
    }
  }

  // Register a change to a single line. Type must be one of "text",
  // "gutter", "class", "widget"
  function regLineChange(cm, line, type) {
    cm.curOp.viewChanged = true;
    var display = cm.display, ext = cm.display.externalMeasured;
    if (ext && line >= ext.lineN && line < ext.lineN + ext.size)
      { display.externalMeasured = null; }

    if (line < display.viewFrom || line >= display.viewTo) { return }
    var lineView = display.view[findViewIndex(cm, line)];
    if (lineView.node == null) { return }
    var arr = lineView.changes || (lineView.changes = []);
    if (indexOf(arr, type) == -1) { arr.push(type); }
  }

  // Clear the view.
  function resetView(cm) {
    cm.display.viewFrom = cm.display.viewTo = cm.doc.first;
    cm.display.view = [];
    cm.display.viewOffset = 0;
  }

  function viewCuttingPoint(cm, oldN, newN, dir) {
    var index = findViewIndex(cm, oldN), diff, view = cm.display.view;
    if (!sawCollapsedSpans || newN == cm.doc.first + cm.doc.size)
      { return {index: index, lineN: newN} }
    var n = cm.display.viewFrom;
    for (var i = 0; i < index; i++)
      { n += view[i].size; }
    if (n != oldN) {
      if (dir > 0) {
        if (index == view.length - 1) { return null }
        diff = (n + view[index].size) - oldN;
        index++;
      } else {
        diff = n - oldN;
      }
      oldN += diff; newN += diff;
    }
    while (visualLineNo(cm.doc, newN) != newN) {
      if (index == (dir < 0 ? 0 : view.length - 1)) { return null }
      newN += dir * view[index - (dir < 0 ? 1 : 0)].size;
      index += dir;
    }
    return {index: index, lineN: newN}
  }

  // Force the view to cover a given range, adding empty view element
  // or clipping off existing ones as needed.
  function adjustView(cm, from, to) {
    var display = cm.display, view = display.view;
    if (view.length == 0 || from >= display.viewTo || to <= display.viewFrom) {
      display.view = buildViewArray(cm, from, to);
      display.viewFrom = from;
    } else {
      if (display.viewFrom > from)
        { display.view = buildViewArray(cm, from, display.viewFrom).concat(display.view); }
      else if (display.viewFrom < from)
        { display.view = display.view.slice(findViewIndex(cm, from)); }
      display.viewFrom = from;
      if (display.viewTo < to)
        { display.view = display.view.concat(buildViewArray(cm, display.viewTo, to)); }
      else if (display.viewTo > to)
        { display.view = display.view.slice(0, findViewIndex(cm, to)); }
    }
    display.viewTo = to;
  }

  // Count the number of lines in the view whose DOM representation is
  // out of date (or nonexistent).
  function countDirtyView(cm) {
    var view = cm.display.view, dirty = 0;
    for (var i = 0; i < view.length; i++) {
      var lineView = view[i];
      if (!lineView.hidden && (!lineView.node || lineView.changes)) { ++dirty; }
    }
    return dirty
  }

  function updateSelection(cm) {
    cm.display.input.showSelection(cm.display.input.prepareSelection());
  }

  function prepareSelection(cm, primary) {
    if ( primary === void 0 ) primary = true;

    var doc = cm.doc, result = {};
    var curFragment = result.cursors = document.createDocumentFragment();
    var selFragment = result.selection = document.createDocumentFragment();

    for (var i = 0; i < doc.sel.ranges.length; i++) {
      if (!primary && i == doc.sel.primIndex) { continue }
      var range$$1 = doc.sel.ranges[i];
      if (range$$1.from().line >= cm.display.viewTo || range$$1.to().line < cm.display.viewFrom) { continue }
      var collapsed = range$$1.empty();
      if (collapsed || cm.options.showCursorWhenSelecting)
        { drawSelectionCursor(cm, range$$1.head, curFragment); }
      if (!collapsed)
        { drawSelectionRange(cm, range$$1, selFragment); }
    }
    return result
  }

  // Draws a cursor for the given range
  function drawSelectionCursor(cm, head, output) {
    var pos = cursorCoords(cm, head, "div", null, null, !cm.options.singleCursorHeightPerLine);

    var cursor = output.appendChild(elt("div", "\u00a0", "CodeMirror-cursor"));
    cursor.style.left = pos.left + "px";
    cursor.style.top = pos.top + "px";
    cursor.style.height = Math.max(0, pos.bottom - pos.top) * cm.options.cursorHeight + "px";

    if (pos.other) {
      // Secondary cursor, shown when on a 'jump' in bi-directional text
      var otherCursor = output.appendChild(elt("div", "\u00a0", "CodeMirror-cursor CodeMirror-secondarycursor"));
      otherCursor.style.display = "";
      otherCursor.style.left = pos.other.left + "px";
      otherCursor.style.top = pos.other.top + "px";
      otherCursor.style.height = (pos.other.bottom - pos.other.top) * .85 + "px";
    }
  }

  function cmpCoords(a, b) { return a.top - b.top || a.left - b.left }

  // Draws the given range as a highlighted selection
  function drawSelectionRange(cm, range$$1, output) {
    var display = cm.display, doc = cm.doc;
    var fragment = document.createDocumentFragment();
    var padding = paddingH(cm.display), leftSide = padding.left;
    var rightSide = Math.max(display.sizerWidth, displayWidth(cm) - display.sizer.offsetLeft) - padding.right;
    var docLTR = doc.direction == "ltr";

    function add(left, top, width, bottom) {
      if (top < 0) { top = 0; }
      top = Math.round(top);
      bottom = Math.round(bottom);
      fragment.appendChild(elt("div", null, "CodeMirror-selected", ("position: absolute; left: " + left + "px;\n                             top: " + top + "px; width: " + (width == null ? rightSide - left : width) + "px;\n                             height: " + (bottom - top) + "px")));
    }

    function drawForLine(line, fromArg, toArg) {
      var lineObj = getLine(doc, line);
      var lineLen = lineObj.text.length;
      var start, end;
      function coords(ch, bias) {
        return charCoords(cm, Pos(line, ch), "div", lineObj, bias)
      }

      function wrapX(pos, dir, side) {
        var extent = wrappedLineExtentChar(cm, lineObj, null, pos);
        var prop = (dir == "ltr") == (side == "after") ? "left" : "right";
        var ch = side == "after" ? extent.begin : extent.end - (/\s/.test(lineObj.text.charAt(extent.end - 1)) ? 2 : 1);
        return coords(ch, prop)[prop]
      }

      var order = getOrder(lineObj, doc.direction);
      iterateBidiSections(order, fromArg || 0, toArg == null ? lineLen : toArg, function (from, to, dir, i) {
        var ltr = dir == "ltr";
        var fromPos = coords(from, ltr ? "left" : "right");
        var toPos = coords(to - 1, ltr ? "right" : "left");

        var openStart = fromArg == null && from == 0, openEnd = toArg == null && to == lineLen;
        var first = i == 0, last = !order || i == order.length - 1;
        if (toPos.top - fromPos.top <= 3) { // Single line
          var openLeft = (docLTR ? openStart : openEnd) && first;
          var openRight = (docLTR ? openEnd : openStart) && last;
          var left = openLeft ? leftSide : (ltr ? fromPos : toPos).left;
          var right = openRight ? rightSide : (ltr ? toPos : fromPos).right;
          add(left, fromPos.top, right - left, fromPos.bottom);
        } else { // Multiple lines
          var topLeft, topRight, botLeft, botRight;
          if (ltr) {
            topLeft = docLTR && openStart && first ? leftSide : fromPos.left;
            topRight = docLTR ? rightSide : wrapX(from, dir, "before");
            botLeft = docLTR ? leftSide : wrapX(to, dir, "after");
            botRight = docLTR && openEnd && last ? rightSide : toPos.right;
          } else {
            topLeft = !docLTR ? leftSide : wrapX(from, dir, "before");
            topRight = !docLTR && openStart && first ? rightSide : fromPos.right;
            botLeft = !docLTR && openEnd && last ? leftSide : toPos.left;
            botRight = !docLTR ? rightSide : wrapX(to, dir, "after");
          }
          add(topLeft, fromPos.top, topRight - topLeft, fromPos.bottom);
          if (fromPos.bottom < toPos.top) { add(leftSide, fromPos.bottom, null, toPos.top); }
          add(botLeft, toPos.top, botRight - botLeft, toPos.bottom);
        }

        if (!start || cmpCoords(fromPos, start) < 0) { start = fromPos; }
        if (cmpCoords(toPos, start) < 0) { start = toPos; }
        if (!end || cmpCoords(fromPos, end) < 0) { end = fromPos; }
        if (cmpCoords(toPos, end) < 0) { end = toPos; }
      });
      return {start: start, end: end}
    }

    var sFrom = range$$1.from(), sTo = range$$1.to();
    if (sFrom.line == sTo.line) {
      drawForLine(sFrom.line, sFrom.ch, sTo.ch);
    } else {
      var fromLine = getLine(doc, sFrom.line), toLine = getLine(doc, sTo.line);
      var singleVLine = visualLine(fromLine) == visualLine(toLine);
      var leftEnd = drawForLine(sFrom.line, sFrom.ch, singleVLine ? fromLine.text.length + 1 : null).end;
      var rightStart = drawForLine(sTo.line, singleVLine ? 0 : null, sTo.ch).start;
      if (singleVLine) {
        if (leftEnd.top < rightStart.top - 2) {
          add(leftEnd.right, leftEnd.top, null, leftEnd.bottom);
          add(leftSide, rightStart.top, rightStart.left, rightStart.bottom);
        } else {
          add(leftEnd.right, leftEnd.top, rightStart.left - leftEnd.right, leftEnd.bottom);
        }
      }
      if (leftEnd.bottom < rightStart.top)
        { add(leftSide, leftEnd.bottom, null, rightStart.top); }
    }

    output.appendChild(fragment);
  }

  // Cursor-blinking
  function restartBlink(cm) {
    if (!cm.state.focused) { return }
    var display = cm.display;
    clearInterval(display.blinker);
    var on = true;
    display.cursorDiv.style.visibility = "";
    if (cm.options.cursorBlinkRate > 0)
      { display.blinker = setInterval(function () { return display.cursorDiv.style.visibility = (on = !on) ? "" : "hidden"; },
        cm.options.cursorBlinkRate); }
    else if (cm.options.cursorBlinkRate < 0)
      { display.cursorDiv.style.visibility = "hidden"; }
  }

  function ensureFocus(cm) {
    if (!cm.state.focused) { cm.display.input.focus(); onFocus(cm); }
  }

  function delayBlurEvent(cm) {
    cm.state.delayingBlurEvent = true;
    setTimeout(function () { if (cm.state.delayingBlurEvent) {
      cm.state.delayingBlurEvent = false;
      onBlur(cm);
    } }, 100);
  }

  function onFocus(cm, e) {
    if (cm.state.delayingBlurEvent) { cm.state.delayingBlurEvent = false; }

    if (cm.options.readOnly == "nocursor") { return }
    if (!cm.state.focused) {
      signal(cm, "focus", cm, e);
      cm.state.focused = true;
      addClass(cm.display.wrapper, "CodeMirror-focused");
      // This test prevents this from firing when a context
      // menu is closed (since the input reset would kill the
      // select-all detection hack)
      if (!cm.curOp && cm.display.selForContextMenu != cm.doc.sel) {
        cm.display.input.reset();
        if (webkit) { setTimeout(function () { return cm.display.input.reset(true); }, 20); } // Issue #1730
      }
      cm.display.input.receivedFocus();
    }
    restartBlink(cm);
  }
  function onBlur(cm, e) {
    if (cm.state.delayingBlurEvent) { return }

    if (cm.state.focused) {
      signal(cm, "blur", cm, e);
      cm.state.focused = false;
      rmClass(cm.display.wrapper, "CodeMirror-focused");
    }
    clearInterval(cm.display.blinker);
    setTimeout(function () { if (!cm.state.focused) { cm.display.shift = false; } }, 150);
  }

  // Read the actual heights of the rendered lines, and update their
  // stored heights to match.
  function updateHeightsInViewport(cm) {
    var display = cm.display;
    var prevBottom = display.lineDiv.offsetTop;
    for (var i = 0; i < display.view.length; i++) {
      var cur = display.view[i], wrapping = cm.options.lineWrapping;
      var height = (void 0), width = 0;
      if (cur.hidden) { continue }
      if (ie && ie_version < 8) {
        var bot = cur.node.offsetTop + cur.node.offsetHeight;
        height = bot - prevBottom;
        prevBottom = bot;
      } else {
        var box = cur.node.getBoundingClientRect();
        height = box.bottom - box.top;
        // Check that lines don't extend past the right of the current
        // editor width
        if (!wrapping && cur.text.firstChild)
          { width = cur.text.firstChild.getBoundingClientRect().right - box.left - 1; }
      }
      var diff = cur.line.height - height;
      if (diff > .005 || diff < -.005) {
        updateLineHeight(cur.line, height);
        updateWidgetHeight(cur.line);
        if (cur.rest) { for (var j = 0; j < cur.rest.length; j++)
          { updateWidgetHeight(cur.rest[j]); } }
      }
      if (width > cm.display.sizerWidth) {
        var chWidth = Math.ceil(width / charWidth(cm.display));
        if (chWidth > cm.display.maxLineLength) {
          cm.display.maxLineLength = chWidth;
          cm.display.maxLine = cur.line;
          cm.display.maxLineChanged = true;
        }
      }
    }
  }

  // Read and store the height of line widgets associated with the
  // given line.
  function updateWidgetHeight(line) {
    if (line.widgets) { for (var i = 0; i < line.widgets.length; ++i) {
      var w = line.widgets[i], parent = w.node.parentNode;
      if (parent) { w.height = parent.offsetHeight; }
    } }
  }

  // Compute the lines that are visible in a given viewport (defaults
  // the the current scroll position). viewport may contain top,
  // height, and ensure (see op.scrollToPos) properties.
  function visibleLines(display, doc, viewport) {
    var top = viewport && viewport.top != null ? Math.max(0, viewport.top) : display.scroller.scrollTop;
    top = Math.floor(top - paddingTop(display));
    var bottom = viewport && viewport.bottom != null ? viewport.bottom : top + display.wrapper.clientHeight;

    var from = lineAtHeight(doc, top), to = lineAtHeight(doc, bottom);
    // Ensure is a {from: {line, ch}, to: {line, ch}} object, and
    // forces those lines into the viewport (if possible).
    if (viewport && viewport.ensure) {
      var ensureFrom = viewport.ensure.from.line, ensureTo = viewport.ensure.to.line;
      if (ensureFrom < from) {
        from = ensureFrom;
        to = lineAtHeight(doc, heightAtLine(getLine(doc, ensureFrom)) + display.wrapper.clientHeight);
      } else if (Math.min(ensureTo, doc.lastLine()) >= to) {
        from = lineAtHeight(doc, heightAtLine(getLine(doc, ensureTo)) - display.wrapper.clientHeight);
        to = ensureTo;
      }
    }
    return {from: from, to: Math.max(to, from + 1)}
  }

  // SCROLLING THINGS INTO VIEW

  // If an editor sits on the top or bottom of the window, partially
  // scrolled out of view, this ensures that the cursor is visible.
  function maybeScrollWindow(cm, rect) {
    if (signalDOMEvent(cm, "scrollCursorIntoView")) { return }

    var display = cm.display, box = display.sizer.getBoundingClientRect(), doScroll = null;
    if (rect.top + box.top < 0) { doScroll = true; }
    else if (rect.bottom + box.top > (window.innerHeight || document.documentElement.clientHeight)) { doScroll = false; }
    if (doScroll != null && !phantom) {
      var scrollNode = elt("div", "\u200b", null, ("position: absolute;\n                         top: " + (rect.top - display.viewOffset - paddingTop(cm.display)) + "px;\n                         height: " + (rect.bottom - rect.top + scrollGap(cm) + display.barHeight) + "px;\n                         left: " + (rect.left) + "px; width: " + (Math.max(2, rect.right - rect.left)) + "px;"));
      cm.display.lineSpace.appendChild(scrollNode);
      scrollNode.scrollIntoView(doScroll);
      cm.display.lineSpace.removeChild(scrollNode);
    }
  }

  // Scroll a given position into view (immediately), verifying that
  // it actually became visible (as line heights are accurately
  // measured, the position of something may 'drift' during drawing).
  function scrollPosIntoView(cm, pos, end, margin) {
    if (margin == null) { margin = 0; }
    var rect;
    if (!cm.options.lineWrapping && pos == end) {
      // Set pos and end to the cursor positions around the character pos sticks to
      // If pos.sticky == "before", that is around pos.ch - 1, otherwise around pos.ch
      // If pos == Pos(_, 0, "before"), pos and end are unchanged
      pos = pos.ch ? Pos(pos.line, pos.sticky == "before" ? pos.ch - 1 : pos.ch, "after") : pos;
      end = pos.sticky == "before" ? Pos(pos.line, pos.ch + 1, "before") : pos;
    }
    for (var limit = 0; limit < 5; limit++) {
      var changed = false;
      var coords = cursorCoords(cm, pos);
      var endCoords = !end || end == pos ? coords : cursorCoords(cm, end);
      rect = {left: Math.min(coords.left, endCoords.left),
              top: Math.min(coords.top, endCoords.top) - margin,
              right: Math.max(coords.left, endCoords.left),
              bottom: Math.max(coords.bottom, endCoords.bottom) + margin};
      var scrollPos = calculateScrollPos(cm, rect);
      var startTop = cm.doc.scrollTop, startLeft = cm.doc.scrollLeft;
      if (scrollPos.scrollTop != null) {
        updateScrollTop(cm, scrollPos.scrollTop);
        if (Math.abs(cm.doc.scrollTop - startTop) > 1) { changed = true; }
      }
      if (scrollPos.scrollLeft != null) {
        setScrollLeft(cm, scrollPos.scrollLeft);
        if (Math.abs(cm.doc.scrollLeft - startLeft) > 1) { changed = true; }
      }
      if (!changed) { break }
    }
    return rect
  }

  // Scroll a given set of coordinates into view (immediately).
  function scrollIntoView(cm, rect) {
    var scrollPos = calculateScrollPos(cm, rect);
    if (scrollPos.scrollTop != null) { updateScrollTop(cm, scrollPos.scrollTop); }
    if (scrollPos.scrollLeft != null) { setScrollLeft(cm, scrollPos.scrollLeft); }
  }

  // Calculate a new scroll position needed to scroll the given
  // rectangle into view. Returns an object with scrollTop and
  // scrollLeft properties. When these are undefined, the
  // vertical/horizontal position does not need to be adjusted.
  function calculateScrollPos(cm, rect) {
    var display = cm.display, snapMargin = textHeight(cm.display);
    if (rect.top < 0) { rect.top = 0; }
    var screentop = cm.curOp && cm.curOp.scrollTop != null ? cm.curOp.scrollTop : display.scroller.scrollTop;
    var screen = displayHeight(cm), result = {};
    if (rect.bottom - rect.top > screen) { rect.bottom = rect.top + screen; }
    var docBottom = cm.doc.height + paddingVert(display);
    var atTop = rect.top < snapMargin, atBottom = rect.bottom > docBottom - snapMargin;
    if (rect.top < screentop) {
      result.scrollTop = atTop ? 0 : rect.top;
    } else if (rect.bottom > screentop + screen) {
      var newTop = Math.min(rect.top, (atBottom ? docBottom : rect.bottom) - screen);
      if (newTop != screentop) { result.scrollTop = newTop; }
    }

    var screenleft = cm.curOp && cm.curOp.scrollLeft != null ? cm.curOp.scrollLeft : display.scroller.scrollLeft;
    var screenw = displayWidth(cm) - (cm.options.fixedGutter ? display.gutters.offsetWidth : 0);
    var tooWide = rect.right - rect.left > screenw;
    if (tooWide) { rect.right = rect.left + screenw; }
    if (rect.left < 10)
      { result.scrollLeft = 0; }
    else if (rect.left < screenleft)
      { result.scrollLeft = Math.max(0, rect.left - (tooWide ? 0 : 10)); }
    else if (rect.right > screenw + screenleft - 3)
      { result.scrollLeft = rect.right + (tooWide ? 0 : 10) - screenw; }
    return result
  }

  // Store a relative adjustment to the scroll position in the current
  // operation (to be applied when the operation finishes).
  function addToScrollTop(cm, top) {
    if (top == null) { return }
    resolveScrollToPos(cm);
    cm.curOp.scrollTop = (cm.curOp.scrollTop == null ? cm.doc.scrollTop : cm.curOp.scrollTop) + top;
  }

  // Make sure that at the end of the operation the current cursor is
  // shown.
  function ensureCursorVisible(cm) {
    resolveScrollToPos(cm);
    var cur = cm.getCursor();
    cm.curOp.scrollToPos = {from: cur, to: cur, margin: cm.options.cursorScrollMargin};
  }

  function scrollToCoords(cm, x, y) {
    if (x != null || y != null) { resolveScrollToPos(cm); }
    if (x != null) { cm.curOp.scrollLeft = x; }
    if (y != null) { cm.curOp.scrollTop = y; }
  }

  function scrollToRange(cm, range$$1) {
    resolveScrollToPos(cm);
    cm.curOp.scrollToPos = range$$1;
  }

  // When an operation has its scrollToPos property set, and another
  // scroll action is applied before the end of the operation, this
  // 'simulates' scrolling that position into view in a cheap way, so
  // that the effect of intermediate scroll commands is not ignored.
  function resolveScrollToPos(cm) {
    var range$$1 = cm.curOp.scrollToPos;
    if (range$$1) {
      cm.curOp.scrollToPos = null;
      var from = estimateCoords(cm, range$$1.from), to = estimateCoords(cm, range$$1.to);
      scrollToCoordsRange(cm, from, to, range$$1.margin);
    }
  }

  function scrollToCoordsRange(cm, from, to, margin) {
    var sPos = calculateScrollPos(cm, {
      left: Math.min(from.left, to.left),
      top: Math.min(from.top, to.top) - margin,
      right: Math.max(from.right, to.right),
      bottom: Math.max(from.bottom, to.bottom) + margin
    });
    scrollToCoords(cm, sPos.scrollLeft, sPos.scrollTop);
  }

  // Sync the scrollable area and scrollbars, ensure the viewport
  // covers the visible area.
  function updateScrollTop(cm, val) {
    if (Math.abs(cm.doc.scrollTop - val) < 2) { return }
    if (!gecko) { updateDisplaySimple(cm, {top: val}); }
    setScrollTop(cm, val, true);
    if (gecko) { updateDisplaySimple(cm); }
    startWorker(cm, 100);
  }

  function setScrollTop(cm, val, forceScroll) {
    val = Math.min(cm.display.scroller.scrollHeight - cm.display.scroller.clientHeight, val);
    if (cm.display.scroller.scrollTop == val && !forceScroll) { return }
    cm.doc.scrollTop = val;
    cm.display.scrollbars.setScrollTop(val);
    if (cm.display.scroller.scrollTop != val) { cm.display.scroller.scrollTop = val; }
  }

  // Sync scroller and scrollbar, ensure the gutter elements are
  // aligned.
  function setScrollLeft(cm, val, isScroller, forceScroll) {
    val = Math.min(val, cm.display.scroller.scrollWidth - cm.display.scroller.clientWidth);
    if ((isScroller ? val == cm.doc.scrollLeft : Math.abs(cm.doc.scrollLeft - val) < 2) && !forceScroll) { return }
    cm.doc.scrollLeft = val;
    alignHorizontally(cm);
    if (cm.display.scroller.scrollLeft != val) { cm.display.scroller.scrollLeft = val; }
    cm.display.scrollbars.setScrollLeft(val);
  }

  // SCROLLBARS

  // Prepare DOM reads needed to update the scrollbars. Done in one
  // shot to minimize update/measure roundtrips.
  function measureForScrollbars(cm) {
    var d = cm.display, gutterW = d.gutters.offsetWidth;
    var docH = Math.round(cm.doc.height + paddingVert(cm.display));
    return {
      clientHeight: d.scroller.clientHeight,
      viewHeight: d.wrapper.clientHeight,
      scrollWidth: d.scroller.scrollWidth, clientWidth: d.scroller.clientWidth,
      viewWidth: d.wrapper.clientWidth,
      barLeft: cm.options.fixedGutter ? gutterW : 0,
      docHeight: docH,
      scrollHeight: docH + scrollGap(cm) + d.barHeight,
      nativeBarWidth: d.nativeBarWidth,
      gutterWidth: gutterW
    }
  }

  var NativeScrollbars = function(place, scroll, cm) {
    this.cm = cm;
    var vert = this.vert = elt("div", [elt("div", null, null, "min-width: 1px")], "CodeMirror-vscrollbar");
    var horiz = this.horiz = elt("div", [elt("div", null, null, "height: 100%; min-height: 1px")], "CodeMirror-hscrollbar");
    vert.tabIndex = horiz.tabIndex = -1;
    place(vert); place(horiz);

    on(vert, "scroll", function () {
      if (vert.clientHeight) { scroll(vert.scrollTop, "vertical"); }
    });
    on(horiz, "scroll", function () {
      if (horiz.clientWidth) { scroll(horiz.scrollLeft, "horizontal"); }
    });

    this.checkedZeroWidth = false;
    // Need to set a minimum width to see the scrollbar on IE7 (but must not set it on IE8).
    if (ie && ie_version < 8) { this.horiz.style.minHeight = this.vert.style.minWidth = "18px"; }
  };

  NativeScrollbars.prototype.update = function (measure) {
    var needsH = measure.scrollWidth > measure.clientWidth + 1;
    var needsV = measure.scrollHeight > measure.clientHeight + 1;
    var sWidth = measure.nativeBarWidth;

    if (needsV) {
      this.vert.style.display = "block";
      this.vert.style.bottom = needsH ? sWidth + "px" : "0";
      var totalHeight = measure.viewHeight - (needsH ? sWidth : 0);
      // A bug in IE8 can cause this value to be negative, so guard it.
      this.vert.firstChild.style.height =
        Math.max(0, measure.scrollHeight - measure.clientHeight + totalHeight) + "px";
    } else {
      this.vert.style.display = "";
      this.vert.firstChild.style.height = "0";
    }

    if (needsH) {
      this.horiz.style.display = "block";
      this.horiz.style.right = needsV ? sWidth + "px" : "0";
      this.horiz.style.left = measure.barLeft + "px";
      var totalWidth = measure.viewWidth - measure.barLeft - (needsV ? sWidth : 0);
      this.horiz.firstChild.style.width =
        Math.max(0, measure.scrollWidth - measure.clientWidth + totalWidth) + "px";
    } else {
      this.horiz.style.display = "";
      this.horiz.firstChild.style.width = "0";
    }

    if (!this.checkedZeroWidth && measure.clientHeight > 0) {
      if (sWidth == 0) { this.zeroWidthHack(); }
      this.checkedZeroWidth = true;
    }

    return {right: needsV ? sWidth : 0, bottom: needsH ? sWidth : 0}
  };

  NativeScrollbars.prototype.setScrollLeft = function (pos) {
    if (this.horiz.scrollLeft != pos) { this.horiz.scrollLeft = pos; }
    if (this.disableHoriz) { this.enableZeroWidthBar(this.horiz, this.disableHoriz, "horiz"); }
  };

  NativeScrollbars.prototype.setScrollTop = function (pos) {
    if (this.vert.scrollTop != pos) { this.vert.scrollTop = pos; }
    if (this.disableVert) { this.enableZeroWidthBar(this.vert, this.disableVert, "vert"); }
  };

  NativeScrollbars.prototype.zeroWidthHack = function () {
    var w = mac && !mac_geMountainLion ? "12px" : "18px";
    this.horiz.style.height = this.vert.style.width = w;
    this.horiz.style.pointerEvents = this.vert.style.pointerEvents = "none";
    this.disableHoriz = new Delayed;
    this.disableVert = new Delayed;
  };

  NativeScrollbars.prototype.enableZeroWidthBar = function (bar, delay, type) {
    bar.style.pointerEvents = "auto";
    function maybeDisable() {
      // To find out whether the scrollbar is still visible, we
      // check whether the element under the pixel in the bottom
      // right corner of the scrollbar box is the scrollbar box
      // itself (when the bar is still visible) or its filler child
      // (when the bar is hidden). If it is still visible, we keep
      // it enabled, if it's hidden, we disable pointer events.
      var box = bar.getBoundingClientRect();
      var elt$$1 = type == "vert" ? document.elementFromPoint(box.right - 1, (box.top + box.bottom) / 2)
          : document.elementFromPoint((box.right + box.left) / 2, box.bottom - 1);
      if (elt$$1 != bar) { bar.style.pointerEvents = "none"; }
      else { delay.set(1000, maybeDisable); }
    }
    delay.set(1000, maybeDisable);
  };

  NativeScrollbars.prototype.clear = function () {
    var parent = this.horiz.parentNode;
    parent.removeChild(this.horiz);
    parent.removeChild(this.vert);
  };

  var NullScrollbars = function () {};

  NullScrollbars.prototype.update = function () { return {bottom: 0, right: 0} };
  NullScrollbars.prototype.setScrollLeft = function () {};
  NullScrollbars.prototype.setScrollTop = function () {};
  NullScrollbars.prototype.clear = function () {};

  function updateScrollbars(cm, measure) {
    if (!measure) { measure = measureForScrollbars(cm); }
    var startWidth = cm.display.barWidth, startHeight = cm.display.barHeight;
    updateScrollbarsInner(cm, measure);
    for (var i = 0; i < 4 && startWidth != cm.display.barWidth || startHeight != cm.display.barHeight; i++) {
      if (startWidth != cm.display.barWidth && cm.options.lineWrapping)
        { updateHeightsInViewport(cm); }
      updateScrollbarsInner(cm, measureForScrollbars(cm));
      startWidth = cm.display.barWidth; startHeight = cm.display.barHeight;
    }
  }

  // Re-synchronize the fake scrollbars with the actual size of the
  // content.
  function updateScrollbarsInner(cm, measure) {
    var d = cm.display;
    var sizes = d.scrollbars.update(measure);

    d.sizer.style.paddingRight = (d.barWidth = sizes.right) + "px";
    d.sizer.style.paddingBottom = (d.barHeight = sizes.bottom) + "px";
    d.heightForcer.style.borderBottom = sizes.bottom + "px solid transparent";

    if (sizes.right && sizes.bottom) {
      d.scrollbarFiller.style.display = "block";
      d.scrollbarFiller.style.height = sizes.bottom + "px";
      d.scrollbarFiller.style.width = sizes.right + "px";
    } else { d.scrollbarFiller.style.display = ""; }
    if (sizes.bottom && cm.options.coverGutterNextToScrollbar && cm.options.fixedGutter) {
      d.gutterFiller.style.display = "block";
      d.gutterFiller.style.height = sizes.bottom + "px";
      d.gutterFiller.style.width = measure.gutterWidth + "px";
    } else { d.gutterFiller.style.display = ""; }
  }

  var scrollbarModel = {"native": NativeScrollbars, "null": NullScrollbars};

  function initScrollbars(cm) {
    if (cm.display.scrollbars) {
      cm.display.scrollbars.clear();
      if (cm.display.scrollbars.addClass)
        { rmClass(cm.display.wrapper, cm.display.scrollbars.addClass); }
    }

    cm.display.scrollbars = new scrollbarModel[cm.options.scrollbarStyle](function (node) {
      cm.display.wrapper.insertBefore(node, cm.display.scrollbarFiller);
      // Prevent clicks in the scrollbars from killing focus
      on(node, "mousedown", function () {
        if (cm.state.focused) { setTimeout(function () { return cm.display.input.focus(); }, 0); }
      });
      node.setAttribute("cm-not-content", "true");
    }, function (pos, axis) {
      if (axis == "horizontal") { setScrollLeft(cm, pos); }
      else { updateScrollTop(cm, pos); }
    }, cm);
    if (cm.display.scrollbars.addClass)
      { addClass(cm.display.wrapper, cm.display.scrollbars.addClass); }
  }

  // Operations are used to wrap a series of changes to the editor
  // state in such a way that each change won't have to update the
  // cursor and display (which would be awkward, slow, and
  // error-prone). Instead, display updates are batched and then all
  // combined and executed at once.

  var nextOpId = 0;
  // Start a new operation.
  function startOperation(cm) {
    cm.curOp = {
      cm: cm,
      viewChanged: false,      // Flag that indicates that lines might need to be redrawn
      startHeight: cm.doc.height, // Used to detect need to update scrollbar
      forceUpdate: false,      // Used to force a redraw
      updateInput: 0,       // Whether to reset the input textarea
      typing: false,           // Whether this reset should be careful to leave existing text (for compositing)
      changeObjs: null,        // Accumulated changes, for firing change events
      cursorActivityHandlers: null, // Set of handlers to fire cursorActivity on
      cursorActivityCalled: 0, // Tracks which cursorActivity handlers have been called already
      selectionChanged: false, // Whether the selection needs to be redrawn
      updateMaxLine: false,    // Set when the widest line needs to be determined anew
      scrollLeft: null, scrollTop: null, // Intermediate scroll position, not pushed to DOM yet
      scrollToPos: null,       // Used to scroll to a specific position
      focus: false,
      id: ++nextOpId           // Unique ID
    };
    pushOperation(cm.curOp);
  }

  // Finish an operation, updating the display and signalling delayed events
  function endOperation(cm) {
    var op = cm.curOp;
    if (op) { finishOperation(op, function (group) {
      for (var i = 0; i < group.ops.length; i++)
        { group.ops[i].cm.curOp = null; }
      endOperations(group);
    }); }
  }

  // The DOM updates done when an operation finishes are batched so
  // that the minimum number of relayouts are required.
  function endOperations(group) {
    var ops = group.ops;
    for (var i = 0; i < ops.length; i++) // Read DOM
      { endOperation_R1(ops[i]); }
    for (var i$1 = 0; i$1 < ops.length; i$1++) // Write DOM (maybe)
      { endOperation_W1(ops[i$1]); }
    for (var i$2 = 0; i$2 < ops.length; i$2++) // Read DOM
      { endOperation_R2(ops[i$2]); }
    for (var i$3 = 0; i$3 < ops.length; i$3++) // Write DOM (maybe)
      { endOperation_W2(ops[i$3]); }
    for (var i$4 = 0; i$4 < ops.length; i$4++) // Read DOM
      { endOperation_finish(ops[i$4]); }
  }

  function endOperation_R1(op) {
    var cm = op.cm, display = cm.display;
    maybeClipScrollbars(cm);
    if (op.updateMaxLine) { findMaxLine(cm); }

    op.mustUpdate = op.viewChanged || op.forceUpdate || op.scrollTop != null ||
      op.scrollToPos && (op.scrollToPos.from.line < display.viewFrom ||
                         op.scrollToPos.to.line >= display.viewTo) ||
      display.maxLineChanged && cm.options.lineWrapping;
    op.update = op.mustUpdate &&
      new DisplayUpdate(cm, op.mustUpdate && {top: op.scrollTop, ensure: op.scrollToPos}, op.forceUpdate);
  }

  function endOperation_W1(op) {
    op.updatedDisplay = op.mustUpdate && updateDisplayIfNeeded(op.cm, op.update);
  }

  function endOperation_R2(op) {
    var cm = op.cm, display = cm.display;
    if (op.updatedDisplay) { updateHeightsInViewport(cm); }

    op.barMeasure = measureForScrollbars(cm);

    // If the max line changed since it was last measured, measure it,
    // and ensure the document's width matches it.
    // updateDisplay_W2 will use these properties to do the actual resizing
    if (display.maxLineChanged && !cm.options.lineWrapping) {
      op.adjustWidthTo = measureChar(cm, display.maxLine, display.maxLine.text.length).left + 3;
      cm.display.sizerWidth = op.adjustWidthTo;
      op.barMeasure.scrollWidth =
        Math.max(display.scroller.clientWidth, display.sizer.offsetLeft + op.adjustWidthTo + scrollGap(cm) + cm.display.barWidth);
      op.maxScrollLeft = Math.max(0, display.sizer.offsetLeft + op.adjustWidthTo - displayWidth(cm));
    }

    if (op.updatedDisplay || op.selectionChanged)
      { op.preparedSelection = display.input.prepareSelection(); }
  }

  function endOperation_W2(op) {
    var cm = op.cm;

    if (op.adjustWidthTo != null) {
      cm.display.sizer.style.minWidth = op.adjustWidthTo + "px";
      if (op.maxScrollLeft < cm.doc.scrollLeft)
        { setScrollLeft(cm, Math.min(cm.display.scroller.scrollLeft, op.maxScrollLeft), true); }
      cm.display.maxLineChanged = false;
    }

    var takeFocus = op.focus && op.focus == activeElt();
    if (op.preparedSelection)
      { cm.display.input.showSelection(op.preparedSelection, takeFocus); }
    if (op.updatedDisplay || op.startHeight != cm.doc.height)
      { updateScrollbars(cm, op.barMeasure); }
    if (op.updatedDisplay)
      { setDocumentHeight(cm, op.barMeasure); }

    if (op.selectionChanged) { restartBlink(cm); }

    if (cm.state.focused && op.updateInput)
      { cm.display.input.reset(op.typing); }
    if (takeFocus) { ensureFocus(op.cm); }
  }

  function endOperation_finish(op) {
    var cm = op.cm, display = cm.display, doc = cm.doc;

    if (op.updatedDisplay) { postUpdateDisplay(cm, op.update); }

    // Abort mouse wheel delta measurement, when scrolling explicitly
    if (display.wheelStartX != null && (op.scrollTop != null || op.scrollLeft != null || op.scrollToPos))
      { display.wheelStartX = display.wheelStartY = null; }

    // Propagate the scroll position to the actual DOM scroller
    if (op.scrollTop != null) { setScrollTop(cm, op.scrollTop, op.forceScroll); }

    if (op.scrollLeft != null) { setScrollLeft(cm, op.scrollLeft, true, true); }
    // If we need to scroll a specific position into view, do so.
    if (op.scrollToPos) {
      var rect = scrollPosIntoView(cm, clipPos(doc, op.scrollToPos.from),
                                   clipPos(doc, op.scrollToPos.to), op.scrollToPos.margin);
      maybeScrollWindow(cm, rect);
    }

    // Fire events for markers that are hidden/unidden by editing or
    // undoing
    var hidden = op.maybeHiddenMarkers, unhidden = op.maybeUnhiddenMarkers;
    if (hidden) { for (var i = 0; i < hidden.length; ++i)
      { if (!hidden[i].lines.length) { signal(hidden[i], "hide"); } } }
    if (unhidden) { for (var i$1 = 0; i$1 < unhidden.length; ++i$1)
      { if (unhidden[i$1].lines.length) { signal(unhidden[i$1], "unhide"); } } }

    if (display.wrapper.offsetHeight)
      { doc.scrollTop = cm.display.scroller.scrollTop; }

    // Fire change events, and delayed event handlers
    if (op.changeObjs)
      { signal(cm, "changes", cm, op.changeObjs); }
    if (op.update)
      { op.update.finish(); }
  }

  // Run the given function in an operation
  function runInOp(cm, f) {
    if (cm.curOp) { return f() }
    startOperation(cm);
    try { return f() }
    finally { endOperation(cm); }
  }
  // Wraps a function in an operation. Returns the wrapped function.
  function operation(cm, f) {
    return function() {
      if (cm.curOp) { return f.apply(cm, arguments) }
      startOperation(cm);
      try { return f.apply(cm, arguments) }
      finally { endOperation(cm); }
    }
  }
  // Used to add methods to editor and doc instances, wrapping them in
  // operations.
  function methodOp(f) {
    return function() {
      if (this.curOp) { return f.apply(this, arguments) }
      startOperation(this);
      try { return f.apply(this, arguments) }
      finally { endOperation(this); }
    }
  }
  function docMethodOp(f) {
    return function() {
      var cm = this.cm;
      if (!cm || cm.curOp) { return f.apply(this, arguments) }
      startOperation(cm);
      try { return f.apply(this, arguments) }
      finally { endOperation(cm); }
    }
  }

  // HIGHLIGHT WORKER

  function startWorker(cm, time) {
    if (cm.doc.highlightFrontier < cm.display.viewTo)
      { cm.state.highlight.set(time, bind(highlightWorker, cm)); }
  }

  function highlightWorker(cm) {
    var doc = cm.doc;
    if (doc.highlightFrontier >= cm.display.viewTo) { return }
    var end = +new Date + cm.options.workTime;
    var context = getContextBefore(cm, doc.highlightFrontier);
    var changedLines = [];

    doc.iter(context.line, Math.min(doc.first + doc.size, cm.display.viewTo + 500), function (line) {
      if (context.line >= cm.display.viewFrom) { // Visible
        var oldStyles = line.styles;
        var resetState = line.text.length > cm.options.maxHighlightLength ? copyState(doc.mode, context.state) : null;
        var highlighted = highlightLine(cm, line, context, true);
        if (resetState) { context.state = resetState; }
        line.styles = highlighted.styles;
        var oldCls = line.styleClasses, newCls = highlighted.classes;
        if (newCls) { line.styleClasses = newCls; }
        else if (oldCls) { line.styleClasses = null; }
        var ischange = !oldStyles || oldStyles.length != line.styles.length ||
          oldCls != newCls && (!oldCls || !newCls || oldCls.bgClass != newCls.bgClass || oldCls.textClass != newCls.textClass);
        for (var i = 0; !ischange && i < oldStyles.length; ++i) { ischange = oldStyles[i] != line.styles[i]; }
        if (ischange) { changedLines.push(context.line); }
        line.stateAfter = context.save();
        context.nextLine();
      } else {
        if (line.text.length <= cm.options.maxHighlightLength)
          { processLine(cm, line.text, context); }
        line.stateAfter = context.line % 5 == 0 ? context.save() : null;
        context.nextLine();
      }
      if (+new Date > end) {
        startWorker(cm, cm.options.workDelay);
        return true
      }
    });
    doc.highlightFrontier = context.line;
    doc.modeFrontier = Math.max(doc.modeFrontier, context.line);
    if (changedLines.length) { runInOp(cm, function () {
      for (var i = 0; i < changedLines.length; i++)
        { regLineChange(cm, changedLines[i], "text"); }
    }); }
  }

  // DISPLAY DRAWING

  var DisplayUpdate = function(cm, viewport, force) {
    var display = cm.display;

    this.viewport = viewport;
    // Store some values that we'll need later (but don't want to force a relayout for)
    this.visible = visibleLines(display, cm.doc, viewport);
    this.editorIsHidden = !display.wrapper.offsetWidth;
    this.wrapperHeight = display.wrapper.clientHeight;
    this.wrapperWidth = display.wrapper.clientWidth;
    this.oldDisplayWidth = displayWidth(cm);
    this.force = force;
    this.dims = getDimensions(cm);
    this.events = [];
  };

  DisplayUpdate.prototype.signal = function (emitter, type) {
    if (hasHandler(emitter, type))
      { this.events.push(arguments); }
  };
  DisplayUpdate.prototype.finish = function () {
      var this$1 = this;

    for (var i = 0; i < this.events.length; i++)
      { signal.apply(null, this$1.events[i]); }
  };

  function maybeClipScrollbars(cm) {
    var display = cm.display;
    if (!display.scrollbarsClipped && display.scroller.offsetWidth) {
      display.nativeBarWidth = display.scroller.offsetWidth - display.scroller.clientWidth;
      display.heightForcer.style.height = scrollGap(cm) + "px";
      display.sizer.style.marginBottom = -display.nativeBarWidth + "px";
      display.sizer.style.borderRightWidth = scrollGap(cm) + "px";
      display.scrollbarsClipped = true;
    }
  }

  function selectionSnapshot(cm) {
    if (cm.hasFocus()) { return null }
    var active = activeElt();
    if (!active || !contains(cm.display.lineDiv, active)) { return null }
    var result = {activeElt: active};
    if (window.getSelection) {
      var sel = window.getSelection();
      if (sel.anchorNode && sel.extend && contains(cm.display.lineDiv, sel.anchorNode)) {
        result.anchorNode = sel.anchorNode;
        result.anchorOffset = sel.anchorOffset;
        result.focusNode = sel.focusNode;
        result.focusOffset = sel.focusOffset;
      }
    }
    return result
  }

  function restoreSelection(snapshot) {
    if (!snapshot || !snapshot.activeElt || snapshot.activeElt == activeElt()) { return }
    snapshot.activeElt.focus();
    if (snapshot.anchorNode && contains(document.body, snapshot.anchorNode) && contains(document.body, snapshot.focusNode)) {
      var sel = window.getSelection(), range$$1 = document.createRange();
      range$$1.setEnd(snapshot.anchorNode, snapshot.anchorOffset);
      range$$1.collapse(false);
      sel.removeAllRanges();
      sel.addRange(range$$1);
      sel.extend(snapshot.focusNode, snapshot.focusOffset);
    }
  }

  // Does the actual updating of the line display. Bails out
  // (returning false) when there is nothing to be done and forced is
  // false.
  function updateDisplayIfNeeded(cm, update) {
    var display = cm.display, doc = cm.doc;

    if (update.editorIsHidden) {
      resetView(cm);
      return false
    }

    // Bail out if the visible area is already rendered and nothing changed.
    if (!update.force &&
        update.visible.from >= display.viewFrom && update.visible.to <= display.viewTo &&
        (display.updateLineNumbers == null || display.updateLineNumbers >= display.viewTo) &&
        display.renderedView == display.view && countDirtyView(cm) == 0)
      { return false }

    if (maybeUpdateLineNumberWidth(cm)) {
      resetView(cm);
      update.dims = getDimensions(cm);
    }

    // Compute a suitable new viewport (from & to)
    var end = doc.first + doc.size;
    var from = Math.max(update.visible.from - cm.options.viewportMargin, doc.first);
    var to = Math.min(end, update.visible.to + cm.options.viewportMargin);
    if (display.viewFrom < from && from - display.viewFrom < 20) { from = Math.max(doc.first, display.viewFrom); }
    if (display.viewTo > to && display.viewTo - to < 20) { to = Math.min(end, display.viewTo); }
    if (sawCollapsedSpans) {
      from = visualLineNo(cm.doc, from);
      to = visualLineEndNo(cm.doc, to);
    }

    var different = from != display.viewFrom || to != display.viewTo ||
      display.lastWrapHeight != update.wrapperHeight || display.lastWrapWidth != update.wrapperWidth;
    adjustView(cm, from, to);

    display.viewOffset = heightAtLine(getLine(cm.doc, display.viewFrom));
    // Position the mover div to align with the current scroll position
    cm.display.mover.style.top = display.viewOffset + "px";

    var toUpdate = countDirtyView(cm);
    if (!different && toUpdate == 0 && !update.force && display.renderedView == display.view &&
        (display.updateLineNumbers == null || display.updateLineNumbers >= display.viewTo))
      { return false }

    // For big changes, we hide the enclosing element during the
    // update, since that speeds up the operations on most browsers.
    var selSnapshot = selectionSnapshot(cm);
    if (toUpdate > 4) { display.lineDiv.style.display = "none"; }
    patchDisplay(cm, display.updateLineNumbers, update.dims);
    if (toUpdate > 4) { display.lineDiv.style.display = ""; }
    display.renderedView = display.view;
    // There might have been a widget with a focused element that got
    // hidden or updated, if so re-focus it.
    restoreSelection(selSnapshot);

    // Prevent selection and cursors from interfering with the scroll
    // width and height.
    removeChildren(display.cursorDiv);
    removeChildren(display.selectionDiv);
    display.gutters.style.height = display.sizer.style.minHeight = 0;

    if (different) {
      display.lastWrapHeight = update.wrapperHeight;
      display.lastWrapWidth = update.wrapperWidth;
      startWorker(cm, 400);
    }

    display.updateLineNumbers = null;

    return true
  }

  function postUpdateDisplay(cm, update) {
    var viewport = update.viewport;

    for (var first = true;; first = false) {
      if (!first || !cm.options.lineWrapping || update.oldDisplayWidth == displayWidth(cm)) {
        // Clip forced viewport to actual scrollable area.
        if (viewport && viewport.top != null)
          { viewport = {top: Math.min(cm.doc.height + paddingVert(cm.display) - displayHeight(cm), viewport.top)}; }
        // Updated line heights might result in the drawn area not
        // actually covering the viewport. Keep looping until it does.
        update.visible = visibleLines(cm.display, cm.doc, viewport);
        if (update.visible.from >= cm.display.viewFrom && update.visible.to <= cm.display.viewTo)
          { break }
      }
      if (!updateDisplayIfNeeded(cm, update)) { break }
      updateHeightsInViewport(cm);
      var barMeasure = measureForScrollbars(cm);
      updateSelection(cm);
      updateScrollbars(cm, barMeasure);
      setDocumentHeight(cm, barMeasure);
      update.force = false;
    }

    update.signal(cm, "update", cm);
    if (cm.display.viewFrom != cm.display.reportedViewFrom || cm.display.viewTo != cm.display.reportedViewTo) {
      update.signal(cm, "viewportChange", cm, cm.display.viewFrom, cm.display.viewTo);
      cm.display.reportedViewFrom = cm.display.viewFrom; cm.display.reportedViewTo = cm.display.viewTo;
    }
  }

  function updateDisplaySimple(cm, viewport) {
    var update = new DisplayUpdate(cm, viewport);
    if (updateDisplayIfNeeded(cm, update)) {
      updateHeightsInViewport(cm);
      postUpdateDisplay(cm, update);
      var barMeasure = measureForScrollbars(cm);
      updateSelection(cm);
      updateScrollbars(cm, barMeasure);
      setDocumentHeight(cm, barMeasure);
      update.finish();
    }
  }

  // Sync the actual display DOM structure with display.view, removing
  // nodes for lines that are no longer in view, and creating the ones
  // that are not there yet, and updating the ones that are out of
  // date.
  function patchDisplay(cm, updateNumbersFrom, dims) {
    var display = cm.display, lineNumbers = cm.options.lineNumbers;
    var container = display.lineDiv, cur = container.firstChild;

    function rm(node) {
      var next = node.nextSibling;
      // Works around a throw-scroll bug in OS X Webkit
      if (webkit && mac && cm.display.currentWheelTarget == node)
        { node.style.display = "none"; }
      else
        { node.parentNode.removeChild(node); }
      return next
    }

    var view = display.view, lineN = display.viewFrom;
    // Loop over the elements in the view, syncing cur (the DOM nodes
    // in display.lineDiv) with the view as we go.
    for (var i = 0; i < view.length; i++) {
      var lineView = view[i];
      if (lineView.hidden) ; else if (!lineView.node || lineView.node.parentNode != container) { // Not drawn yet
        var node = buildLineElement(cm, lineView, lineN, dims);
        container.insertBefore(node, cur);
      } else { // Already drawn
        while (cur != lineView.node) { cur = rm(cur); }
        var updateNumber = lineNumbers && updateNumbersFrom != null &&
          updateNumbersFrom <= lineN && lineView.lineNumber;
        if (lineView.changes) {
          if (indexOf(lineView.changes, "gutter") > -1) { updateNumber = false; }
          updateLineForChanges(cm, lineView, lineN, dims);
        }
        if (updateNumber) {
          removeChildren(lineView.lineNumber);
          lineView.lineNumber.appendChild(document.createTextNode(lineNumberFor(cm.options, lineN)));
        }
        cur = lineView.node.nextSibling;
      }
      lineN += lineView.size;
    }
    while (cur) { cur = rm(cur); }
  }

  function updateGutterSpace(display) {
    var width = display.gutters.offsetWidth;
    display.sizer.style.marginLeft = width + "px";
  }

  function setDocumentHeight(cm, measure) {
    cm.display.sizer.style.minHeight = measure.docHeight + "px";
    cm.display.heightForcer.style.top = measure.docHeight + "px";
    cm.display.gutters.style.height = (measure.docHeight + cm.display.barHeight + scrollGap(cm)) + "px";
  }

  // Re-align line numbers and gutter marks to compensate for
  // horizontal scrolling.
  function alignHorizontally(cm) {
    var display = cm.display, view = display.view;
    if (!display.alignWidgets && (!display.gutters.firstChild || !cm.options.fixedGutter)) { return }
    var comp = compensateForHScroll(display) - display.scroller.scrollLeft + cm.doc.scrollLeft;
    var gutterW = display.gutters.offsetWidth, left = comp + "px";
    for (var i = 0; i < view.length; i++) { if (!view[i].hidden) {
      if (cm.options.fixedGutter) {
        if (view[i].gutter)
          { view[i].gutter.style.left = left; }
        if (view[i].gutterBackground)
          { view[i].gutterBackground.style.left = left; }
      }
      var align = view[i].alignable;
      if (align) { for (var j = 0; j < align.length; j++)
        { align[j].style.left = left; } }
    } }
    if (cm.options.fixedGutter)
      { display.gutters.style.left = (comp + gutterW) + "px"; }
  }

  // Used to ensure that the line number gutter is still the right
  // size for the current document size. Returns true when an update
  // is needed.
  function maybeUpdateLineNumberWidth(cm) {
    if (!cm.options.lineNumbers) { return false }
    var doc = cm.doc, last = lineNumberFor(cm.options, doc.first + doc.size - 1), display = cm.display;
    if (last.length != display.lineNumChars) {
      var test = display.measure.appendChild(elt("div", [elt("div", last)],
                                                 "CodeMirror-linenumber CodeMirror-gutter-elt"));
      var innerW = test.firstChild.offsetWidth, padding = test.offsetWidth - innerW;
      display.lineGutter.style.width = "";
      display.lineNumInnerWidth = Math.max(innerW, display.lineGutter.offsetWidth - padding) + 1;
      display.lineNumWidth = display.lineNumInnerWidth + padding;
      display.lineNumChars = display.lineNumInnerWidth ? last.length : -1;
      display.lineGutter.style.width = display.lineNumWidth + "px";
      updateGutterSpace(cm.display);
      return true
    }
    return false
  }

  function getGutters(gutters, lineNumbers) {
    var result = [], sawLineNumbers = false;
    for (var i = 0; i < gutters.length; i++) {
      var name = gutters[i], style = null;
      if (typeof name != "string") { style = name.style; name = name.className; }
      if (name == "CodeMirror-linenumbers") {
        if (!lineNumbers) { continue }
        else { sawLineNumbers = true; }
      }
      result.push({className: name, style: style});
    }
    if (lineNumbers && !sawLineNumbers) { result.push({className: "CodeMirror-linenumbers", style: null}); }
    return result
  }

  // Rebuild the gutter elements, ensure the margin to the left of the
  // code matches their width.
  function renderGutters(display) {
    var gutters = display.gutters, specs = display.gutterSpecs;
    removeChildren(gutters);
    display.lineGutter = null;
    for (var i = 0; i < specs.length; ++i) {
      var ref = specs[i];
      var className = ref.className;
      var style = ref.style;
      var gElt = gutters.appendChild(elt("div", null, "CodeMirror-gutter " + className));
      if (style) { gElt.style.cssText = style; }
      if (className == "CodeMirror-linenumbers") {
        display.lineGutter = gElt;
        gElt.style.width = (display.lineNumWidth || 1) + "px";
      }
    }
    gutters.style.display = specs.length ? "" : "none";
    updateGutterSpace(display);
  }

  function updateGutters(cm) {
    renderGutters(cm.display);
    regChange(cm);
    alignHorizontally(cm);
  }

  // The display handles the DOM integration, both for input reading
  // and content drawing. It holds references to DOM nodes and
  // display-related state.

  function Display(place, doc, input, options) {
    var d = this;
    this.input = input;

    // Covers bottom-right square when both scrollbars are present.
    d.scrollbarFiller = elt("div", null, "CodeMirror-scrollbar-filler");
    d.scrollbarFiller.setAttribute("cm-not-content", "true");
    // Covers bottom of gutter when coverGutterNextToScrollbar is on
    // and h scrollbar is present.
    d.gutterFiller = elt("div", null, "CodeMirror-gutter-filler");
    d.gutterFiller.setAttribute("cm-not-content", "true");
    // Will contain the actual code, positioned to cover the viewport.
    d.lineDiv = eltP("div", null, "CodeMirror-code");
    // Elements are added to these to represent selection and cursors.
    d.selectionDiv = elt("div", null, null, "position: relative; z-index: 1");
    d.cursorDiv = elt("div", null, "CodeMirror-cursors");
    // A visibility: hidden element used to find the size of things.
    d.measure = elt("div", null, "CodeMirror-measure");
    // When lines outside of the viewport are measured, they are drawn in this.
    d.lineMeasure = elt("div", null, "CodeMirror-measure");
    // Wraps everything that needs to exist inside the vertically-padded coordinate system
    d.lineSpace = eltP("div", [d.measure, d.lineMeasure, d.selectionDiv, d.cursorDiv, d.lineDiv],
                      null, "position: relative; outline: none");
    var lines = eltP("div", [d.lineSpace], "CodeMirror-lines");
    // Moved around its parent to cover visible view.
    d.mover = elt("div", [lines], null, "position: relative");
    // Set to the height of the document, allowing scrolling.
    d.sizer = elt("div", [d.mover], "CodeMirror-sizer");
    d.sizerWidth = null;
    // Behavior of elts with overflow: auto and padding is
    // inconsistent across browsers. This is used to ensure the
    // scrollable area is big enough.
    d.heightForcer = elt("div", null, null, "position: absolute; height: " + scrollerGap + "px; width: 1px;");
    // Will contain the gutters, if any.
    d.gutters = elt("div", null, "CodeMirror-gutters");
    d.lineGutter = null;
    // Actual scrollable element.
    d.scroller = elt("div", [d.sizer, d.heightForcer, d.gutters], "CodeMirror-scroll");
    d.scroller.setAttribute("tabIndex", "-1");
    // The element in which the editor lives.
    d.wrapper = elt("div", [d.scrollbarFiller, d.gutterFiller, d.scroller], "CodeMirror");

    // Work around IE7 z-index bug (not perfect, hence IE7 not really being supported)
    if (ie && ie_version < 8) { d.gutters.style.zIndex = -1; d.scroller.style.paddingRight = 0; }
    if (!webkit && !(gecko && mobile)) { d.scroller.draggable = true; }

    if (place) {
      if (place.appendChild) { place.appendChild(d.wrapper); }
      else { place(d.wrapper); }
    }

    // Current rendered range (may be bigger than the view window).
    d.viewFrom = d.viewTo = doc.first;
    d.reportedViewFrom = d.reportedViewTo = doc.first;
    // Information about the rendered lines.
    d.view = [];
    d.renderedView = null;
    // Holds info about a single rendered line when it was rendered
    // for measurement, while not in view.
    d.externalMeasured = null;
    // Empty space (in pixels) above the view
    d.viewOffset = 0;
    d.lastWrapHeight = d.lastWrapWidth = 0;
    d.updateLineNumbers = null;

    d.nativeBarWidth = d.barHeight = d.barWidth = 0;
    d.scrollbarsClipped = false;

    // Used to only resize the line number gutter when necessary (when
    // the amount of lines crosses a boundary that makes its width change)
    d.lineNumWidth = d.lineNumInnerWidth = d.lineNumChars = null;
    // Set to true when a non-horizontal-scrolling line widget is
    // added. As an optimization, line widget aligning is skipped when
    // this is false.
    d.alignWidgets = false;

    d.cachedCharWidth = d.cachedTextHeight = d.cachedPaddingH = null;

    // Tracks the maximum line length so that the horizontal scrollbar
    // can be kept static when scrolling.
    d.maxLine = null;
    d.maxLineLength = 0;
    d.maxLineChanged = false;

    // Used for measuring wheel scrolling granularity
    d.wheelDX = d.wheelDY = d.wheelStartX = d.wheelStartY = null;

    // True when shift is held down.
    d.shift = false;

    // Used to track whether anything happened since the context menu
    // was opened.
    d.selForContextMenu = null;

    d.activeTouch = null;

    d.gutterSpecs = getGutters(options.gutters, options.lineNumbers);
    renderGutters(d);

    input.init(d);
  }

  // Since the delta values reported on mouse wheel events are
  // unstandardized between browsers and even browser versions, and
  // generally horribly unpredictable, this code starts by measuring
  // the scroll effect that the first few mouse wheel events have,
  // and, from that, detects the way it can convert deltas to pixel
  // offsets afterwards.
  //
  // The reason we want to know the amount a wheel event will scroll
  // is that it gives us a chance to update the display before the
  // actual scrolling happens, reducing flickering.

  var wheelSamples = 0, wheelPixelsPerUnit = null;
  // Fill in a browser-detected starting value on browsers where we
  // know one. These don't have to be accurate -- the result of them
  // being wrong would just be a slight flicker on the first wheel
  // scroll (if it is large enough).
  if (ie) { wheelPixelsPerUnit = -.53; }
  else if (gecko) { wheelPixelsPerUnit = 15; }
  else if (chrome) { wheelPixelsPerUnit = -.7; }
  else if (safari) { wheelPixelsPerUnit = -1/3; }

  function wheelEventDelta(e) {
    var dx = e.wheelDeltaX, dy = e.wheelDeltaY;
    if (dx == null && e.detail && e.axis == e.HORIZONTAL_AXIS) { dx = e.detail; }
    if (dy == null && e.detail && e.axis == e.VERTICAL_AXIS) { dy = e.detail; }
    else if (dy == null) { dy = e.wheelDelta; }
    return {x: dx, y: dy}
  }
  function wheelEventPixels(e) {
    var delta = wheelEventDelta(e);
    delta.x *= wheelPixelsPerUnit;
    delta.y *= wheelPixelsPerUnit;
    return delta
  }

  function onScrollWheel(cm, e) {
    var delta = wheelEventDelta(e), dx = delta.x, dy = delta.y;

    var display = cm.display, scroll = display.scroller;
    // Quit if there's nothing to scroll here
    var canScrollX = scroll.scrollWidth > scroll.clientWidth;
    var canScrollY = scroll.scrollHeight > scroll.clientHeight;
    if (!(dx && canScrollX || dy && canScrollY)) { return }

    // Webkit browsers on OS X abort momentum scrolls when the target
    // of the scroll event is removed from the scrollable element.
    // This hack (see related code in patchDisplay) makes sure the
    // element is kept around.
    if (dy && mac && webkit) {
      outer: for (var cur = e.target, view = display.view; cur != scroll; cur = cur.parentNode) {
        for (var i = 0; i < view.length; i++) {
          if (view[i].node == cur) {
            cm.display.currentWheelTarget = cur;
            break outer
          }
        }
      }
    }

    // On some browsers, horizontal scrolling will cause redraws to
    // happen before the gutter has been realigned, causing it to
    // wriggle around in a most unseemly way. When we have an
    // estimated pixels/delta value, we just handle horizontal
    // scrolling entirely here. It'll be slightly off from native, but
    // better than glitching out.
    if (dx && !gecko && !presto && wheelPixelsPerUnit != null) {
      if (dy && canScrollY)
        { updateScrollTop(cm, Math.max(0, scroll.scrollTop + dy * wheelPixelsPerUnit)); }
      setScrollLeft(cm, Math.max(0, scroll.scrollLeft + dx * wheelPixelsPerUnit));
      // Only prevent default scrolling if vertical scrolling is
      // actually possible. Otherwise, it causes vertical scroll
      // jitter on OSX trackpads when deltaX is small and deltaY
      // is large (issue #3579)
      if (!dy || (dy && canScrollY))
        { e_preventDefault(e); }
      display.wheelStartX = null; // Abort measurement, if in progress
      return
    }

    // 'Project' the visible viewport to cover the area that is being
    // scrolled into view (if we know enough to estimate it).
    if (dy && wheelPixelsPerUnit != null) {
      var pixels = dy * wheelPixelsPerUnit;
      var top = cm.doc.scrollTop, bot = top + display.wrapper.clientHeight;
      if (pixels < 0) { top = Math.max(0, top + pixels - 50); }
      else { bot = Math.min(cm.doc.height, bot + pixels + 50); }
      updateDisplaySimple(cm, {top: top, bottom: bot});
    }

    if (wheelSamples < 20) {
      if (display.wheelStartX == null) {
        display.wheelStartX = scroll.scrollLeft; display.wheelStartY = scroll.scrollTop;
        display.wheelDX = dx; display.wheelDY = dy;
        setTimeout(function () {
          if (display.wheelStartX == null) { return }
          var movedX = scroll.scrollLeft - display.wheelStartX;
          var movedY = scroll.scrollTop - display.wheelStartY;
          var sample = (movedY && display.wheelDY && movedY / display.wheelDY) ||
            (movedX && display.wheelDX && movedX / display.wheelDX);
          display.wheelStartX = display.wheelStartY = null;
          if (!sample) { return }
          wheelPixelsPerUnit = (wheelPixelsPerUnit * wheelSamples + sample) / (wheelSamples + 1);
          ++wheelSamples;
        }, 200);
      } else {
        display.wheelDX += dx; display.wheelDY += dy;
      }
    }
  }

  // Selection objects are immutable. A new one is created every time
  // the selection changes. A selection is one or more non-overlapping
  // (and non-touching) ranges, sorted, and an integer that indicates
  // which one is the primary selection (the one that's scrolled into
  // view, that getCursor returns, etc).
  var Selection = function(ranges, primIndex) {
    this.ranges = ranges;
    this.primIndex = primIndex;
  };

  Selection.prototype.primary = function () { return this.ranges[this.primIndex] };

  Selection.prototype.equals = function (other) {
      var this$1 = this;

    if (other == this) { return true }
    if (other.primIndex != this.primIndex || other.ranges.length != this.ranges.length) { return false }
    for (var i = 0; i < this.ranges.length; i++) {
      var here = this$1.ranges[i], there = other.ranges[i];
      if (!equalCursorPos(here.anchor, there.anchor) || !equalCursorPos(here.head, there.head)) { return false }
    }
    return true
  };

  Selection.prototype.deepCopy = function () {
      var this$1 = this;

    var out = [];
    for (var i = 0; i < this.ranges.length; i++)
      { out[i] = new Range(copyPos(this$1.ranges[i].anchor), copyPos(this$1.ranges[i].head)); }
    return new Selection(out, this.primIndex)
  };

  Selection.prototype.somethingSelected = function () {
      var this$1 = this;

    for (var i = 0; i < this.ranges.length; i++)
      { if (!this$1.ranges[i].empty()) { return true } }
    return false
  };

  Selection.prototype.contains = function (pos, end) {
      var this$1 = this;

    if (!end) { end = pos; }
    for (var i = 0; i < this.ranges.length; i++) {
      var range = this$1.ranges[i];
      if (cmp(end, range.from()) >= 0 && cmp(pos, range.to()) <= 0)
        { return i }
    }
    return -1
  };

  var Range = function(anchor, head) {
    this.anchor = anchor; this.head = head;
  };

  Range.prototype.from = function () { return minPos(this.anchor, this.head) };
  Range.prototype.to = function () { return maxPos(this.anchor, this.head) };
  Range.prototype.empty = function () { return this.head.line == this.anchor.line && this.head.ch == this.anchor.ch };

  // Take an unsorted, potentially overlapping set of ranges, and
  // build a selection out of it. 'Consumes' ranges array (modifying
  // it).
  function normalizeSelection(cm, ranges, primIndex) {
    var mayTouch = cm && cm.options.selectionsMayTouch;
    var prim = ranges[primIndex];
    ranges.sort(function (a, b) { return cmp(a.from(), b.from()); });
    primIndex = indexOf(ranges, prim);
    for (var i = 1; i < ranges.length; i++) {
      var cur = ranges[i], prev = ranges[i - 1];
      var diff = cmp(prev.to(), cur.from());
      if (mayTouch && !cur.empty() ? diff > 0 : diff >= 0) {
        var from = minPos(prev.from(), cur.from()), to = maxPos(prev.to(), cur.to());
        var inv = prev.empty() ? cur.from() == cur.head : prev.from() == prev.head;
        if (i <= primIndex) { --primIndex; }
        ranges.splice(--i, 2, new Range(inv ? to : from, inv ? from : to));
      }
    }
    return new Selection(ranges, primIndex)
  }

  function simpleSelection(anchor, head) {
    return new Selection([new Range(anchor, head || anchor)], 0)
  }

  // Compute the position of the end of a change (its 'to' property
  // refers to the pre-change end).
  function changeEnd(change) {
    if (!change.text) { return change.to }
    return Pos(change.from.line + change.text.length - 1,
               lst(change.text).length + (change.text.length == 1 ? change.from.ch : 0))
  }

  // Adjust a position to refer to the post-change position of the
  // same text, or the end of the change if the change covers it.
  function adjustForChange(pos, change) {
    if (cmp(pos, change.from) < 0) { return pos }
    if (cmp(pos, change.to) <= 0) { return changeEnd(change) }

    var line = pos.line + change.text.length - (change.to.line - change.from.line) - 1, ch = pos.ch;
    if (pos.line == change.to.line) { ch += changeEnd(change).ch - change.to.ch; }
    return Pos(line, ch)
  }

  function computeSelAfterChange(doc, change) {
    var out = [];
    for (var i = 0; i < doc.sel.ranges.length; i++) {
      var range = doc.sel.ranges[i];
      out.push(new Range(adjustForChange(range.anchor, change),
                         adjustForChange(range.head, change)));
    }
    return normalizeSelection(doc.cm, out, doc.sel.primIndex)
  }

  function offsetPos(pos, old, nw) {
    if (pos.line == old.line)
      { return Pos(nw.line, pos.ch - old.ch + nw.ch) }
    else
      { return Pos(nw.line + (pos.line - old.line), pos.ch) }
  }

  // Used by replaceSelections to allow moving the selection to the
  // start or around the replaced test. Hint may be "start" or "around".
  function computeReplacedSel(doc, changes, hint) {
    var out = [];
    var oldPrev = Pos(doc.first, 0), newPrev = oldPrev;
    for (var i = 0; i < changes.length; i++) {
      var change = changes[i];
      var from = offsetPos(change.from, oldPrev, newPrev);
      var to = offsetPos(changeEnd(change), oldPrev, newPrev);
      oldPrev = change.to;
      newPrev = to;
      if (hint == "around") {
        var range = doc.sel.ranges[i], inv = cmp(range.head, range.anchor) < 0;
        out[i] = new Range(inv ? to : from, inv ? from : to);
      } else {
        out[i] = new Range(from, from);
      }
    }
    return new Selection(out, doc.sel.primIndex)
  }

  // Used to get the editor into a consistent state again when options change.

  function loadMode(cm) {
    cm.doc.mode = getMode(cm.options, cm.doc.modeOption);
    resetModeState(cm);
  }

  function resetModeState(cm) {
    cm.doc.iter(function (line) {
      if (line.stateAfter) { line.stateAfter = null; }
      if (line.styles) { line.styles = null; }
    });
    cm.doc.modeFrontier = cm.doc.highlightFrontier = cm.doc.first;
    startWorker(cm, 100);
    cm.state.modeGen++;
    if (cm.curOp) { regChange(cm); }
  }

  // DOCUMENT DATA STRUCTURE

  // By default, updates that start and end at the beginning of a line
  // are treated specially, in order to make the association of line
  // widgets and marker elements with the text behave more intuitive.
  function isWholeLineUpdate(doc, change) {
    return change.from.ch == 0 && change.to.ch == 0 && lst(change.text) == "" &&
      (!doc.cm || doc.cm.options.wholeLineUpdateBefore)
  }

  // Perform a change on the document data structure.
  function updateDoc(doc, change, markedSpans, estimateHeight$$1) {
    function spansFor(n) {return markedSpans ? markedSpans[n] : null}
    function update(line, text, spans) {
      updateLine(line, text, spans, estimateHeight$$1);
      signalLater(line, "change", line, change);
    }
    function linesFor(start, end) {
      var result = [];
      for (var i = start; i < end; ++i)
        { result.push(new Line(text[i], spansFor(i), estimateHeight$$1)); }
      return result
    }

    var from = change.from, to = change.to, text = change.text;
    var firstLine = getLine(doc, from.line), lastLine = getLine(doc, to.line);
    var lastText = lst(text), lastSpans = spansFor(text.length - 1), nlines = to.line - from.line;

    // Adjust the line structure
    if (change.full) {
      doc.insert(0, linesFor(0, text.length));
      doc.remove(text.length, doc.size - text.length);
    } else if (isWholeLineUpdate(doc, change)) {
      // This is a whole-line replace. Treated specially to make
      // sure line objects move the way they are supposed to.
      var added = linesFor(0, text.length - 1);
      update(lastLine, lastLine.text, lastSpans);
      if (nlines) { doc.remove(from.line, nlines); }
      if (added.length) { doc.insert(from.line, added); }
    } else if (firstLine == lastLine) {
      if (text.length == 1) {
        update(firstLine, firstLine.text.slice(0, from.ch) + lastText + firstLine.text.slice(to.ch), lastSpans);
      } else {
        var added$1 = linesFor(1, text.length - 1);
        added$1.push(new Line(lastText + firstLine.text.slice(to.ch), lastSpans, estimateHeight$$1));
        update(firstLine, firstLine.text.slice(0, from.ch) + text[0], spansFor(0));
        doc.insert(from.line + 1, added$1);
      }
    } else if (text.length == 1) {
      update(firstLine, firstLine.text.slice(0, from.ch) + text[0] + lastLine.text.slice(to.ch), spansFor(0));
      doc.remove(from.line + 1, nlines);
    } else {
      update(firstLine, firstLine.text.slice(0, from.ch) + text[0], spansFor(0));
      update(lastLine, lastText + lastLine.text.slice(to.ch), lastSpans);
      var added$2 = linesFor(1, text.length - 1);
      if (nlines > 1) { doc.remove(from.line + 1, nlines - 1); }
      doc.insert(from.line + 1, added$2);
    }

    signalLater(doc, "change", doc, change);
  }

  // Call f for all linked documents.
  function linkedDocs(doc, f, sharedHistOnly) {
    function propagate(doc, skip, sharedHist) {
      if (doc.linked) { for (var i = 0; i < doc.linked.length; ++i) {
        var rel = doc.linked[i];
        if (rel.doc == skip) { continue }
        var shared = sharedHist && rel.sharedHist;
        if (sharedHistOnly && !shared) { continue }
        f(rel.doc, shared);
        propagate(rel.doc, doc, shared);
      } }
    }
    propagate(doc, null, true);
  }

  // Attach a document to an editor.
  function attachDoc(cm, doc) {
    if (doc.cm) { throw new Error("This document is already in use.") }
    cm.doc = doc;
    doc.cm = cm;
    estimateLineHeights(cm);
    loadMode(cm);
    setDirectionClass(cm);
    if (!cm.options.lineWrapping) { findMaxLine(cm); }
    cm.options.mode = doc.modeOption;
    regChange(cm);
  }

  function setDirectionClass(cm) {
  (cm.doc.direction == "rtl" ? addClass : rmClass)(cm.display.lineDiv, "CodeMirror-rtl");
  }

  function directionChanged(cm) {
    runInOp(cm, function () {
      setDirectionClass(cm);
      regChange(cm);
    });
  }

  function History(startGen) {
    // Arrays of change events and selections. Doing something adds an
    // event to done and clears undo. Undoing moves events from done
    // to undone, redoing moves them in the other direction.
    this.done = []; this.undone = [];
    this.undoDepth = Infinity;
    // Used to track when changes can be merged into a single undo
    // event
    this.lastModTime = this.lastSelTime = 0;
    this.lastOp = this.lastSelOp = null;
    this.lastOrigin = this.lastSelOrigin = null;
    // Used by the isClean() method
    this.generation = this.maxGeneration = startGen || 1;
  }

  // Create a history change event from an updateDoc-style change
  // object.
  function historyChangeFromChange(doc, change) {
    var histChange = {from: copyPos(change.from), to: changeEnd(change), text: getBetween(doc, change.from, change.to)};
    attachLocalSpans(doc, histChange, change.from.line, change.to.line + 1);
    linkedDocs(doc, function (doc) { return attachLocalSpans(doc, histChange, change.from.line, change.to.line + 1); }, true);
    return histChange
  }

  // Pop all selection events off the end of a history array. Stop at
  // a change event.
  function clearSelectionEvents(array) {
    while (array.length) {
      var last = lst(array);
      if (last.ranges) { array.pop(); }
      else { break }
    }
  }

  // Find the top change event in the history. Pop off selection
  // events that are in the way.
  function lastChangeEvent(hist, force) {
    if (force) {
      clearSelectionEvents(hist.done);
      return lst(hist.done)
    } else if (hist.done.length && !lst(hist.done).ranges) {
      return lst(hist.done)
    } else if (hist.done.length > 1 && !hist.done[hist.done.length - 2].ranges) {
      hist.done.pop();
      return lst(hist.done)
    }
  }

  // Register a change in the history. Merges changes that are within
  // a single operation, or are close together with an origin that
  // allows merging (starting with "+") into a single event.
  function addChangeToHistory(doc, change, selAfter, opId) {
    var hist = doc.history;
    hist.undone.length = 0;
    var time = +new Date, cur;
    var last;

    if ((hist.lastOp == opId ||
         hist.lastOrigin == change.origin && change.origin &&
         ((change.origin.charAt(0) == "+" && hist.lastModTime > time - (doc.cm ? doc.cm.options.historyEventDelay : 500)) ||
          change.origin.charAt(0) == "*")) &&
        (cur = lastChangeEvent(hist, hist.lastOp == opId))) {
      // Merge this change into the last event
      last = lst(cur.changes);
      if (cmp(change.from, change.to) == 0 && cmp(change.from, last.to) == 0) {
        // Optimized case for simple insertion -- don't want to add
        // new changesets for every character typed
        last.to = changeEnd(change);
      } else {
        // Add new sub-event
        cur.changes.push(historyChangeFromChange(doc, change));
      }
    } else {
      // Can not be merged, start a new event.
      var before = lst(hist.done);
      if (!before || !before.ranges)
        { pushSelectionToHistory(doc.sel, hist.done); }
      cur = {changes: [historyChangeFromChange(doc, change)],
             generation: hist.generation};
      hist.done.push(cur);
      while (hist.done.length > hist.undoDepth) {
        hist.done.shift();
        if (!hist.done[0].ranges) { hist.done.shift(); }
      }
    }
    hist.done.push(selAfter);
    hist.generation = ++hist.maxGeneration;
    hist.lastModTime = hist.lastSelTime = time;
    hist.lastOp = hist.lastSelOp = opId;
    hist.lastOrigin = hist.lastSelOrigin = change.origin;

    if (!last) { signal(doc, "historyAdded"); }
  }

  function selectionEventCanBeMerged(doc, origin, prev, sel) {
    var ch = origin.charAt(0);
    return ch == "*" ||
      ch == "+" &&
      prev.ranges.length == sel.ranges.length &&
      prev.somethingSelected() == sel.somethingSelected() &&
      new Date - doc.history.lastSelTime <= (doc.cm ? doc.cm.options.historyEventDelay : 500)
  }

  // Called whenever the selection changes, sets the new selection as
  // the pending selection in the history, and pushes the old pending
  // selection into the 'done' array when it was significantly
  // different (in number of selected ranges, emptiness, or time).
  function addSelectionToHistory(doc, sel, opId, options) {
    var hist = doc.history, origin = options && options.origin;

    // A new event is started when the previous origin does not match
    // the current, or the origins don't allow matching. Origins
    // starting with * are always merged, those starting with + are
    // merged when similar and close together in time.
    if (opId == hist.lastSelOp ||
        (origin && hist.lastSelOrigin == origin &&
         (hist.lastModTime == hist.lastSelTime && hist.lastOrigin == origin ||
          selectionEventCanBeMerged(doc, origin, lst(hist.done), sel))))
      { hist.done[hist.done.length - 1] = sel; }
    else
      { pushSelectionToHistory(sel, hist.done); }

    hist.lastSelTime = +new Date;
    hist.lastSelOrigin = origin;
    hist.lastSelOp = opId;
    if (options && options.clearRedo !== false)
      { clearSelectionEvents(hist.undone); }
  }

  function pushSelectionToHistory(sel, dest) {
    var top = lst(dest);
    if (!(top && top.ranges && top.equals(sel)))
      { dest.push(sel); }
  }

  // Used to store marked span information in the history.
  function attachLocalSpans(doc, change, from, to) {
    var existing = change["spans_" + doc.id], n = 0;
    doc.iter(Math.max(doc.first, from), Math.min(doc.first + doc.size, to), function (line) {
      if (line.markedSpans)
        { (existing || (existing = change["spans_" + doc.id] = {}))[n] = line.markedSpans; }
      ++n;
    });
  }

  // When un/re-doing restores text containing marked spans, those
  // that have been explicitly cleared should not be restored.
  function removeClearedSpans(spans) {
    if (!spans) { return null }
    var out;
    for (var i = 0; i < spans.length; ++i) {
      if (spans[i].marker.explicitlyCleared) { if (!out) { out = spans.slice(0, i); } }
      else if (out) { out.push(spans[i]); }
    }
    return !out ? spans : out.length ? out : null
  }

  // Retrieve and filter the old marked spans stored in a change event.
  function getOldSpans(doc, change) {
    var found = change["spans_" + doc.id];
    if (!found) { return null }
    var nw = [];
    for (var i = 0; i < change.text.length; ++i)
      { nw.push(removeClearedSpans(found[i])); }
    return nw
  }

  // Used for un/re-doing changes from the history. Combines the
  // result of computing the existing spans with the set of spans that
  // existed in the history (so that deleting around a span and then
  // undoing brings back the span).
  function mergeOldSpans(doc, change) {
    var old = getOldSpans(doc, change);
    var stretched = stretchSpansOverChange(doc, change);
    if (!old) { return stretched }
    if (!stretched) { return old }

    for (var i = 0; i < old.length; ++i) {
      var oldCur = old[i], stretchCur = stretched[i];
      if (oldCur && stretchCur) {
        spans: for (var j = 0; j < stretchCur.length; ++j) {
          var span = stretchCur[j];
          for (var k = 0; k < oldCur.length; ++k)
            { if (oldCur[k].marker == span.marker) { continue spans } }
          oldCur.push(span);
        }
      } else if (stretchCur) {
        old[i] = stretchCur;
      }
    }
    return old
  }

  // Used both to provide a JSON-safe object in .getHistory, and, when
  // detaching a document, to split the history in two
  function copyHistoryArray(events, newGroup, instantiateSel) {
    var copy = [];
    for (var i = 0; i < events.length; ++i) {
      var event = events[i];
      if (event.ranges) {
        copy.push(instantiateSel ? Selection.prototype.deepCopy.call(event) : event);
        continue
      }
      var changes = event.changes, newChanges = [];
      copy.push({changes: newChanges});
      for (var j = 0; j < changes.length; ++j) {
        var change = changes[j], m = (void 0);
        newChanges.push({from: change.from, to: change.to, text: change.text});
        if (newGroup) { for (var prop in change) { if (m = prop.match(/^spans_(\d+)$/)) {
          if (indexOf(newGroup, Number(m[1])) > -1) {
            lst(newChanges)[prop] = change[prop];
            delete change[prop];
          }
        } } }
      }
    }
    return copy
  }

  // The 'scroll' parameter given to many of these indicated whether
  // the new cursor position should be scrolled into view after
  // modifying the selection.

  // If shift is held or the extend flag is set, extends a range to
  // include a given position (and optionally a second position).
  // Otherwise, simply returns the range between the given positions.
  // Used for cursor motion and such.
  function extendRange(range, head, other, extend) {
    if (extend) {
      var anchor = range.anchor;
      if (other) {
        var posBefore = cmp(head, anchor) < 0;
        if (posBefore != (cmp(other, anchor) < 0)) {
          anchor = head;
          head = other;
        } else if (posBefore != (cmp(head, other) < 0)) {
          head = other;
        }
      }
      return new Range(anchor, head)
    } else {
      return new Range(other || head, head)
    }
  }

  // Extend the primary selection range, discard the rest.
  function extendSelection(doc, head, other, options, extend) {
    if (extend == null) { extend = doc.cm && (doc.cm.display.shift || doc.extend); }
    setSelection(doc, new Selection([extendRange(doc.sel.primary(), head, other, extend)], 0), options);
  }

  // Extend all selections (pos is an array of selections with length
  // equal the number of selections)
  function extendSelections(doc, heads, options) {
    var out = [];
    var extend = doc.cm && (doc.cm.display.shift || doc.extend);
    for (var i = 0; i < doc.sel.ranges.length; i++)
      { out[i] = extendRange(doc.sel.ranges[i], heads[i], null, extend); }
    var newSel = normalizeSelection(doc.cm, out, doc.sel.primIndex);
    setSelection(doc, newSel, options);
  }

  // Updates a single range in the selection.
  function replaceOneSelection(doc, i, range, options) {
    var ranges = doc.sel.ranges.slice(0);
    ranges[i] = range;
    setSelection(doc, normalizeSelection(doc.cm, ranges, doc.sel.primIndex), options);
  }

  // Reset the selection to a single range.
  function setSimpleSelection(doc, anchor, head, options) {
    setSelection(doc, simpleSelection(anchor, head), options);
  }

  // Give beforeSelectionChange handlers a change to influence a
  // selection update.
  function filterSelectionChange(doc, sel, options) {
    var obj = {
      ranges: sel.ranges,
      update: function(ranges) {
        var this$1 = this;

        this.ranges = [];
        for (var i = 0; i < ranges.length; i++)
          { this$1.ranges[i] = new Range(clipPos(doc, ranges[i].anchor),
                                     clipPos(doc, ranges[i].head)); }
      },
      origin: options && options.origin
    };
    signal(doc, "beforeSelectionChange", doc, obj);
    if (doc.cm) { signal(doc.cm, "beforeSelectionChange", doc.cm, obj); }
    if (obj.ranges != sel.ranges) { return normalizeSelection(doc.cm, obj.ranges, obj.ranges.length - 1) }
    else { return sel }
  }

  function setSelectionReplaceHistory(doc, sel, options) {
    var done = doc.history.done, last = lst(done);
    if (last && last.ranges) {
      done[done.length - 1] = sel;
      setSelectionNoUndo(doc, sel, options);
    } else {
      setSelection(doc, sel, options);
    }
  }

  // Set a new selection.
  function setSelection(doc, sel, options) {
    setSelectionNoUndo(doc, sel, options);
    addSelectionToHistory(doc, doc.sel, doc.cm ? doc.cm.curOp.id : NaN, options);
  }

  function setSelectionNoUndo(doc, sel, options) {
    if (hasHandler(doc, "beforeSelectionChange") || doc.cm && hasHandler(doc.cm, "beforeSelectionChange"))
      { sel = filterSelectionChange(doc, sel, options); }

    var bias = options && options.bias ||
      (cmp(sel.primary().head, doc.sel.primary().head) < 0 ? -1 : 1);
    setSelectionInner(doc, skipAtomicInSelection(doc, sel, bias, true));

    if (!(options && options.scroll === false) && doc.cm)
      { ensureCursorVisible(doc.cm); }
  }

  function setSelectionInner(doc, sel) {
    if (sel.equals(doc.sel)) { return }

    doc.sel = sel;

    if (doc.cm) {
      doc.cm.curOp.updateInput = 1;
      doc.cm.curOp.selectionChanged = true;
      signalCursorActivity(doc.cm);
    }
    signalLater(doc, "cursorActivity", doc);
  }

  // Verify that the selection does not partially select any atomic
  // marked ranges.
  function reCheckSelection(doc) {
    setSelectionInner(doc, skipAtomicInSelection(doc, doc.sel, null, false));
  }

  // Return a selection that does not partially select any atomic
  // ranges.
  function skipAtomicInSelection(doc, sel, bias, mayClear) {
    var out;
    for (var i = 0; i < sel.ranges.length; i++) {
      var range = sel.ranges[i];
      var old = sel.ranges.length == doc.sel.ranges.length && doc.sel.ranges[i];
      var newAnchor = skipAtomic(doc, range.anchor, old && old.anchor, bias, mayClear);
      var newHead = skipAtomic(doc, range.head, old && old.head, bias, mayClear);
      if (out || newAnchor != range.anchor || newHead != range.head) {
        if (!out) { out = sel.ranges.slice(0, i); }
        out[i] = new Range(newAnchor, newHead);
      }
    }
    return out ? normalizeSelection(doc.cm, out, sel.primIndex) : sel
  }

  function skipAtomicInner(doc, pos, oldPos, dir, mayClear) {
    var line = getLine(doc, pos.line);
    if (line.markedSpans) { for (var i = 0; i < line.markedSpans.length; ++i) {
      var sp = line.markedSpans[i], m = sp.marker;
      if ((sp.from == null || (m.inclusiveLeft ? sp.from <= pos.ch : sp.from < pos.ch)) &&
          (sp.to == null || (m.inclusiveRight ? sp.to >= pos.ch : sp.to > pos.ch))) {
        if (mayClear) {
          signal(m, "beforeCursorEnter");
          if (m.explicitlyCleared) {
            if (!line.markedSpans) { break }
            else {--i; continue}
          }
        }
        if (!m.atomic) { continue }

        if (oldPos) {
          var near = m.find(dir < 0 ? 1 : -1), diff = (void 0);
          if (dir < 0 ? m.inclusiveRight : m.inclusiveLeft)
            { near = movePos(doc, near, -dir, near && near.line == pos.line ? line : null); }
          if (near && near.line == pos.line && (diff = cmp(near, oldPos)) && (dir < 0 ? diff < 0 : diff > 0))
            { return skipAtomicInner(doc, near, pos, dir, mayClear) }
        }

        var far = m.find(dir < 0 ? -1 : 1);
        if (dir < 0 ? m.inclusiveLeft : m.inclusiveRight)
          { far = movePos(doc, far, dir, far.line == pos.line ? line : null); }
        return far ? skipAtomicInner(doc, far, pos, dir, mayClear) : null
      }
    } }
    return pos
  }

  // Ensure a given position is not inside an atomic range.
  function skipAtomic(doc, pos, oldPos, bias, mayClear) {
    var dir = bias || 1;
    var found = skipAtomicInner(doc, pos, oldPos, dir, mayClear) ||
        (!mayClear && skipAtomicInner(doc, pos, oldPos, dir, true)) ||
        skipAtomicInner(doc, pos, oldPos, -dir, mayClear) ||
        (!mayClear && skipAtomicInner(doc, pos, oldPos, -dir, true));
    if (!found) {
      doc.cantEdit = true;
      return Pos(doc.first, 0)
    }
    return found
  }

  function movePos(doc, pos, dir, line) {
    if (dir < 0 && pos.ch == 0) {
      if (pos.line > doc.first) { return clipPos(doc, Pos(pos.line - 1)) }
      else { return null }
    } else if (dir > 0 && pos.ch == (line || getLine(doc, pos.line)).text.length) {
      if (pos.line < doc.first + doc.size - 1) { return Pos(pos.line + 1, 0) }
      else { return null }
    } else {
      return new Pos(pos.line, pos.ch + dir)
    }
  }

  function selectAll(cm) {
    cm.setSelection(Pos(cm.firstLine(), 0), Pos(cm.lastLine()), sel_dontScroll);
  }

  // UPDATING

  // Allow "beforeChange" event handlers to influence a change
  function filterChange(doc, change, update) {
    var obj = {
      canceled: false,
      from: change.from,
      to: change.to,
      text: change.text,
      origin: change.origin,
      cancel: function () { return obj.canceled = true; }
    };
    if (update) { obj.update = function (from, to, text, origin) {
      if (from) { obj.from = clipPos(doc, from); }
      if (to) { obj.to = clipPos(doc, to); }
      if (text) { obj.text = text; }
      if (origin !== undefined) { obj.origin = origin; }
    }; }
    signal(doc, "beforeChange", doc, obj);
    if (doc.cm) { signal(doc.cm, "beforeChange", doc.cm, obj); }

    if (obj.canceled) {
      if (doc.cm) { doc.cm.curOp.updateInput = 2; }
      return null
    }
    return {from: obj.from, to: obj.to, text: obj.text, origin: obj.origin}
  }

  // Apply a change to a document, and add it to the document's
  // history, and propagating it to all linked documents.
  function makeChange(doc, change, ignoreReadOnly) {
    if (doc.cm) {
      if (!doc.cm.curOp) { return operation(doc.cm, makeChange)(doc, change, ignoreReadOnly) }
      if (doc.cm.state.suppressEdits) { return }
    }

    if (hasHandler(doc, "beforeChange") || doc.cm && hasHandler(doc.cm, "beforeChange")) {
      change = filterChange(doc, change, true);
      if (!change) { return }
    }

    // Possibly split or suppress the update based on the presence
    // of read-only spans in its range.
    var split = sawReadOnlySpans && !ignoreReadOnly && removeReadOnlyRanges(doc, change.from, change.to);
    if (split) {
      for (var i = split.length - 1; i >= 0; --i)
        { makeChangeInner(doc, {from: split[i].from, to: split[i].to, text: i ? [""] : change.text, origin: change.origin}); }
    } else {
      makeChangeInner(doc, change);
    }
  }

  function makeChangeInner(doc, change) {
    if (change.text.length == 1 && change.text[0] == "" && cmp(change.from, change.to) == 0) { return }
    var selAfter = computeSelAfterChange(doc, change);
    addChangeToHistory(doc, change, selAfter, doc.cm ? doc.cm.curOp.id : NaN);

    makeChangeSingleDoc(doc, change, selAfter, stretchSpansOverChange(doc, change));
    var rebased = [];

    linkedDocs(doc, function (doc, sharedHist) {
      if (!sharedHist && indexOf(rebased, doc.history) == -1) {
        rebaseHist(doc.history, change);
        rebased.push(doc.history);
      }
      makeChangeSingleDoc(doc, change, null, stretchSpansOverChange(doc, change));
    });
  }

  // Revert a change stored in a document's history.
  function makeChangeFromHistory(doc, type, allowSelectionOnly) {
    var suppress = doc.cm && doc.cm.state.suppressEdits;
    if (suppress && !allowSelectionOnly) { return }

    var hist = doc.history, event, selAfter = doc.sel;
    var source = type == "undo" ? hist.done : hist.undone, dest = type == "undo" ? hist.undone : hist.done;

    // Verify that there is a useable event (so that ctrl-z won't
    // needlessly clear selection events)
    var i = 0;
    for (; i < source.length; i++) {
      event = source[i];
      if (allowSelectionOnly ? event.ranges && !event.equals(doc.sel) : !event.ranges)
        { break }
    }
    if (i == source.length) { return }
    hist.lastOrigin = hist.lastSelOrigin = null;

    for (;;) {
      event = source.pop();
      if (event.ranges) {
        pushSelectionToHistory(event, dest);
        if (allowSelectionOnly && !event.equals(doc.sel)) {
          setSelection(doc, event, {clearRedo: false});
          return
        }
        selAfter = event;
      } else if (suppress) {
        source.push(event);
        return
      } else { break }
    }

    // Build up a reverse change object to add to the opposite history
    // stack (redo when undoing, and vice versa).
    var antiChanges = [];
    pushSelectionToHistory(selAfter, dest);
    dest.push({changes: antiChanges, generation: hist.generation});
    hist.generation = event.generation || ++hist.maxGeneration;

    var filter = hasHandler(doc, "beforeChange") || doc.cm && hasHandler(doc.cm, "beforeChange");

    var loop = function ( i ) {
      var change = event.changes[i];
      change.origin = type;
      if (filter && !filterChange(doc, change, false)) {
        source.length = 0;
        return {}
      }

      antiChanges.push(historyChangeFromChange(doc, change));

      var after = i ? computeSelAfterChange(doc, change) : lst(source);
      makeChangeSingleDoc(doc, change, after, mergeOldSpans(doc, change));
      if (!i && doc.cm) { doc.cm.scrollIntoView({from: change.from, to: changeEnd(change)}); }
      var rebased = [];

      // Propagate to the linked documents
      linkedDocs(doc, function (doc, sharedHist) {
        if (!sharedHist && indexOf(rebased, doc.history) == -1) {
          rebaseHist(doc.history, change);
          rebased.push(doc.history);
        }
        makeChangeSingleDoc(doc, change, null, mergeOldSpans(doc, change));
      });
    };

    for (var i$1 = event.changes.length - 1; i$1 >= 0; --i$1) {
      var returned = loop( i$1 );

      if ( returned ) return returned.v;
    }
  }

  // Sub-views need their line numbers shifted when text is added
  // above or below them in the parent document.
  function shiftDoc(doc, distance) {
    if (distance == 0) { return }
    doc.first += distance;
    doc.sel = new Selection(map(doc.sel.ranges, function (range) { return new Range(
      Pos(range.anchor.line + distance, range.anchor.ch),
      Pos(range.head.line + distance, range.head.ch)
    ); }), doc.sel.primIndex);
    if (doc.cm) {
      regChange(doc.cm, doc.first, doc.first - distance, distance);
      for (var d = doc.cm.display, l = d.viewFrom; l < d.viewTo; l++)
        { regLineChange(doc.cm, l, "gutter"); }
    }
  }

  // More lower-level change function, handling only a single document
  // (not linked ones).
  function makeChangeSingleDoc(doc, change, selAfter, spans) {
    if (doc.cm && !doc.cm.curOp)
      { return operation(doc.cm, makeChangeSingleDoc)(doc, change, selAfter, spans) }

    if (change.to.line < doc.first) {
      shiftDoc(doc, change.text.length - 1 - (change.to.line - change.from.line));
      return
    }
    if (change.from.line > doc.lastLine()) { return }

    // Clip the change to the size of this doc
    if (change.from.line < doc.first) {
      var shift = change.text.length - 1 - (doc.first - change.from.line);
      shiftDoc(doc, shift);
      change = {from: Pos(doc.first, 0), to: Pos(change.to.line + shift, change.to.ch),
                text: [lst(change.text)], origin: change.origin};
    }
    var last = doc.lastLine();
    if (change.to.line > last) {
      change = {from: change.from, to: Pos(last, getLine(doc, last).text.length),
                text: [change.text[0]], origin: change.origin};
    }

    change.removed = getBetween(doc, change.from, change.to);

    if (!selAfter) { selAfter = computeSelAfterChange(doc, change); }
    if (doc.cm) { makeChangeSingleDocInEditor(doc.cm, change, spans); }
    else { updateDoc(doc, change, spans); }
    setSelectionNoUndo(doc, selAfter, sel_dontScroll);
  }

  // Handle the interaction of a change to a document with the editor
  // that this document is part of.
  function makeChangeSingleDocInEditor(cm, change, spans) {
    var doc = cm.doc, display = cm.display, from = change.from, to = change.to;

    var recomputeMaxLength = false, checkWidthStart = from.line;
    if (!cm.options.lineWrapping) {
      checkWidthStart = lineNo(visualLine(getLine(doc, from.line)));
      doc.iter(checkWidthStart, to.line + 1, function (line) {
        if (line == display.maxLine) {
          recomputeMaxLength = true;
          return true
        }
      });
    }

    if (doc.sel.contains(change.from, change.to) > -1)
      { signalCursorActivity(cm); }

    updateDoc(doc, change, spans, estimateHeight(cm));

    if (!cm.options.lineWrapping) {
      doc.iter(checkWidthStart, from.line + change.text.length, function (line) {
        var len = lineLength(line);
        if (len > display.maxLineLength) {
          display.maxLine = line;
          display.maxLineLength = len;
          display.maxLineChanged = true;
          recomputeMaxLength = false;
        }
      });
      if (recomputeMaxLength) { cm.curOp.updateMaxLine = true; }
    }

    retreatFrontier(doc, from.line);
    startWorker(cm, 400);

    var lendiff = change.text.length - (to.line - from.line) - 1;
    // Remember that these lines changed, for updating the display
    if (change.full)
      { regChange(cm); }
    else if (from.line == to.line && change.text.length == 1 && !isWholeLineUpdate(cm.doc, change))
      { regLineChange(cm, from.line, "text"); }
    else
      { regChange(cm, from.line, to.line + 1, lendiff); }

    var changesHandler = hasHandler(cm, "changes"), changeHandler = hasHandler(cm, "change");
    if (changeHandler || changesHandler) {
      var obj = {
        from: from, to: to,
        text: change.text,
        removed: change.removed,
        origin: change.origin
      };
      if (changeHandler) { signalLater(cm, "change", cm, obj); }
      if (changesHandler) { (cm.curOp.changeObjs || (cm.curOp.changeObjs = [])).push(obj); }
    }
    cm.display.selForContextMenu = null;
  }

  function replaceRange(doc, code, from, to, origin) {
    var assign;

    if (!to) { to = from; }
    if (cmp(to, from) < 0) { (assign = [to, from], from = assign[0], to = assign[1]); }
    if (typeof code == "string") { code = doc.splitLines(code); }
    makeChange(doc, {from: from, to: to, text: code, origin: origin});
  }

  // Rebasing/resetting history to deal with externally-sourced changes

  function rebaseHistSelSingle(pos, from, to, diff) {
    if (to < pos.line) {
      pos.line += diff;
    } else if (from < pos.line) {
      pos.line = from;
      pos.ch = 0;
    }
  }

  // Tries to rebase an array of history events given a change in the
  // document. If the change touches the same lines as the event, the
  // event, and everything 'behind' it, is discarded. If the change is
  // before the event, the event's positions are updated. Uses a
  // copy-on-write scheme for the positions, to avoid having to
  // reallocate them all on every rebase, but also avoid problems with
  // shared position objects being unsafely updated.
  function rebaseHistArray(array, from, to, diff) {
    for (var i = 0; i < array.length; ++i) {
      var sub = array[i], ok = true;
      if (sub.ranges) {
        if (!sub.copied) { sub = array[i] = sub.deepCopy(); sub.copied = true; }
        for (var j = 0; j < sub.ranges.length; j++) {
          rebaseHistSelSingle(sub.ranges[j].anchor, from, to, diff);
          rebaseHistSelSingle(sub.ranges[j].head, from, to, diff);
        }
        continue
      }
      for (var j$1 = 0; j$1 < sub.changes.length; ++j$1) {
        var cur = sub.changes[j$1];
        if (to < cur.from.line) {
          cur.from = Pos(cur.from.line + diff, cur.from.ch);
          cur.to = Pos(cur.to.line + diff, cur.to.ch);
        } else if (from <= cur.to.line) {
          ok = false;
          break
        }
      }
      if (!ok) {
        array.splice(0, i + 1);
        i = 0;
      }
    }
  }

  function rebaseHist(hist, change) {
    var from = change.from.line, to = change.to.line, diff = change.text.length - (to - from) - 1;
    rebaseHistArray(hist.done, from, to, diff);
    rebaseHistArray(hist.undone, from, to, diff);
  }

  // Utility for applying a change to a line by handle or number,
  // returning the number and optionally registering the line as
  // changed.
  function changeLine(doc, handle, changeType, op) {
    var no = handle, line = handle;
    if (typeof handle == "number") { line = getLine(doc, clipLine(doc, handle)); }
    else { no = lineNo(handle); }
    if (no == null) { return null }
    if (op(line, no) && doc.cm) { regLineChange(doc.cm, no, changeType); }
    return line
  }

  // The document is represented as a BTree consisting of leaves, with
  // chunk of lines in them, and branches, with up to ten leaves or
  // other branch nodes below them. The top node is always a branch
  // node, and is the document object itself (meaning it has
  // additional methods and properties).
  //
  // All nodes have parent links. The tree is used both to go from
  // line numbers to line objects, and to go from objects to numbers.
  // It also indexes by height, and is used to convert between height
  // and line object, and to find the total height of the document.
  //
  // See also http://marijnhaverbeke.nl/blog/codemirror-line-tree.html

  function LeafChunk(lines) {
    var this$1 = this;

    this.lines = lines;
    this.parent = null;
    var height = 0;
    for (var i = 0; i < lines.length; ++i) {
      lines[i].parent = this$1;
      height += lines[i].height;
    }
    this.height = height;
  }

  LeafChunk.prototype = {
    chunkSize: function() { return this.lines.length },

    // Remove the n lines at offset 'at'.
    removeInner: function(at, n) {
      var this$1 = this;

      for (var i = at, e = at + n; i < e; ++i) {
        var line = this$1.lines[i];
        this$1.height -= line.height;
        cleanUpLine(line);
        signalLater(line, "delete");
      }
      this.lines.splice(at, n);
    },

    // Helper used to collapse a small branch into a single leaf.
    collapse: function(lines) {
      lines.push.apply(lines, this.lines);
    },

    // Insert the given array of lines at offset 'at', count them as
    // having the given height.
    insertInner: function(at, lines, height) {
      var this$1 = this;

      this.height += height;
      this.lines = this.lines.slice(0, at).concat(lines).concat(this.lines.slice(at));
      for (var i = 0; i < lines.length; ++i) { lines[i].parent = this$1; }
    },

    // Used to iterate over a part of the tree.
    iterN: function(at, n, op) {
      var this$1 = this;

      for (var e = at + n; at < e; ++at)
        { if (op(this$1.lines[at])) { return true } }
    }
  };

  function BranchChunk(children) {
    var this$1 = this;

    this.children = children;
    var size = 0, height = 0;
    for (var i = 0; i < children.length; ++i) {
      var ch = children[i];
      size += ch.chunkSize(); height += ch.height;
      ch.parent = this$1;
    }
    this.size = size;
    this.height = height;
    this.parent = null;
  }

  BranchChunk.prototype = {
    chunkSize: function() { return this.size },

    removeInner: function(at, n) {
      var this$1 = this;

      this.size -= n;
      for (var i = 0; i < this.children.length; ++i) {
        var child = this$1.children[i], sz = child.chunkSize();
        if (at < sz) {
          var rm = Math.min(n, sz - at), oldHeight = child.height;
          child.removeInner(at, rm);
          this$1.height -= oldHeight - child.height;
          if (sz == rm) { this$1.children.splice(i--, 1); child.parent = null; }
          if ((n -= rm) == 0) { break }
          at = 0;
        } else { at -= sz; }
      }
      // If the result is smaller than 25 lines, ensure that it is a
      // single leaf node.
      if (this.size - n < 25 &&
          (this.children.length > 1 || !(this.children[0] instanceof LeafChunk))) {
        var lines = [];
        this.collapse(lines);
        this.children = [new LeafChunk(lines)];
        this.children[0].parent = this;
      }
    },

    collapse: function(lines) {
      var this$1 = this;

      for (var i = 0; i < this.children.length; ++i) { this$1.children[i].collapse(lines); }
    },

    insertInner: function(at, lines, height) {
      var this$1 = this;

      this.size += lines.length;
      this.height += height;
      for (var i = 0; i < this.children.length; ++i) {
        var child = this$1.children[i], sz = child.chunkSize();
        if (at <= sz) {
          child.insertInner(at, lines, height);
          if (child.lines && child.lines.length > 50) {
            // To avoid memory thrashing when child.lines is huge (e.g. first view of a large file), it's never spliced.
            // Instead, small slices are taken. They're taken in order because sequential memory accesses are fastest.
            var remaining = child.lines.length % 25 + 25;
            for (var pos = remaining; pos < child.lines.length;) {
              var leaf = new LeafChunk(child.lines.slice(pos, pos += 25));
              child.height -= leaf.height;
              this$1.children.splice(++i, 0, leaf);
              leaf.parent = this$1;
            }
            child.lines = child.lines.slice(0, remaining);
            this$1.maybeSpill();
          }
          break
        }
        at -= sz;
      }
    },

    // When a node has grown, check whether it should be split.
    maybeSpill: function() {
      if (this.children.length <= 10) { return }
      var me = this;
      do {
        var spilled = me.children.splice(me.children.length - 5, 5);
        var sibling = new BranchChunk(spilled);
        if (!me.parent) { // Become the parent node
          var copy = new BranchChunk(me.children);
          copy.parent = me;
          me.children = [copy, sibling];
          me = copy;
       } else {
          me.size -= sibling.size;
          me.height -= sibling.height;
          var myIndex = indexOf(me.parent.children, me);
          me.parent.children.splice(myIndex + 1, 0, sibling);
        }
        sibling.parent = me.parent;
      } while (me.children.length > 10)
      me.parent.maybeSpill();
    },

    iterN: function(at, n, op) {
      var this$1 = this;

      for (var i = 0; i < this.children.length; ++i) {
        var child = this$1.children[i], sz = child.chunkSize();
        if (at < sz) {
          var used = Math.min(n, sz - at);
          if (child.iterN(at, used, op)) { return true }
          if ((n -= used) == 0) { break }
          at = 0;
        } else { at -= sz; }
      }
    }
  };

  // Line widgets are block elements displayed above or below a line.

  var LineWidget = function(doc, node, options) {
    var this$1 = this;

    if (options) { for (var opt in options) { if (options.hasOwnProperty(opt))
      { this$1[opt] = options[opt]; } } }
    this.doc = doc;
    this.node = node;
  };

  LineWidget.prototype.clear = function () {
      var this$1 = this;

    var cm = this.doc.cm, ws = this.line.widgets, line = this.line, no = lineNo(line);
    if (no == null || !ws) { return }
    for (var i = 0; i < ws.length; ++i) { if (ws[i] == this$1) { ws.splice(i--, 1); } }
    if (!ws.length) { line.widgets = null; }
    var height = widgetHeight(this);
    updateLineHeight(line, Math.max(0, line.height - height));
    if (cm) {
      runInOp(cm, function () {
        adjustScrollWhenAboveVisible(cm, line, -height);
        regLineChange(cm, no, "widget");
      });
      signalLater(cm, "lineWidgetCleared", cm, this, no);
    }
  };

  LineWidget.prototype.changed = function () {
      var this$1 = this;

    var oldH = this.height, cm = this.doc.cm, line = this.line;
    this.height = null;
    var diff = widgetHeight(this) - oldH;
    if (!diff) { return }
    if (!lineIsHidden(this.doc, line)) { updateLineHeight(line, line.height + diff); }
    if (cm) {
      runInOp(cm, function () {
        cm.curOp.forceUpdate = true;
        adjustScrollWhenAboveVisible(cm, line, diff);
        signalLater(cm, "lineWidgetChanged", cm, this$1, lineNo(line));
      });
    }
  };
  eventMixin(LineWidget);

  function adjustScrollWhenAboveVisible(cm, line, diff) {
    if (heightAtLine(line) < ((cm.curOp && cm.curOp.scrollTop) || cm.doc.scrollTop))
      { addToScrollTop(cm, diff); }
  }

  function addLineWidget(doc, handle, node, options) {
    var widget = new LineWidget(doc, node, options);
    var cm = doc.cm;
    if (cm && widget.noHScroll) { cm.display.alignWidgets = true; }
    changeLine(doc, handle, "widget", function (line) {
      var widgets = line.widgets || (line.widgets = []);
      if (widget.insertAt == null) { widgets.push(widget); }
      else { widgets.splice(Math.min(widgets.length - 1, Math.max(0, widget.insertAt)), 0, widget); }
      widget.line = line;
      if (cm && !lineIsHidden(doc, line)) {
        var aboveVisible = heightAtLine(line) < doc.scrollTop;
        updateLineHeight(line, line.height + widgetHeight(widget));
        if (aboveVisible) { addToScrollTop(cm, widget.height); }
        cm.curOp.forceUpdate = true;
      }
      return true
    });
    if (cm) { signalLater(cm, "lineWidgetAdded", cm, widget, typeof handle == "number" ? handle : lineNo(handle)); }
    return widget
  }

  // TEXTMARKERS

  // Created with markText and setBookmark methods. A TextMarker is a
  // handle that can be used to clear or find a marked position in the
  // document. Line objects hold arrays (markedSpans) containing
  // {from, to, marker} object pointing to such marker objects, and
  // indicating that such a marker is present on that line. Multiple
  // lines may point to the same marker when it spans across lines.
  // The spans will have null for their from/to properties when the
  // marker continues beyond the start/end of the line. Markers have
  // links back to the lines they currently touch.

  // Collapsed markers have unique ids, in order to be able to order
  // them, which is needed for uniquely determining an outer marker
  // when they overlap (they may nest, but not partially overlap).
  var nextMarkerId = 0;

  var TextMarker = function(doc, type) {
    this.lines = [];
    this.type = type;
    this.doc = doc;
    this.id = ++nextMarkerId;
  };

  // Clear the marker.
  TextMarker.prototype.clear = function () {
      var this$1 = this;

    if (this.explicitlyCleared) { return }
    var cm = this.doc.cm, withOp = cm && !cm.curOp;
    if (withOp) { startOperation(cm); }
    if (hasHandler(this, "clear")) {
      var found = this.find();
      if (found) { signalLater(this, "clear", found.from, found.to); }
    }
    var min = null, max = null;
    for (var i = 0; i < this.lines.length; ++i) {
      var line = this$1.lines[i];
      var span = getMarkedSpanFor(line.markedSpans, this$1);
      if (cm && !this$1.collapsed) { regLineChange(cm, lineNo(line), "text"); }
      else if (cm) {
        if (span.to != null) { max = lineNo(line); }
        if (span.from != null) { min = lineNo(line); }
      }
      line.markedSpans = removeMarkedSpan(line.markedSpans, span);
      if (span.from == null && this$1.collapsed && !lineIsHidden(this$1.doc, line) && cm)
        { updateLineHeight(line, textHeight(cm.display)); }
    }
    if (cm && this.collapsed && !cm.options.lineWrapping) { for (var i$1 = 0; i$1 < this.lines.length; ++i$1) {
      var visual = visualLine(this$1.lines[i$1]), len = lineLength(visual);
      if (len > cm.display.maxLineLength) {
        cm.display.maxLine = visual;
        cm.display.maxLineLength = len;
        cm.display.maxLineChanged = true;
      }
    } }

    if (min != null && cm && this.collapsed) { regChange(cm, min, max + 1); }
    this.lines.length = 0;
    this.explicitlyCleared = true;
    if (this.atomic && this.doc.cantEdit) {
      this.doc.cantEdit = false;
      if (cm) { reCheckSelection(cm.doc); }
    }
    if (cm) { signalLater(cm, "markerCleared", cm, this, min, max); }
    if (withOp) { endOperation(cm); }
    if (this.parent) { this.parent.clear(); }
  };

  // Find the position of the marker in the document. Returns a {from,
  // to} object by default. Side can be passed to get a specific side
  // -- 0 (both), -1 (left), or 1 (right). When lineObj is true, the
  // Pos objects returned contain a line object, rather than a line
  // number (used to prevent looking up the same line twice).
  TextMarker.prototype.find = function (side, lineObj) {
      var this$1 = this;

    if (side == null && this.type == "bookmark") { side = 1; }
    var from, to;
    for (var i = 0; i < this.lines.length; ++i) {
      var line = this$1.lines[i];
      var span = getMarkedSpanFor(line.markedSpans, this$1);
      if (span.from != null) {
        from = Pos(lineObj ? line : lineNo(line), span.from);
        if (side == -1) { return from }
      }
      if (span.to != null) {
        to = Pos(lineObj ? line : lineNo(line), span.to);
        if (side == 1) { return to }
      }
    }
    return from && {from: from, to: to}
  };

  // Signals that the marker's widget changed, and surrounding layout
  // should be recomputed.
  TextMarker.prototype.changed = function () {
      var this$1 = this;

    var pos = this.find(-1, true), widget = this, cm = this.doc.cm;
    if (!pos || !cm) { return }
    runInOp(cm, function () {
      var line = pos.line, lineN = lineNo(pos.line);
      var view = findViewForLine(cm, lineN);
      if (view) {
        clearLineMeasurementCacheFor(view);
        cm.curOp.selectionChanged = cm.curOp.forceUpdate = true;
      }
      cm.curOp.updateMaxLine = true;
      if (!lineIsHidden(widget.doc, line) && widget.height != null) {
        var oldHeight = widget.height;
        widget.height = null;
        var dHeight = widgetHeight(widget) - oldHeight;
        if (dHeight)
          { updateLineHeight(line, line.height + dHeight); }
      }
      signalLater(cm, "markerChanged", cm, this$1);
    });
  };

  TextMarker.prototype.attachLine = function (line) {
    if (!this.lines.length && this.doc.cm) {
      var op = this.doc.cm.curOp;
      if (!op.maybeHiddenMarkers || indexOf(op.maybeHiddenMarkers, this) == -1)
        { (op.maybeUnhiddenMarkers || (op.maybeUnhiddenMarkers = [])).push(this); }
    }
    this.lines.push(line);
  };

  TextMarker.prototype.detachLine = function (line) {
    this.lines.splice(indexOf(this.lines, line), 1);
    if (!this.lines.length && this.doc.cm) {
      var op = this.doc.cm.curOp
      ;(op.maybeHiddenMarkers || (op.maybeHiddenMarkers = [])).push(this);
    }
  };
  eventMixin(TextMarker);

  // Create a marker, wire it up to the right lines, and
  function markText(doc, from, to, options, type) {
    // Shared markers (across linked documents) are handled separately
    // (markTextShared will call out to this again, once per
    // document).
    if (options && options.shared) { return markTextShared(doc, from, to, options, type) }
    // Ensure we are in an operation.
    if (doc.cm && !doc.cm.curOp) { return operation(doc.cm, markText)(doc, from, to, options, type) }

    var marker = new TextMarker(doc, type), diff = cmp(from, to);
    if (options) { copyObj(options, marker, false); }
    // Don't connect empty markers unless clearWhenEmpty is false
    if (diff > 0 || diff == 0 && marker.clearWhenEmpty !== false)
      { return marker }
    if (marker.replacedWith) {
      // Showing up as a widget implies collapsed (widget replaces text)
      marker.collapsed = true;
      marker.widgetNode = eltP("span", [marker.replacedWith], "CodeMirror-widget");
      if (!options.handleMouseEvents) { marker.widgetNode.setAttribute("cm-ignore-events", "true"); }
      if (options.insertLeft) { marker.widgetNode.insertLeft = true; }
    }
    if (marker.collapsed) {
      if (conflictingCollapsedRange(doc, from.line, from, to, marker) ||
          from.line != to.line && conflictingCollapsedRange(doc, to.line, from, to, marker))
        { throw new Error("Inserting collapsed marker partially overlapping an existing one") }
      seeCollapsedSpans();
    }

    if (marker.addToHistory)
      { addChangeToHistory(doc, {from: from, to: to, origin: "markText"}, doc.sel, NaN); }

    var curLine = from.line, cm = doc.cm, updateMaxLine;
    doc.iter(curLine, to.line + 1, function (line) {
      if (cm && marker.collapsed && !cm.options.lineWrapping && visualLine(line) == cm.display.maxLine)
        { updateMaxLine = true; }
      if (marker.collapsed && curLine != from.line) { updateLineHeight(line, 0); }
      addMarkedSpan(line, new MarkedSpan(marker,
                                         curLine == from.line ? from.ch : null,
                                         curLine == to.line ? to.ch : null));
      ++curLine;
    });
    // lineIsHidden depends on the presence of the spans, so needs a second pass
    if (marker.collapsed) { doc.iter(from.line, to.line + 1, function (line) {
      if (lineIsHidden(doc, line)) { updateLineHeight(line, 0); }
    }); }

    if (marker.clearOnEnter) { on(marker, "beforeCursorEnter", function () { return marker.clear(); }); }

    if (marker.readOnly) {
      seeReadOnlySpans();
      if (doc.history.done.length || doc.history.undone.length)
        { doc.clearHistory(); }
    }
    if (marker.collapsed) {
      marker.id = ++nextMarkerId;
      marker.atomic = true;
    }
    if (cm) {
      // Sync editor state
      if (updateMaxLine) { cm.curOp.updateMaxLine = true; }
      if (marker.collapsed)
        { regChange(cm, from.line, to.line + 1); }
      else if (marker.className || marker.startStyle || marker.endStyle || marker.css ||
               marker.attributes || marker.title)
        { for (var i = from.line; i <= to.line; i++) { regLineChange(cm, i, "text"); } }
      if (marker.atomic) { reCheckSelection(cm.doc); }
      signalLater(cm, "markerAdded", cm, marker);
    }
    return marker
  }

  // SHARED TEXTMARKERS

  // A shared marker spans multiple linked documents. It is
  // implemented as a meta-marker-object controlling multiple normal
  // markers.
  var SharedTextMarker = function(markers, primary) {
    var this$1 = this;

    this.markers = markers;
    this.primary = primary;
    for (var i = 0; i < markers.length; ++i)
      { markers[i].parent = this$1; }
  };

  SharedTextMarker.prototype.clear = function () {
      var this$1 = this;

    if (this.explicitlyCleared) { return }
    this.explicitlyCleared = true;
    for (var i = 0; i < this.markers.length; ++i)
      { this$1.markers[i].clear(); }
    signalLater(this, "clear");
  };

  SharedTextMarker.prototype.find = function (side, lineObj) {
    return this.primary.find(side, lineObj)
  };
  eventMixin(SharedTextMarker);

  function markTextShared(doc, from, to, options, type) {
    options = copyObj(options);
    options.shared = false;
    var markers = [markText(doc, from, to, options, type)], primary = markers[0];
    var widget = options.widgetNode;
    linkedDocs(doc, function (doc) {
      if (widget) { options.widgetNode = widget.cloneNode(true); }
      markers.push(markText(doc, clipPos(doc, from), clipPos(doc, to), options, type));
      for (var i = 0; i < doc.linked.length; ++i)
        { if (doc.linked[i].isParent) { return } }
      primary = lst(markers);
    });
    return new SharedTextMarker(markers, primary)
  }

  function findSharedMarkers(doc) {
    return doc.findMarks(Pos(doc.first, 0), doc.clipPos(Pos(doc.lastLine())), function (m) { return m.parent; })
  }

  function copySharedMarkers(doc, markers) {
    for (var i = 0; i < markers.length; i++) {
      var marker = markers[i], pos = marker.find();
      var mFrom = doc.clipPos(pos.from), mTo = doc.clipPos(pos.to);
      if (cmp(mFrom, mTo)) {
        var subMark = markText(doc, mFrom, mTo, marker.primary, marker.primary.type);
        marker.markers.push(subMark);
        subMark.parent = marker;
      }
    }
  }

  function detachSharedMarkers(markers) {
    var loop = function ( i ) {
      var marker = markers[i], linked = [marker.primary.doc];
      linkedDocs(marker.primary.doc, function (d) { return linked.push(d); });
      for (var j = 0; j < marker.markers.length; j++) {
        var subMarker = marker.markers[j];
        if (indexOf(linked, subMarker.doc) == -1) {
          subMarker.parent = null;
          marker.markers.splice(j--, 1);
        }
      }
    };

    for (var i = 0; i < markers.length; i++) loop( i );
  }

  var nextDocId = 0;
  var Doc = function(text, mode, firstLine, lineSep, direction) {
    if (!(this instanceof Doc)) { return new Doc(text, mode, firstLine, lineSep, direction) }
    if (firstLine == null) { firstLine = 0; }

    BranchChunk.call(this, [new LeafChunk([new Line("", null)])]);
    this.first = firstLine;
    this.scrollTop = this.scrollLeft = 0;
    this.cantEdit = false;
    this.cleanGeneration = 1;
    this.modeFrontier = this.highlightFrontier = firstLine;
    var start = Pos(firstLine, 0);
    this.sel = simpleSelection(start);
    this.history = new History(null);
    this.id = ++nextDocId;
    this.modeOption = mode;
    this.lineSep = lineSep;
    this.direction = (direction == "rtl") ? "rtl" : "ltr";
    this.extend = false;

    if (typeof text == "string") { text = this.splitLines(text); }
    updateDoc(this, {from: start, to: start, text: text});
    setSelection(this, simpleSelection(start), sel_dontScroll);
  };

  Doc.prototype = createObj(BranchChunk.prototype, {
    constructor: Doc,
    // Iterate over the document. Supports two forms -- with only one
    // argument, it calls that for each line in the document. With
    // three, it iterates over the range given by the first two (with
    // the second being non-inclusive).
    iter: function(from, to, op) {
      if (op) { this.iterN(from - this.first, to - from, op); }
      else { this.iterN(this.first, this.first + this.size, from); }
    },

    // Non-public interface for adding and removing lines.
    insert: function(at, lines) {
      var height = 0;
      for (var i = 0; i < lines.length; ++i) { height += lines[i].height; }
      this.insertInner(at - this.first, lines, height);
    },
    remove: function(at, n) { this.removeInner(at - this.first, n); },

    // From here, the methods are part of the public interface. Most
    // are also available from CodeMirror (editor) instances.

    getValue: function(lineSep) {
      var lines = getLines(this, this.first, this.first + this.size);
      if (lineSep === false) { return lines }
      return lines.join(lineSep || this.lineSeparator())
    },
    setValue: docMethodOp(function(code) {
      var top = Pos(this.first, 0), last = this.first + this.size - 1;
      makeChange(this, {from: top, to: Pos(last, getLine(this, last).text.length),
                        text: this.splitLines(code), origin: "setValue", full: true}, true);
      if (this.cm) { scrollToCoords(this.cm, 0, 0); }
      setSelection(this, simpleSelection(top), sel_dontScroll);
    }),
    replaceRange: function(code, from, to, origin) {
      from = clipPos(this, from);
      to = to ? clipPos(this, to) : from;
      replaceRange(this, code, from, to, origin);
    },
    getRange: function(from, to, lineSep) {
      var lines = getBetween(this, clipPos(this, from), clipPos(this, to));
      if (lineSep === false) { return lines }
      return lines.join(lineSep || this.lineSeparator())
    },

    getLine: function(line) {var l = this.getLineHandle(line); return l && l.text},

    getLineHandle: function(line) {if (isLine(this, line)) { return getLine(this, line) }},
    getLineNumber: function(line) {return lineNo(line)},

    getLineHandleVisualStart: function(line) {
      if (typeof line == "number") { line = getLine(this, line); }
      return visualLine(line)
    },

    lineCount: function() {return this.size},
    firstLine: function() {return this.first},
    lastLine: function() {return this.first + this.size - 1},

    clipPos: function(pos) {return clipPos(this, pos)},

    getCursor: function(start) {
      var range$$1 = this.sel.primary(), pos;
      if (start == null || start == "head") { pos = range$$1.head; }
      else if (start == "anchor") { pos = range$$1.anchor; }
      else if (start == "end" || start == "to" || start === false) { pos = range$$1.to(); }
      else { pos = range$$1.from(); }
      return pos
    },
    listSelections: function() { return this.sel.ranges },
    somethingSelected: function() {return this.sel.somethingSelected()},

    setCursor: docMethodOp(function(line, ch, options) {
      setSimpleSelection(this, clipPos(this, typeof line == "number" ? Pos(line, ch || 0) : line), null, options);
    }),
    setSelection: docMethodOp(function(anchor, head, options) {
      setSimpleSelection(this, clipPos(this, anchor), clipPos(this, head || anchor), options);
    }),
    extendSelection: docMethodOp(function(head, other, options) {
      extendSelection(this, clipPos(this, head), other && clipPos(this, other), options);
    }),
    extendSelections: docMethodOp(function(heads, options) {
      extendSelections(this, clipPosArray(this, heads), options);
    }),
    extendSelectionsBy: docMethodOp(function(f, options) {
      var heads = map(this.sel.ranges, f);
      extendSelections(this, clipPosArray(this, heads), options);
    }),
    setSelections: docMethodOp(function(ranges, primary, options) {
      var this$1 = this;

      if (!ranges.length) { return }
      var out = [];
      for (var i = 0; i < ranges.length; i++)
        { out[i] = new Range(clipPos(this$1, ranges[i].anchor),
                           clipPos(this$1, ranges[i].head)); }
      if (primary == null) { primary = Math.min(ranges.length - 1, this.sel.primIndex); }
      setSelection(this, normalizeSelection(this.cm, out, primary), options);
    }),
    addSelection: docMethodOp(function(anchor, head, options) {
      var ranges = this.sel.ranges.slice(0);
      ranges.push(new Range(clipPos(this, anchor), clipPos(this, head || anchor)));
      setSelection(this, normalizeSelection(this.cm, ranges, ranges.length - 1), options);
    }),

    getSelection: function(lineSep) {
      var this$1 = this;

      var ranges = this.sel.ranges, lines;
      for (var i = 0; i < ranges.length; i++) {
        var sel = getBetween(this$1, ranges[i].from(), ranges[i].to());
        lines = lines ? lines.concat(sel) : sel;
      }
      if (lineSep === false) { return lines }
      else { return lines.join(lineSep || this.lineSeparator()) }
    },
    getSelections: function(lineSep) {
      var this$1 = this;

      var parts = [], ranges = this.sel.ranges;
      for (var i = 0; i < ranges.length; i++) {
        var sel = getBetween(this$1, ranges[i].from(), ranges[i].to());
        if (lineSep !== false) { sel = sel.join(lineSep || this$1.lineSeparator()); }
        parts[i] = sel;
      }
      return parts
    },
    replaceSelection: function(code, collapse, origin) {
      var dup = [];
      for (var i = 0; i < this.sel.ranges.length; i++)
        { dup[i] = code; }
      this.replaceSelections(dup, collapse, origin || "+input");
    },
    replaceSelections: docMethodOp(function(code, collapse, origin) {
      var this$1 = this;

      var changes = [], sel = this.sel;
      for (var i = 0; i < sel.ranges.length; i++) {
        var range$$1 = sel.ranges[i];
        changes[i] = {from: range$$1.from(), to: range$$1.to(), text: this$1.splitLines(code[i]), origin: origin};
      }
      var newSel = collapse && collapse != "end" && computeReplacedSel(this, changes, collapse);
      for (var i$1 = changes.length - 1; i$1 >= 0; i$1--)
        { makeChange(this$1, changes[i$1]); }
      if (newSel) { setSelectionReplaceHistory(this, newSel); }
      else if (this.cm) { ensureCursorVisible(this.cm); }
    }),
    undo: docMethodOp(function() {makeChangeFromHistory(this, "undo");}),
    redo: docMethodOp(function() {makeChangeFromHistory(this, "redo");}),
    undoSelection: docMethodOp(function() {makeChangeFromHistory(this, "undo", true);}),
    redoSelection: docMethodOp(function() {makeChangeFromHistory(this, "redo", true);}),

    setExtending: function(val) {this.extend = val;},
    getExtending: function() {return this.extend},

    historySize: function() {
      var hist = this.history, done = 0, undone = 0;
      for (var i = 0; i < hist.done.length; i++) { if (!hist.done[i].ranges) { ++done; } }
      for (var i$1 = 0; i$1 < hist.undone.length; i$1++) { if (!hist.undone[i$1].ranges) { ++undone; } }
      return {undo: done, redo: undone}
    },
    clearHistory: function() {this.history = new History(this.history.maxGeneration);},

    markClean: function() {
      this.cleanGeneration = this.changeGeneration(true);
    },
    changeGeneration: function(forceSplit) {
      if (forceSplit)
        { this.history.lastOp = this.history.lastSelOp = this.history.lastOrigin = null; }
      return this.history.generation
    },
    isClean: function (gen) {
      return this.history.generation == (gen || this.cleanGeneration)
    },

    getHistory: function() {
      return {done: copyHistoryArray(this.history.done),
              undone: copyHistoryArray(this.history.undone)}
    },
    setHistory: function(histData) {
      var hist = this.history = new History(this.history.maxGeneration);
      hist.done = copyHistoryArray(histData.done.slice(0), null, true);
      hist.undone = copyHistoryArray(histData.undone.slice(0), null, true);
    },

    setGutterMarker: docMethodOp(function(line, gutterID, value) {
      return changeLine(this, line, "gutter", function (line) {
        var markers = line.gutterMarkers || (line.gutterMarkers = {});
        markers[gutterID] = value;
        if (!value && isEmpty(markers)) { line.gutterMarkers = null; }
        return true
      })
    }),

    clearGutter: docMethodOp(function(gutterID) {
      var this$1 = this;

      this.iter(function (line) {
        if (line.gutterMarkers && line.gutterMarkers[gutterID]) {
          changeLine(this$1, line, "gutter", function () {
            line.gutterMarkers[gutterID] = null;
            if (isEmpty(line.gutterMarkers)) { line.gutterMarkers = null; }
            return true
          });
        }
      });
    }),

    lineInfo: function(line) {
      var n;
      if (typeof line == "number") {
        if (!isLine(this, line)) { return null }
        n = line;
        line = getLine(this, line);
        if (!line) { return null }
      } else {
        n = lineNo(line);
        if (n == null) { return null }
      }
      return {line: n, handle: line, text: line.text, gutterMarkers: line.gutterMarkers,
              textClass: line.textClass, bgClass: line.bgClass, wrapClass: line.wrapClass,
              widgets: line.widgets}
    },

    addLineClass: docMethodOp(function(handle, where, cls) {
      return changeLine(this, handle, where == "gutter" ? "gutter" : "class", function (line) {
        var prop = where == "text" ? "textClass"
                 : where == "background" ? "bgClass"
                 : where == "gutter" ? "gutterClass" : "wrapClass";
        if (!line[prop]) { line[prop] = cls; }
        else if (classTest(cls).test(line[prop])) { return false }
        else { line[prop] += " " + cls; }
        return true
      })
    }),
    removeLineClass: docMethodOp(function(handle, where, cls) {
      return changeLine(this, handle, where == "gutter" ? "gutter" : "class", function (line) {
        var prop = where == "text" ? "textClass"
                 : where == "background" ? "bgClass"
                 : where == "gutter" ? "gutterClass" : "wrapClass";
        var cur = line[prop];
        if (!cur) { return false }
        else if (cls == null) { line[prop] = null; }
        else {
          var found = cur.match(classTest(cls));
          if (!found) { return false }
          var end = found.index + found[0].length;
          line[prop] = cur.slice(0, found.index) + (!found.index || end == cur.length ? "" : " ") + cur.slice(end) || null;
        }
        return true
      })
    }),

    addLineWidget: docMethodOp(function(handle, node, options) {
      return addLineWidget(this, handle, node, options)
    }),
    removeLineWidget: function(widget) { widget.clear(); },

    markText: function(from, to, options) {
      return markText(this, clipPos(this, from), clipPos(this, to), options, options && options.type || "range")
    },
    setBookmark: function(pos, options) {
      var realOpts = {replacedWith: options && (options.nodeType == null ? options.widget : options),
                      insertLeft: options && options.insertLeft,
                      clearWhenEmpty: false, shared: options && options.shared,
                      handleMouseEvents: options && options.handleMouseEvents};
      pos = clipPos(this, pos);
      return markText(this, pos, pos, realOpts, "bookmark")
    },
    findMarksAt: function(pos) {
      pos = clipPos(this, pos);
      var markers = [], spans = getLine(this, pos.line).markedSpans;
      if (spans) { for (var i = 0; i < spans.length; ++i) {
        var span = spans[i];
        if ((span.from == null || span.from <= pos.ch) &&
            (span.to == null || span.to >= pos.ch))
          { markers.push(span.marker.parent || span.marker); }
      } }
      return markers
    },
    findMarks: function(from, to, filter) {
      from = clipPos(this, from); to = clipPos(this, to);
      var found = [], lineNo$$1 = from.line;
      this.iter(from.line, to.line + 1, function (line) {
        var spans = line.markedSpans;
        if (spans) { for (var i = 0; i < spans.length; i++) {
          var span = spans[i];
          if (!(span.to != null && lineNo$$1 == from.line && from.ch >= span.to ||
                span.from == null && lineNo$$1 != from.line ||
                span.from != null && lineNo$$1 == to.line && span.from >= to.ch) &&
              (!filter || filter(span.marker)))
            { found.push(span.marker.parent || span.marker); }
        } }
        ++lineNo$$1;
      });
      return found
    },
    getAllMarks: function() {
      var markers = [];
      this.iter(function (line) {
        var sps = line.markedSpans;
        if (sps) { for (var i = 0; i < sps.length; ++i)
          { if (sps[i].from != null) { markers.push(sps[i].marker); } } }
      });
      return markers
    },

    posFromIndex: function(off) {
      var ch, lineNo$$1 = this.first, sepSize = this.lineSeparator().length;
      this.iter(function (line) {
        var sz = line.text.length + sepSize;
        if (sz > off) { ch = off; return true }
        off -= sz;
        ++lineNo$$1;
      });
      return clipPos(this, Pos(lineNo$$1, ch))
    },
    indexFromPos: function (coords) {
      coords = clipPos(this, coords);
      var index = coords.ch;
      if (coords.line < this.first || coords.ch < 0) { return 0 }
      var sepSize = this.lineSeparator().length;
      this.iter(this.first, coords.line, function (line) { // iter aborts when callback returns a truthy value
        index += line.text.length + sepSize;
      });
      return index
    },

    copy: function(copyHistory) {
      var doc = new Doc(getLines(this, this.first, this.first + this.size),
                        this.modeOption, this.first, this.lineSep, this.direction);
      doc.scrollTop = this.scrollTop; doc.scrollLeft = this.scrollLeft;
      doc.sel = this.sel;
      doc.extend = false;
      if (copyHistory) {
        doc.history.undoDepth = this.history.undoDepth;
        doc.setHistory(this.getHistory());
      }
      return doc
    },

    linkedDoc: function(options) {
      if (!options) { options = {}; }
      var from = this.first, to = this.first + this.size;
      if (options.from != null && options.from > from) { from = options.from; }
      if (options.to != null && options.to < to) { to = options.to; }
      var copy = new Doc(getLines(this, from, to), options.mode || this.modeOption, from, this.lineSep, this.direction);
      if (options.sharedHist) { copy.history = this.history
      ; }(this.linked || (this.linked = [])).push({doc: copy, sharedHist: options.sharedHist});
      copy.linked = [{doc: this, isParent: true, sharedHist: options.sharedHist}];
      copySharedMarkers(copy, findSharedMarkers(this));
      return copy
    },
    unlinkDoc: function(other) {
      var this$1 = this;

      if (other instanceof CodeMirror) { other = other.doc; }
      if (this.linked) { for (var i = 0; i < this.linked.length; ++i) {
        var link = this$1.linked[i];
        if (link.doc != other) { continue }
        this$1.linked.splice(i, 1);
        other.unlinkDoc(this$1);
        detachSharedMarkers(findSharedMarkers(this$1));
        break
      } }
      // If the histories were shared, split them again
      if (other.history == this.history) {
        var splitIds = [other.id];
        linkedDocs(other, function (doc) { return splitIds.push(doc.id); }, true);
        other.history = new History(null);
        other.history.done = copyHistoryArray(this.history.done, splitIds);
        other.history.undone = copyHistoryArray(this.history.undone, splitIds);
      }
    },
    iterLinkedDocs: function(f) {linkedDocs(this, f);},

    getMode: function() {return this.mode},
    getEditor: function() {return this.cm},

    splitLines: function(str) {
      if (this.lineSep) { return str.split(this.lineSep) }
      return splitLinesAuto(str)
    },
    lineSeparator: function() { return this.lineSep || "\n" },

    setDirection: docMethodOp(function (dir) {
      if (dir != "rtl") { dir = "ltr"; }
      if (dir == this.direction) { return }
      this.direction = dir;
      this.iter(function (line) { return line.order = null; });
      if (this.cm) { directionChanged(this.cm); }
    })
  });

  // Public alias.
  Doc.prototype.eachLine = Doc.prototype.iter;

  // Kludge to work around strange IE behavior where it'll sometimes
  // re-fire a series of drag-related events right after the drop (#1551)
  var lastDrop = 0;

  function onDrop(e) {
    var cm = this;
    clearDragCursor(cm);
    if (signalDOMEvent(cm, e) || eventInWidget(cm.display, e))
      { return }
    e_preventDefault(e);
    if (ie) { lastDrop = +new Date; }
    var pos = posFromMouse(cm, e, true), files = e.dataTransfer.files;
    if (!pos || cm.isReadOnly()) { return }
    // Might be a file drop, in which case we simply extract the text
    // and insert it.
    if (files && files.length && window.FileReader && window.File) {
      var n = files.length, text = Array(n), read = 0;
      var loadFile = function (file, i) {
        if (cm.options.allowDropFileTypes &&
            indexOf(cm.options.allowDropFileTypes, file.type) == -1)
          { return }

        var reader = new FileReader;
        reader.onload = operation(cm, function () {
          var content = reader.result;
          if (/[\x00-\x08\x0e-\x1f]{2}/.test(content)) { content = ""; }
          text[i] = content;
          if (++read == n) {
            pos = clipPos(cm.doc, pos);
            var change = {from: pos, to: pos,
                          text: cm.doc.splitLines(text.join(cm.doc.lineSeparator())),
                          origin: "paste"};
            makeChange(cm.doc, change);
            setSelectionReplaceHistory(cm.doc, simpleSelection(pos, changeEnd(change)));
          }
        });
        reader.readAsText(file);
      };
      for (var i = 0; i < n; ++i) { loadFile(files[i], i); }
    } else { // Normal drop
      // Don't do a replace if the drop happened inside of the selected text.
      if (cm.state.draggingText && cm.doc.sel.contains(pos) > -1) {
        cm.state.draggingText(e);
        // Ensure the editor is re-focused
        setTimeout(function () { return cm.display.input.focus(); }, 20);
        return
      }
      try {
        var text$1 = e.dataTransfer.getData("Text");
        if (text$1) {
          var selected;
          if (cm.state.draggingText && !cm.state.draggingText.copy)
            { selected = cm.listSelections(); }
          setSelectionNoUndo(cm.doc, simpleSelection(pos, pos));
          if (selected) { for (var i$1 = 0; i$1 < selected.length; ++i$1)
            { replaceRange(cm.doc, "", selected[i$1].anchor, selected[i$1].head, "drag"); } }
          cm.replaceSelection(text$1, "around", "paste");
          cm.display.input.focus();
        }
      }
      catch(e){}
    }
  }

  function onDragStart(cm, e) {
    if (ie && (!cm.state.draggingText || +new Date - lastDrop < 100)) { e_stop(e); return }
    if (signalDOMEvent(cm, e) || eventInWidget(cm.display, e)) { return }

    e.dataTransfer.setData("Text", cm.getSelection());
    e.dataTransfer.effectAllowed = "copyMove";

    // Use dummy image instead of default browsers image.
    // Recent Safari (~6.0.2) have a tendency to segfault when this happens, so we don't do it there.
    if (e.dataTransfer.setDragImage && !safari) {
      var img = elt("img", null, null, "position: fixed; left: 0; top: 0;");
      img.src = "data:image/gif;base64,R0lGODlhAQABAAAAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw==";
      if (presto) {
        img.width = img.height = 1;
        cm.display.wrapper.appendChild(img);
        // Force a relayout, or Opera won't use our image for some obscure reason
        img._top = img.offsetTop;
      }
      e.dataTransfer.setDragImage(img, 0, 0);
      if (presto) { img.parentNode.removeChild(img); }
    }
  }

  function onDragOver(cm, e) {
    var pos = posFromMouse(cm, e);
    if (!pos) { return }
    var frag = document.createDocumentFragment();
    drawSelectionCursor(cm, pos, frag);
    if (!cm.display.dragCursor) {
      cm.display.dragCursor = elt("div", null, "CodeMirror-cursors CodeMirror-dragcursors");
      cm.display.lineSpace.insertBefore(cm.display.dragCursor, cm.display.cursorDiv);
    }
    removeChildrenAndAdd(cm.display.dragCursor, frag);
  }

  function clearDragCursor(cm) {
    if (cm.display.dragCursor) {
      cm.display.lineSpace.removeChild(cm.display.dragCursor);
      cm.display.dragCursor = null;
    }
  }

  // These must be handled carefully, because naively registering a
  // handler for each editor will cause the editors to never be
  // garbage collected.

  function forEachCodeMirror(f) {
    if (!document.getElementsByClassName) { return }
    var byClass = document.getElementsByClassName("CodeMirror"), editors = [];
    for (var i = 0; i < byClass.length; i++) {
      var cm = byClass[i].CodeMirror;
      if (cm) { editors.push(cm); }
    }
    if (editors.length) { editors[0].operation(function () {
      for (var i = 0; i < editors.length; i++) { f(editors[i]); }
    }); }
  }

  var globalsRegistered = false;
  function ensureGlobalHandlers() {
    if (globalsRegistered) { return }
    registerGlobalHandlers();
    globalsRegistered = true;
  }
  function registerGlobalHandlers() {
    // When the window resizes, we need to refresh active editors.
    var resizeTimer;
    on(window, "resize", function () {
      if (resizeTimer == null) { resizeTimer = setTimeout(function () {
        resizeTimer = null;
        forEachCodeMirror(onResize);
      }, 100); }
    });
    // When the window loses focus, we want to show the editor as blurred
    on(window, "blur", function () { return forEachCodeMirror(onBlur); });
  }
  // Called when the window resizes
  function onResize(cm) {
    var d = cm.display;
    // Might be a text scaling operation, clear size caches.
    d.cachedCharWidth = d.cachedTextHeight = d.cachedPaddingH = null;
    d.scrollbarsClipped = false;
    cm.setSize();
  }

  var keyNames = {
    3: "Pause", 8: "Backspace", 9: "Tab", 13: "Enter", 16: "Shift", 17: "Ctrl", 18: "Alt",
    19: "Pause", 20: "CapsLock", 27: "Esc", 32: "Space", 33: "PageUp", 34: "PageDown", 35: "End",
    36: "Home", 37: "Left", 38: "Up", 39: "Right", 40: "Down", 44: "PrintScrn", 45: "Insert",
    46: "Delete", 59: ";", 61: "=", 91: "Mod", 92: "Mod", 93: "Mod",
    106: "*", 107: "=", 109: "-", 110: ".", 111: "/", 145: "ScrollLock",
    173: "-", 186: ";", 187: "=", 188: ",", 189: "-", 190: ".", 191: "/", 192: "` + "`" + `", 219: "[", 220: "\\",
	221: "]", 222: "'", 63232: "Up", 63233: "Down", 63234: "Left", 63235: "Right", 63272: "Delete",
		63273: "Home", 63275: "End", 63276: "PageUp", 63277: "PageDown", 63302: "Insert"
};

// Number keys
for (var i = 0; i < 10; i++) { keyNames[i + 48] = keyNames[i + 96] = String(i); }
// Alphabetic keys
for (var i$1 = 65; i$1 <= 90; i$1++) { keyNames[i$1] = String.fromCharCode(i$1); }
// Function keys
for (var i$2 = 1; i$2 <= 12; i$2++) { keyNames[i$2 + 111] = keyNames[i$2 + 63235] = "F" + i$2; }

var keyMap = {};

keyMap.basic = {
"Left": "goCharLeft", "Right": "goCharRight", "Up": "goLineUp", "Down": "goLineDown",
"End": "goLineEnd", "Home": "goLineStartSmart", "PageUp": "goPageUp", "PageDown": "goPageDown",
"Delete": "delCharAfter", "Backspace": "delCharBefore", "Shift-Backspace": "delCharBefore",
"Tab": "defaultTab", "Shift-Tab": "indentAuto",
"Enter": "newlineAndIndent", "Insert": "toggleOverwrite",
"Esc": "singleSelection"
};
// Note that the save and find-related commands aren't defined by
// default. User code or addons can define them. Unknown commands
// are simply ignored.
keyMap.pcDefault = {
"Ctrl-A": "selectAll", "Ctrl-D": "deleteLine", "Ctrl-Z": "undo", "Shift-Ctrl-Z": "redo", "Ctrl-Y": "redo",
"Ctrl-Home": "goDocStart", "Ctrl-End": "goDocEnd", "Ctrl-Up": "goLineUp", "Ctrl-Down": "goLineDown",
"Ctrl-Left": "goGroupLeft", "Ctrl-Right": "goGroupRight", "Alt-Left": "goLineStart", "Alt-Right": "goLineEnd",
"Ctrl-Backspace": "delGroupBefore", "Ctrl-Delete": "delGroupAfter", "Ctrl-S": "save", "Ctrl-F": "find",
"Ctrl-G": "findNext", "Shift-Ctrl-G": "findPrev", "Shift-Ctrl-F": "replace", "Shift-Ctrl-R": "replaceAll",
"Ctrl-[": "indentLess", "Ctrl-]": "indentMore",
"Ctrl-U": "undoSelection", "Shift-Ctrl-U": "redoSelection", "Alt-U": "redoSelection",
"fallthrough": "basic"
};
// Very basic readline/emacs-style bindings, which are standard on Mac.
keyMap.emacsy = {
"Ctrl-F": "goCharRight", "Ctrl-B": "goCharLeft", "Ctrl-P": "goLineUp", "Ctrl-N": "goLineDown",
"Alt-F": "goWordRight", "Alt-B": "goWordLeft", "Ctrl-A": "goLineStart", "Ctrl-E": "goLineEnd",
"Ctrl-V": "goPageDown", "Shift-Ctrl-V": "goPageUp", "Ctrl-D": "delCharAfter", "Ctrl-H": "delCharBefore",
"Alt-D": "delWordAfter", "Alt-Backspace": "delWordBefore", "Ctrl-K": "killLine", "Ctrl-T": "transposeChars",
"Ctrl-O": "openLine"
};
keyMap.macDefault = {
"Cmd-A": "selectAll", "Cmd-D": "deleteLine", "Cmd-Z": "undo", "Shift-Cmd-Z": "redo", "Cmd-Y": "redo",
"Cmd-Home": "goDocStart", "Cmd-Up": "goDocStart", "Cmd-End": "goDocEnd", "Cmd-Down": "goDocEnd", "Alt-Left": "goGroupLeft",
"Alt-Right": "goGroupRight", "Cmd-Left": "goLineLeft", "Cmd-Right": "goLineRight", "Alt-Backspace": "delGroupBefore",
"Ctrl-Alt-Backspace": "delGroupAfter", "Alt-Delete": "delGroupAfter", "Cmd-S": "save", "Cmd-F": "find",
"Cmd-G": "findNext", "Shift-Cmd-G": "findPrev", "Cmd-Alt-F": "replace", "Shift-Cmd-Alt-F": "replaceAll",
"Cmd-[": "indentLess", "Cmd-]": "indentMore", "Cmd-Backspace": "delWrappedLineLeft", "Cmd-Delete": "delWrappedLineRight",
"Cmd-U": "undoSelection", "Shift-Cmd-U": "redoSelection", "Ctrl-Up": "goDocStart", "Ctrl-Down": "goDocEnd",
"fallthrough": ["basic", "emacsy"]
};
keyMap["default"] = mac ? keyMap.macDefault : keyMap.pcDefault;

// KEYMAP DISPATCH

function normalizeKeyName(name) {
var parts = name.split(/-(?!$)/);
name = parts[parts.length - 1];
var alt, ctrl, shift, cmd;
for (var i = 0; i < parts.length - 1; i++) {
var mod = parts[i];
if (/^(cmd|meta|m)$/i.test(mod)) { cmd = true; }
else if (/^a(lt)?$/i.test(mod)) { alt = true; }
else if (/^(c|ctrl|control)$/i.test(mod)) { ctrl = true; }
else if (/^s(hift)?$/i.test(mod)) { shift = true; }
else { throw new Error("Unrecognized modifier name: " + mod) }
}
if (alt) { name = "Alt-" + name; }
if (ctrl) { name = "Ctrl-" + name; }
if (cmd) { name = "Cmd-" + name; }
if (shift) { name = "Shift-" + name; }
return name
}

// This is a kludge to keep keymaps mostly working as raw objects
// (backwards compatibility) while at the same time support features
// like normalization and multi-stroke key bindings. It compiles a
// new normalized keymap, and then updates the old object to reflect
// this.
function normalizeKeyMap(keymap) {
var copy = {};
for (var keyname in keymap) { if (keymap.hasOwnProperty(keyname)) {
var value = keymap[keyname];
if (/^(name|fallthrough|(de|at)tach)$/.test(keyname)) { continue }
if (value == "...") { delete keymap[keyname]; continue }

var keys = map(keyname.split(" "), normalizeKeyName);
for (var i = 0; i < keys.length; i++) {
var val = (void 0), name = (void 0);
if (i == keys.length - 1) {
name = keys.join(" ");
val = value;
} else {
name = keys.slice(0, i + 1).join(" ");
val = "...";
}
var prev = copy[name];
if (!prev) { copy[name] = val; }
else if (prev != val) { throw new Error("Inconsistent bindings for " + name) }
}
delete keymap[keyname];
} }
for (var prop in copy) { keymap[prop] = copy[prop]; }
return keymap
}

function lookupKey(key, map$$1, handle, context) {
map$$1 = getKeyMap(map$$1);
var found = map$$1.call ? map$$1.call(key, context) : map$$1[key];
if (found === false) { return "nothing" }
if (found === "...") { return "multi" }
if (found != null && handle(found)) { return "handled" }

if (map$$1.fallthrough) {
if (Object.prototype.toString.call(map$$1.fallthrough) != "[object Array]")
{ return lookupKey(key, map$$1.fallthrough, handle, context) }
for (var i = 0; i < map$$1.fallthrough.length; i++) {
var result = lookupKey(key, map$$1.fallthrough[i], handle, context);
if (result) { return result }
}
}
}

// Modifier key presses don't count as 'real' key presses for the
// purpose of keymap fallthrough.
function isModifierKey(value) {
var name = typeof value == "string" ? value : keyNames[value.keyCode];
return name == "Ctrl" || name == "Alt" || name == "Shift" || name == "Mod"
}

function addModifierNames(name, event, noShift) {
var base = name;
if (event.altKey && base != "Alt") { name = "Alt-" + name; }
if ((flipCtrlCmd ? event.metaKey : event.ctrlKey) && base != "Ctrl") { name = "Ctrl-" + name; }
if ((flipCtrlCmd ? event.ctrlKey : event.metaKey) && base != "Cmd") { name = "Cmd-" + name; }
if (!noShift && event.shiftKey && base != "Shift") { name = "Shift-" + name; }
return name
}

// Look up the name of a key as indicated by an event object.
function keyName(event, noShift) {
if (presto && event.keyCode == 34 && event["char"]) { return false }
var name = keyNames[event.keyCode];
if (name == null || event.altGraphKey) { return false }
// Ctrl-ScrollLock has keyCode 3, same as Ctrl-Pause,
// so we'll use event.code when available (Chrome 48+, FF 38+, Safari 10.1+)
if (event.keyCode == 3 && event.code) { name = event.code; }
return addModifierNames(name, event, noShift)
}

function getKeyMap(val) {
return typeof val == "string" ? keyMap[val] : val
}

// Helper for deleting text near the selection(s), used to implement
// backspace, delete, and similar functionality.
function deleteNearSelection(cm, compute) {
var ranges = cm.doc.sel.ranges, kill = [];
// Build up a set of ranges to kill first, merging overlapping
// ranges.
for (var i = 0; i < ranges.length; i++) {
var toKill = compute(ranges[i]);
while (kill.length && cmp(toKill.from, lst(kill).to) <= 0) {
var replaced = kill.pop();
if (cmp(replaced.from, toKill.from) < 0) {
toKill.from = replaced.from;
break
}
}
kill.push(toKill);
}
// Next, remove those actual ranges.
runInOp(cm, function () {
for (var i = kill.length - 1; i >= 0; i--)
{ replaceRange(cm.doc, "", kill[i].from, kill[i].to, "+delete"); }
ensureCursorVisible(cm);
});
}

function moveCharLogically(line, ch, dir) {
var target = skipExtendingChars(line.text, ch + dir, dir);
return target < 0 || target > line.text.length ? null : target
}

function moveLogically(line, start, dir) {
var ch = moveCharLogically(line, start.ch, dir);
return ch == null ? null : new Pos(start.line, ch, dir < 0 ? "after" : "before")
}

function endOfLine(visually, cm, lineObj, lineNo, dir) {
if (visually) {
var order = getOrder(lineObj, cm.doc.direction);
if (order) {
var part = dir < 0 ? lst(order) : order[0];
var moveInStorageOrder = (dir < 0) == (part.level == 1);
var sticky = moveInStorageOrder ? "after" : "before";
var ch;
// With a wrapped rtl chunk (possibly spanning multiple bidi parts),
// it could be that the last bidi part is not on the last visual line,
// since visual lines contain content order-consecutive chunks.
// Thus, in rtl, we are looking for the first (content-order) character
// in the rtl chunk that is on the last line (that is, the same line
// as the last (content-order) character).
if (part.level > 0 || cm.doc.direction == "rtl") {
var prep = prepareMeasureForLine(cm, lineObj);
ch = dir < 0 ? lineObj.text.length - 1 : 0;
var targetTop = measureCharPrepared(cm, prep, ch).top;
ch = findFirst(function (ch) { return measureCharPrepared(cm, prep, ch).top == targetTop; }, (dir < 0) == (part.level == 1) ? part.from : part.to - 1, ch);
if (sticky == "before") { ch = moveCharLogically(lineObj, ch, 1); }
} else { ch = dir < 0 ? part.to : part.from; }
return new Pos(lineNo, ch, sticky)
}
}
return new Pos(lineNo, dir < 0 ? lineObj.text.length : 0, dir < 0 ? "before" : "after")
}

function moveVisually(cm, line, start, dir) {
var bidi = getOrder(line, cm.doc.direction);
if (!bidi) { return moveLogically(line, start, dir) }
if (start.ch >= line.text.length) {
start.ch = line.text.length;
start.sticky = "before";
} else if (start.ch <= 0) {
start.ch = 0;
start.sticky = "after";
}
var partPos = getBidiPartAt(bidi, start.ch, start.sticky), part = bidi[partPos];
if (cm.doc.direction == "ltr" && part.level % 2 == 0 && (dir > 0 ? part.to > start.ch : part.from < start.ch)) {
// Case 1: We move within an ltr part in an ltr editor. Even with wrapped lines,
// nothing interesting happens.
return moveLogically(line, start, dir)
}

var mv = function (pos, dir) { return moveCharLogically(line, pos instanceof Pos ? pos.ch : pos, dir); };
var prep;
var getWrappedLineExtent = function (ch) {
if (!cm.options.lineWrapping) { return {begin: 0, end: line.text.length} }
prep = prep || prepareMeasureForLine(cm, line);
return wrappedLineExtentChar(cm, line, prep, ch)
};
var wrappedLineExtent = getWrappedLineExtent(start.sticky == "before" ? mv(start, -1) : start.ch);

if (cm.doc.direction == "rtl" || part.level == 1) {
var moveInStorageOrder = (part.level == 1) == (dir < 0);
var ch = mv(start, moveInStorageOrder ? 1 : -1);
if (ch != null && (!moveInStorageOrder ? ch >= part.from && ch >= wrappedLineExtent.begin : ch <= part.to && ch <= wrappedLineExtent.end)) {
// Case 2: We move within an rtl part or in an rtl editor on the same visual line
var sticky = moveInStorageOrder ? "before" : "after";
return new Pos(start.line, ch, sticky)
}
}

// Case 3: Could not move within this bidi part in this visual line, so leave
// the current bidi part

var searchInVisualLine = function (partPos, dir, wrappedLineExtent) {
var getRes = function (ch, moveInStorageOrder) { return moveInStorageOrder
? new Pos(start.line, mv(ch, 1), "before")
: new Pos(start.line, ch, "after"); };

for (; partPos >= 0 && partPos < bidi.length; partPos += dir) {
var part = bidi[partPos];
var moveInStorageOrder = (dir > 0) == (part.level != 1);
var ch = moveInStorageOrder ? wrappedLineExtent.begin : mv(wrappedLineExtent.end, -1);
if (part.from <= ch && ch < part.to) { return getRes(ch, moveInStorageOrder) }
ch = moveInStorageOrder ? part.from : mv(part.to, -1);
if (wrappedLineExtent.begin <= ch && ch < wrappedLineExtent.end) { return getRes(ch, moveInStorageOrder) }
}
};

// Case 3a: Look for other bidi parts on the same visual line
var res = searchInVisualLine(partPos + dir, dir, wrappedLineExtent);
if (res) { return res }

// Case 3b: Look for other bidi parts on the next visual line
var nextCh = dir > 0 ? wrappedLineExtent.end : mv(wrappedLineExtent.begin, -1);
if (nextCh != null && !(dir > 0 && nextCh == line.text.length)) {
res = searchInVisualLine(dir > 0 ? 0 : bidi.length - 1, dir, getWrappedLineExtent(nextCh));
if (res) { return res }
}

// Case 4: Nowhere to move
return null
}

// Commands are parameter-less actions that can be performed on an
// editor, mostly used for keybindings.
var commands = {
selectAll: selectAll,
singleSelection: function (cm) { return cm.setSelection(cm.getCursor("anchor"), cm.getCursor("head"), sel_dontScroll); },
killLine: function (cm) { return deleteNearSelection(cm, function (range) {
if (range.empty()) {
var len = getLine(cm.doc, range.head.line).text.length;
if (range.head.ch == len && range.head.line < cm.lastLine())
{ return {from: range.head, to: Pos(range.head.line + 1, 0)} }
else
{ return {from: range.head, to: Pos(range.head.line, len)} }
} else {
return {from: range.from(), to: range.to()}
}
}); },
deleteLine: function (cm) { return deleteNearSelection(cm, function (range) { return ({
from: Pos(range.from().line, 0),
to: clipPos(cm.doc, Pos(range.to().line + 1, 0))
}); }); },
delLineLeft: function (cm) { return deleteNearSelection(cm, function (range) { return ({
from: Pos(range.from().line, 0), to: range.from()
}); }); },
delWrappedLineLeft: function (cm) { return deleteNearSelection(cm, function (range) {
var top = cm.charCoords(range.head, "div").top + 5;
var leftPos = cm.coordsChar({left: 0, top: top}, "div");
return {from: leftPos, to: range.from()}
}); },
delWrappedLineRight: function (cm) { return deleteNearSelection(cm, function (range) {
var top = cm.charCoords(range.head, "div").top + 5;
var rightPos = cm.coordsChar({left: cm.display.lineDiv.offsetWidth + 100, top: top}, "div");
return {from: range.from(), to: rightPos }
}); },
undo: function (cm) { return cm.undo(); },
redo: function (cm) { return cm.redo(); },
undoSelection: function (cm) { return cm.undoSelection(); },
redoSelection: function (cm) { return cm.redoSelection(); },
goDocStart: function (cm) { return cm.extendSelection(Pos(cm.firstLine(), 0)); },
goDocEnd: function (cm) { return cm.extendSelection(Pos(cm.lastLine())); },
goLineStart: function (cm) { return cm.extendSelectionsBy(function (range) { return lineStart(cm, range.head.line); },
{origin: "+move", bias: 1}
); },
goLineStartSmart: function (cm) { return cm.extendSelectionsBy(function (range) { return lineStartSmart(cm, range.head); },
{origin: "+move", bias: 1}
); },
goLineEnd: function (cm) { return cm.extendSelectionsBy(function (range) { return lineEnd(cm, range.head.line); },
{origin: "+move", bias: -1}
); },
goLineRight: function (cm) { return cm.extendSelectionsBy(function (range) {
var top = cm.cursorCoords(range.head, "div").top + 5;
return cm.coordsChar({left: cm.display.lineDiv.offsetWidth + 100, top: top}, "div")
}, sel_move); },
goLineLeft: function (cm) { return cm.extendSelectionsBy(function (range) {
var top = cm.cursorCoords(range.head, "div").top + 5;
return cm.coordsChar({left: 0, top: top}, "div")
}, sel_move); },
goLineLeftSmart: function (cm) { return cm.extendSelectionsBy(function (range) {
var top = cm.cursorCoords(range.head, "div").top + 5;
var pos = cm.coordsChar({left: 0, top: top}, "div");
if (pos.ch < cm.getLine(pos.line).search(/\S/)) { return lineStartSmart(cm, range.head) }
return pos
}, sel_move); },
goLineUp: function (cm) { return cm.moveV(-1, "line"); },
goLineDown: function (cm) { return cm.moveV(1, "line"); },
goPageUp: function (cm) { return cm.moveV(-1, "page"); },
goPageDown: function (cm) { return cm.moveV(1, "page"); },
goCharLeft: function (cm) { return cm.moveH(-1, "char"); },
goCharRight: function (cm) { return cm.moveH(1, "char"); },
goColumnLeft: function (cm) { return cm.moveH(-1, "column"); },
goColumnRight: function (cm) { return cm.moveH(1, "column"); },
goWordLeft: function (cm) { return cm.moveH(-1, "word"); },
goGroupRight: function (cm) { return cm.moveH(1, "group"); },
goGroupLeft: function (cm) { return cm.moveH(-1, "group"); },
goWordRight: function (cm) { return cm.moveH(1, "word"); },
delCharBefore: function (cm) { return cm.deleteH(-1, "char"); },
delCharAfter: function (cm) { return cm.deleteH(1, "char"); },
delWordBefore: function (cm) { return cm.deleteH(-1, "word"); },
delWordAfter: function (cm) { return cm.deleteH(1, "word"); },
delGroupBefore: function (cm) { return cm.deleteH(-1, "group"); },
delGroupAfter: function (cm) { return cm.deleteH(1, "group"); },
indentAuto: function (cm) { return cm.indentSelection("smart"); },
indentMore: function (cm) { return cm.indentSelection("add"); },
indentLess: function (cm) { return cm.indentSelection("subtract"); },
insertTab: function (cm) { return cm.replaceSelection("\t"); },
insertSoftTab: function (cm) {
var spaces = [], ranges = cm.listSelections(), tabSize = cm.options.tabSize;
for (var i = 0; i < ranges.length; i++) {
var pos = ranges[i].from();
var col = countColumn(cm.getLine(pos.line), pos.ch, tabSize);
spaces.push(spaceStr(tabSize - col % tabSize));
}
cm.replaceSelections(spaces);
},
defaultTab: function (cm) {
if (cm.somethingSelected()) { cm.indentSelection("add"); }
else { cm.execCommand("insertTab"); }
},
// Swap the two chars left and right of each selection's head.
// Move cursor behind the two swapped characters afterwards.
//
// Doesn't consider line feeds a character.
// Doesn't scan more than one line above to find a character.
// Doesn't do anything on an empty line.
// Doesn't do anything with non-empty selections.
transposeChars: function (cm) { return runInOp(cm, function () {
var ranges = cm.listSelections(), newSel = [];
for (var i = 0; i < ranges.length; i++) {
if (!ranges[i].empty()) { continue }
var cur = ranges[i].head, line = getLine(cm.doc, cur.line).text;
if (line) {
if (cur.ch == line.length) { cur = new Pos(cur.line, cur.ch - 1); }
if (cur.ch > 0) {
cur = new Pos(cur.line, cur.ch + 1);
cm.replaceRange(line.charAt(cur.ch - 1) + line.charAt(cur.ch - 2),
Pos(cur.line, cur.ch - 2), cur, "+transpose");
} else if (cur.line > cm.doc.first) {
var prev = getLine(cm.doc, cur.line - 1).text;
if (prev) {
cur = new Pos(cur.line, 1);
cm.replaceRange(line.charAt(0) + cm.doc.lineSeparator() +
prev.charAt(prev.length - 1),
Pos(cur.line - 1, prev.length - 1), cur, "+transpose");
}
}
}
newSel.push(new Range(cur, cur));
}
cm.setSelections(newSel);
}); },
newlineAndIndent: function (cm) { return runInOp(cm, function () {
var sels = cm.listSelections();
for (var i = sels.length - 1; i >= 0; i--)
{ cm.replaceRange(cm.doc.lineSeparator(), sels[i].anchor, sels[i].head, "+input"); }
sels = cm.listSelections();
for (var i$1 = 0; i$1 < sels.length; i$1++)
{ cm.indentLine(sels[i$1].from().line, null, true); }
ensureCursorVisible(cm);
}); },
openLine: function (cm) { return cm.replaceSelection("\n", "start"); },
toggleOverwrite: function (cm) { return cm.toggleOverwrite(); }
};


function lineStart(cm, lineN) {
var line = getLine(cm.doc, lineN);
var visual = visualLine(line);
if (visual != line) { lineN = lineNo(visual); }
return endOfLine(true, cm, visual, lineN, 1)
}
function lineEnd(cm, lineN) {
var line = getLine(cm.doc, lineN);
var visual = visualLineEnd(line);
if (visual != line) { lineN = lineNo(visual); }
return endOfLine(true, cm, line, lineN, -1)
}
function lineStartSmart(cm, pos) {
var start = lineStart(cm, pos.line);
var line = getLine(cm.doc, start.line);
var order = getOrder(line, cm.doc.direction);
if (!order || order[0].level == 0) {
var firstNonWS = Math.max(0, line.text.search(/\S/));
var inWS = pos.line == start.line && pos.ch <= firstNonWS && pos.ch;
return Pos(start.line, inWS ? 0 : firstNonWS, start.sticky)
}
return start
}

// Run a handler that was bound to a key.
function doHandleBinding(cm, bound, dropShift) {
if (typeof bound == "string") {
bound = commands[bound];
if (!bound) { return false }
}
// Ensure previous input has been read, so that the handler sees a
// consistent view of the document
cm.display.input.ensurePolled();
var prevShift = cm.display.shift, done = false;
try {
if (cm.isReadOnly()) { cm.state.suppressEdits = true; }
if (dropShift) { cm.display.shift = false; }
done = bound(cm) != Pass;
} finally {
cm.display.shift = prevShift;
cm.state.suppressEdits = false;
}
return done
}

function lookupKeyForEditor(cm, name, handle) {
for (var i = 0; i < cm.state.keyMaps.length; i++) {
var result = lookupKey(name, cm.state.keyMaps[i], handle, cm);
if (result) { return result }
}
return (cm.options.extraKeys && lookupKey(name, cm.options.extraKeys, handle, cm))
|| lookupKey(name, cm.options.keyMap, handle, cm)
}

// Note that, despite the name, this function is also used to check
// for bound mouse clicks.

var stopSeq = new Delayed;

function dispatchKey(cm, name, e, handle) {
var seq = cm.state.keySeq;
if (seq) {
if (isModifierKey(name)) { return "handled" }
if (/\'$/.test(name))
{ cm.state.keySeq = null; }
else
{ stopSeq.set(50, function () {
if (cm.state.keySeq == seq) {
cm.state.keySeq = null;
cm.display.input.reset();
}
}); }
if (dispatchKeyInner(cm, seq + " " + name, e, handle)) { return true }
}
return dispatchKeyInner(cm, name, e, handle)
}

function dispatchKeyInner(cm, name, e, handle) {
var result = lookupKeyForEditor(cm, name, handle);

if (result == "multi")
{ cm.state.keySeq = name; }
if (result == "handled")
{ signalLater(cm, "keyHandled", cm, name, e); }

if (result == "handled" || result == "multi") {
e_preventDefault(e);
restartBlink(cm);
}

return !!result
}

// Handle a key from the keydown event.
function handleKeyBinding(cm, e) {
var name = keyName(e, true);
if (!name) { return false }

if (e.shiftKey && !cm.state.keySeq) {
// First try to resolve full name (including 'Shift-'). Failing
// that, see if there is a cursor-motion command (starting with
// 'go') bound to the keyname without 'Shift-'.
return dispatchKey(cm, "Shift-" + name, e, function (b) { return doHandleBinding(cm, b, true); })
|| dispatchKey(cm, name, e, function (b) {
if (typeof b == "string" ? /^go[A-Z]/.test(b) : b.motion)
{ return doHandleBinding(cm, b) }
})
} else {
return dispatchKey(cm, name, e, function (b) { return doHandleBinding(cm, b); })
}
}

// Handle a key from the keypress event
function handleCharBinding(cm, e, ch) {
return dispatchKey(cm, "'" + ch + "'", e, function (b) { return doHandleBinding(cm, b, true); })
}

var lastStoppedKey = null;
function onKeyDown(e) {
var cm = this;
cm.curOp.focus = activeElt();
if (signalDOMEvent(cm, e)) { return }
// IE does strange things with escape.
if (ie && ie_version < 11 && e.keyCode == 27) { e.returnValue = false; }
var code = e.keyCode;
cm.display.shift = code == 16 || e.shiftKey;
var handled = handleKeyBinding(cm, e);
if (presto) {
lastStoppedKey = handled ? code : null;
// Opera has no cut event... we try to at least catch the key combo
if (!handled && code == 88 && !hasCopyEvent && (mac ? e.metaKey : e.ctrlKey))
{ cm.replaceSelection("", null, "cut"); }
}

// Turn mouse into crosshair when Alt is held on Mac.
if (code == 18 && !/\bCodeMirror-crosshair\b/.test(cm.display.lineDiv.className))
{ showCrossHair(cm); }
}

function showCrossHair(cm) {
var lineDiv = cm.display.lineDiv;
addClass(lineDiv, "CodeMirror-crosshair");

function up(e) {
if (e.keyCode == 18 || !e.altKey) {
rmClass(lineDiv, "CodeMirror-crosshair");
off(document, "keyup", up);
off(document, "mouseover", up);
}
}
on(document, "keyup", up);
on(document, "mouseover", up);
}

function onKeyUp(e) {
if (e.keyCode == 16) { this.doc.sel.shift = false; }
signalDOMEvent(this, e);
}

function onKeyPress(e) {
var cm = this;
if (eventInWidget(cm.display, e) || signalDOMEvent(cm, e) || e.ctrlKey && !e.altKey || mac && e.metaKey) { return }
var keyCode = e.keyCode, charCode = e.charCode;
if (presto && keyCode == lastStoppedKey) {lastStoppedKey = null; e_preventDefault(e); return}
if ((presto && (!e.which || e.which < 10)) && handleKeyBinding(cm, e)) { return }
var ch = String.fromCharCode(charCode == null ? keyCode : charCode);
// Some browsers fire keypress events for backspace
if (ch == "\x08") { return }
if (handleCharBinding(cm, e, ch)) { return }
cm.display.input.onKeyPress(e);
}

var DOUBLECLICK_DELAY = 400;

var PastClick = function(time, pos, button) {
this.time = time;
this.pos = pos;
this.button = button;
};

PastClick.prototype.compare = function (time, pos, button) {
return this.time + DOUBLECLICK_DELAY > time &&
cmp(pos, this.pos) == 0 && button == this.button
};

var lastClick, lastDoubleClick;
function clickRepeat(pos, button) {
var now = +new Date;
if (lastDoubleClick && lastDoubleClick.compare(now, pos, button)) {
lastClick = lastDoubleClick = null;
return "triple"
} else if (lastClick && lastClick.compare(now, pos, button)) {
lastDoubleClick = new PastClick(now, pos, button);
lastClick = null;
return "double"
} else {
lastClick = new PastClick(now, pos, button);
lastDoubleClick = null;
return "single"
}
}

// A mouse down can be a single click, double click, triple click,
// start of selection drag, start of text drag, new cursor
// (ctrl-click), rectangle drag (alt-drag), or xwin
// middle-click-paste. Or it might be a click on something we should
// not interfere with, such as a scrollbar or widget.
function onMouseDown(e) {
var cm = this, display = cm.display;
if (signalDOMEvent(cm, e) || display.activeTouch && display.input.supportsTouch()) { return }
display.input.ensurePolled();
display.shift = e.shiftKey;

if (eventInWidget(display, e)) {
if (!webkit) {
// Briefly turn off draggability, to allow widgets to do
// normal dragging things.
display.scroller.draggable = false;
setTimeout(function () { return display.scroller.draggable = true; }, 100);
}
return
}
if (clickInGutter(cm, e)) { return }
var pos = posFromMouse(cm, e), button = e_button(e), repeat = pos ? clickRepeat(pos, button) : "single";
window.focus();

// #3261: make sure, that we're not starting a second selection
if (button == 1 && cm.state.selectingText)
{ cm.state.selectingText(e); }

if (pos && handleMappedButton(cm, button, pos, repeat, e)) { return }

if (button == 1) {
if (pos) { leftButtonDown(cm, pos, repeat, e); }
else if (e_target(e) == display.scroller) { e_preventDefault(e); }
} else if (button == 2) {
if (pos) { extendSelection(cm.doc, pos); }
setTimeout(function () { return display.input.focus(); }, 20);
} else if (button == 3) {
if (captureRightClick) { cm.display.input.onContextMenu(e); }
else { delayBlurEvent(cm); }
}
}

function handleMappedButton(cm, button, pos, repeat, event) {
var name = "Click";
if (repeat == "double") { name = "Double" + name; }
else if (repeat == "triple") { name = "Triple" + name; }
name = (button == 1 ? "Left" : button == 2 ? "Middle" : "Right") + name;

return dispatchKey(cm,  addModifierNames(name, event), event, function (bound) {
if (typeof bound == "string") { bound = commands[bound]; }
if (!bound) { return false }
var done = false;
try {
if (cm.isReadOnly()) { cm.state.suppressEdits = true; }
done = bound(cm, pos) != Pass;
} finally {
cm.state.suppressEdits = false;
}
return done
})
}

function configureMouse(cm, repeat, event) {
var option = cm.getOption("configureMouse");
var value = option ? option(cm, repeat, event) : {};
if (value.unit == null) {
var rect = chromeOS ? event.shiftKey && event.metaKey : event.altKey;
value.unit = rect ? "rectangle" : repeat == "single" ? "char" : repeat == "double" ? "word" : "line";
}
if (value.extend == null || cm.doc.extend) { value.extend = cm.doc.extend || event.shiftKey; }
if (value.addNew == null) { value.addNew = mac ? event.metaKey : event.ctrlKey; }
if (value.moveOnDrag == null) { value.moveOnDrag = !(mac ? event.altKey : event.ctrlKey); }
return value
}

function leftButtonDown(cm, pos, repeat, event) {
if (ie) { setTimeout(bind(ensureFocus, cm), 0); }
else { cm.curOp.focus = activeElt(); }

var behavior = configureMouse(cm, repeat, event);

var sel = cm.doc.sel, contained;
if (cm.options.dragDrop && dragAndDrop && !cm.isReadOnly() &&
repeat == "single" && (contained = sel.contains(pos)) > -1 &&
(cmp((contained = sel.ranges[contained]).from(), pos) < 0 || pos.xRel > 0) &&
(cmp(contained.to(), pos) > 0 || pos.xRel < 0))
{ leftButtonStartDrag(cm, event, pos, behavior); }
else
{ leftButtonSelect(cm, event, pos, behavior); }
}

// Start a text drag. When it ends, see if any dragging actually
// happen, and treat as a click if it didn't.
function leftButtonStartDrag(cm, event, pos, behavior) {
var display = cm.display, moved = false;
var dragEnd = operation(cm, function (e) {
if (webkit) { display.scroller.draggable = false; }
cm.state.draggingText = false;
off(display.wrapper.ownerDocument, "mouseup", dragEnd);
off(display.wrapper.ownerDocument, "mousemove", mouseMove);
off(display.scroller, "dragstart", dragStart);
off(display.scroller, "drop", dragEnd);
if (!moved) {
e_preventDefault(e);
if (!behavior.addNew)
{ extendSelection(cm.doc, pos, null, null, behavior.extend); }
// Work around unexplainable focus problem in IE9 (#2127) and Chrome (#3081)
if (webkit || ie && ie_version == 9)
{ setTimeout(function () {display.wrapper.ownerDocument.body.focus(); display.input.focus();}, 20); }
else
{ display.input.focus(); }
}
});
var mouseMove = function(e2) {
moved = moved || Math.abs(event.clientX - e2.clientX) + Math.abs(event.clientY - e2.clientY) >= 10;
};
var dragStart = function () { return moved = true; };
// Let the drag handler handle this.
if (webkit) { display.scroller.draggable = true; }
cm.state.draggingText = dragEnd;
dragEnd.copy = !behavior.moveOnDrag;
// IE's approach to draggable
if (display.scroller.dragDrop) { display.scroller.dragDrop(); }
on(display.wrapper.ownerDocument, "mouseup", dragEnd);
on(display.wrapper.ownerDocument, "mousemove", mouseMove);
on(display.scroller, "dragstart", dragStart);
on(display.scroller, "drop", dragEnd);

delayBlurEvent(cm);
setTimeout(function () { return display.input.focus(); }, 20);
}

function rangeForUnit(cm, pos, unit) {
if (unit == "char") { return new Range(pos, pos) }
if (unit == "word") { return cm.findWordAt(pos) }
if (unit == "line") { return new Range(Pos(pos.line, 0), clipPos(cm.doc, Pos(pos.line + 1, 0))) }
var result = unit(cm, pos);
return new Range(result.from, result.to)
}

// Normal selection, as opposed to text dragging.
function leftButtonSelect(cm, event, start, behavior) {
var display = cm.display, doc = cm.doc;
e_preventDefault(event);

var ourRange, ourIndex, startSel = doc.sel, ranges = startSel.ranges;
if (behavior.addNew && !behavior.extend) {
ourIndex = doc.sel.contains(start);
if (ourIndex > -1)
{ ourRange = ranges[ourIndex]; }
else
{ ourRange = new Range(start, start); }
} else {
ourRange = doc.sel.primary();
ourIndex = doc.sel.primIndex;
}

if (behavior.unit == "rectangle") {
if (!behavior.addNew) { ourRange = new Range(start, start); }
start = posFromMouse(cm, event, true, true);
ourIndex = -1;
} else {
var range$$1 = rangeForUnit(cm, start, behavior.unit);
if (behavior.extend)
{ ourRange = extendRange(ourRange, range$$1.anchor, range$$1.head, behavior.extend); }
else
{ ourRange = range$$1; }
}

if (!behavior.addNew) {
ourIndex = 0;
setSelection(doc, new Selection([ourRange], 0), sel_mouse);
startSel = doc.sel;
} else if (ourIndex == -1) {
ourIndex = ranges.length;
setSelection(doc, normalizeSelection(cm, ranges.concat([ourRange]), ourIndex),
{scroll: false, origin: "*mouse"});
} else if (ranges.length > 1 && ranges[ourIndex].empty() && behavior.unit == "char" && !behavior.extend) {
setSelection(doc, normalizeSelection(cm, ranges.slice(0, ourIndex).concat(ranges.slice(ourIndex + 1)), 0),
{scroll: false, origin: "*mouse"});
startSel = doc.sel;
} else {
replaceOneSelection(doc, ourIndex, ourRange, sel_mouse);
}

var lastPos = start;
function extendTo(pos) {
if (cmp(lastPos, pos) == 0) { return }
lastPos = pos;

if (behavior.unit == "rectangle") {
var ranges = [], tabSize = cm.options.tabSize;
var startCol = countColumn(getLine(doc, start.line).text, start.ch, tabSize);
var posCol = countColumn(getLine(doc, pos.line).text, pos.ch, tabSize);
var left = Math.min(startCol, posCol), right = Math.max(startCol, posCol);
for (var line = Math.min(start.line, pos.line), end = Math.min(cm.lastLine(), Math.max(start.line, pos.line));
line <= end; line++) {
var text = getLine(doc, line).text, leftPos = findColumn(text, left, tabSize);
if (left == right)
{ ranges.push(new Range(Pos(line, leftPos), Pos(line, leftPos))); }
else if (text.length > leftPos)
{ ranges.push(new Range(Pos(line, leftPos), Pos(line, findColumn(text, right, tabSize)))); }
}
if (!ranges.length) { ranges.push(new Range(start, start)); }
setSelection(doc, normalizeSelection(cm, startSel.ranges.slice(0, ourIndex).concat(ranges), ourIndex),
{origin: "*mouse", scroll: false});
cm.scrollIntoView(pos);
} else {
var oldRange = ourRange;
var range$$1 = rangeForUnit(cm, pos, behavior.unit);
var anchor = oldRange.anchor, head;
if (cmp(range$$1.anchor, anchor) > 0) {
head = range$$1.head;
anchor = minPos(oldRange.from(), range$$1.anchor);
} else {
head = range$$1.anchor;
anchor = maxPos(oldRange.to(), range$$1.head);
}
var ranges$1 = startSel.ranges.slice(0);
ranges$1[ourIndex] = bidiSimplify(cm, new Range(clipPos(doc, anchor), head));
setSelection(doc, normalizeSelection(cm, ranges$1, ourIndex), sel_mouse);
}
}

var editorSize = display.wrapper.getBoundingClientRect();
// Used to ensure timeout re-tries don't fire when another extend
// happened in the meantime (clearTimeout isn't reliable -- at
// least on Chrome, the timeouts still happen even when cleared,
// if the clear happens after their scheduled firing time).
var counter = 0;

function extend(e) {
var curCount = ++counter;
var cur = posFromMouse(cm, e, true, behavior.unit == "rectangle");
if (!cur) { return }
if (cmp(cur, lastPos) != 0) {
cm.curOp.focus = activeElt();
extendTo(cur);
var visible = visibleLines(display, doc);
if (cur.line >= visible.to || cur.line < visible.from)
{ setTimeout(operation(cm, function () {if (counter == curCount) { extend(e); }}), 150); }
} else {
var outside = e.clientY < editorSize.top ? -20 : e.clientY > editorSize.bottom ? 20 : 0;
if (outside) { setTimeout(operation(cm, function () {
if (counter != curCount) { return }
display.scroller.scrollTop += outside;
extend(e);
}), 50); }
}
}

function done(e) {
cm.state.selectingText = false;
counter = Infinity;
// If e is null or undefined we interpret this as someone trying
// to explicitly cancel the selection rather than the user
// letting go of the mouse button.
if (e) {
e_preventDefault(e);
display.input.focus();
}
off(display.wrapper.ownerDocument, "mousemove", move);
off(display.wrapper.ownerDocument, "mouseup", up);
doc.history.lastSelOrigin = null;
}

var move = operation(cm, function (e) {
if (e.buttons === 0 || !e_button(e)) { done(e); }
else { extend(e); }
});
var up = operation(cm, done);
cm.state.selectingText = up;
on(display.wrapper.ownerDocument, "mousemove", move);
on(display.wrapper.ownerDocument, "mouseup", up);
}

// Used when mouse-selecting to adjust the anchor to the proper side
// of a bidi jump depending on the visual position of the head.
function bidiSimplify(cm, range$$1) {
var anchor = range$$1.anchor;
var head = range$$1.head;
var anchorLine = getLine(cm.doc, anchor.line);
if (cmp(anchor, head) == 0 && anchor.sticky == head.sticky) { return range$$1 }
var order = getOrder(anchorLine);
if (!order) { return range$$1 }
var index = getBidiPartAt(order, anchor.ch, anchor.sticky), part = order[index];
if (part.from != anchor.ch && part.to != anchor.ch) { return range$$1 }
var boundary = index + ((part.from == anchor.ch) == (part.level != 1) ? 0 : 1);
if (boundary == 0 || boundary == order.length) { return range$$1 }

// Compute the relative visual position of the head compared to the
// anchor (<0 is to the left, >0 to the right)
var leftSide;
if (head.line != anchor.line) {
leftSide = (head.line - anchor.line) * (cm.doc.direction == "ltr" ? 1 : -1) > 0;
} else {
var headIndex = getBidiPartAt(order, head.ch, head.sticky);
var dir = headIndex - index || (head.ch - anchor.ch) * (part.level == 1 ? -1 : 1);
if (headIndex == boundary - 1 || headIndex == boundary)
{ leftSide = dir < 0; }
else
{ leftSide = dir > 0; }
}

var usePart = order[boundary + (leftSide ? -1 : 0)];
var from = leftSide == (usePart.level == 1);
var ch = from ? usePart.from : usePart.to, sticky = from ? "after" : "before";
return anchor.ch == ch && anchor.sticky == sticky ? range$$1 : new Range(new Pos(anchor.line, ch, sticky), head)
}


// Determines whether an event happened in the gutter, and fires the
// handlers for the corresponding event.
function gutterEvent(cm, e, type, prevent) {
var mX, mY;
if (e.touches) {
mX = e.touches[0].clientX;
mY = e.touches[0].clientY;
} else {
try { mX = e.clientX; mY = e.clientY; }
catch(e) { return false }
}
if (mX >= Math.floor(cm.display.gutters.getBoundingClientRect().right)) { return false }
if (prevent) { e_preventDefault(e); }

var display = cm.display;
var lineBox = display.lineDiv.getBoundingClientRect();

if (mY > lineBox.bottom || !hasHandler(cm, type)) { return e_defaultPrevented(e) }
mY -= lineBox.top - display.viewOffset;

for (var i = 0; i < cm.display.gutterSpecs.length; ++i) {
var g = display.gutters.childNodes[i];
if (g && g.getBoundingClientRect().right >= mX) {
var line = lineAtHeight(cm.doc, mY);
var gutter = cm.display.gutterSpecs[i];
signal(cm, type, cm, line, gutter.className, e);
return e_defaultPrevented(e)
}
}
}

function clickInGutter(cm, e) {
return gutterEvent(cm, e, "gutterClick", true)
}

// CONTEXT MENU HANDLING

// To make the context menu work, we need to briefly unhide the
// textarea (making it as unobtrusive as possible) to let the
// right-click take effect on it.
function onContextMenu(cm, e) {
if (eventInWidget(cm.display, e) || contextMenuInGutter(cm, e)) { return }
if (signalDOMEvent(cm, e, "contextmenu")) { return }
if (!captureRightClick) { cm.display.input.onContextMenu(e); }
}

function contextMenuInGutter(cm, e) {
if (!hasHandler(cm, "gutterContextMenu")) { return false }
return gutterEvent(cm, e, "gutterContextMenu", false)
}

function themeChanged(cm) {
cm.display.wrapper.className = cm.display.wrapper.className.replace(/\s*cm-s-\S+/g, "") +
cm.options.theme.replace(/(^|\s)\s*/g, " cm-s-");
clearCaches(cm);
}

var Init = {toString: function(){return "CodeMirror.Init"}};

var defaults = {};
var optionHandlers = {};

function defineOptions(CodeMirror) {
var optionHandlers = CodeMirror.optionHandlers;

function option(name, deflt, handle, notOnInit) {
CodeMirror.defaults[name] = deflt;
if (handle) { optionHandlers[name] =
notOnInit ? function (cm, val, old) {if (old != Init) { handle(cm, val, old); }} : handle; }
}

CodeMirror.defineOption = option;

// Passed to option handlers when there is no old value.
CodeMirror.Init = Init;

// These two are, on init, called from the constructor because they
// have to be initialized before the editor can start at all.
option("value", "", function (cm, val) { return cm.setValue(val); }, true);
option("mode", null, function (cm, val) {
cm.doc.modeOption = val;
loadMode(cm);
}, true);

option("indentUnit", 2, loadMode, true);
option("indentWithTabs", false);
option("smartIndent", true);
option("tabSize", 4, function (cm) {
resetModeState(cm);
clearCaches(cm);
regChange(cm);
}, true);

option("lineSeparator", null, function (cm, val) {
cm.doc.lineSep = val;
if (!val) { return }
var newBreaks = [], lineNo = cm.doc.first;
cm.doc.iter(function (line) {
for (var pos = 0;;) {
var found = line.text.indexOf(val, pos);
if (found == -1) { break }
pos = found + val.length;
newBreaks.push(Pos(lineNo, found));
}
lineNo++;
});
for (var i = newBreaks.length - 1; i >= 0; i--)
{ replaceRange(cm.doc, val, newBreaks[i], Pos(newBreaks[i].line, newBreaks[i].ch + val.length)); }
});
option("specialChars", /[\u0000-\u001f\u007f-\u009f\u00ad\u061c\u200b-\u200f\u2028\u2029\ufeff]/g, function (cm, val, old) {
cm.state.specialChars = new RegExp(val.source + (val.test("\t") ? "" : "|\t"), "g");
if (old != Init) { cm.refresh(); }
});
option("specialCharPlaceholder", defaultSpecialCharPlaceholder, function (cm) { return cm.refresh(); }, true);
option("electricChars", true);
option("inputStyle", mobile ? "contenteditable" : "textarea", function () {
throw new Error("inputStyle can not (yet) be changed in a running editor") // FIXME
}, true);
option("spellcheck", false, function (cm, val) { return cm.getInputField().spellcheck = val; }, true);
option("autocorrect", false, function (cm, val) { return cm.getInputField().autocorrect = val; }, true);
option("autocapitalize", false, function (cm, val) { return cm.getInputField().autocapitalize = val; }, true);
option("rtlMoveVisually", !windows);
option("wholeLineUpdateBefore", true);

option("theme", "default", function (cm) {
themeChanged(cm);
updateGutters(cm);
}, true);
option("keyMap", "default", function (cm, val, old) {
var next = getKeyMap(val);
var prev = old != Init && getKeyMap(old);
if (prev && prev.detach) { prev.detach(cm, next); }
if (next.attach) { next.attach(cm, prev || null); }
});
option("extraKeys", null);
option("configureMouse", null);

option("lineWrapping", false, wrappingChanged, true);
option("gutters", [], function (cm, val) {
cm.display.gutterSpecs = getGutters(val, cm.options.lineNumbers);
updateGutters(cm);
}, true);
option("fixedGutter", true, function (cm, val) {
cm.display.gutters.style.left = val ? compensateForHScroll(cm.display) + "px" : "0";
cm.refresh();
}, true);
option("coverGutterNextToScrollbar", false, function (cm) { return updateScrollbars(cm); }, true);
option("scrollbarStyle", "native", function (cm) {
initScrollbars(cm);
updateScrollbars(cm);
cm.display.scrollbars.setScrollTop(cm.doc.scrollTop);
cm.display.scrollbars.setScrollLeft(cm.doc.scrollLeft);
}, true);
option("lineNumbers", false, function (cm, val) {
cm.display.gutterSpecs = getGutters(cm.options.gutters, val);
updateGutters(cm);
}, true);
option("firstLineNumber", 1, updateGutters, true);
option("lineNumberFormatter", function (integer) { return integer; }, updateGutters, true);
option("showCursorWhenSelecting", false, updateSelection, true);

option("resetSelectionOnContextMenu", true);
option("lineWiseCopyCut", true);
option("pasteLinesPerSelection", true);
option("selectionsMayTouch", false);

option("readOnly", false, function (cm, val) {
if (val == "nocursor") {
onBlur(cm);
cm.display.input.blur();
}
cm.display.input.readOnlyChanged(val);
});
option("disableInput", false, function (cm, val) {if (!val) { cm.display.input.reset(); }}, true);
option("dragDrop", true, dragDropChanged);
option("allowDropFileTypes", null);

option("cursorBlinkRate", 530);
option("cursorScrollMargin", 0);
option("cursorHeight", 1, updateSelection, true);
option("singleCursorHeightPerLine", true, updateSelection, true);
option("workTime", 100);
option("workDelay", 100);
option("flattenSpans", true, resetModeState, true);
option("addModeClass", false, resetModeState, true);
option("pollInterval", 100);
option("undoDepth", 200, function (cm, val) { return cm.doc.history.undoDepth = val; });
option("historyEventDelay", 1250);
option("viewportMargin", 10, function (cm) { return cm.refresh(); }, true);
option("maxHighlightLength", 10000, resetModeState, true);
option("moveInputWithCursor", true, function (cm, val) {
if (!val) { cm.display.input.resetPosition(); }
});

option("tabindex", null, function (cm, val) { return cm.display.input.getField().tabIndex = val || ""; });
option("autofocus", null);
option("direction", "ltr", function (cm, val) { return cm.doc.setDirection(val); }, true);
option("phrases", null);
}

function dragDropChanged(cm, value, old) {
var wasOn = old && old != Init;
if (!value != !wasOn) {
var funcs = cm.display.dragFunctions;
var toggle = value ? on : off;
toggle(cm.display.scroller, "dragstart", funcs.start);
toggle(cm.display.scroller, "dragenter", funcs.enter);
toggle(cm.display.scroller, "dragover", funcs.over);
toggle(cm.display.scroller, "dragleave", funcs.leave);
toggle(cm.display.scroller, "drop", funcs.drop);
}
}

function wrappingChanged(cm) {
if (cm.options.lineWrapping) {
addClass(cm.display.wrapper, "CodeMirror-wrap");
cm.display.sizer.style.minWidth = "";
cm.display.sizerWidth = null;
} else {
rmClass(cm.display.wrapper, "CodeMirror-wrap");
findMaxLine(cm);
}
estimateLineHeights(cm);
regChange(cm);
clearCaches(cm);
setTimeout(function () { return updateScrollbars(cm); }, 100);
}

// A CodeMirror instance represents an editor. This is the object
// that user code is usually dealing with.

function CodeMirror(place, options) {
var this$1 = this;

if (!(this instanceof CodeMirror)) { return new CodeMirror(place, options) }

this.options = options = options ? copyObj(options) : {};
// Determine effective options based on given values and defaults.
copyObj(defaults, options, false);

var doc = options.value;
if (typeof doc == "string") { doc = new Doc(doc, options.mode, null, options.lineSeparator, options.direction); }
else if (options.mode) { doc.modeOption = options.mode; }
this.doc = doc;

var input = new CodeMirror.inputStyles[options.inputStyle](this);
var display = this.display = new Display(place, doc, input, options);
display.wrapper.CodeMirror = this;
themeChanged(this);
if (options.lineWrapping)
{ this.display.wrapper.className += " CodeMirror-wrap"; }
initScrollbars(this);

this.state = {
keyMaps: [],  // stores maps added by addKeyMap
overlays: [], // highlighting overlays, as added by addOverlay
modeGen: 0,   // bumped when mode/overlay changes, used to invalidate highlighting info
overwrite: false,
delayingBlurEvent: false,
focused: false,
suppressEdits: false, // used to disable editing during key handlers when in readOnly mode
pasteIncoming: -1, cutIncoming: -1, // help recognize paste/cut edits in input.poll
selectingText: false,
draggingText: false,
highlight: new Delayed(), // stores highlight worker timeout
keySeq: null,  // Unfinished key sequence
specialChars: null
};

if (options.autofocus && !mobile) { display.input.focus(); }

// Override magic textarea content restore that IE sometimes does
// on our hidden textarea on reload
if (ie && ie_version < 11) { setTimeout(function () { return this$1.display.input.reset(true); }, 20); }

registerEventHandlers(this);
ensureGlobalHandlers();

startOperation(this);
this.curOp.forceUpdate = true;
attachDoc(this, doc);

if ((options.autofocus && !mobile) || this.hasFocus())
{ setTimeout(bind(onFocus, this), 20); }
else
{ onBlur(this); }

for (var opt in optionHandlers) { if (optionHandlers.hasOwnProperty(opt))
{ optionHandlers[opt](this$1, options[opt], Init); } }
maybeUpdateLineNumberWidth(this);
if (options.finishInit) { options.finishInit(this); }
for (var i = 0; i < initHooks.length; ++i) { initHooks[i](this$1); }
endOperation(this);
// Suppress optimizelegibility in Webkit, since it breaks text
// measuring on line wrapping boundaries.
if (webkit && options.lineWrapping &&
getComputedStyle(display.lineDiv).textRendering == "optimizelegibility")
{ display.lineDiv.style.textRendering = "auto"; }
}

// The default configuration options.
CodeMirror.defaults = defaults;
// Functions to run when options are changed.
CodeMirror.optionHandlers = optionHandlers;

// Attach the necessary event handlers when initializing the editor
function registerEventHandlers(cm) {
var d = cm.display;
on(d.scroller, "mousedown", operation(cm, onMouseDown));
// Older IE's will not fire a second mousedown for a double click
if (ie && ie_version < 11)
{ on(d.scroller, "dblclick", operation(cm, function (e) {
if (signalDOMEvent(cm, e)) { return }
var pos = posFromMouse(cm, e);
if (!pos || clickInGutter(cm, e) || eventInWidget(cm.display, e)) { return }
e_preventDefault(e);
var word = cm.findWordAt(pos);
extendSelection(cm.doc, word.anchor, word.head);
})); }
else
{ on(d.scroller, "dblclick", function (e) { return signalDOMEvent(cm, e) || e_preventDefault(e); }); }
// Some browsers fire contextmenu *after* opening the menu, at
// which point we can't mess with it anymore. Context menu is
// handled in onMouseDown for these browsers.
on(d.scroller, "contextmenu", function (e) { return onContextMenu(cm, e); });

// Used to suppress mouse event handling when a touch happens
var touchFinished, prevTouch = {end: 0};
function finishTouch() {
if (d.activeTouch) {
touchFinished = setTimeout(function () { return d.activeTouch = null; }, 1000);
prevTouch = d.activeTouch;
prevTouch.end = +new Date;
}
}
function isMouseLikeTouchEvent(e) {
if (e.touches.length != 1) { return false }
var touch = e.touches[0];
return touch.radiusX <= 1 && touch.radiusY <= 1
}
function farAway(touch, other) {
if (other.left == null) { return true }
var dx = other.left - touch.left, dy = other.top - touch.top;
return dx * dx + dy * dy > 20 * 20
}
on(d.scroller, "touchstart", function (e) {
if (!signalDOMEvent(cm, e) && !isMouseLikeTouchEvent(e) && !clickInGutter(cm, e)) {
d.input.ensurePolled();
clearTimeout(touchFinished);
var now = +new Date;
d.activeTouch = {start: now, moved: false,
prev: now - prevTouch.end <= 300 ? prevTouch : null};
if (e.touches.length == 1) {
d.activeTouch.left = e.touches[0].pageX;
d.activeTouch.top = e.touches[0].pageY;
}
}
});
on(d.scroller, "touchmove", function () {
if (d.activeTouch) { d.activeTouch.moved = true; }
});
on(d.scroller, "touchend", function (e) {
var touch = d.activeTouch;
if (touch && !eventInWidget(d, e) && touch.left != null &&
!touch.moved && new Date - touch.start < 300) {
var pos = cm.coordsChar(d.activeTouch, "page"), range;
if (!touch.prev || farAway(touch, touch.prev)) // Single tap
{ range = new Range(pos, pos); }
else if (!touch.prev.prev || farAway(touch, touch.prev.prev)) // Double tap
{ range = cm.findWordAt(pos); }
else // Triple tap
{ range = new Range(Pos(pos.line, 0), clipPos(cm.doc, Pos(pos.line + 1, 0))); }
cm.setSelection(range.anchor, range.head);
cm.focus();
e_preventDefault(e);
}
finishTouch();
});
on(d.scroller, "touchcancel", finishTouch);

// Sync scrolling between fake scrollbars and real scrollable
// area, ensure viewport is updated when scrolling.
on(d.scroller, "scroll", function () {
if (d.scroller.clientHeight) {
updateScrollTop(cm, d.scroller.scrollTop);
setScrollLeft(cm, d.scroller.scrollLeft, true);
signal(cm, "scroll", cm);
}
});

// Listen to wheel events in order to try and update the viewport on time.
on(d.scroller, "mousewheel", function (e) { return onScrollWheel(cm, e); });
on(d.scroller, "DOMMouseScroll", function (e) { return onScrollWheel(cm, e); });

// Prevent wrapper from ever scrolling
on(d.wrapper, "scroll", function () { return d.wrapper.scrollTop = d.wrapper.scrollLeft = 0; });

d.dragFunctions = {
enter: function (e) {if (!signalDOMEvent(cm, e)) { e_stop(e); }},
over: function (e) {if (!signalDOMEvent(cm, e)) { onDragOver(cm, e); e_stop(e); }},
start: function (e) { return onDragStart(cm, e); },
drop: operation(cm, onDrop),
leave: function (e) {if (!signalDOMEvent(cm, e)) { clearDragCursor(cm); }}
};

var inp = d.input.getField();
on(inp, "keyup", function (e) { return onKeyUp.call(cm, e); });
on(inp, "keydown", operation(cm, onKeyDown));
on(inp, "keypress", operation(cm, onKeyPress));
on(inp, "focus", function (e) { return onFocus(cm, e); });
on(inp, "blur", function (e) { return onBlur(cm, e); });
}

var initHooks = [];
CodeMirror.defineInitHook = function (f) { return initHooks.push(f); };

// Indent the given line. The how parameter can be "smart",
// "add"/null, "subtract", or "prev". When aggressive is false
// (typically set to true for forced single-line indents), empty
// lines are not indented, and places where the mode returns Pass
// are left alone.
function indentLine(cm, n, how, aggressive) {
var doc = cm.doc, state;
if (how == null) { how = "add"; }
if (how == "smart") {
// Fall back to "prev" when the mode doesn't have an indentation
// method.
if (!doc.mode.indent) { how = "prev"; }
else { state = getContextBefore(cm, n).state; }
}

var tabSize = cm.options.tabSize;
var line = getLine(doc, n), curSpace = countColumn(line.text, null, tabSize);
if (line.stateAfter) { line.stateAfter = null; }
var curSpaceString = line.text.match(/^\s*/)[0], indentation;
if (!aggressive && !/\S/.test(line.text)) {
indentation = 0;
how = "not";
} else if (how == "smart") {
indentation = doc.mode.indent(state, line.text.slice(curSpaceString.length), line.text);
if (indentation == Pass || indentation > 150) {
if (!aggressive) { return }
how = "prev";
}
}
if (how == "prev") {
if (n > doc.first) { indentation = countColumn(getLine(doc, n-1).text, null, tabSize); }
else { indentation = 0; }
} else if (how == "add") {
indentation = curSpace + cm.options.indentUnit;
} else if (how == "subtract") {
indentation = curSpace - cm.options.indentUnit;
} else if (typeof how == "number") {
indentation = curSpace + how;
}
indentation = Math.max(0, indentation);

var indentString = "", pos = 0;
if (cm.options.indentWithTabs)
{ for (var i = Math.floor(indentation / tabSize); i; --i) {pos += tabSize; indentString += "\t";} }
if (pos < indentation) { indentString += spaceStr(indentation - pos); }

if (indentString != curSpaceString) {
replaceRange(doc, indentString, Pos(n, 0), Pos(n, curSpaceString.length), "+input");
line.stateAfter = null;
return true
} else {
// Ensure that, if the cursor was in the whitespace at the start
// of the line, it is moved to the end of that space.
for (var i$1 = 0; i$1 < doc.sel.ranges.length; i$1++) {
var range = doc.sel.ranges[i$1];
if (range.head.line == n && range.head.ch < curSpaceString.length) {
var pos$1 = Pos(n, curSpaceString.length);
replaceOneSelection(doc, i$1, new Range(pos$1, pos$1));
break
}
}
}
}

// This will be set to a {lineWise: bool, text: [string]} object, so
// that, when pasting, we know what kind of selections the copied
// text was made out of.
var lastCopied = null;

function setLastCopied(newLastCopied) {
lastCopied = newLastCopied;
}

function applyTextInput(cm, inserted, deleted, sel, origin) {
var doc = cm.doc;
cm.display.shift = false;
if (!sel) { sel = doc.sel; }

var recent = +new Date - 200;
var paste = origin == "paste" || cm.state.pasteIncoming > recent;
var textLines = splitLinesAuto(inserted), multiPaste = null;
// When pasting N lines into N selections, insert one line per selection
if (paste && sel.ranges.length > 1) {
if (lastCopied && lastCopied.text.join("\n") == inserted) {
if (sel.ranges.length % lastCopied.text.length == 0) {
multiPaste = [];
for (var i = 0; i < lastCopied.text.length; i++)
{ multiPaste.push(doc.splitLines(lastCopied.text[i])); }
}
} else if (textLines.length == sel.ranges.length && cm.options.pasteLinesPerSelection) {
multiPaste = map(textLines, function (l) { return [l]; });
}
}

var updateInput = cm.curOp.updateInput;
// Normal behavior is to insert the new text into every selection
for (var i$1 = sel.ranges.length - 1; i$1 >= 0; i$1--) {
var range$$1 = sel.ranges[i$1];
var from = range$$1.from(), to = range$$1.to();
if (range$$1.empty()) {
if (deleted && deleted > 0) // Handle deletion
{ from = Pos(from.line, from.ch - deleted); }
else if (cm.state.overwrite && !paste) // Handle overwrite
{ to = Pos(to.line, Math.min(getLine(doc, to.line).text.length, to.ch + lst(textLines).length)); }
else if (paste && lastCopied && lastCopied.lineWise && lastCopied.text.join("\n") == inserted)
{ from = to = Pos(from.line, 0); }
}
var changeEvent = {from: from, to: to, text: multiPaste ? multiPaste[i$1 % multiPaste.length] : textLines,
origin: origin || (paste ? "paste" : cm.state.cutIncoming > recent ? "cut" : "+input")};
makeChange(cm.doc, changeEvent);
signalLater(cm, "inputRead", cm, changeEvent);
}
if (inserted && !paste)
{ triggerElectric(cm, inserted); }

ensureCursorVisible(cm);
if (cm.curOp.updateInput < 2) { cm.curOp.updateInput = updateInput; }
cm.curOp.typing = true;
cm.state.pasteIncoming = cm.state.cutIncoming = -1;
}

function handlePaste(e, cm) {
var pasted = e.clipboardData && e.clipboardData.getData("Text");
if (pasted) {
e.preventDefault();
if (!cm.isReadOnly() && !cm.options.disableInput)
{ runInOp(cm, function () { return applyTextInput(cm, pasted, 0, null, "paste"); }); }
return true
}
}

function triggerElectric(cm, inserted) {
// When an 'electric' character is inserted, immediately trigger a reindent
if (!cm.options.electricChars || !cm.options.smartIndent) { return }
var sel = cm.doc.sel;

for (var i = sel.ranges.length - 1; i >= 0; i--) {
var range$$1 = sel.ranges[i];
if (range$$1.head.ch > 100 || (i && sel.ranges[i - 1].head.line == range$$1.head.line)) { continue }
var mode = cm.getModeAt(range$$1.head);
var indented = false;
if (mode.electricChars) {
for (var j = 0; j < mode.electricChars.length; j++)
{ if (inserted.indexOf(mode.electricChars.charAt(j)) > -1) {
indented = indentLine(cm, range$$1.head.line, "smart");
break
} }
} else if (mode.electricInput) {
if (mode.electricInput.test(getLine(cm.doc, range$$1.head.line).text.slice(0, range$$1.head.ch)))
{ indented = indentLine(cm, range$$1.head.line, "smart"); }
}
if (indented) { signalLater(cm, "electricInput", cm, range$$1.head.line); }
}
}

function copyableRanges(cm) {
var text = [], ranges = [];
for (var i = 0; i < cm.doc.sel.ranges.length; i++) {
var line = cm.doc.sel.ranges[i].head.line;
var lineRange = {anchor: Pos(line, 0), head: Pos(line + 1, 0)};
ranges.push(lineRange);
text.push(cm.getRange(lineRange.anchor, lineRange.head));
}
return {text: text, ranges: ranges}
}

function disableBrowserMagic(field, spellcheck, autocorrect, autocapitalize) {
field.setAttribute("autocorrect", autocorrect ? "" : "off");
field.setAttribute("autocapitalize", autocapitalize ? "" : "off");
field.setAttribute("spellcheck", !!spellcheck);
}

function hiddenTextarea() {
var te = elt("textarea", null, null, "position: absolute; bottom: -1em; padding: 0; width: 1px; height: 1em; outline: none");
var div = elt("div", [te], null, "overflow: hidden; position: relative; width: 3px; height: 0px;");
// The textarea is kept positioned near the cursor to prevent the
// fact that it'll be scrolled into view on input from scrolling
// our fake cursor out of view. On webkit, when wrap=off, paste is
// very slow. So make the area wide instead.
if (webkit) { te.style.width = "1000px"; }
else { te.setAttribute("wrap", "off"); }
// If border: 0; -- iOS fails to open keyboard (issue #1287)
if (ios) { te.style.border = "1px solid black"; }
disableBrowserMagic(te);
return div
}

// The publicly visible API. Note that methodOp(f) means
// This is not the complete set of editor methods. Most of the
// methods defined on the Doc type are also injected into
// CodeMirror.prototype, for backwards compatibility and
// convenience.

function addEditorMethods(CodeMirror) {
var optionHandlers = CodeMirror.optionHandlers;

var helpers = CodeMirror.helpers = {};

CodeMirror.prototype = {
constructor: CodeMirror,
focus: function(){window.focus(); this.display.input.focus();},

setOption: function(option, value) {
var options = this.options, old = options[option];
if (options[option] == value && option != "mode") { return }
options[option] = value;
if (optionHandlers.hasOwnProperty(option))
{ operation(this, optionHandlers[option])(this, value, old); }
signal(this, "optionChange", this, option);
},

getOption: function(option) {return this.options[option]},
getDoc: function() {return this.doc},

addKeyMap: function(map$$1, bottom) {
this.state.keyMaps[bottom ? "push" : "unshift"](getKeyMap(map$$1));
},
removeKeyMap: function(map$$1) {
var maps = this.state.keyMaps;
for (var i = 0; i < maps.length; ++i)
{ if (maps[i] == map$$1 || maps[i].name == map$$1) {
maps.splice(i, 1);
return true
} }
},

addOverlay: methodOp(function(spec, options) {
var mode = spec.token ? spec : CodeMirror.getMode(this.options, spec);
if (mode.startState) { throw new Error("Overlays may not be stateful.") }
insertSorted(this.state.overlays,
{mode: mode, modeSpec: spec, opaque: options && options.opaque,
priority: (options && options.priority) || 0},
function (overlay) { return overlay.priority; });
this.state.modeGen++;
regChange(this);
}),
removeOverlay: methodOp(function(spec) {
var this$1 = this;

var overlays = this.state.overlays;
for (var i = 0; i < overlays.length; ++i) {
var cur = overlays[i].modeSpec;
if (cur == spec || typeof spec == "string" && cur.name == spec) {
overlays.splice(i, 1);
this$1.state.modeGen++;
regChange(this$1);
return
}
}
}),

indentLine: methodOp(function(n, dir, aggressive) {
if (typeof dir != "string" && typeof dir != "number") {
if (dir == null) { dir = this.options.smartIndent ? "smart" : "prev"; }
else { dir = dir ? "add" : "subtract"; }
}
if (isLine(this.doc, n)) { indentLine(this, n, dir, aggressive); }
}),
indentSelection: methodOp(function(how) {
var this$1 = this;

var ranges = this.doc.sel.ranges, end = -1;
for (var i = 0; i < ranges.length; i++) {
var range$$1 = ranges[i];
if (!range$$1.empty()) {
var from = range$$1.from(), to = range$$1.to();
var start = Math.max(end, from.line);
end = Math.min(this$1.lastLine(), to.line - (to.ch ? 0 : 1)) + 1;
for (var j = start; j < end; ++j)
{ indentLine(this$1, j, how); }
var newRanges = this$1.doc.sel.ranges;
if (from.ch == 0 && ranges.length == newRanges.length && newRanges[i].from().ch > 0)
{ replaceOneSelection(this$1.doc, i, new Range(from, newRanges[i].to()), sel_dontScroll); }
} else if (range$$1.head.line > end) {
indentLine(this$1, range$$1.head.line, how, true);
end = range$$1.head.line;
if (i == this$1.doc.sel.primIndex) { ensureCursorVisible(this$1); }
}
}
}),

// Fetch the parser token for a given character. Useful for hacks
// that want to inspect the mode state (say, for completion).
getTokenAt: function(pos, precise) {
return takeToken(this, pos, precise)
},

getLineTokens: function(line, precise) {
return takeToken(this, Pos(line), precise, true)
},

getTokenTypeAt: function(pos) {
pos = clipPos(this.doc, pos);
var styles = getLineStyles(this, getLine(this.doc, pos.line));
var before = 0, after = (styles.length - 1) / 2, ch = pos.ch;
var type;
if (ch == 0) { type = styles[2]; }
else { for (;;) {
var mid = (before + after) >> 1;
if ((mid ? styles[mid * 2 - 1] : 0) >= ch) { after = mid; }
else if (styles[mid * 2 + 1] < ch) { before = mid + 1; }
else { type = styles[mid * 2 + 2]; break }
} }
var cut = type ? type.indexOf("overlay ") : -1;
return cut < 0 ? type : cut == 0 ? null : type.slice(0, cut - 1)
},

getModeAt: function(pos) {
var mode = this.doc.mode;
if (!mode.innerMode) { return mode }
return CodeMirror.innerMode(mode, this.getTokenAt(pos).state).mode
},

getHelper: function(pos, type) {
return this.getHelpers(pos, type)[0]
},

getHelpers: function(pos, type) {
var this$1 = this;

var found = [];
if (!helpers.hasOwnProperty(type)) { return found }
var help = helpers[type], mode = this.getModeAt(pos);
if (typeof mode[type] == "string") {
if (help[mode[type]]) { found.push(help[mode[type]]); }
} else if (mode[type]) {
for (var i = 0; i < mode[type].length; i++) {
var val = help[mode[type][i]];
if (val) { found.push(val); }
}
} else if (mode.helperType && help[mode.helperType]) {
found.push(help[mode.helperType]);
} else if (help[mode.name]) {
found.push(help[mode.name]);
}
for (var i$1 = 0; i$1 < help._global.length; i$1++) {
var cur = help._global[i$1];
if (cur.pred(mode, this$1) && indexOf(found, cur.val) == -1)
{ found.push(cur.val); }
}
return found
},

getStateAfter: function(line, precise) {
var doc = this.doc;
line = clipLine(doc, line == null ? doc.first + doc.size - 1: line);
return getContextBefore(this, line + 1, precise).state
},

cursorCoords: function(start, mode) {
var pos, range$$1 = this.doc.sel.primary();
if (start == null) { pos = range$$1.head; }
else if (typeof start == "object") { pos = clipPos(this.doc, start); }
else { pos = start ? range$$1.from() : range$$1.to(); }
return cursorCoords(this, pos, mode || "page")
},

charCoords: function(pos, mode) {
return charCoords(this, clipPos(this.doc, pos), mode || "page")
},

coordsChar: function(coords, mode) {
coords = fromCoordSystem(this, coords, mode || "page");
return coordsChar(this, coords.left, coords.top)
},

lineAtHeight: function(height, mode) {
height = fromCoordSystem(this, {top: height, left: 0}, mode || "page").top;
return lineAtHeight(this.doc, height + this.display.viewOffset)
},
heightAtLine: function(line, mode, includeWidgets) {
var end = false, lineObj;
if (typeof line == "number") {
var last = this.doc.first + this.doc.size - 1;
if (line < this.doc.first) { line = this.doc.first; }
else if (line > last) { line = last; end = true; }
lineObj = getLine(this.doc, line);
} else {
lineObj = line;
}
return intoCoordSystem(this, lineObj, {top: 0, left: 0}, mode || "page", includeWidgets || end).top +
(end ? this.doc.height - heightAtLine(lineObj) : 0)
},

defaultTextHeight: function() { return textHeight(this.display) },
defaultCharWidth: function() { return charWidth(this.display) },

getViewport: function() { return {from: this.display.viewFrom, to: this.display.viewTo}},

addWidget: function(pos, node, scroll, vert, horiz) {
var display = this.display;
pos = cursorCoords(this, clipPos(this.doc, pos));
var top = pos.bottom, left = pos.left;
node.style.position = "absolute";
node.setAttribute("cm-ignore-events", "true");
this.display.input.setUneditable(node);
display.sizer.appendChild(node);
if (vert == "over") {
top = pos.top;
} else if (vert == "above" || vert == "near") {
var vspace = Math.max(display.wrapper.clientHeight, this.doc.height),
hspace = Math.max(display.sizer.clientWidth, display.lineSpace.clientWidth);
// Default to positioning above (if specified and possible); otherwise default to positioning below
if ((vert == 'above' || pos.bottom + node.offsetHeight > vspace) && pos.top > node.offsetHeight)
{ top = pos.top - node.offsetHeight; }
else if (pos.bottom + node.offsetHeight <= vspace)
{ top = pos.bottom; }
if (left + node.offsetWidth > hspace)
{ left = hspace - node.offsetWidth; }
}
node.style.top = top + "px";
node.style.left = node.style.right = "";
if (horiz == "right") {
left = display.sizer.clientWidth - node.offsetWidth;
node.style.right = "0px";
} else {
if (horiz == "left") { left = 0; }
else if (horiz == "middle") { left = (display.sizer.clientWidth - node.offsetWidth) / 2; }
node.style.left = left + "px";
}
if (scroll)
{ scrollIntoView(this, {left: left, top: top, right: left + node.offsetWidth, bottom: top + node.offsetHeight}); }
},

triggerOnKeyDown: methodOp(onKeyDown),
triggerOnKeyPress: methodOp(onKeyPress),
triggerOnKeyUp: onKeyUp,
triggerOnMouseDown: methodOp(onMouseDown),

execCommand: function(cmd) {
if (commands.hasOwnProperty(cmd))
{ return commands[cmd].call(null, this) }
},

triggerElectric: methodOp(function(text) { triggerElectric(this, text); }),

findPosH: function(from, amount, unit, visually) {
var this$1 = this;

var dir = 1;
if (amount < 0) { dir = -1; amount = -amount; }
var cur = clipPos(this.doc, from);
for (var i = 0; i < amount; ++i) {
cur = findPosH(this$1.doc, cur, dir, unit, visually);
if (cur.hitSide) { break }
}
return cur
},

moveH: methodOp(function(dir, unit) {
var this$1 = this;

this.extendSelectionsBy(function (range$$1) {
if (this$1.display.shift || this$1.doc.extend || range$$1.empty())
{ return findPosH(this$1.doc, range$$1.head, dir, unit, this$1.options.rtlMoveVisually) }
else
{ return dir < 0 ? range$$1.from() : range$$1.to() }
}, sel_move);
}),

deleteH: methodOp(function(dir, unit) {
var sel = this.doc.sel, doc = this.doc;
if (sel.somethingSelected())
{ doc.replaceSelection("", null, "+delete"); }
else
{ deleteNearSelection(this, function (range$$1) {
var other = findPosH(doc, range$$1.head, dir, unit, false);
return dir < 0 ? {from: other, to: range$$1.head} : {from: range$$1.head, to: other}
}); }
}),

findPosV: function(from, amount, unit, goalColumn) {
var this$1 = this;

var dir = 1, x = goalColumn;
if (amount < 0) { dir = -1; amount = -amount; }
var cur = clipPos(this.doc, from);
for (var i = 0; i < amount; ++i) {
var coords = cursorCoords(this$1, cur, "div");
if (x == null) { x = coords.left; }
else { coords.left = x; }
cur = findPosV(this$1, coords, dir, unit);
if (cur.hitSide) { break }
}
return cur
},

moveV: methodOp(function(dir, unit) {
var this$1 = this;

var doc = this.doc, goals = [];
var collapse = !this.display.shift && !doc.extend && doc.sel.somethingSelected();
doc.extendSelectionsBy(function (range$$1) {
if (collapse)
{ return dir < 0 ? range$$1.from() : range$$1.to() }
var headPos = cursorCoords(this$1, range$$1.head, "div");
if (range$$1.goalColumn != null) { headPos.left = range$$1.goalColumn; }
goals.push(headPos.left);
var pos = findPosV(this$1, headPos, dir, unit);
if (unit == "page" && range$$1 == doc.sel.primary())
{ addToScrollTop(this$1, charCoords(this$1, pos, "div").top - headPos.top); }
return pos
}, sel_move);
if (goals.length) { for (var i = 0; i < doc.sel.ranges.length; i++)
{ doc.sel.ranges[i].goalColumn = goals[i]; } }
}),

// Find the word at the given position (as returned by coordsChar).
findWordAt: function(pos) {
var doc = this.doc, line = getLine(doc, pos.line).text;
var start = pos.ch, end = pos.ch;
if (line) {
var helper = this.getHelper(pos, "wordChars");
if ((pos.sticky == "before" || end == line.length) && start) { --start; } else { ++end; }
var startChar = line.charAt(start);
var check = isWordChar(startChar, helper)
? function (ch) { return isWordChar(ch, helper); }
: /\s/.test(startChar) ? function (ch) { return /\s/.test(ch); }
: function (ch) { return (!/\s/.test(ch) && !isWordChar(ch)); };
while (start > 0 && check(line.charAt(start - 1))) { --start; }
while (end < line.length && check(line.charAt(end))) { ++end; }
}
return new Range(Pos(pos.line, start), Pos(pos.line, end))
},

toggleOverwrite: function(value) {
if (value != null && value == this.state.overwrite) { return }
if (this.state.overwrite = !this.state.overwrite)
{ addClass(this.display.cursorDiv, "CodeMirror-overwrite"); }
else
{ rmClass(this.display.cursorDiv, "CodeMirror-overwrite"); }

signal(this, "overwriteToggle", this, this.state.overwrite);
},
hasFocus: function() { return this.display.input.getField() == activeElt() },
isReadOnly: function() { return !!(this.options.readOnly || this.doc.cantEdit) },

scrollTo: methodOp(function (x, y) { scrollToCoords(this, x, y); }),
getScrollInfo: function() {
var scroller = this.display.scroller;
return {left: scroller.scrollLeft, top: scroller.scrollTop,
height: scroller.scrollHeight - scrollGap(this) - this.display.barHeight,
width: scroller.scrollWidth - scrollGap(this) - this.display.barWidth,
clientHeight: displayHeight(this), clientWidth: displayWidth(this)}
},

scrollIntoView: methodOp(function(range$$1, margin) {
if (range$$1 == null) {
range$$1 = {from: this.doc.sel.primary().head, to: null};
if (margin == null) { margin = this.options.cursorScrollMargin; }
} else if (typeof range$$1 == "number") {
range$$1 = {from: Pos(range$$1, 0), to: null};
} else if (range$$1.from == null) {
range$$1 = {from: range$$1, to: null};
}
if (!range$$1.to) { range$$1.to = range$$1.from; }
range$$1.margin = margin || 0;

if (range$$1.from.line != null) {
scrollToRange(this, range$$1);
} else {
scrollToCoordsRange(this, range$$1.from, range$$1.to, range$$1.margin);
}
}),

setSize: methodOp(function(width, height) {
var this$1 = this;

var interpret = function (val) { return typeof val == "number" || /^\d+$/.test(String(val)) ? val + "px" : val; };
if (width != null) { this.display.wrapper.style.width = interpret(width); }
if (height != null) { this.display.wrapper.style.height = interpret(height); }
if (this.options.lineWrapping) { clearLineMeasurementCache(this); }
var lineNo$$1 = this.display.viewFrom;
this.doc.iter(lineNo$$1, this.display.viewTo, function (line) {
if (line.widgets) { for (var i = 0; i < line.widgets.length; i++)
{ if (line.widgets[i].noHScroll) { regLineChange(this$1, lineNo$$1, "widget"); break } } }
++lineNo$$1;
});
this.curOp.forceUpdate = true;
signal(this, "refresh", this);
}),

operation: function(f){return runInOp(this, f)},
startOperation: function(){return startOperation(this)},
endOperation: function(){return endOperation(this)},

refresh: methodOp(function() {
var oldHeight = this.display.cachedTextHeight;
regChange(this);
this.curOp.forceUpdate = true;
clearCaches(this);
scrollToCoords(this, this.doc.scrollLeft, this.doc.scrollTop);
updateGutterSpace(this.display);
if (oldHeight == null || Math.abs(oldHeight - textHeight(this.display)) > .5)
{ estimateLineHeights(this); }
signal(this, "refresh", this);
}),

swapDoc: methodOp(function(doc) {
var old = this.doc;
old.cm = null;
// Cancel the current text selection if any (#5821)
if (this.state.selectingText) { this.state.selectingText(); }
attachDoc(this, doc);
clearCaches(this);
this.display.input.reset();
scrollToCoords(this, doc.scrollLeft, doc.scrollTop);
this.curOp.forceScroll = true;
signalLater(this, "swapDoc", this, old);
return old
}),

phrase: function(phraseText) {
var phrases = this.options.phrases;
return phrases && Object.prototype.hasOwnProperty.call(phrases, phraseText) ? phrases[phraseText] : phraseText
},

getInputField: function(){return this.display.input.getField()},
getWrapperElement: function(){return this.display.wrapper},
getScrollerElement: function(){return this.display.scroller},
getGutterElement: function(){return this.display.gutters}
};
eventMixin(CodeMirror);

CodeMirror.registerHelper = function(type, name, value) {
if (!helpers.hasOwnProperty(type)) { helpers[type] = CodeMirror[type] = {_global: []}; }
helpers[type][name] = value;
};
CodeMirror.registerGlobalHelper = function(type, name, predicate, value) {
CodeMirror.registerHelper(type, name, value);
helpers[type]._global.push({pred: predicate, val: value});
};
}

// Used for horizontal relative motion. Dir is -1 or 1 (left or
// right), unit can be "char", "column" (like char, but doesn't
// cross line boundaries), "word" (across next word), or "group" (to
// the start of next group of word or non-word-non-whitespace
// chars). The visually param controls whether, in right-to-left
// text, direction 1 means to move towards the next index in the
// string, or towards the character to the right of the current
// position. The resulting position will have a hitSide=true
// property if it reached the end of the document.
function findPosH(doc, pos, dir, unit, visually) {
var oldPos = pos;
var origDir = dir;
var lineObj = getLine(doc, pos.line);
function findNextLine() {
var l = pos.line + dir;
if (l < doc.first || l >= doc.first + doc.size) { return false }
pos = new Pos(l, pos.ch, pos.sticky);
return lineObj = getLine(doc, l)
}
function moveOnce(boundToLine) {
var next;
if (visually) {
next = moveVisually(doc.cm, lineObj, pos, dir);
} else {
next = moveLogically(lineObj, pos, dir);
}
if (next == null) {
if (!boundToLine && findNextLine())
{ pos = endOfLine(visually, doc.cm, lineObj, pos.line, dir); }
else
{ return false }
} else {
pos = next;
}
return true
}

if (unit == "char") {
moveOnce();
} else if (unit == "column") {
moveOnce(true);
} else if (unit == "word" || unit == "group") {
var sawType = null, group = unit == "group";
var helper = doc.cm && doc.cm.getHelper(pos, "wordChars");
for (var first = true;; first = false) {
if (dir < 0 && !moveOnce(!first)) { break }
var cur = lineObj.text.charAt(pos.ch) || "\n";
var type = isWordChar(cur, helper) ? "w"
: group && cur == "\n" ? "n"
: !group || /\s/.test(cur) ? null
: "p";
if (group && !first && !type) { type = "s"; }
if (sawType && sawType != type) {
if (dir < 0) {dir = 1; moveOnce(); pos.sticky = "after";}
break
}

if (type) { sawType = type; }
if (dir > 0 && !moveOnce(!first)) { break }
}
}
var result = skipAtomic(doc, pos, oldPos, origDir, true);
if (equalCursorPos(oldPos, result)) { result.hitSide = true; }
return result
}

// For relative vertical movement. Dir may be -1 or 1. Unit can be
// "page" or "line". The resulting position will have a hitSide=true
// property if it reached the end of the document.
function findPosV(cm, pos, dir, unit) {
var doc = cm.doc, x = pos.left, y;
if (unit == "page") {
var pageSize = Math.min(cm.display.wrapper.clientHeight, window.innerHeight || document.documentElement.clientHeight);
var moveAmount = Math.max(pageSize - .5 * textHeight(cm.display), 3);
y = (dir > 0 ? pos.bottom : pos.top) + dir * moveAmount;

} else if (unit == "line") {
y = dir > 0 ? pos.bottom + 3 : pos.top - 3;
}
var target;
for (;;) {
target = coordsChar(cm, x, y);
if (!target.outside) { break }
if (dir < 0 ? y <= 0 : y >= doc.height) { target.hitSide = true; break }
y += dir * 5;
}
return target
}

// CONTENTEDITABLE INPUT STYLE

var ContentEditableInput = function(cm) {
this.cm = cm;
this.lastAnchorNode = this.lastAnchorOffset = this.lastFocusNode = this.lastFocusOffset = null;
this.polling = new Delayed();
this.composing = null;
this.gracePeriod = false;
this.readDOMTimeout = null;
};

ContentEditableInput.prototype.init = function (display) {
var this$1 = this;

var input = this, cm = input.cm;
var div = input.div = display.lineDiv;
disableBrowserMagic(div, cm.options.spellcheck, cm.options.autocorrect, cm.options.autocapitalize);

on(div, "paste", function (e) {
if (signalDOMEvent(cm, e) || handlePaste(e, cm)) { return }
// IE doesn't fire input events, so we schedule a read for the pasted content in this way
if (ie_version <= 11) { setTimeout(operation(cm, function () { return this$1.updateFromDOM(); }), 20); }
});

on(div, "compositionstart", function (e) {
this$1.composing = {data: e.data, done: false};
});
on(div, "compositionupdate", function (e) {
if (!this$1.composing) { this$1.composing = {data: e.data, done: false}; }
});
on(div, "compositionend", function (e) {
if (this$1.composing) {
if (e.data != this$1.composing.data) { this$1.readFromDOMSoon(); }
this$1.composing.done = true;
}
});

on(div, "touchstart", function () { return input.forceCompositionEnd(); });

on(div, "input", function () {
if (!this$1.composing) { this$1.readFromDOMSoon(); }
});

function onCopyCut(e) {
if (signalDOMEvent(cm, e)) { return }
if (cm.somethingSelected()) {
setLastCopied({lineWise: false, text: cm.getSelections()});
if (e.type == "cut") { cm.replaceSelection("", null, "cut"); }
} else if (!cm.options.lineWiseCopyCut) {
return
} else {
var ranges = copyableRanges(cm);
setLastCopied({lineWise: true, text: ranges.text});
if (e.type == "cut") {
cm.operation(function () {
cm.setSelections(ranges.ranges, 0, sel_dontScroll);
cm.replaceSelection("", null, "cut");
});
}
}
if (e.clipboardData) {
e.clipboardData.clearData();
var content = lastCopied.text.join("\n");
// iOS exposes the clipboard API, but seems to discard content inserted into it
e.clipboardData.setData("Text", content);
if (e.clipboardData.getData("Text") == content) {
e.preventDefault();
return
}
}
// Old-fashioned briefly-focus-a-textarea hack
var kludge = hiddenTextarea(), te = kludge.firstChild;
cm.display.lineSpace.insertBefore(kludge, cm.display.lineSpace.firstChild);
te.value = lastCopied.text.join("\n");
var hadFocus = document.activeElement;
selectInput(te);
setTimeout(function () {
cm.display.lineSpace.removeChild(kludge);
hadFocus.focus();
if (hadFocus == div) { input.showPrimarySelection(); }
}, 50);
}
on(div, "copy", onCopyCut);
on(div, "cut", onCopyCut);
};

ContentEditableInput.prototype.prepareSelection = function () {
var result = prepareSelection(this.cm, false);
result.focus = this.cm.state.focused;
return result
};

ContentEditableInput.prototype.showSelection = function (info, takeFocus) {
if (!info || !this.cm.display.view.length) { return }
if (info.focus || takeFocus) { this.showPrimarySelection(); }
this.showMultipleSelections(info);
};

ContentEditableInput.prototype.getSelection = function () {
return this.cm.display.wrapper.ownerDocument.getSelection()
};

ContentEditableInput.prototype.showPrimarySelection = function () {
var sel = this.getSelection(), cm = this.cm, prim = cm.doc.sel.primary();
var from = prim.from(), to = prim.to();

if (cm.display.viewTo == cm.display.viewFrom || from.line >= cm.display.viewTo || to.line < cm.display.viewFrom) {
sel.removeAllRanges();
return
}

var curAnchor = domToPos(cm, sel.anchorNode, sel.anchorOffset);
var curFocus = domToPos(cm, sel.focusNode, sel.focusOffset);
if (curAnchor && !curAnchor.bad && curFocus && !curFocus.bad &&
cmp(minPos(curAnchor, curFocus), from) == 0 &&
cmp(maxPos(curAnchor, curFocus), to) == 0)
{ return }

var view = cm.display.view;
var start = (from.line >= cm.display.viewFrom && posToDOM(cm, from)) ||
{node: view[0].measure.map[2], offset: 0};
var end = to.line < cm.display.viewTo && posToDOM(cm, to);
if (!end) {
var measure = view[view.length - 1].measure;
var map$$1 = measure.maps ? measure.maps[measure.maps.length - 1] : measure.map;
end = {node: map$$1[map$$1.length - 1], offset: map$$1[map$$1.length - 2] - map$$1[map$$1.length - 3]};
}

if (!start || !end) {
sel.removeAllRanges();
return
}

var old = sel.rangeCount && sel.getRangeAt(0), rng;
try { rng = range(start.node, start.offset, end.offset, end.node); }
catch(e) {} // Our model of the DOM might be outdated, in which case the range we try to set can be impossible
if (rng) {
if (!gecko && cm.state.focused) {
sel.collapse(start.node, start.offset);
if (!rng.collapsed) {
sel.removeAllRanges();
sel.addRange(rng);
}
} else {
sel.removeAllRanges();
sel.addRange(rng);
}
if (old && sel.anchorNode == null) { sel.addRange(old); }
else if (gecko) { this.startGracePeriod(); }
}
this.rememberSelection();
};

ContentEditableInput.prototype.startGracePeriod = function () {
var this$1 = this;

clearTimeout(this.gracePeriod);
this.gracePeriod = setTimeout(function () {
this$1.gracePeriod = false;
if (this$1.selectionChanged())
{ this$1.cm.operation(function () { return this$1.cm.curOp.selectionChanged = true; }); }
}, 20);
};

ContentEditableInput.prototype.showMultipleSelections = function (info) {
removeChildrenAndAdd(this.cm.display.cursorDiv, info.cursors);
removeChildrenAndAdd(this.cm.display.selectionDiv, info.selection);
};

ContentEditableInput.prototype.rememberSelection = function () {
var sel = this.getSelection();
this.lastAnchorNode = sel.anchorNode; this.lastAnchorOffset = sel.anchorOffset;
this.lastFocusNode = sel.focusNode; this.lastFocusOffset = sel.focusOffset;
};

ContentEditableInput.prototype.selectionInEditor = function () {
var sel = this.getSelection();
if (!sel.rangeCount) { return false }
var node = sel.getRangeAt(0).commonAncestorContainer;
return contains(this.div, node)
};

ContentEditableInput.prototype.focus = function () {
if (this.cm.options.readOnly != "nocursor") {
if (!this.selectionInEditor())
{ this.showSelection(this.prepareSelection(), true); }
this.div.focus();
}
};
ContentEditableInput.prototype.blur = function () { this.div.blur(); };
ContentEditableInput.prototype.getField = function () { return this.div };

ContentEditableInput.prototype.supportsTouch = function () { return true };

ContentEditableInput.prototype.receivedFocus = function () {
var input = this;
if (this.selectionInEditor())
{ this.pollSelection(); }
else
{ runInOp(this.cm, function () { return input.cm.curOp.selectionChanged = true; }); }

function poll() {
if (input.cm.state.focused) {
input.pollSelection();
input.polling.set(input.cm.options.pollInterval, poll);
}
}
this.polling.set(this.cm.options.pollInterval, poll);
};

ContentEditableInput.prototype.selectionChanged = function () {
var sel = this.getSelection();
return sel.anchorNode != this.lastAnchorNode || sel.anchorOffset != this.lastAnchorOffset ||
sel.focusNode != this.lastFocusNode || sel.focusOffset != this.lastFocusOffset
};

ContentEditableInput.prototype.pollSelection = function () {
if (this.readDOMTimeout != null || this.gracePeriod || !this.selectionChanged()) { return }
var sel = this.getSelection(), cm = this.cm;
// On Android Chrome (version 56, at least), backspacing into an
// uneditable block element will put the cursor in that element,
// and then, because it's not editable, hide the virtual keyboard.
// Because Android doesn't allow us to actually detect backspace
// presses in a sane way, this code checks for when that happens
// and simulates a backspace press in this case.
if (android && chrome && this.cm.display.gutterSpecs.length && isInGutter(sel.anchorNode)) {
this.cm.triggerOnKeyDown({type: "keydown", keyCode: 8, preventDefault: Math.abs});
this.blur();
this.focus();
return
}
if (this.composing) { return }
this.rememberSelection();
var anchor = domToPos(cm, sel.anchorNode, sel.anchorOffset);
var head = domToPos(cm, sel.focusNode, sel.focusOffset);
if (anchor && head) { runInOp(cm, function () {
setSelection(cm.doc, simpleSelection(anchor, head), sel_dontScroll);
if (anchor.bad || head.bad) { cm.curOp.selectionChanged = true; }
}); }
};

ContentEditableInput.prototype.pollContent = function () {
if (this.readDOMTimeout != null) {
clearTimeout(this.readDOMTimeout);
this.readDOMTimeout = null;
}

var cm = this.cm, display = cm.display, sel = cm.doc.sel.primary();
var from = sel.from(), to = sel.to();
if (from.ch == 0 && from.line > cm.firstLine())
{ from = Pos(from.line - 1, getLine(cm.doc, from.line - 1).length); }
if (to.ch == getLine(cm.doc, to.line).text.length && to.line < cm.lastLine())
{ to = Pos(to.line + 1, 0); }
if (from.line < display.viewFrom || to.line > display.viewTo - 1) { return false }

var fromIndex, fromLine, fromNode;
if (from.line == display.viewFrom || (fromIndex = findViewIndex(cm, from.line)) == 0) {
fromLine = lineNo(display.view[0].line);
fromNode = display.view[0].node;
} else {
fromLine = lineNo(display.view[fromIndex].line);
fromNode = display.view[fromIndex - 1].node.nextSibling;
}
var toIndex = findViewIndex(cm, to.line);
var toLine, toNode;
if (toIndex == display.view.length - 1) {
toLine = display.viewTo - 1;
toNode = display.lineDiv.lastChild;
} else {
toLine = lineNo(display.view[toIndex + 1].line) - 1;
toNode = display.view[toIndex + 1].node.previousSibling;
}

if (!fromNode) { return false }
var newText = cm.doc.splitLines(domTextBetween(cm, fromNode, toNode, fromLine, toLine));
var oldText = getBetween(cm.doc, Pos(fromLine, 0), Pos(toLine, getLine(cm.doc, toLine).text.length));
while (newText.length > 1 && oldText.length > 1) {
if (lst(newText) == lst(oldText)) { newText.pop(); oldText.pop(); toLine--; }
else if (newText[0] == oldText[0]) { newText.shift(); oldText.shift(); fromLine++; }
else { break }
}

var cutFront = 0, cutEnd = 0;
var newTop = newText[0], oldTop = oldText[0], maxCutFront = Math.min(newTop.length, oldTop.length);
while (cutFront < maxCutFront && newTop.charCodeAt(cutFront) == oldTop.charCodeAt(cutFront))
{ ++cutFront; }
var newBot = lst(newText), oldBot = lst(oldText);
var maxCutEnd = Math.min(newBot.length - (newText.length == 1 ? cutFront : 0),
oldBot.length - (oldText.length == 1 ? cutFront : 0));
while (cutEnd < maxCutEnd &&
newBot.charCodeAt(newBot.length - cutEnd - 1) == oldBot.charCodeAt(oldBot.length - cutEnd - 1))
{ ++cutEnd; }
// Try to move start of change to start of selection if ambiguous
if (newText.length == 1 && oldText.length == 1 && fromLine == from.line) {
while (cutFront && cutFront > from.ch &&
newBot.charCodeAt(newBot.length - cutEnd - 1) == oldBot.charCodeAt(oldBot.length - cutEnd - 1)) {
cutFront--;
cutEnd++;
}
}

newText[newText.length - 1] = newBot.slice(0, newBot.length - cutEnd).replace(/^\u200b+/, "");
newText[0] = newText[0].slice(cutFront).replace(/\u200b+$/, "");

var chFrom = Pos(fromLine, cutFront);
var chTo = Pos(toLine, oldText.length ? lst(oldText).length - cutEnd : 0);
if (newText.length > 1 || newText[0] || cmp(chFrom, chTo)) {
replaceRange(cm.doc, newText, chFrom, chTo, "+input");
return true
}
};

ContentEditableInput.prototype.ensurePolled = function () {
this.forceCompositionEnd();
};
ContentEditableInput.prototype.reset = function () {
this.forceCompositionEnd();
};
ContentEditableInput.prototype.forceCompositionEnd = function () {
if (!this.composing) { return }
clearTimeout(this.readDOMTimeout);
this.composing = null;
this.updateFromDOM();
this.div.blur();
this.div.focus();
};
ContentEditableInput.prototype.readFromDOMSoon = function () {
var this$1 = this;

if (this.readDOMTimeout != null) { return }
this.readDOMTimeout = setTimeout(function () {
this$1.readDOMTimeout = null;
if (this$1.composing) {
if (this$1.composing.done) { this$1.composing = null; }
else { return }
}
this$1.updateFromDOM();
}, 80);
};

ContentEditableInput.prototype.updateFromDOM = function () {
var this$1 = this;

if (this.cm.isReadOnly() || !this.pollContent())
{ runInOp(this.cm, function () { return regChange(this$1.cm); }); }
};

ContentEditableInput.prototype.setUneditable = function (node) {
node.contentEditable = "false";
};

ContentEditableInput.prototype.onKeyPress = function (e) {
if (e.charCode == 0 || this.composing) { return }
e.preventDefault();
if (!this.cm.isReadOnly())
{ operation(this.cm, applyTextInput)(this.cm, String.fromCharCode(e.charCode == null ? e.keyCode : e.charCode), 0); }
};

ContentEditableInput.prototype.readOnlyChanged = function (val) {
this.div.contentEditable = String(val != "nocursor");
};

ContentEditableInput.prototype.onContextMenu = function () {};
ContentEditableInput.prototype.resetPosition = function () {};

ContentEditableInput.prototype.needsContentAttribute = true;

function posToDOM(cm, pos) {
var view = findViewForLine(cm, pos.line);
if (!view || view.hidden) { return null }
var line = getLine(cm.doc, pos.line);
var info = mapFromLineView(view, line, pos.line);

var order = getOrder(line, cm.doc.direction), side = "left";
if (order) {
var partPos = getBidiPartAt(order, pos.ch);
side = partPos % 2 ? "right" : "left";
}
var result = nodeAndOffsetInLineMap(info.map, pos.ch, side);
result.offset = result.collapse == "right" ? result.end : result.start;
return result
}

function isInGutter(node) {
for (var scan = node; scan; scan = scan.parentNode)
{ if (/CodeMirror-gutter-wrapper/.test(scan.className)) { return true } }
return false
}

function badPos(pos, bad) { if (bad) { pos.bad = true; } return pos }

function domTextBetween(cm, from, to, fromLine, toLine) {
var text = "", closing = false, lineSep = cm.doc.lineSeparator(), extraLinebreak = false;
function recognizeMarker(id) { return function (marker) { return marker.id == id; } }
function close() {
if (closing) {
text += lineSep;
if (extraLinebreak) { text += lineSep; }
closing = extraLinebreak = false;
}
}
function addText(str) {
if (str) {
close();
text += str;
}
}
function walk(node) {
if (node.nodeType == 1) {
var cmText = node.getAttribute("cm-text");
if (cmText) {
addText(cmText);
return
}
var markerID = node.getAttribute("cm-marker"), range$$1;
if (markerID) {
var found = cm.findMarks(Pos(fromLine, 0), Pos(toLine + 1, 0), recognizeMarker(+markerID));
if (found.length && (range$$1 = found[0].find(0)))
{ addText(getBetween(cm.doc, range$$1.from, range$$1.to).join(lineSep)); }
return
}
if (node.getAttribute("contenteditable") == "false") { return }
var isBlock = /^(pre|div|p|li|table|br)$/i.test(node.nodeName);
if (!/^br$/i.test(node.nodeName) && node.textContent.length == 0) { return }

if (isBlock) { close(); }
for (var i = 0; i < node.childNodes.length; i++)
{ walk(node.childNodes[i]); }

if (/^(pre|p)$/i.test(node.nodeName)) { extraLinebreak = true; }
if (isBlock) { closing = true; }
} else if (node.nodeType == 3) {
addText(node.nodeValue.replace(/\u200b/g, "").replace(/\u00a0/g, " "));
}
}
for (;;) {
walk(from);
if (from == to) { break }
from = from.nextSibling;
extraLinebreak = false;
}
return text
}

function domToPos(cm, node, offset) {
var lineNode;
if (node == cm.display.lineDiv) {
lineNode = cm.display.lineDiv.childNodes[offset];
if (!lineNode) { return badPos(cm.clipPos(Pos(cm.display.viewTo - 1)), true) }
node = null; offset = 0;
} else {
for (lineNode = node;; lineNode = lineNode.parentNode) {
if (!lineNode || lineNode == cm.display.lineDiv) { return null }
if (lineNode.parentNode && lineNode.parentNode == cm.display.lineDiv) { break }
}
}
for (var i = 0; i < cm.display.view.length; i++) {
var lineView = cm.display.view[i];
if (lineView.node == lineNode)
{ return locateNodeInLineView(lineView, node, offset) }
}
}

function locateNodeInLineView(lineView, node, offset) {
var wrapper = lineView.text.firstChild, bad = false;
if (!node || !contains(wrapper, node)) { return badPos(Pos(lineNo(lineView.line), 0), true) }
if (node == wrapper) {
bad = true;
node = wrapper.childNodes[offset];
offset = 0;
if (!node) {
var line = lineView.rest ? lst(lineView.rest) : lineView.line;
return badPos(Pos(lineNo(line), line.text.length), bad)
}
}

var textNode = node.nodeType == 3 ? node : null, topNode = node;
if (!textNode && node.childNodes.length == 1 && node.firstChild.nodeType == 3) {
textNode = node.firstChild;
if (offset) { offset = textNode.nodeValue.length; }
}
while (topNode.parentNode != wrapper) { topNode = topNode.parentNode; }
var measure = lineView.measure, maps = measure.maps;

function find(textNode, topNode, offset) {
for (var i = -1; i < (maps ? maps.length : 0); i++) {
var map$$1 = i < 0 ? measure.map : maps[i];
for (var j = 0; j < map$$1.length; j += 3) {
var curNode = map$$1[j + 2];
if (curNode == textNode || curNode == topNode) {
var line = lineNo(i < 0 ? lineView.line : lineView.rest[i]);
var ch = map$$1[j] + offset;
if (offset < 0 || curNode != textNode) { ch = map$$1[j + (offset ? 1 : 0)]; }
return Pos(line, ch)
}
}
}
}
var found = find(textNode, topNode, offset);
if (found) { return badPos(found, bad) }

// FIXME this is all really shaky. might handle the few cases it needs to handle, but likely to cause problems
for (var after = topNode.nextSibling, dist = textNode ? textNode.nodeValue.length - offset : 0; after; after = after.nextSibling) {
found = find(after, after.firstChild, 0);
if (found)
{ return badPos(Pos(found.line, found.ch - dist), bad) }
else
{ dist += after.textContent.length; }
}
for (var before = topNode.previousSibling, dist$1 = offset; before; before = before.previousSibling) {
found = find(before, before.firstChild, -1);
if (found)
{ return badPos(Pos(found.line, found.ch + dist$1), bad) }
else
{ dist$1 += before.textContent.length; }
}
}

// TEXTAREA INPUT STYLE

var TextareaInput = function(cm) {
this.cm = cm;
// See input.poll and input.reset
this.prevInput = "";

// Flag that indicates whether we expect input to appear real soon
// now (after some event like 'keypress' or 'input') and are
// polling intensively.
this.pollingFast = false;
// Self-resetting timeout for the poller
this.polling = new Delayed();
// Used to work around IE issue with selection being forgotten when focus moves away from textarea
this.hasSelection = false;
this.composing = null;
};

TextareaInput.prototype.init = function (display) {
var this$1 = this;

var input = this, cm = this.cm;
this.createField(display);
var te = this.textarea;

display.wrapper.insertBefore(this.wrapper, display.wrapper.firstChild);

// Needed to hide big blue blinking cursor on Mobile Safari (doesn't seem to work in iOS 8 anymore)
if (ios) { te.style.width = "0px"; }

on(te, "input", function () {
if (ie && ie_version >= 9 && this$1.hasSelection) { this$1.hasSelection = null; }
input.poll();
});

on(te, "paste", function (e) {
if (signalDOMEvent(cm, e) || handlePaste(e, cm)) { return }

cm.state.pasteIncoming = +new Date;
input.fastPoll();
});

function prepareCopyCut(e) {
if (signalDOMEvent(cm, e)) { return }
if (cm.somethingSelected()) {
setLastCopied({lineWise: false, text: cm.getSelections()});
} else if (!cm.options.lineWiseCopyCut) {
return
} else {
var ranges = copyableRanges(cm);
setLastCopied({lineWise: true, text: ranges.text});
if (e.type == "cut") {
cm.setSelections(ranges.ranges, null, sel_dontScroll);
} else {
input.prevInput = "";
te.value = ranges.text.join("\n");
selectInput(te);
}
}
if (e.type == "cut") { cm.state.cutIncoming = +new Date; }
}
on(te, "cut", prepareCopyCut);
on(te, "copy", prepareCopyCut);

on(display.scroller, "paste", function (e) {
if (eventInWidget(display, e) || signalDOMEvent(cm, e)) { return }
if (!te.dispatchEvent) {
cm.state.pasteIncoming = +new Date;
input.focus();
return
}
var event = new Event("paste");
event.clipboardData = e.clipboardData;
te.dispatchEvent(event);
});

// Prevent normal selection in the editor (we handle our own)
on(display.lineSpace, "selectstart", function (e) {
if (!eventInWidget(display, e)) { e_preventDefault(e); }
});

on(te, "compositionstart", function () {
var start = cm.getCursor("from");
if (input.composing) { input.composing.range.clear(); }
input.composing = {
start: start,
range: cm.markText(start, cm.getCursor("to"), {className: "CodeMirror-composing"})
};
});
on(te, "compositionend", function () {
if (input.composing) {
input.poll();
input.composing.range.clear();
input.composing = null;
}
});
};

TextareaInput.prototype.createField = function (_display) {
// Wraps and hides input textarea
this.wrapper = hiddenTextarea();
// The semihidden textarea that is focused when the editor is
// focused, and receives input.
this.textarea = this.wrapper.firstChild;
};

TextareaInput.prototype.prepareSelection = function () {
// Redraw the selection and/or cursor
var cm = this.cm, display = cm.display, doc = cm.doc;
var result = prepareSelection(cm);

// Move the hidden textarea near the cursor to prevent scrolling artifacts
if (cm.options.moveInputWithCursor) {
var headPos = cursorCoords(cm, doc.sel.primary().head, "div");
var wrapOff = display.wrapper.getBoundingClientRect(), lineOff = display.lineDiv.getBoundingClientRect();
result.teTop = Math.max(0, Math.min(display.wrapper.clientHeight - 10,
headPos.top + lineOff.top - wrapOff.top));
result.teLeft = Math.max(0, Math.min(display.wrapper.clientWidth - 10,
headPos.left + lineOff.left - wrapOff.left));
}

return result
};

TextareaInput.prototype.showSelection = function (drawn) {
var cm = this.cm, display = cm.display;
removeChildrenAndAdd(display.cursorDiv, drawn.cursors);
removeChildrenAndAdd(display.selectionDiv, drawn.selection);
if (drawn.teTop != null) {
this.wrapper.style.top = drawn.teTop + "px";
this.wrapper.style.left = drawn.teLeft + "px";
}
};

// Reset the input to correspond to the selection (or to be empty,
// when not typing and nothing is selected)
TextareaInput.prototype.reset = function (typing) {
if (this.contextMenuPending || this.composing) { return }
var cm = this.cm;
if (cm.somethingSelected()) {
this.prevInput = "";
var content = cm.getSelection();
this.textarea.value = content;
if (cm.state.focused) { selectInput(this.textarea); }
if (ie && ie_version >= 9) { this.hasSelection = content; }
} else if (!typing) {
this.prevInput = this.textarea.value = "";
if (ie && ie_version >= 9) { this.hasSelection = null; }
}
};

TextareaInput.prototype.getField = function () { return this.textarea };

TextareaInput.prototype.supportsTouch = function () { return false };

TextareaInput.prototype.focus = function () {
if (this.cm.options.readOnly != "nocursor" && (!mobile || activeElt() != this.textarea)) {
try { this.textarea.focus(); }
catch (e) {} // IE8 will throw if the textarea is display: none or not in DOM
}
};

TextareaInput.prototype.blur = function () { this.textarea.blur(); };

TextareaInput.prototype.resetPosition = function () {
this.wrapper.style.top = this.wrapper.style.left = 0;
};

TextareaInput.prototype.receivedFocus = function () { this.slowPoll(); };

// Poll for input changes, using the normal rate of polling. This
// runs as long as the editor is focused.
TextareaInput.prototype.slowPoll = function () {
var this$1 = this;

if (this.pollingFast) { return }
this.polling.set(this.cm.options.pollInterval, function () {
this$1.poll();
if (this$1.cm.state.focused) { this$1.slowPoll(); }
});
};

// When an event has just come in that is likely to add or change
// something in the input textarea, we poll faster, to ensure that
// the change appears on the screen quickly.
TextareaInput.prototype.fastPoll = function () {
var missed = false, input = this;
input.pollingFast = true;
function p() {
var changed = input.poll();
if (!changed && !missed) {missed = true; input.polling.set(60, p);}
else {input.pollingFast = false; input.slowPoll();}
}
input.polling.set(20, p);
};

// Read input from the textarea, and update the document to match.
// When something is selected, it is present in the textarea, and
// selected (unless it is huge, in which case a placeholder is
// used). When nothing is selected, the cursor sits after previously
// seen text (can be empty), which is stored in prevInput (we must
// not reset the textarea when typing, because that breaks IME).
TextareaInput.prototype.poll = function () {
var this$1 = this;

var cm = this.cm, input = this.textarea, prevInput = this.prevInput;
// Since this is called a *lot*, try to bail out as cheaply as
// possible when it is clear that nothing happened. hasSelection
// will be the case when there is a lot of text in the textarea,
// in which case reading its value would be expensive.
if (this.contextMenuPending || !cm.state.focused ||
(hasSelection(input) && !prevInput && !this.composing) ||
cm.isReadOnly() || cm.options.disableInput || cm.state.keySeq)
{ return false }

var text = input.value;
// If nothing changed, bail.
if (text == prevInput && !cm.somethingSelected()) { return false }
// Work around nonsensical selection resetting in IE9/10, and
// inexplicable appearance of private area unicode characters on
// some key combos in Mac (#2689).
if (ie && ie_version >= 9 && this.hasSelection === text ||
mac && /[\uf700-\uf7ff]/.test(text)) {
cm.display.input.reset();
return false
}

if (cm.doc.sel == cm.display.selForContextMenu) {
var first = text.charCodeAt(0);
if (first == 0x200b && !prevInput) { prevInput = "\u200b"; }
if (first == 0x21da) { this.reset(); return this.cm.execCommand("undo") }
}
// Find the part of the input that is actually new
var same = 0, l = Math.min(prevInput.length, text.length);
while (same < l && prevInput.charCodeAt(same) == text.charCodeAt(same)) { ++same; }

runInOp(cm, function () {
applyTextInput(cm, text.slice(same), prevInput.length - same,
null, this$1.composing ? "*compose" : null);

// Don't leave long text in the textarea, since it makes further polling slow
if (text.length > 1000 || text.indexOf("\n") > -1) { input.value = this$1.prevInput = ""; }
else { this$1.prevInput = text; }

if (this$1.composing) {
this$1.composing.range.clear();
this$1.composing.range = cm.markText(this$1.composing.start, cm.getCursor("to"),
{className: "CodeMirror-composing"});
}
});
return true
};

TextareaInput.prototype.ensurePolled = function () {
if (this.pollingFast && this.poll()) { this.pollingFast = false; }
};

TextareaInput.prototype.onKeyPress = function () {
if (ie && ie_version >= 9) { this.hasSelection = null; }
this.fastPoll();
};

TextareaInput.prototype.onContextMenu = function (e) {
var input = this, cm = input.cm, display = cm.display, te = input.textarea;
if (input.contextMenuPending) { input.contextMenuPending(); }
var pos = posFromMouse(cm, e), scrollPos = display.scroller.scrollTop;
if (!pos || presto) { return } // Opera is difficult.

// Reset the current text selection only if the click is done outside of the selection
// and 'resetSelectionOnContextMenu' option is true.
var reset = cm.options.resetSelectionOnContextMenu;
if (reset && cm.doc.sel.contains(pos) == -1)
{ operation(cm, setSelection)(cm.doc, simpleSelection(pos), sel_dontScroll); }

var oldCSS = te.style.cssText, oldWrapperCSS = input.wrapper.style.cssText;
var wrapperBox = input.wrapper.offsetParent.getBoundingClientRect();
input.wrapper.style.cssText = "position: static";
te.style.cssText = "position: absolute; width: 30px; height: 30px;\n      top: " + (e.clientY - wrapperBox.top - 5) + "px; left: " + (e.clientX - wrapperBox.left - 5) + "px;\n      z-index: 1000; background: " + (ie ? "rgba(255, 255, 255, .05)" : "transparent") + ";\n      outline: none; border-width: 0; outline: none; overflow: hidden; opacity: .05; filter: alpha(opacity=5);";
var oldScrollY;
if (webkit) { oldScrollY = window.scrollY; } // Work around Chrome issue (#2712)
display.input.focus();
if (webkit) { window.scrollTo(null, oldScrollY); }
display.input.reset();
// Adds "Select all" to context menu in FF
if (!cm.somethingSelected()) { te.value = input.prevInput = " "; }
input.contextMenuPending = rehide;
display.selForContextMenu = cm.doc.sel;
clearTimeout(display.detectingSelectAll);

// Select-all will be greyed out if there's nothing to select, so
// this adds a zero-width space so that we can later check whether
// it got selected.
function prepareSelectAllHack() {
if (te.selectionStart != null) {
var selected = cm.somethingSelected();
var extval = "\u200b" + (selected ? te.value : "");
te.value = "\u21da"; // Used to catch context-menu undo
te.value = extval;
input.prevInput = selected ? "" : "\u200b";
te.selectionStart = 1; te.selectionEnd = extval.length;
// Re-set this, in case some other handler touched the
// selection in the meantime.
display.selForContextMenu = cm.doc.sel;
}
}
function rehide() {
if (input.contextMenuPending != rehide) { return }
input.contextMenuPending = false;
input.wrapper.style.cssText = oldWrapperCSS;
te.style.cssText = oldCSS;
if (ie && ie_version < 9) { display.scrollbars.setScrollTop(display.scroller.scrollTop = scrollPos); }

// Try to detect the user choosing select-all
if (te.selectionStart != null) {
if (!ie || (ie && ie_version < 9)) { prepareSelectAllHack(); }
var i = 0, poll = function () {
if (display.selForContextMenu == cm.doc.sel && te.selectionStart == 0 &&
te.selectionEnd > 0 && input.prevInput == "\u200b") {
operation(cm, selectAll)(cm);
} else if (i++ < 10) {
display.detectingSelectAll = setTimeout(poll, 500);
} else {
display.selForContextMenu = null;
display.input.reset();
}
};
display.detectingSelectAll = setTimeout(poll, 200);
}
}

if (ie && ie_version >= 9) { prepareSelectAllHack(); }
if (captureRightClick) {
e_stop(e);
var mouseup = function () {
off(window, "mouseup", mouseup);
setTimeout(rehide, 20);
};
on(window, "mouseup", mouseup);
} else {
setTimeout(rehide, 50);
}
};

TextareaInput.prototype.readOnlyChanged = function (val) {
if (!val) { this.reset(); }
this.textarea.disabled = val == "nocursor";
};

TextareaInput.prototype.setUneditable = function () {};

TextareaInput.prototype.needsContentAttribute = false;

function fromTextArea(textarea, options) {
options = options ? copyObj(options) : {};
options.value = textarea.value;
if (!options.tabindex && textarea.tabIndex)
{ options.tabindex = textarea.tabIndex; }
if (!options.placeholder && textarea.placeholder)
{ options.placeholder = textarea.placeholder; }
// Set autofocus to true if this textarea is focused, or if it has
// autofocus and no other element is focused.
if (options.autofocus == null) {
var hasFocus = activeElt();
options.autofocus = hasFocus == textarea ||
textarea.getAttribute("autofocus") != null && hasFocus == document.body;
}

function save() {textarea.value = cm.getValue();}

var realSubmit;
if (textarea.form) {
on(textarea.form, "submit", save);
// Deplorable hack to make the submit method do the right thing.
if (!options.leaveSubmitMethodAlone) {
var form = textarea.form;
realSubmit = form.submit;
try {
var wrappedSubmit = form.submit = function () {
save();
form.submit = realSubmit;
form.submit();
form.submit = wrappedSubmit;
};
} catch(e) {}
}
}

options.finishInit = function (cm) {
cm.save = save;
cm.getTextArea = function () { return textarea; };
cm.toTextArea = function () {
cm.toTextArea = isNaN; // Prevent this from being ran twice
save();
textarea.parentNode.removeChild(cm.getWrapperElement());
textarea.style.display = "";
if (textarea.form) {
off(textarea.form, "submit", save);
if (typeof textarea.form.submit == "function")
{ textarea.form.submit = realSubmit; }
}
};
};

textarea.style.display = "none";
var cm = CodeMirror(function (node) { return textarea.parentNode.insertBefore(node, textarea.nextSibling); },
options);
return cm
}

function addLegacyProps(CodeMirror) {
CodeMirror.off = off;
CodeMirror.on = on;
CodeMirror.wheelEventPixels = wheelEventPixels;
CodeMirror.Doc = Doc;
CodeMirror.splitLines = splitLinesAuto;
CodeMirror.countColumn = countColumn;
CodeMirror.findColumn = findColumn;
CodeMirror.isWordChar = isWordCharBasic;
CodeMirror.Pass = Pass;
CodeMirror.signal = signal;
CodeMirror.Line = Line;
CodeMirror.changeEnd = changeEnd;
CodeMirror.scrollbarModel = scrollbarModel;
CodeMirror.Pos = Pos;
CodeMirror.cmpPos = cmp;
CodeMirror.modes = modes;
CodeMirror.mimeModes = mimeModes;
CodeMirror.resolveMode = resolveMode;
CodeMirror.getMode = getMode;
CodeMirror.modeExtensions = modeExtensions;
CodeMirror.extendMode = extendMode;
CodeMirror.copyState = copyState;
CodeMirror.startState = startState;
CodeMirror.innerMode = innerMode;
CodeMirror.commands = commands;
CodeMirror.keyMap = keyMap;
CodeMirror.keyName = keyName;
CodeMirror.isModifierKey = isModifierKey;
CodeMirror.lookupKey = lookupKey;
CodeMirror.normalizeKeyMap = normalizeKeyMap;
CodeMirror.StringStream = StringStream;
CodeMirror.SharedTextMarker = SharedTextMarker;
CodeMirror.TextMarker = TextMarker;
CodeMirror.LineWidget = LineWidget;
CodeMirror.e_preventDefault = e_preventDefault;
CodeMirror.e_stopPropagation = e_stopPropagation;
CodeMirror.e_stop = e_stop;
CodeMirror.addClass = addClass;
CodeMirror.contains = contains;
CodeMirror.rmClass = rmClass;
CodeMirror.keyNames = keyNames;
}

// EDITOR CONSTRUCTOR

defineOptions(CodeMirror);

addEditorMethods(CodeMirror);

// Set up methods on CodeMirror's prototype to redirect to the editor's document.
var dontDelegate = "iter insert remove copy getEditor constructor".split(" ");
for (var prop in Doc.prototype) { if (Doc.prototype.hasOwnProperty(prop) && indexOf(dontDelegate, prop) < 0)
{ CodeMirror.prototype[prop] = (function(method) {
return function() {return method.apply(this.doc, arguments)}
})(Doc.prototype[prop]); } }

eventMixin(Doc);
CodeMirror.inputStyles = {"textarea": TextareaInput, "contenteditable": ContentEditableInput};

// Extra arguments are stored as the mode's dependencies, which is
// used by (legacy) mechanisms like loadmode.js to automatically
// load a mode. (Preferred mechanism is the require/define calls.)
CodeMirror.defineMode = function(name/*, mode, â€¦*/) {
if (!CodeMirror.defaults.mode && name != "null") { CodeMirror.defaults.mode = name; }
defineMode.apply(this, arguments);
};

CodeMirror.defineMIME = defineMIME;

// Minimal default mode.
CodeMirror.defineMode("null", function () { return ({token: function (stream) { return stream.skipToEnd(); }}); });
CodeMirror.defineMIME("text/plain", "null");

// EXTENSIONS

CodeMirror.defineExtension = function (name, func) {
	CodeMirror.prototype[name] = func;
};
CodeMirror.defineDocExtension = function (name, func) {
	Doc.prototype[name] = func;
};

CodeMirror.fromTextArea = fromTextArea;

addLegacyProps(CodeMirror);

CodeMirror.version = "5.46.0";

return CodeMirror;

})));





// CodeMirror, copyright (c) by Marijn Haverbeke and others
// Distributed under an MIT license: https://codemirror.net/LICENSE

(function(mod) {
  if (typeof exports == "object" && typeof module == "object") // CommonJS
    mod(require("../../lib/codemirror"));
  else if (typeof define == "function" && define.amd) // AMD
    define(["../../lib/codemirror"], mod);
  else // Plain browser env
    mod(CodeMirror);
})(function(CodeMirror) {
"use strict";

CodeMirror.defineMode("go", function(config) {
  var indentUnit = config.indentUnit;

  var keywords = {
    "break":true, "case":true, "chan":true, "const":true, "continue":true,
    "default":true, "defer":true, "else":true, "fallthrough":true, "for":true,
    "func":true, "go":true, "goto":true, "if":true, "import":true,
    "interface":true, "map":true, "package":true, "range":true, "return":true,
    "select":true, "struct":true, "switch":true, "type":true, "var":true,
    "bool":true, "byte":true, "complex64":true, "complex128":true,
    "float32":true, "float64":true, "int8":true, "int16":true, "int32":true,
    "int64":true, "string":true, "uint8":true, "uint16":true, "uint32":true,
    "uint64":true, "int":true, "uint":true, "uintptr":true, "error": true,
    "rune":true
  };

  var atoms = {
    "true":true, "false":true, "iota":true, "nil":true, "append":true,
    "cap":true, "close":true, "complex":true, "copy":true, "delete":true, "imag":true,
    "len":true, "make":true, "new":true, "panic":true, "print":true,
    "println":true, "real":true, "recover":true
  };

  var isOperatorChar = /[+\-*&^%:=<>!|\/]/;

  var curPunc;

  function tokenBase(stream, state) {
    var ch = stream.next();
    if (ch == '"' || ch == "'" || ch == "` + "`" + `") {
	state.tokenize = tokenString(ch);
	return state.tokenize(stream, state);
}
if (/[\d\.]/.test(ch)) {
if (ch == ".") {
stream.match(/^[0-9]+([eE][\-+]?[0-9]+)?/);
} else if (ch == "0") {
stream.match(/^[xX][0-9a-fA-F]+/) || stream.match(/^0[0-7]+/);
} else {
stream.match(/^[0-9]*\.?[0-9]*([eE][\-+]?[0-9]+)?/);
}
return "number";
}
if (/[\[\]{}\(\),;\:\.]/.test(ch)) {
curPunc = ch;
return null;
}
if (ch == "/") {
if (stream.eat("*")) {
state.tokenize = tokenComment;
return tokenComment(stream, state);
}
if (stream.eat("/")) {
stream.skipToEnd();
return "comment";
}
}
if (isOperatorChar.test(ch)) {
stream.eatWhile(isOperatorChar);
return "operator";
}
stream.eatWhile(/[\w\$_\xa1-\uffff]/);
var cur = stream.current();
if (keywords.propertyIsEnumerable(cur)) {
if (cur == "case" || cur == "default") curPunc = "case";
return "keyword";
}
if (atoms.propertyIsEnumerable(cur)) return "atom";
return "variable";
}

function tokenString(quote) {
return function(stream, state) {
var escaped = false, next, end = false;
while ((next = stream.next()) != null) {
if (next == quote && !escaped) {end = true; break;}
escaped = !escaped && quote != "` + "`" + `" && next == "\\";
}
if (end || !(escaped || quote == "` + "`" + `"))
state.tokenize = tokenBase;
return "string";
};
}

function tokenComment(stream, state) {
var maybeEnd = false, ch;
while (ch = stream.next()) {
if (ch == "/" && maybeEnd) {
state.tokenize = tokenBase;
break;
}
maybeEnd = (ch == "*");
}
return "comment";
}

function Context(indented, column, type, align, prev) {
this.indented = indented;
this.column = column;
this.type = type;
this.align = align;
this.prev = prev;
}
function pushContext(state, col, type) {
return state.context = new Context(state.indented, col, type, null, state.context);
}
function popContext(state) {
if (!state.context.prev) return;
var t = state.context.type;
if (t == ")" || t == "]" || t == "}")
state.indented = state.context.indented;
return state.context = state.context.prev;
}

// Interface

return {
startState: function(basecolumn) {
return {
tokenize: null,
context: new Context((basecolumn || 0) - indentUnit, 0, "top", false),
indented: 0,
startOfLine: true
};
},

token: function(stream, state) {
var ctx = state.context;
if (stream.sol()) {
if (ctx.align == null) ctx.align = false;
state.indented = stream.indentation();
state.startOfLine = true;
if (ctx.type == "case") ctx.type = "}";
}
if (stream.eatSpace()) return null;
curPunc = null;
var style = (state.tokenize || tokenBase)(stream, state);
if (style == "comment") return style;
if (ctx.align == null) ctx.align = true;

if (curPunc == "{") pushContext(state, stream.column(), "}");
else if (curPunc == "[") pushContext(state, stream.column(), "]");
else if (curPunc == "(") pushContext(state, stream.column(), ")");
else if (curPunc == "case") ctx.type = "case";
else if (curPunc == "}" && ctx.type == "}") popContext(state);
else if (curPunc == ctx.type) popContext(state);
state.startOfLine = false;
return style;
},

indent: function(state, textAfter) {
if (state.tokenize != tokenBase && state.tokenize != null) return CodeMirror.Pass;
var ctx = state.context, firstChar = textAfter && textAfter.charAt(0);
if (ctx.type == "case" && /^(?:case|default)\b/.test(textAfter)) {
state.context.type = "}";
return ctx.indented;
}
var closing = firstChar == ctx.type;
if (ctx.align) return ctx.column + (closing ? 0 : 1);
else return ctx.indented + (closing ? 0 : indentUnit);
},

electricChars: "{}):",
closeBrackets: "()[]{}''\"\"` + "`" + "`" + `",
fold: "brace",
blockCommentStart: "/*",
blockCommentEnd: "*/",
lineComment: "//"
};
});

CodeMirror.defineMIME("text/x-go", "go");

});

`)
}
