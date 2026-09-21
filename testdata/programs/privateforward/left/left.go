package left

type I interface{ get() int }

func Call(v I) int { return v.get() }

type S struct{ Value int }

func (s *S) get() int {
	if s == nil {
		return -1
	}
	s.Value++
	return s.Value
}

type V struct{ Value int }

func (v V) get() int { v.Value++; return v.Value }

type Number int

func (n Number) get() int { return int(n) }

type hidden struct{ Value int }

func (v hidden) get() int { return v.Value }

type Outer struct{ hidden }

func Make() Outer { return Outer{hidden{9}} }

type Generic[T any] interface{ item() T }
type Box[T any] struct{ Value T }

func (b Box[T]) item() T         { return b.Value }
func Read[T any](v Generic[T]) T { return v.item() }
