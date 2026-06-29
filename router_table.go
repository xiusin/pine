// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"net/http"
	"os"
	"reflect"
	"runtime"

	"github.com/landoop/tableprinter"
)

type RouterTableRow struct {
	Method string `header:"METHOD"`
	Path   string `header:"PATH"`
	// Alias   string `header:"ALIASES"`
	Name    string `header:"NAME"`
	Handler string `header:"HANDLER"`
}

func (r *Router) DumpRouteTable() {
	p := tableprinter.New(os.Stdout)
	p.BorderTop, p.BorderBottom, p.BorderLeft, p.BorderRight = true, true, true, true
	p.CenterSeparator, p.ColumnSeparator, p.RowSeparator = "│", "│", "─"
	p.HeaderBgColor, p.HeaderFgColor = 40, 32

	var tables []RouterTableRow

	// appendRows 将一份路由表快照追加到 tables, 跳过静默注册的 OPTIONS 别名.
	appendRows := func(tbl []tableEntry) {
		for _, e := range tbl {
			if e.Method == http.MethodOptions {
				continue
			}
			tables = append(tables, RouterTableRow{
				Method:  e.Method,
				Path:    e.Path,
				Name:    e.Name,
				Handler: runtime.FuncForPC(reflect.ValueOf(e.Handler).Pointer()).Name(),
			})
		}
	}

	// 主树路由表.
	appendRows(r.app.tree.table)
	// 子域路由表 (按 host 前缀注册的路由位于各子树).
	for _, sub := range r.app.tree.hostTrees {
		appendRows(sub.table)
	}

	p.Print(tables)
}
