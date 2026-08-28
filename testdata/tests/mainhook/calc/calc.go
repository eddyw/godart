// Package calc holds the code that the test of the TestMain hook reads.
package calc

var Prefix = ""

func Label(name string) string { return Prefix + name }
