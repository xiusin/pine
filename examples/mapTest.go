package main

import "fmt"

func main() {
	//m := map[string]interface{}{}
	//m1 := m
	//m2 := m
	//m1["sss"] = "name"
	//m2["sss11"] = "name"
	//fmt.Println(m1, m2)

	m1 := get()
	m2 := get()
	m1["sss"] = "name"
	m2["sss11"] = "name"
	fmt.Println(m1, m2)

}

func get() map[string]interface{} {
	return map[string]interface{}{}
}
