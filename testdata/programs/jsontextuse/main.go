// The program proves that encoding/json/jsontext reads a token stream, writes
// one, and puts a value into its canonical form.
package main

import (
	"encoding/json/jsontext"
	"fmt"
	"strings"
)

func main() {
	dec := jsontext.NewDecoder(strings.NewReader(`{"k":[1,2],"s":"x","f":1.5,"b":true,"n":null}`))
	for {
		tok, err := dec.ReadToken()
		if err != nil {
			break
		}
		fmt.Print(tok.Kind(), tok.String(), " ")
	}
	fmt.Println()

	val := jsontext.Value(`{"b":2,"a":{"d":4,"c":3}}`)
	if err := val.Canonicalize(); err != nil {
		panic(err)
	}
	fmt.Println("canonical", val.String())

	compact := jsontext.Value("{\n  \"a\": 1\n}")
	if err := compact.Compact(); err != nil {
		panic(err)
	}
	fmt.Println("compact", compact.String())

	var out strings.Builder
	enc := jsontext.NewEncoder(&out)
	for _, tok := range []jsontext.Token{
		jsontext.BeginObject,
		jsontext.String("name"),
		jsontext.String("Ada"),
		jsontext.String("age"),
		jsontext.Int(36),
		jsontext.String("rate"),
		jsontext.Float(1.5),
		jsontext.String("live"),
		jsontext.True,
		jsontext.EndObject,
	} {
		if err := enc.WriteToken(tok); err != nil {
			panic(err)
		}
	}
	fmt.Println("written", out.String())

	bad := jsontext.NewDecoder(strings.NewReader(`{"a":}`))
	for {
		_, err := bad.ReadToken()
		if err != nil {
			fmt.Println("error", err)
			break
		}
	}

	fmt.Println("valid", jsontext.Value(`{"a":1}`).IsValid(),
		jsontext.Value(`{"a":}`).IsValid())
}
