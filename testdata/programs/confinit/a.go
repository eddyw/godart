// The file a.go stands first in the order of the names of the files. Go gives
// each variable its value in the order of the dependencies, and it then runs
// each init function in the order of the files.
package main

// steps records each step of the start of the program.
var steps []string

func note(s string) string {
	steps = append(steps, s)
	return s
}

func noteValue(s string, v int) int {
	steps = append(steps, s)
	return v
}

// aFirst reads no other variable, thus it takes its value early.
var aFirst = note("a.go: aFirst")

// aReadsB reads a variable of the file b.go, thus that variable takes its
// value first.
var aReadsB = note("a.go: aReadsB reads " + bEarly)

// aFromFunc reads a function that reads another variable.
var aFromFunc = noteValue("a.go: aFromFunc", readsC())

func readsC() int { return cValue * 2 }

func init() {
	note("a.go: init 1")
}

func init() {
	note("a.go: init 2")
}
