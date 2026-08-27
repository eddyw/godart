package main

func producer(ch chan int, n int) {
	for i := 0; i < n; i++ {
		ch <- i
	}
	close(ch)
}

func sum(ch chan int, done chan int) {
	total := 0
	for v := range ch {
		total += v
	}
	done <- total
}

func main() {
	ch := make(chan int)
	done := make(chan int)
	go producer(ch, 5)
	go sum(ch, done)
	println(<-done)

	buf := make(chan string, 2)
	buf <- "a"
	buf <- "b"
	println(len(buf), cap(buf))
	println(<-buf, <-buf)
	close(buf)
	v, ok := <-buf
	println(v == "", ok)

	a := make(chan int, 1)
	b := make(chan int, 1)
	a <- 1
	select {
	case x := <-a:
		println("a", x)
	case y := <-b:
		println("b", y)
	}

	select {
	case x := <-b:
		println("b", x)
	default:
		println("no value")
	}

	c := make(chan int, 1)
	select {
	case c <- 9:
		println("sent")
	default:
		println("full")
	}
	println(<-c)

	results := make(chan int, 3)
	for i := 0; i < 3; i++ {
		go func(n int) { results <- n * n }(i)
	}
	total := 0
	for i := 0; i < 3; i++ {
		total += <-results
	}
	println(total)
}
