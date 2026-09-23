package foreign

func Create[T any](value T) any       { return struct{ X T }{value} }
func Read[T any](value any) (T, bool) { x, ok := value.(struct{ X T }); return x.X, ok }

func Pair[A, B any](a A, b B) any {
	return struct {
		X A
		Y B
	}{a, b}
}
func Pointer[T any](v T) (any, *T) { p := &struct{ X T }{v}; return p, &p.X }
func Constrained[T ~int](v T) any  { return struct{ X T }{v} }
func Nested[T any](v T) any {
	return struct {
		X []T
		Y struct{ Z T }
	}{[]T{v}, struct{ Z T }{v}}
}
