// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package logger

import "github.com/fatih/color"


var ColorInfoPrefix = color.GreenString("%s", "[INFO] ")

var InfoPrefix = color.GreenString("%s", "[INFO] ")

var ErroPrefix = color.RedString("%s", "[ERRO] ")

var HttpErroPrefix = color.RedString("%s", "[HTTP ERRO] ")