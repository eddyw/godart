// Package seed holds the functions that the fuzz targets of the module read.
package seed

// Reverse gives the bytes in the opposite order.
func Reverse(data []byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[len(data)-1-i] = b
	}
	return out
}

// Repeat gives the text n times. A count below 1 gives an empty text.
func Repeat(text string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += text
	}
	return out
}
