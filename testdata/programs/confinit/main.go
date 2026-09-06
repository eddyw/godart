// The program proves the order of the start of a Go program: each package
// variable takes its value in the order of the dependencies, a variable that
// reads no other one follows the order of the files and the order of the
// declarations, and each init function runs after that, in the same order.
package main

import "fmt"

func row(label string, value any) {
	fmt.Printf("%-26s %v\n", label, value)
}

// mainLast reads each other variable, thus it takes its value last.
var mainLast = note("main.go: mainLast reads " + bLate)

var mainCount = noteValue("main.go: mainCount", len(steps))

func init() {
	note("main.go: init")
}

func main() {
	fmt.Println("== the order of the start ==")
	for i, s := range steps {
		fmt.Printf("%2d %s\n", i, s)
	}
	fmt.Println("== the values ==")
	row("aFirst", aFirst)
	row("aReadsB", aReadsB)
	row("aFromFunc", aFromFunc)
	row("bEarly", bEarly)
	row("bLate", bLate)
	row("bBase", bBase)
	row("bMiddle", bMiddle)
	row("bChain", bChain)
	row("cValue", cValue)
	row("cOne", cOne)
	row("cTwo", cTwo)
	row("cZero", cZero)
	row("cReadsZero", cReadsZero)
	row("mainLast", mainLast)
	row("mainCount", mainCount)
	row("the count of steps", len(steps))
	fmt.Println("== a write inside main ==")
	cZero = 9
	row("cZero after a write", cZero)
	row("cReadsZero stays", cReadsZero)
}
