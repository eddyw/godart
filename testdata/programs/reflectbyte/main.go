package main

import (
	"fmt"
	"reflect"
)

type header struct {
	Data []byte
	Name []rune
	Flag byte
	Char rune
}

type raw []byte

type mark byte

type letter rune

func main() {
	// Type.String prints the name of the element type, thus uint8 and not
	// byte, and int32 and not rune.
	fmt.Println(reflect.TypeOf(byte(0)), reflect.TypeOf(byte(0)).Kind())
	fmt.Println(reflect.TypeOf(rune(0)), reflect.TypeOf(rune(0)).Kind())
	fmt.Println(reflect.TypeOf([]byte(nil)), reflect.TypeOf([]byte(nil)).Elem())
	fmt.Println(reflect.TypeOf([]rune(nil)), reflect.TypeOf([]rune(nil)).Elem())
	fmt.Println(reflect.TypeOf([4]byte{}), reflect.TypeOf(map[byte]rune(nil)))
	fmt.Println(reflect.TypeOf(&[]byte{}), reflect.TypeOf(make(chan rune)))

	// A named type keeps its own name, and its element reads uint8.
	fmt.Println(reflect.TypeOf(raw(nil)), reflect.TypeOf(raw(nil)).Elem())

	// A named type whose underlying type is an alias reads the kind of the
	// element type.
	fmt.Println(reflect.TypeOf(mark(0)), reflect.TypeOf(mark(0)).Kind())
	fmt.Println(reflect.TypeOf(letter(0)), reflect.TypeOf(letter(0)).Kind())

	// A struct field reads the same rule.
	t := reflect.TypeOf(header{})
	for i := 0; i < t.NumField(); i++ {
		fmt.Println(t.Field(i).Name, t.Field(i).Type, t.Field(i).Type.Kind())
	}
	fmt.Println(t)

	// The verb %T follows the same text.
	fmt.Printf("%T %T %T\n", []byte(nil), rune(0), map[string][]byte(nil))

	// A function type prints the element name too.
	fmt.Println(reflect.TypeOf(func(b []byte) rune { return 0 }))
}
