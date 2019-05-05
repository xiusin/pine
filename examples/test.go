package main

import (
	"fmt"
	"regexp"
	"strings"
)

func main()  {
	compiler := regexp.MustCompile(`([A-Z])`)
	fmt.Println(compiler.ReplaceAllStringFunc("GetEditName", func(s string) string {
		fmt.Println(s)
		return strings.ToLower("_" + s)
	}))
}
