package main

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Signed int64
type Unsigned uint64

func main() {
	for _, x := range []int64{9007199254740993, -9007199254740993, 9223372036854775807, -9223372036854775808} {
		v := reflect.ValueOf(x)
		var dst int64
		reflect.ValueOf(&dst).Elem().SetInt(v.Int())
		fmt.Println(x, v.Int(), dst, v.Convert(reflect.TypeFor[Signed]()).Interface())
		b, err := json.Marshal(x)
		var decoded int64
		read := json.Unmarshal(b, &decoded)
		fmt.Println(string(b), err, decoded, read)
	}
	for _, x := range []uint64{9007199254740993, 9223372036854775809, 18446744073709551615} {
		v := reflect.ValueOf(x)
		var dst uint64
		reflect.ValueOf(&dst).Elem().SetUint(v.Uint())
		fmt.Println(x, v.Uint(), dst, v.Convert(reflect.TypeFor[Unsigned]()).Interface())
		b, err := json.Marshal(x)
		var decoded uint64
		read := json.Unmarshal(b, &decoded)
		fmt.Println(string(b), err, decoded, read)
	}
	var narrow int8
	reflect.ValueOf(&narrow).Elem().SetInt(300)
	fmt.Println("narrow", narrow, reflect.ValueOf(int64(300)).Convert(reflect.TypeFor[int8]()).Interface())
}
