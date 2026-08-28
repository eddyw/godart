package main

import (
	"fmt"
	"time"
)

type point struct{ X, Y int }

func (p point) String() string { return fmt.Sprintf("(%d,%d)", p.X, p.Y) }

type celsius float64

func (c celsius) String() string { return fmt.Sprintf("%v degrees", float64(c)) }

type wrapper struct{ P point }

func main() {
	p := point{1, 2}
	fmt.Println(p)
	fmt.Printf("%v|%s|%d\n", p, p, 4)
	fmt.Println(wrapper{P: p})
	fmt.Println([]point{p, {3, 4}})

	var s fmt.Stringer = p
	fmt.Println(s.String(), s)

	var c celsius = 21.5
	fmt.Println(c)
	fmt.Println(time.Second + 500*time.Millisecond)

	var e error = fmt.Errorf("the cause of %v", p)
	fmt.Println(e)
}
