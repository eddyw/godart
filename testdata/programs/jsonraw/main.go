package main

import (
	"encoding/json"
	"fmt"
)

const source = "  {\n \"b\" : 1e2 ,  \"a\":[1, 2 ,   {} ],\n \"c\" : \"\\u0041\\n\" }  "

// show writes the value and the cause. The call of the method stands in front
// of the read of the value, because the order of the operands of 1 call is not
// the order of the source in Go.
func show(name string, err error, m json.RawMessage) {
	fmt.Printf("%s: %q err=%v\n", name, string(m), err)
}

func copyOf(text string) json.RawMessage {
	return append(json.RawMessage(nil), text...)
}

func main() {
	src := json.RawMessage(source)

	c := copyOf(source)
	show("compact", c.Compact(), c)

	i := copyOf(source)
	show("indent", i.Indent(), i)

	f := copyOf(source)
	show("format", f.Format(), f)

	k := copyOf(source)
	show("canonical", k.Canonicalize(), k)

	fmt.Println("kind", src.Kind(), json.RawMessage(`12`).Kind(),
		json.RawMessage(`"x"`).Kind(), json.RawMessage(`[]`).Kind(),
		json.RawMessage(`true`).Kind(), json.RawMessage(``).Kind())
	fmt.Printf("string %q\n", src.String())

	var empty json.RawMessage
	fmt.Printf("nil string %q clone %v\n", empty.String(), empty.Clone() == nil)
	clone := src.Clone()
	clone[2] = ' '
	fmt.Println("clone holds a copy", string(src) == source)

	fmt.Println("valid", src.IsValid())
	fmt.Println("duplicate", json.RawMessage(`{"a":1,"a":2}`).IsValid())
	fmt.Println("bad utf8", json.RawMessage("\"\xff\"").IsValid())
	fmt.Println("trailing", json.RawMessage(`1 2`).IsValid())
	fmt.Println("lone surrogate", json.RawMessage(`"\ud800"`).IsValid())
	fmt.Println("pair", json.RawMessage(`"😀"`).IsValid())

	dup := json.RawMessage(`{"a":1,"a":2}`)
	fmt.Println("format of a duplicate", dup.Format() != nil)

	bad := json.RawMessage("\"\xff\"")
	fmt.Printf("compact of bad utf8 err=%v %q\n", bad.Compact(), string(bad))

	numbers := copyOf(`[1.0, 100.0, 1e2, 0.1, -0, 1e21, 1e-7, 12345678901234567890]`)
	show("canonical numbers", numbers.Canonicalize(), numbers)

	plain := copyOf(`[1.0, 100.0, 1e2, 0.1]`)
	show("format numbers", plain.Format(), plain)

	sorted := copyOf(`{"b":1,"A":2,"a":3,"":4}`)
	show("sorted", sorted.Canonicalize(), sorted)

	nested := copyOf(`{"x":{"z":1,"y":{"q":2,"p":3}}}`)
	show("nested sort", nested.Canonicalize(), nested)

	escapes := copyOf(`"Aé😀\/\b\f"`)
	show("escapes", escapes.Format(), escapes)

	var none json.RawMessage
	fmt.Println("format of nil", none.Format() != nil)

	deep := copyOf(`{"a":[{"b":[]},[],{}],"c":null}`)
	show("indent deep", deep.Indent(), deep)

	round := copyOf(source)
	_ = round.Format()
	before := string(round)
	_ = round.Format()
	fmt.Println("format is stable", before == string(round))
}
