package main

import (
	"example.com/linkvar/bridge"
	"example.com/linkvar/other"
	"example.com/linkvar/storage"
	"fmt"
	_ "unsafe"
)

//go:linkname number example.com/linkvar/storage.count
var number int

//go:linkname state example.com/linkvar/storage.state
var state struct{ N int }

var address = &number

func main() {
	fmt.Println(number, storage.Read(), other.Read(), state.N)
	fmt.Println(address == storage.Address(), address == other.Address(), address == bridge.Address())
	*address = 21
	fmt.Println(number, storage.Read(), other.Read())
	number++
	fmt.Println(*storage.Address())
	state.N = 5
	fmt.Println(storage.State())
	storage.SetState(9)
	fmt.Println(state.N)
	state = struct{ N int }{11}
	fmt.Println(storage.State())
}
