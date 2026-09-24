package main

import (
	"fmt"
	"reflect"
	"testing"
	"testing/synctest"
)

func main() {
	synctest.Test(new(testing.T), func(t *testing.T) {
		for _, operation := range []string{"receive", "send", "close"} {
			c := make(chan int)
			done := make(chan struct{})
			go func() {
				selected := reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(c)}
				if operation == "send" {
					selected.Dir = reflect.SelectSend
					selected.Send = reflect.ValueOf(42)
				}
				index, value, ok := reflect.Select([]reflect.SelectCase{
					{Dir: reflect.SelectRecv}, selected,
				})
				fmt.Println(operation, index, value.IsValid(), ok)
				if value.IsValid() {
					fmt.Println(value.Int())
				}
				close(done)
			}()
			synctest.Wait()
			switch operation {
			case "receive":
				c <- 31
			case "send":
				if value := <-c; value != 42 {
					panic(value)
				}
			case "close":
				close(c)
			}
			<-done
		}
		c := make(chan int, 1)
		c <- 53
		index, value, ok := reflect.Select([]reflect.SelectCase{
			{Dir: reflect.SelectRecv},
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(c)},
		})
		fmt.Println("immediate", index, value.Int(), ok)
		index, value, ok = reflect.Select([]reflect.SelectCase{
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(c)},
			{Dir: reflect.SelectDefault},
		})
		fmt.Println("default", index, value.IsValid(), ok)
	})
}
