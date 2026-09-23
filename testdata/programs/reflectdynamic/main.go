package main

import (
	"fmt"
	"reflect"
	"testing"
	"testing/synctest"
	"time"
)

func main() {
	type Answer int
	synctest.Test(new(testing.T), func(t *testing.T) {
		for n := 0; n <= 4; n++ {
			results := make([]reflect.Type, n)
			for i := range results {
				results[i] = reflect.TypeOf(Answer(0))
			}
			f := reflect.MakeFunc(reflect.FuncOf(nil, results, false), func([]reflect.Value) []reflect.Value {
				time.Sleep(time.Nanosecond)
				values := make([]reflect.Value, n)
				for i := range values {
					values[i] = reflect.ValueOf(Answer(47 + i))
				}
				return values
			})
			values := f.Call(nil)
			fmt.Print(n)
			for _, value := range values {
				fmt.Print(" ", value.Interface())
			}
			fmt.Println()
		}
	})
}
