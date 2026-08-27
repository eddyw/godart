// Command root holds a test in the root package of the module. The output puts
// bin/main.dart and bin/main_test.dart next to each other.
package main

import "fmt"

// value gives the number that the test reads.
func value() int { return 7 }

func main() { fmt.Println(value()) }
