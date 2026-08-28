package main

type number interface {
	~int | ~float64
}

type text interface {
	~string
}

func join[T text](values []T, sep T) T {
	var out T
	for i, v := range values {
		if i > 0 {
			out += sep
		}
		out += v
	}
	return out
}

// total takes a variadic parameter of the type parameter. The call must read
// the instantiated signature, and not the declaration.
func total[T number](values ...T) T {
	var out T
	for _, v := range values {
		out += v
	}
	return out
}

func sum[T number](values []T) T {
	var total T
	for _, v := range values {
		total += v
	}
	return total
}

func mapSlice[T any, U any](values []T, f func(T) U) []U {
	out := make([]U, 0, len(values))
	for _, v := range values {
		out = append(out, f(v))
	}
	return out
}

func index[T comparable](values []T, want T) int {
	for i, v := range values {
		if v == want {
			return i
		}
	}
	return -1
}

func zero[T any]() T {
	var v T
	return v
}

type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(v T) {
	s.items = append(s.items, v)
}

func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var empty T
		return empty, false
	}
	last := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return last, true
}

func (s Stack[T]) Len() int {
	return len(s.items)
}

func (s Stack[T]) Convert[U any](f func(T) U) []U {
	return mapSlice(s.items, f)
}

type pair[K comparable, V any] struct {
	key   K
	value V
}

// kind reads the real type of a value of a type parameter. The dictionary of
// the type argument holds the description, thus the value reaches an interface.
func kind[T string | []byte](s T) int {
	switch v := any(s).(type) {
	case string:
		return len(v)
	case []byte:
		return -len(v)
	}
	return 0
}

// describe writes the name of the type argument.
func describe[T any](v T) string {
	var x any = v
	switch x.(type) {
	case int:
		return "int"
	case float64:
		return "float64"
	case bool:
		return "bool"
	}
	return "other"
}

func main() {
	println(sum([]int{1, 2, 3}))
	println(sum([]float64{1.5, 2.5}))
	println(join([]string{"a", "b"}, "-"))

	texts := mapSlice([]int{1, 2}, func(v int) string {
		if v == 1 {
			return "one"
		}
		return "two"
	})
	println(len(texts), texts[0], texts[1])

	println(index([]string{"a", "b"}, "b"))
	println(index([]int{1, 2}, 9))

	println(zero[int](), zero[string]() == "")

	var s Stack[int]
	s.Push(1)
	s.Push(2)
	println(s.Len())
	v, ok := s.Pop()
	println(v, ok)

	names := &Stack[string]{}
	names.Push("go")
	println(names.Len())

	lengths := s.Convert(func(v int) int { return v * 10 })
	println(len(lengths), lengths[0])

	p := pair[string, int]{key: "a", value: 1}
	println(p.key, p.value)

	pairs := []pair[string, int]{{"x", 1}, {"y", 2}}
	println(len(pairs), pairs[1].key)

	println(total(1, 2, 3))
	println(total(1.5, 2.5))
	println(total[int]())
	rest := []int{4, 5}
	println(total(rest...))

	println(kind("abc"), kind([]byte("abcd")))
	println(describe(1), describe(1.5), describe(true), describe("x"))
}
