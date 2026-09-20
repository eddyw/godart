package main

import "fmt"

// two gives 2 values, thus a call of it fills 2 arguments.
func two() (int, string) { return 1, "a" }

func three() (int, int, int) { return 7, 8, 9 }

func add(a, b int) int { return a + b }

func addAll(vals ...int) int {
	out := 0
	for _, v := range vals {
		out += v
	}
	return out
}

var calls = 0

// counted gives 2 values and it counts each call. The output proves that the
// inner call runs 1 time.
func counted() (int, int) {
	calls++
	return calls, calls * 10
}

func appended() ([]any, *int) {
	value := 42
	return nil, &value
}

func main() {
	values := append(appended())
	fmt.Println(*values[0].(*int))
	defer func(a, b int) { fmt.Println(a, b) }(counted())
	result := make(chan int)
	go func(a, b int) { result <- a + b }(counted())
	fmt.Println(calls, <-result)
	fmt.Println(two())
	fmt.Println(three())
	fmt.Println(add(two2()))
	fmt.Println(addAll(three()))
	fmt.Printf("%d %s\n", 1, "b")

	fmt.Println(counted())
	fmt.Println(calls)
	fmt.Println(add(counted()))
	fmt.Println(calls)

	// A single value call keeps the plain form.
	fmt.Println(add(1, 2))
}

func two2() (int, int) { return 2, 3 }
