// The file b.go stands second in the order of the names of the files.
package main

// bEarly reads no other variable. The variable aReadsB of a.go reads it, thus
// it takes its value in front of that one.
var bEarly = note("b.go: bEarly")

// bLate reads a variable of a.go.
var bLate = note("b.go: bLate reads " + aFirst)

// bChain reads a chain of 3 variables.
var (
	bChain = noteValue("b.go: bChain", bMiddle+1)
	bBase  = noteValue("b.go: bBase", 1)
)

var bMiddle = noteValue("b.go: bMiddle", bBase*10)

func init() {
	note("b.go: init")
}
