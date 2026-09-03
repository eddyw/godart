// The program proves the order of the evaluation of an expression that holds
// a call. The Go specification orders each call of an expression from left to
// right, and it leaves the order of each other operand open. gc runs each call
// of a statement first, thus each other operand reads its value after the
// calls. Read decision 324.
package main

import "fmt"

type box struct{ v int }

func (b *box) fill() error {
	b.v = 7
	return nil
}

// pair holds the shape of crypto/x509.ParseOID. The method writes the field,
// and the first result reads it after that.
func pair() (box, error) {
	var b box
	return b, b.fill()
}

type point struct{ X, Y int }

type sink struct{ n int }

func (s sink) show(v int) string { return fmt.Sprint(s.n, " ", v) }

// blocking proves the same order inside a function that gives up control. The
// receive stands behind the read, and the read gives the new value.
func blocking(ch chan int) (int, int) {
	y := 1
	go func() {
		y = 9
		ch <- 0
	}()
	return y, <-ch
}

func main() {
	b, err := pair()
	fmt.Println("return", b.v, err)

	x := 1
	set := func(n int) int { x = n; return 0 }

	x = 1
	fmt.Println("args", x, set(2))

	x = 1
	p, q := x, set(3)
	fmt.Println("define", p, q)

	x = 1
	var r, s int
	r, s = x, set(4)
	fmt.Println("assign", r, s)

	x = 1
	fmt.Println("binary", x+set(5))

	// A call in front of a read already matches, thus the output holds no
	// temporary variable for this shape.
	x = 1
	fmt.Println("callfirst", set(6)+x)

	// Each operand in front of the call is a constant, thus this shape needs
	// no temporary variable either.
	x = 1
	fmt.Println("const", 100, set(7))

	x = 1
	fmt.Println("slice", []int{x, set(8)})

	x = 1
	fmt.Println("struct", point{X: x, Y: set(9)})

	arr := []int{10, 20}
	i := 0
	pickIndex := func() int { i = 1; return 0 }
	fmt.Println("index", arr[i]+pickIndex())

	nums := []int{1}
	grow := func() int { nums = append(nums, 2); return 0 }
	fmt.Println("append", len(nums), grow())

	sk := sink{n: 1}
	change := func(n int) int { sk.n = n; return 0 }
	fmt.Println("recv", sk.show(change(5)))

	first := make(chan int, 1)
	second := make(chan int, 1)
	chans := []chan int{first, second}
	k := 0
	pickChan := func() int { k = 1; return 3 }
	chans[k] <- pickChan()
	fmt.Println("send", len(first), len(second))

	x = 1
	report := func() {
		defer fmt.Println("defer", x, set(10))
	}
	report()

	live := make(chan int)
	one, two := blocking(live)
	fmt.Println("blocking", one, two)
}
