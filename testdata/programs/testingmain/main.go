package main

import (
	"flag"
	"fmt"
	"regexp"
	"testing"
)

// examples holds the list that a generated main of Go gives to RunExamples.
func examples() []testing.InternalExample {
	return []testing.InternalExample{
		{
			Name:   "ExampleGood",
			F:      func() { fmt.Println("hello"); fmt.Println("world") },
			Output: "hello\nworld\n",
		},
		{
			Name:      "ExampleUnordered",
			F:         func() { fmt.Println("b"); fmt.Println("a") },
			Output:    "a\nb\n",
			Unordered: true,
		},
		{
			Name:   "ExampleBad",
			F:      func() { fmt.Println("wrong") },
			Output: "right\n",
		},
	}
}

func main() {
	testing.Init()
	fmt.Println("flag test.v", flag.Lookup("test.v") != nil)
	fmt.Println("flag test.run", flag.Lookup("test.run") != nil)
	fmt.Println("flag test.short", flag.Lookup("test.short") != nil)
	// A second call must add no flag a second time.
	testing.Init()
	flag.Parse()

	ok := testing.RunExamples(regexp.MatchString, examples())
	fmt.Println("examples ok", ok)

	// RunTests of Go runs no test outside a generated main, because the list
	// of the processors of the harness stays empty. gart runs the list. The
	// call below thus holds an empty list, and the 2 give the same line.
	fmt.Println("tests ok", testing.RunTests(regexp.MatchString, nil))

	target := testing.InternalFuzzTarget{Name: "FuzzOne"}
	fmt.Println("fuzz target", target.Name, target.Fn == nil)

	testing.RunBenchmarks(regexp.MatchString, nil)
	fmt.Println("short", testing.Short())
	fmt.Println("cover mode", testing.CoverMode() == "", testing.Coverage())
}
