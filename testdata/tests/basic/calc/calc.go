// Package calc holds a few functions with tests.
package calc

// Add gives the sum of 2 integers.
func Add(a, b int) int { return a + b }

// Double gives 2 times the value.
func Double(v int) int { return v * 2 }

// Join puts a separator between each part.
func Join(parts []string, sep string) string {
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += sep
		}
		out += part
	}
	return out
}
