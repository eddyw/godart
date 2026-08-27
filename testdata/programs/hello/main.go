package main

func greet(name string) string {
	return "hello, " + name
}

func main() {
	println(greet("world"))
	n := 0
	for i := 0; i < 5; i++ {
		n += i * i
	}
	println(n)
}
