package main

import (
	"fmt"
	"reflect"
)

type A struct {
	X int `a`
}
type B struct {
	X int `b`
}

func (b *B) Add(n int)         { b.X += n }
func (b B) Read() int          { return b.X }
func from[P ~*A](p P) *B       { return (*B)(p) }
func to[P ~*B](p *A) P         { return P(p) }
func both[P ~*A, Q ~*B](p P) Q { return Q(p) }

func unionFrom[P ~*A | ~*B](p P) *B             { return (*B)(p) }
func unionTo[P ~*A | ~*B](p *A) P               { return P(p) }
func unionBoth[P ~*A | ~*B, Q ~*A | ~*B](p P) Q { return Q(p) }

type AP *A
type I int
type V [2]int

func scalar[P ~*I | ~*int](p P) *int      { return (*int)(p) }
func array[P ~*V | ~*[2]int](p P) *[2]int { return (*[2]int)(p) }

type VA = [1]struct {
	X int `a`
}
type VB = [1]struct {
	X int `b`
}

func structArray[P ~*VA | ~*VB](p P) *VB { return (*VB)(p) }

func viaValue[T any, P ~*T](v T) P    { return P(&v) }
func viaPointer[T any, P ~*T](p *T) P { return P(p) }

func replace[T any, P ~*T](v, next T) P {
	p := P(&v)
	v = next
	return p
}

type C[T any] struct {
	X T `a`
}
type D[T any] struct {
	X T `b`
}

func fromG[T any, P ~*C[T]](p P) *D[T] { return (*D[T])(p) }
func toG[T any, P ~*D[T]](p *C[T]) P   { return P(p) }

func main() {
	a := A{17}
	for _, b := range []*B{from(&a), to[*B](&a), both[*A, *B](&a)} {
		b.Add(2)
		fmt.Println(a.X, b.Read(), &a.X == &b.X, (*A)(b) == &a)
		fmt.Println(b == from(&a), b == to[*B](&a), b == both[*A, *B](&a))
		fmt.Println(reflect.ValueOf(&a).Pointer() == reflect.ValueOf(b).Pointer())
		fmt.Println(map[*B]string{b: "same"}[from(&a)])
		a.X++
		fmt.Println(b.X)
	}
	b := unionFrom(&a)
	b.X++
	fmt.Println(a.X, b == unionFrom(b), b == unionTo[*B](&a), &a == unionTo[*A](&a))
	fmt.Println(b == unionBoth[*A, *B](&a), &a == unionBoth[*B, *A](b))
	fmt.Println(unionFrom((*A)(nil)) == nil, unionTo[*B](nil) == nil, unionBoth[*A, *B](nil) == nil)

	fmt.Println(from((*A)(nil)) == nil, to[*B](nil) == nil, both[*A, *B](nil) == nil)
	fmt.Println(b == from(AP(&a)), AP(&a) == unionBoth[*B, AP](b))
	i := I(31)
	ip := scalar(&i)
	*ip++
	fmt.Println(i, ip == (*int)(&i), scalar((*I)(nil)) == nil)
	v := V{1, 2}
	vp := array(&v)
	vp[1]++
	fmt.Println(v, vp == (*[2]int)(&v), array((*V)(nil)) == nil)

	va := VA{{41}}
	vb := structArray(&va)
	vb[0].X++
	fmt.Println(va[0].X, &va[0].X == &vb[0].X, vb == structArray(vb), vb == (*VB)(&va), structArray((*VA)(nil)) == nil)

	a2 := A{90}
	p2 := viaValue[A, *A](a2)
	p2.X++
	fmt.Println(a2.X, p2.X, viaPointer[A, AP](&a2) == AP(&a2), viaPointer[A, *A](nil) == nil)
	vp2 := viaValue[[2]int, *[2]int]([2]int{51, 52})
	vp2[1]++
	fmt.Println(*vp2)
	ip2 := viaValue[int, *int](61)
	*ip2++
	fmt.Println(*ip2)

	fmt.Println(*replace[A, *A](A{1}, A{2}), *replace[int, *int](1, 2), *replace[[2]int, *[2]int]([2]int{1, 2}, [2]int{3, 4}))

	c := C[string]{"hello"}
	d := fromG(&c)
	d.X += "!"
	fmt.Println(c.X, &c.X == &d.X, d == toG[string, *D[string]](&c))
	fmt.Println(fromG[string]((*C[string])(nil)) == nil, toG[string, *D[string]](nil) == nil)
}
