// The program proves that a block operation inside the body of a range loop
// over a function parks and resumes. The body becomes a Dart closure, and the
// closure must take the machine form when it blocks. Before that rule, the
// receive below gave a junk value, and the loop ran on with it.
package main

import "fmt"

func seq(yield func(int) bool) {
	for i := 1; i <= 3; i++ {
		if !yield(i) {
			return
		}
	}
}

func main() {
	data := make(chan int)
	go func() {
		for i := 10; i <= 30; i += 10 {
			data <- i
		}
	}()
	total := 0
	for v := range seq {
		total += v + <-data
	}
	fmt.Println("total:", total)

	// A break leaves the loop through the closure, and the send of the pair
	// goroutine must still park and resume.
	pair := make(chan string)
	go func() {
		pair <- "a"
		pair <- "b"
	}()
	got := []string{}
	for v := range seq {
		got = append(got, fmt.Sprint(v, <-pair))
		if v == 2 {
			break
		}
	}
	fmt.Println("got:", got)
}
