// The program proves that a method of a generic type takes a pointer to a type
// parameter.
package main

import "fmt"

type box[T any] struct {
	v T
}

func (b *box[T]) set(p *T) {
	b.v = *p
}

func (b *box[T]) read(out *T) {
	*out = b.v
}

func (b box[T]) value() T {
	return b.v
}

type pair[K comparable, V any] struct {
	key K
	val V
}

func (p *pair[K, V]) swap(k *K, v *V) {
	p.key, *k = *k, p.key
	p.val, *v = *v, p.val
}

type setter[T any] interface {
	set(p *T)
	value() T
}

func fill[T any](s setter[T], p *T) T {
	s.set(p)
	return s.value()
}

type keeper[T any] struct {
	seen []T
}

func (k *keeper[T]) set(p *T) {
	k.seen = append(k.seen, *p)
}

func (k *keeper[T]) value() T {
	return k.seen[len(k.seen)-1]
}

func main() {
	var b box[int]
	n := 7
	b.set(&n)
	fmt.Println(b.value())

	var out int
	b.read(&out)
	fmt.Println(out)

	var s box[string]
	text := "hello"
	s.set(&text)
	fmt.Println(s.value())

	var face setter[int] = &box[int]{}
	m := 11
	fmt.Println(fill(face, &m))

	var kept setter[string] = &keeper[string]{}
	word := "kept"
	fmt.Println(fill(kept, &word))

	take := face.set
	m = 13
	take(&m)
	fmt.Println(face.value())

	p := &pair[string, int]{key: "a", val: 1}
	k, v := "b", 2
	p.swap(&k, &v)
	fmt.Println(p.key, p.val, k, v)
}
