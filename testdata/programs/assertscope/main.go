package main

import (
	left "example.com/assertscope/left"
	right "example.com/assertscope/right"
	"fmt"
)

func report(label string, f func()) {
	defer func() {
		if value := recover(); value != nil {
			fmt.Println(label, value.(error).Error())
		}
	}()
	f()
	panic("assertion did not panic")
}

func assert[T any](x any)        { _ = x.(T) }
func assertPointer[T any](x any) { _ = x.(*T) }

func local() any {
	type Value int
	return Value(0)
}

func main() {
	type Value int
	report("scope", func() { _ = local().(Value) })
	report("generic scope", func() { assert[Value](local()) })
	report("named", func() { _ = any(left.Number(0)).(right.Number) })
	report("generic named", func() { assert[right.Number](left.Number(0)) })
	report("pointer", func() { _ = any(new(left.Number)).(*right.Number) })
	report("generic pointer", func() { assertPointer[right.Number](new(left.Number)) })
	report("method pointer", func() { _ = any(new(left.Method)).(*right.Method) })
	report("generic method pointer", func() { assertPointer[right.Method](new(left.Method)) })
	report("generic pointer argument", func() { assert[*right.Method](new(left.Method)) })
	report("private struct", func() { right.AssertPrivate(left.Private()) })
	report("promoted struct", func() { right.AssertPromoted(left.Promoted()) })
}
