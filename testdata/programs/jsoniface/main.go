// The program jsoniface reads a JSON value into an interface that already
// holds a pointer. Go descends into that pointer and writes the value that it
// names, and net/rpc/jsonrpc reads its parameter that way.
package main

import (
	"encoding/json"
	"fmt"
)

// Args holds the 2 operands of a call.
type Args struct {
	A, B int
}

// holder holds 2 fields of an interface type.
type holder struct {
	V any `json:"v"`
	W any `json:"w"`
}

func main() {
	field()
	top()
	array()
	slot()
}

// field reads into a struct field of an interface type.
func field() {
	fmt.Println("== field ==")
	h := holder{V: &Args{}}
	err := json.Unmarshal([]byte(`{"v":{"A":1,"B":2},"w":{"A":3}}`), &h)
	fmt.Printf("err=%v v=%v w=%v\n", err, h.V, h.W)

	// A null gives the interface the value nil, and it keeps the pointer.
	h2 := holder{V: &Args{A: 9}}
	err = json.Unmarshal([]byte(`{"v":null}`), &h2)
	fmt.Printf("err=%v v=%v\n", err, h2.V)

	// A field that holds a value and no pointer takes a fresh map.
	h3 := holder{V: Args{A: 8}}
	err = json.Unmarshal([]byte(`{"v":{"A":4}}`), &h3)
	fmt.Printf("err=%v v=%v\n", err, h3.V)
}

// top reads into the interface that the call names.
func top() {
	fmt.Println("== top ==")
	var v any = &Args{}
	err := json.Unmarshal([]byte(`{"A":5,"B":6}`), &v)
	fmt.Printf("err=%v v=%v\n", err, v)

	var n any
	err = json.Unmarshal([]byte(`{"A":7}`), &n)
	fmt.Printf("err=%v n=%v\n", err, n)
}

// array reads into an array of interfaces. The codec of net/rpc/jsonrpc holds
// exactly this shape.
func array() {
	fmt.Println("== array ==")
	var arr [2]any
	arr[0] = &Args{}
	err := json.Unmarshal([]byte(`[{"A":1,"B":1},{"A":2}]`), &arr)
	fmt.Printf("err=%v 0=%v 1=%v\n", err, arr[0], arr[1])
}

// slot reads a raw message and then the parameter, exactly as the server codec
// of net/rpc/jsonrpc does.
func slot() {
	fmt.Println("== slot ==")
	var head struct {
		Method string           `json:"method"`
		Params *json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal([]byte(`{"method":"M","params":[{"A":6,"B":7}]}`), &head); err != nil {
		fmt.Println("head:", err)
		return
	}
	out := new(Args)
	var params [1]any
	params[0] = out
	err := json.Unmarshal(*head.Params, &params)
	fmt.Printf("err=%v method=%s args=%v\n", err, head.Method, *out)
}
