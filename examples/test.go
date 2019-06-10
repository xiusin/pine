package main

import "fmt"

type lili struct {
	Name string
}

func (lili *lili) fmtPointer()  {
	fmt.Println("pointer")
}

func (lili lili) fmtRef() {
	fmt.Println("fmtRef")
}

func main()  {
	(lili{}).fmtRef()
}