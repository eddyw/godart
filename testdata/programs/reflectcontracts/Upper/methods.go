package Upper

type Sealed interface {
	Z() int
	a() int
}

type Value struct{}

func (Value) Z() int { return 6 }
func (Value) a() int { return 7 }

func Call(value Sealed) int {
	method := value.a
	expression := Sealed.a
	return value.a() + method() + expression(value) + generic(value)
}

func generic[T Sealed](value T) int { return value.a() }
