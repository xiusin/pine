package main

import "fmt"

func main() {
	call(func() {

	})
}

func call(a func())  {
	fmt.Printf("%p", a)
	runcall(a)
}

func runcall(a func())  {
	fmt.Printf("%p", a)
}
