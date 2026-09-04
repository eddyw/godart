// The program proves reflect.Value.Bytes for a defined slice type of bytes.
// Go gives the slice itself, thus a write through the result reaches the
// value. The descriptor of such a type holds its own name, and an assertion to
// []byte fails on that name alone.
package main

import (
	"fmt"
	"reflect"
)

type Raw []byte

type record struct {
	Body Raw
}

func main() {
	r := Raw{1, 2, 3}
	fmt.Println("plain", reflect.ValueOf(r).Bytes())
	fmt.Println("byte", reflect.ValueOf([]byte{4, 5}).Bytes())

	rec := record{Body: Raw{7, 8}}
	f := reflect.ValueOf(&rec).Elem().Field(0)
	b := f.Bytes()
	fmt.Println("field", b)

	// Go gives the slice itself, thus the write reaches the record.
	b[0] = 9
	fmt.Println("after", rec.Body)

	var arr [3]byte = [3]byte{10, 11, 12}
	fmt.Println("array", reflect.ValueOf(&arr).Elem().Bytes())
}
