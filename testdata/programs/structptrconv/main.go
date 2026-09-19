// The program proves that a pointer conversion between 2 defined struct types
// gives a pointer to the same value.
package main

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type inner struct {
	a int
	b string
}

type outer inner

func (o *outer) label() string {
	return fmt.Sprintf("outer(%d,%s)", o.a, o.b)
}

func (i *inner) label() string {
	return fmt.Sprintf("inner(%d,%s)", i.a, i.b)
}

func main() {
	localConversions()
	p := &inner{a: 1, b: "one"}
	q := (*outer)(p)

	fmt.Println(p.label(), q.label())

	q.a = 2
	fmt.Println("write through q reaches p", p.a)

	p.b = "two"
	fmt.Println("write through p reaches q", q.b)

	back := (*inner)(q)
	fmt.Println("round trip", back == p)

	other := (*outer)(p)
	fmt.Println("2 conversions of 1 pointer", q == other)

	var face any = p
	_, isInner := face.(*inner)
	_, isOuter := face.(*outer)
	fmt.Println("identity of p", isInner, isOuter)

	face = q
	_, isInner = face.(*inner)
	_, isOuter = face.(*outer)
	fmt.Println("identity of q", isInner, isOuter)

	fmt.Printf("types %T %T\n", p, q)

	type namer interface{ label() string }
	fmt.Println(face.(namer).label())

	value := outer(*p)
	value.a = 3
	fmt.Println("a value conversion copies", p.a, value.a)
}

type payload struct {
	Count int `json:"count"`
}

func (p *payload) MarshalJSON() ([]byte, error) {
	type Alias payload
	return json.Marshal((*Alias)(p))
}

func (p *payload) UnmarshalJSON(data []byte) error {
	type Alias payload
	return json.Unmarshal(data, (*Alias)(p))
}

func localConversions() {
	type Alias payload
	p := &payload{Count: 4}
	a := (*Alias)(p)
	a.Count++
	fmt.Println("local pointer", p.Count, (*payload)(a) == p, (*Alias)(p) == a)
	var empty *payload
	fmt.Println("local nil", (*Alias)(empty) == nil)
	var boxed any = a
	_, source := boxed.(*payload)
	_, target := boxed.(*Alias)
	fmt.Println("local identity", source, target, reflect.TypeOf(a).Elem().NumMethod())
	reflect.ValueOf(a).Elem().Field(0).SetInt(6)
	fmt.Println("local reflect write", p.Count)
	copy := Alias(*p)
	copy.Count++
	fmt.Println("local copy", copy.Count, p.Count)
	zero := new(Alias)
	literal := Alias{Count: 8}
	values := make([]Alias, 1)
	fmt.Println("local allocation", zero.Count, literal.Count, values[0].Count)
	data, err := json.Marshal(p)
	fmt.Println("local marshal", string(data), err)
	err = json.Unmarshal([]byte(`{"count":9}`), p)
	fmt.Println("local unmarshal", p.Count, err)
}
