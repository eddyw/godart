// Package count counts the runes of a text.
package count

func Runes(text string) int {
	n := 0
	for range text {
		n++
	}
	return n
}
