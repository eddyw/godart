// Package target holds a fuzz target and an example. gart runs neither one.
package target

// Length gives the count of bytes of the text.
func Length(text string) int { return len(text) }
