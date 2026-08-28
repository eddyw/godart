// Package calc holds the code that the table driven test reads.
package calc

func Clamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func Sum(values []int) int {
	total := 0
	for _, v := range values {
		total += v
	}
	return total
}
