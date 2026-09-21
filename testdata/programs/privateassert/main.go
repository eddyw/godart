package main

import (
	"example.com/privateassert/foreign"
	"fmt"
)

type Local int

func (Local) hidden(int) {}
func (Local) Visible()   {}

type Wrong struct{ foreign.Wrong }
type Right struct{ foreign.Right }

func report(f func()) {
	defer func() {
		if value := recover(); value != nil {
			fmt.Println(value.(error).Error())
		}
	}()
	f()
}

func main() {
	report(func() { _ = any(Local(0)).(foreign.Private) })
	report(func() { _ = any(Local(0)).(foreign.Public) })
	report(func() { _ = any(Wrong{}).(foreign.Private) })
	if _, ok := any(Right{}).(foreign.Private); !ok {
		panic("promoted private method")
	}
}
