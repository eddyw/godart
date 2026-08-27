package main

func divmod(a, b int) (int, int) {
	return a / b, a % b
}

func named(a int) (q int, r int) {
	q = a * 2
	r = a + 1
	return
}

func total(values ...int) int {
	n := 0
	for _, v := range values {
		n += v
	}
	return n
}

func apply(f func(int) int, v int) int {
	return f(v)
}

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	q, r := divmod(17, 5)
	println(q, r)

	a, b := named(4)
	println(a, b)

	println(total(), total(1, 2, 3))
	nums := []int{4, 5}
	println(total(nums...))

	double := func(v int) int { return v * 2 }
	println(apply(double, 21))

	count := 0
	inc := func() { count++ }
	inc()
	inc()
	println(count)

	println(fib(10))

	x, y := 1, 2
	x, y = y, x
	println(x, y)
}
