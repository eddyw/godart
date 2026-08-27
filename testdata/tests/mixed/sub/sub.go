// Package sub holds a function that blocks and a function that fails.
package sub

// Sum reads each value of the channel and gives the total.
func Sum(values chan int) int {
	total := 0
	for v := range values {
		total += v
	}
	return total
}

// Broken gives a wrong result on purpose. A test of it must fail.
func Broken() int { return 1 }
