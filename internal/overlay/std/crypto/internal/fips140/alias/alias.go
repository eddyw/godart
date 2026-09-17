//go:build dart

package alias

import "syscall/dart"

func AnyOverlap(x, y []byte) bool {
	return dart.SliceOverlap(x, y)
}

func InexactOverlap(x, y []byte) bool {
	if len(x) == 0 || len(y) == 0 || &x[0] == &y[0] {
		return false
	}
	return AnyOverlap(x, y)
}
