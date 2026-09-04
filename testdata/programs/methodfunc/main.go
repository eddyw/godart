package main

import (
	"fmt"
	"reflect"
	"strings"
)

type greeter struct{ Name string }

func (g greeter) Hello(who string) string { return "hello " + who + ", " + g.Name }

func (g *greeter) Rename(to string) { g.Name = to }

type speaker interface{ Hello(string) string }

func main() {
	t := reflect.TypeOf(greeter{})
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		fmt.Println(m.Name, m.Type, m.Func.IsValid(), m.Func.Kind())
	}

	m, ok := t.MethodByName("Hello")
	fmt.Println("found:", ok)
	out := m.Func.Call([]reflect.Value{
		reflect.ValueOf(greeter{Name: "ana"}),
		reflect.ValueOf("world"),
	})
	fmt.Println(out[0].String())

	// A pointer type holds the method of the value type and its own.
	pt := reflect.TypeOf(&greeter{})
	names := make([]string, 0, pt.NumMethod())
	for i := 0; i < pt.NumMethod(); i++ {
		names = append(names, pt.Method(i).Name)
	}
	fmt.Println(strings.Join(names, ","))

	g := &greeter{Name: "bo"}
	rm, _ := pt.MethodByName("Rename")
	rm.Func.Call([]reflect.Value{reflect.ValueOf(g), reflect.ValueOf("cy")})
	fmt.Println(g.Name)

	// The method of an interface type holds the zero Value, exactly as in Go.
	it := reflect.TypeOf((*speaker)(nil)).Elem()
	fmt.Println(it.Method(0).Name, it.Method(0).Func.IsValid())
}
