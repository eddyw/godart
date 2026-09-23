package main

import "fmt"

func main() {
	first := struct{ F119201 int }{17}
	second := struct{ F140986 string }{"value"}
	fmt.Println(first.F119201, second.F140986)
}
