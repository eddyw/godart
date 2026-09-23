package main

import (
	"example.com/structptrview/foreign"
	"fmt"
	"reflect"
	"weak"
)

type A struct {
	X int `a`
}
type B struct {
	X int `b`
}

func (p *B) Add(n int) { p.X += n }
func (p B) Read() int  { return p.X + 1 }

type G[T any] struct{ X T }
type Node struct {
	Next  *Node
	Value int
}

func main() {
	a := A{17}
	b := (*B)(&a)
	b.Add(2)
	c := (*struct{ X int })(b)
	c.X++
	f := (*foreign.Value)(b)
	f.Add(3)
	fmt.Println(a.X, b.Read(), f.Read(), (*A)(b) == &a, (*B)(&a) == b)
	fmt.Println(&a.X == &b.X, &a.X == &f.X, weak.Make(&a.X) == weak.Make(&b.X))
	fmt.Println(reflect.ValueOf(&a).Pointer() == reflect.ValueOf(b).Pointer(), fmt.Sprintf("%p", &a) == fmt.Sprintf("%p", b))
	fmt.Println(reflect.ValueOf(b).Elem().Field(0).Addr().Interface().(*int) == &a.X)
	reflected := reflect.ValueOf(&a).Convert(reflect.TypeOf(b)).Interface().(*B)
	reflected.X++
	fmt.Println(a.X, reflected == b)
	var face any = b
	_, wrong := face.(*A)
	fmt.Println(wrong, face.(interface{ Read() int }).Read())
	copied := *b
	copied.X++
	fmt.Println(copied.X, a.X)
	g := (*G[int])(&a)
	g.X++
	fmt.Println(a.X, (*A)(g) == &a)
	node := Node{Value: 29}
	n := (*struct {
		Next  *Node
		Value int
	})(&node)
	n.Next = &node
	n.Value++
	fmt.Println(node.Value, n.Next == &node, (*Node)(n) == &node)
	var nilA *A
	fmt.Println((*B)(nilA) == nil)
	fmt.Println(reflect.TypeFor[chan<- int]().Name() == "")
	fmt.Println(map[*B]string{b: "found"}[(*B)(&a)])
	t1 := reflect.StructOf([]reflect.StructField{{Name: "RuntimeField", Type: reflect.TypeFor[int](), Tag: "first"}})
	t2 := reflect.StructOf([]reflect.StructField{{Name: "RuntimeField", Type: reflect.TypeFor[int](), Tag: "second"}})
	v1 := reflect.New(t1)
	v2 := v1.Convert(reflect.PointerTo(t2))
	v2.Elem().Field(0).SetInt(47)
	fmt.Println(v1.Elem().Field(0).Int(), v1.Pointer() == v2.Pointer(), v2.Convert(v1.Type()).Interface() == v1.Interface())
	fmt.Println(v1.Elem().Field(0).Addr().Interface() == v2.Elem().Field(0).Addr().Interface())
	fmt.Println("complete")
}
