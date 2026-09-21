package impl

var Initialized int

func init()                   { Initialized++ }
func twice(n int) int         { return n * 2 }
func Function() func(int) int { return twice }
func wait(n int) int {
	c := make(chan int)
	go func() { c <- n + 1 }()
	return <-c
}
