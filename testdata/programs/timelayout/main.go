package main

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

func main() {
	t := reflect.TypeFor[time.Time]()
	fmt.Println(t.Size(), t.Align(), unsafe.Sizeof(time.Time{}), unsafe.Alignof(time.Time{}))
	fmt.Println(t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fmt.Println(f.Name, f.Type, f.Offset, f.IsExported())
	}
	var pair struct {
		T time.Time
		B byte
	}
	fmt.Println(unsafe.Sizeof(pair), unsafe.Offsetof(pair.B))
	for _, value := range []time.Time{{}, time.Unix(-1, 123).UTC(), time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)} {
		copy := value.Add(0)
		fmt.Println(copy == value, reflect.DeepEqual(copy, value), value.Equal(copy))
	}
}
