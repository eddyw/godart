package main

import (
	"fmt"
	"reflect"
)

func main() {
	static := []reflect.Type{
		reflect.TypeOf(struct {
			X int "\xff"
		}{}),
		reflect.TypeOf(struct {
			X int "\xfe"
		}{}),
		reflect.TypeOf(struct {
			X int "\x00\n\t\"\\"
		}{}),
		reflect.TypeOf(struct {
			X int "json:\"a\xff\""
		}{}),
	}
	for _, t := range static {
		f := t.Field(0)
		fmt.Printf("static %x %q\n", string(f.Tag), t.String())
		dynamic := reflect.StructOf([]reflect.StructField{f})
		fmt.Printf("equal %t %x %q\n", dynamic == t, string(dynamic.Field(0).Tag), dynamic.String())
		fmt.Printf("lookup %x\n", f.Tag.Get("json"))
	}
	seen := map[reflect.Type]bool{}
	for _, tag := range []string{"", "json:\"x\"", "\xff", "\xfe", "\xc0\x80", "\x00\n\t\"\\", "é世界😀", "\u00a0\u200b", "json:\"a\xff\""} {
		f := []reflect.StructField{{Name: "X", Type: reflect.TypeOf(int(0)), Tag: reflect.StructTag(tag)}}
		t := reflect.StructOf(f)
		fmt.Printf("dynamic %x %q %t %t\n", string(t.Field(0).Tag), t.String(), seen[t], reflect.StructOf(f) == t)
		seen[t] = true
	}
	fmt.Println("types", len(seen))
}
