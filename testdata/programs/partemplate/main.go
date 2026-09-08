// The witness of a call of a nil function of text/template and of the text of
// a reflect.Value inside an error.
package main

import (
	"fmt"
	"os"
	"reflect"
	"text/template"
)

type holder struct {
	F func() string
}

func run(name, text string, data any) {
	t, err := template.New(name).Parse(text)
	if err != nil {
		fmt.Println(name, "parse", err)
		return
	}
	err = t.Execute(os.Stdout, data)
	fmt.Println()
	fmt.Println(name, "exec", err)
}

func main() {
	run("nilcall", "{{call .F}}", holder{})
	run("eq", "{{eq . .}}", []int{1})
	fmt.Printf("%s|\n", reflect.ValueOf([]int{1}))
	fmt.Printf("%v|\n", reflect.ValueOf([]int{1}))
	fmt.Printf("%d|\n", reflect.ValueOf(7))
	fmt.Printf("%s|\n", reflect.ValueOf("text"))
}
