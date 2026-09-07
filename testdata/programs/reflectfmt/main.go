// The program proves 2 rules of reflect that the package fmt and the package
// text/template read. fmt writes the value that a reflect.Value holds at the
// top level, and its String method below that level. A call of a nil function
// value panics with the text of Go, and the template reports that text.
package main

import (
	"fmt"
	"os"
	"reflect"
	"text/template"
)

type hidden struct{ n int }

func values() {

	fmt.Printf("%s|%v|%d|\n", reflect.ValueOf([]int{1}), reflect.ValueOf([]int{1}), reflect.ValueOf(5))
	fmt.Printf("%v|%s|%q|\n", reflect.ValueOf("text"), reflect.ValueOf("text"), reflect.ValueOf("text"))
	fmt.Printf("%v|\n", reflect.Value{})
	fmt.Printf("%v|\n", []any{reflect.Value{}})
	fmt.Printf("%v|%+v|\n", reflect.ValueOf(hidden{2}), reflect.ValueOf(hidden{2}))
	v := reflect.ValueOf(hidden{3}).Field(0)
	fmt.Printf("%v|%d|\n", v, v)
	fmt.Printf("%T|\n", reflect.ValueOf(1))
}

func nilCall() {

	var f func() string
	t := template.Must(template.New("x").Parse("{{call .}}"))
	err := t.Execute(os.Stdout, f)
	fmt.Println("\ntemplate", err)
	defer func() { fmt.Println("recovered", recover()) }()
	reflect.ValueOf(f).Call(nil)
}

func main() {
	values()
	nilCall()
}
