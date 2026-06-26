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
	// Name    string `header:"NAME"`
	Handler string `header:"HANDLER"`
}

func (r *Router) DumpRouteTable() {
	p := tableprinter.New(os.Stdout)
	p.BorderTop, p.BorderBottom, p.BorderLeft, p.BorderRight = true, true, true, true
	p.CenterSeparator, p.ColumnSeparator, p.RowSeparator = "│", "│", "─"
	p.HeaderBgColor, p.HeaderFgColor = 40, 32

	var tables []RouterTableRow

	// 遍历基数树路由表快照 (已排除静默注册的 catch-all 基路径与 OPTIONS 别名).
	for _, e := range r.app.tree.table {
		if e.Method == http.MethodOptions {
			continue
		}
		tables = append(tables, RouterTableRow{
			Method:  e.Method,
			Path:    e.Path,
			Handler: runtime.FuncForPC(reflect.ValueOf(e.Handler).Pointer()).Name(),
		})
	}

	p.Print(tables)
}
