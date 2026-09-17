//go:build ignore

package main

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	println("fib", fib(12))
	n := 0
	for i := 0; i < 0; i++ {
		n += i
	}
	println("zero loop", n)
	if n != 0 {
		println("wrong")
	} else {
		println("branch", true)
	}
}
