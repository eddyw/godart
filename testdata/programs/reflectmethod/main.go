// The program proves the method API of the package reflect. Type.NumMethod
// and Type.Method give the exported methods in the order of the name.
// Value.Method gives a func value, and Call runs it with the receiver bound.
package main

import (
	"fmt"
	"reflect"
)

type counter struct {
	n int
}

func (c *counter) Add(d int) int          { c.n += d; return c.n }
func (c *counter) Name() string           { return "counter" }
func (c *counter) reset()                 { c.n = 0 }
func (c counter) Value() int              { return c.n }
func (c counter) Scale(f float64) float64 { return float64(c.n) * f }

func main() {
	c := &counter{n: 3}
	t := reflect.TypeOf(c)
	fmt.Println("methods:", t.NumMethod())
	for i := 0; i < t.NumMethod(); i++ {
		fmt.Println(i, t.Method(i).Name)
	}

	v := reflect.ValueOf(c)
	add := v.Method(0)
	mt := add.Type()
	fmt.Println("in:", mt.NumIn(), "out:", mt.NumOut(), "kind:", mt.In(0).Kind())
	out := add.Call([]reflect.Value{reflect.ValueOf(4)})
	fmt.Println("add:", out[0].Interface())
	fmt.Println("value:", v.Method(3).Call(nil)[0].Interface())

	scale := v.Method(2)
	fmt.Println("scale:", scale.Call([]reflect.Value{reflect.ValueOf(2.5)})[0].Interface())
	fmt.Println("after:", c.n)

	var p *counter
	fmt.Println("nil ptr:", reflect.ValueOf(p).IsNil(), reflect.ValueOf(c).IsNil())
	var s []int
	fmt.Println("nil slice:", reflect.ValueOf(s).IsNil(), reflect.ValueOf([]int{1}).IsNil())

	m := map[string]int{"a": 1, "b": 2}
	mv := reflect.ValueOf(m)
	keys := mv.MapKeys()
	fmt.Println("keys:", len(keys))
	total := 0
	for _, k := range keys {
		total += int(mv.MapIndex(k).Int())
	}
	fmt.Println("total:", total)
	fmt.Println("absent:", mv.MapIndex(reflect.ValueOf("c")).IsValid())
	var nilMap map[string]int
	fmt.Println("nil map:", reflect.ValueOf(nilMap).IsNil())
}
