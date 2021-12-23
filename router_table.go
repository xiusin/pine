// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"github.com/landoop/tableprinter"
	"os"
	"reflect"
	"runtime"
)

type RouterTableRow struct {
	Method  string `header:"METHOD"`
	Path    string `header:"PATH"`
	Alias   string `header:"ALIASES"`
	Name    string `header:"NAME"`
	Handler string `header:"HANDLER"`
}

func (r *Router) DumpRouteTable() {
	p := tableprinter.New(os.Stdout)
	p.BorderTop, p.BorderBottom, p.BorderLeft, p.BorderRight = true, true, true, true
	p.CenterSeparator, p.ColumnSeparator, p.RowSeparator = "│", "│", "─"
	p.HeaderBgColor, p.HeaderFgColor = 40, 32

	var tables []RouterTableRow

	for method, routers := range r.methodRoutes {
		if len(routers) == 0 {
			continue
		}
		for s, entry := range routers {
			pc := make([]uintptr, 1)
			runtime.Callers(2, pc)
			tables = append(tables, RouterTableRow{
				Method:  method,
				Path:    s,
				Alias:   "",
				Name:    "",
				Handler: runtime.FuncForPC(reflect.ValueOf(entry.Handle).Pointer()).Name(),
			})
		}
	}
	p.Print(tables)
}
