// The file c.go stands last in the order of the names of the files.
package main

// cValue holds no initializer that reads another variable.
var cValue = noteValue("c.go: cValue", 21)

// cGroup holds 2 names of 1 declaration. Each name reads the same call.
var cOne, cTwo = pairOf()

func pairOf() (int, int) {
	note("c.go: pairOf")
	return 1, 2
}

// cZero holds no initializer, thus it takes the zero value first.
var cZero int

var cReadsZero = noteValue("c.go: cReadsZero", cZero+5)

func init() {
	note("c.go: init")
}
