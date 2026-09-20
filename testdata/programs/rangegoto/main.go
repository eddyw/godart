package main

import (
	"fmt"
	"runtime"
)

func sequence(yield func(int) bool) {
	defer fmt.Print("exit ")
	for i := 0; i < 3; i++ {
		if !yield(i) {
			return
		}
	}
}

func restart() int {
	n := 0
again:
	for range sequence {
		runtime.Gosched()
		n++
		if n < 3 {
			goto again
		}
		goto done
	}
done:
	return n
}

func nested() int {
	n := 0
outer:
	for range sequence {
	inside:
		for range sequence {
			runtime.Gosched()
			n++
			if n < 2 {
				goto inside
			}
			break outer
		}
	}
	return n
}

func local() int {
	for range sequence {
		f := func() int {
			n := 0
		again:
			for range sequence {
				runtime.Gosched()
				n++
				if n == 1 {
					goto again
				}
				return n
			}
			return -1
		}
		return f()
	}
	return -1
}

func main() {
	fmt.Println(restart())
	fmt.Println(nested())
	fmt.Println(local())
}
