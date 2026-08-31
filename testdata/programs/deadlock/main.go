package main

import "fmt"

// waitFor blocks on a channel that no goroutine fills. The frame chain of the
// goroutine holds this function under main, thus the deadlock report of the
// runtime walks 2 frames.
func waitFor(ch chan int) int {
	return <-ch
}

func main() {
	fmt.Println("start")
	other := make(chan string)
	go func() {
		other <- "never read"
	}()
	fmt.Println(waitFor(make(chan int)))
}
