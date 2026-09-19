// The program proves the order of the evaluation at 5 positions that a
// temporary variable in front of the statement cannot reach. The condition of
// a for loop, the third clause of a for loop, and the value of a case each run
// more than 1 time. The initializer of a package variable stands outside each
// statement. A conversion of a string reads its operand early under gc.
package main

import "fmt"

var gx = 1

func gset(n int) int { gx = n; return 0 }

// Go gives sum its value after gx. The call writes gx, and the read of gx
// gives the new value.
var sum = gx + gset(2)

func main() {
	fmt.Println("global", sum, gx)

	x := 1
	set := func(n int) int { x = n; return 0 }

	x = 1
	n := 0
	for x+set(3) < 3 {
		n++
	}
	fmt.Println("forcond", n, x)

	x = 1
	rounds := 0
	for i := 0; i < 2; i, _ = x, set(5) {
		rounds++
	}
	fmt.Println("forpost", rounds, x)

	x = 1
	hit := "no"
	switch 3 {
	case x + set(3):
		hit = "yes"
	}
	fmt.Println("case", hit, x)

	s := "ab"
	grow := func() int { s = "cd"; return 0 }
	fmt.Println("convert", string([]byte(s)), grow(), s)

	s = "ab"
	growRunes := func() int { s = "cd"; return 0 }
	fmt.Println("runes", string([]rune(s)), growRunes(), s)
}
