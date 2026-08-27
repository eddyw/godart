package target

import (
	"fmt"
	"testing"
)

func TestLength(t *testing.T) {
	if got := Length("abc"); got != 3 {
		t.Errorf("Length gives %d, and 3 is the count", got)
	}
}

func BenchmarkLength(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Length("abc")
	}
}

func FuzzLength(f *testing.F) {
	f.Add("abc")
	f.Fuzz(func(t *testing.T, text string) { _ = Length(text) })
}

func ExampleLength() {
	fmt.Println(Length("abc"))
	// Output: 3
}
