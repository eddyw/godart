package calc

import "testing"

// sink stops the Dart compiler from a removal of a benchmark body. The compiler
// removes code that no other code reads.
var sink int

func BenchmarkAdd(b *testing.B) {
	total := 0
	for i := 0; i < b.N; i++ {
		total = Add(total, 1)
	}
	sink = total
}

func BenchmarkDouble(b *testing.B) {
	cases := []doubleCase{{name: "one", in: 1, want: 2}, {name: "big", in: 1000, want: 2000}}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			total := 0
			for i := 0; i < b.N; i++ {
				total += Double(c.in)
			}
			sink = total
		})
	}
}

// BenchmarkLoop reads the Loop method. Such a body runs 1 time, and Loop raises
// the count of iterations itself.
func BenchmarkLoop(b *testing.B) {
	total := 0
	for b.Loop() {
		total = Add(total, 1)
	}
	sink = total
}
