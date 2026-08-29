package main

import (
	"fmt"
	"reflect"
)

type errorT struct{ msg string }

func (e errorT) Error() string { return "errorT: " + e.msg }

type pathError struct{ op string }

func (e *pathError) Error() string { return "pathError: " + e.op }

type timeouter interface{ Timeout() bool }

func main() {
	var i int
	var s []string
	var m map[string]int
	var a [3]int
	var e errorT
	var p *pathError
	var err error
	var to timeouter

	for _, t := range []reflect.Type{
		reflect.TypeOf(i),
		reflect.TypeOf(s),
		reflect.TypeOf(m),
		reflect.TypeOf(a),
		reflect.TypeOf(e),
		reflect.TypeOf(&e),
		reflect.TypeOf(p),
		reflect.TypeOf(&p),
		reflect.TypeOf(&err),
		reflect.TypeOf(&to),
	} {
		fmt.Println(t.String(), t.Kind(), t.Comparable())
	}

	for _, t := range []reflect.Type{
		reflect.TypeOf(&i),
		reflect.TypeOf(s),
		reflect.TypeOf(m),
		reflect.TypeOf(a),
		reflect.TypeOf(&e),
		reflect.TypeOf(&p),
		reflect.TypeOf(&err),
	} {
		fmt.Println(t.String(), "->", t.Elem().String(), t.Elem().Kind())
	}

	// A pointer to an interface reads and writes the interface value.
	err = errorT{"first"}
	rerr := reflect.ValueOf(&err)
	fmt.Println(rerr.Elem().Interface())
	rerr.Elem().Set(reflect.Zero(reflect.TypeOf(&err).Elem()))
	fmt.Println(err == nil, rerr.Elem().Interface() == nil)

	// A pointer to a struct writes each field.
	e = errorT{"second"}
	re := reflect.ValueOf(&e)
	fmt.Println(re.Elem().Interface())
	re.Elem().Set(reflect.Zero(reflect.TypeOf(&e).Elem()))
	fmt.Println(e, re.Elem().Interface())

	// A pointer to a pointer writes the box.
	p = &pathError{"open"}
	rp := reflect.ValueOf(&p)
	fmt.Println(rp.Elem().Interface())
	rp.Elem().Set(reflect.Zero(reflect.TypeOf(&p).Elem()))
	fmt.Println(p == nil)

	// A pointer to an integer writes the box.
	i = 7
	ri := reflect.ValueOf(&i)
	fmt.Println(ri.Elem().Interface())
	ri.Elem().Set(reflect.Zero(reflect.TypeOf(&i).Elem()))
	fmt.Println(i)

	// A write of a value that is not the zero value.
	rerr.Elem().Set(reflect.ValueOf(errorT{"third"}))
	fmt.Println(err)
	ri.Elem().Set(reflect.ValueOf(42))
	fmt.Println(i)

	fmt.Println(reflect.DeepEqual(errorT{"a"}, errorT{"a"}))
	fmt.Println(reflect.ValueOf(3).Interface())
}
