package main

import "fmt"

// Go runs a method with a pointer receiver when the receiver is nil. The
// method holds the test of the nil value.

type user struct{ name string }

func (u *user) Name() string {
	if u == nil {
		return "<none>"
	}
	return u.name
}

func (u *user) rename(s string) {
	if u == nil {
		return
	}
	u.name = s
}

type namer interface{ Name() string }

type node struct {
	v    int
	next *node
}

func (n *node) sum() int {
	if n == nil {
		return 0
	}
	return n.v + n.next.sum()
}

func (n *node) depth() int {
	if n == nil {
		return 0
	}
	d := n.next.depth()
	return d + 1
}

type counter struct{ n int }

func (c *counter) inc() { c.n++ }

func (c counter) get() int { return c.n }

type box[T any] struct{ v T }

func (b *box[T]) set(v T) { b.v = v }
func (b *box[T]) get() T  { return b.v }

func main() {
	var u *user
	fmt.Println(u.Name())
	u.rename("x")
	fmt.Println(u.Name())

	real := &user{name: "a"}
	fmt.Println(real.Name())
	real.rename("b")
	fmt.Println(real.Name())

	var iface namer = u
	fmt.Println(iface.Name(), iface == nil)

	l := &node{v: 1, next: &node{v: 2, next: &node{v: 3}}}
	fmt.Println(l.sum(), l.depth())
	var empty *node
	fmt.Println(empty.sum(), empty.depth())

	// A value receiver, and the address of a variable.
	c := counter{}
	c.inc()
	c.inc()
	fmt.Println(c.get())

	// A method value of a nil receiver.
	f := u.Name
	fmt.Println(f())

	// A method of a generic type stays a member of its class.
	b := &box[string]{}
	b.set("g")
	fmt.Println(b.get())
}
