//go:build ignore

package main

func main() {
	max := 9223372036854775807
	println(max + 1)
	n := 9007199254740993
	println(n, n*3, n-1)
	value := 12
	{
		value := 7
		println("inner", value)
	}
	println("outer", value)
	var zero int
	var flag bool
	println(zero, flag)
}
