package cover

// Classify names a number: "low", "high", or "zero".
func Classify(n int) string {
	if n > 10 {
		return "high"
	}
	if n > 0 {
		return "low"
	}
	if n < 0 {
		return "negative"
	}
	return "zero"
}

// Unused runs in no test. The count of the coverage thus stays below 100.
func Unused(n int) int {
	total := 0
	for i := 0; i < n; i++ {
		total += i
	}
	return total
}
