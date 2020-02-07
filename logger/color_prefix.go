package logger

import "github.com/fatih/color"


var ColorInfoPrefix = color.GreenString("%s", "[INFO] ")

var InfoPrefix = color.GreenString("%s", "[INFO] ")

var ErroPrefix = color.RedString("%s", "[ERRO] ")

var HttpErroPrefix = color.RedString("%s", "[HTTP ERRO] ")