package main

import (
	"fmt"

	"example.com/initorder/base"
)

var greeting = prefix + name

var prefix = "hello "

var name = who()

func who() string { return target }

var target = "go"

// A variable with no initializer takes the zero value.
var zero int

var fromBase = base.Total + len(base.Label)

func init() {
	fmt.Println("init sees", greeting, zero)
}

func main() {
	fmt.Println(greeting)
	fmt.Println(base.Total, base.Label, base.Suffix)
	fmt.Println(fromBase)
}
