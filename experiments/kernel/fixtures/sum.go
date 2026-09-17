//go:build ignore

package main

func square(x int) int { return x * x }
func main() {
	sum := 0
	for i := 0; i < 5; i++ {
		sum += square(i)
	}
	if sum == 30 {
		println("sum", sum)
	} else {
		println("bad", sum)
	}
}
