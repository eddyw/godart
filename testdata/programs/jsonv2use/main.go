package main

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"strings"
)

type point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type celsius float64

func main() {
	b, err := json.Marshal(point{X: 3, Y: 4})
	fmt.Println(string(b), err)

	var p point
	err = json.Unmarshal([]byte(`{"x":7,"y":9}`), &p)
	fmt.Println(p.X, p.Y, err)

	// A type specific marshaler reads the Encoder of the option set. The
	// function value takes the pointer of a type parameter inside the package,
	// thus the emitter wraps it.
	ms := json.JoinMarshalers(
		json.MarshalFunc(func(c celsius) ([]byte, error) {
			return fmt.Appendf(nil, `"%.1fC"`, float64(c)), nil
		}),
		json.MarshalToFunc(func(enc *jsontext.Encoder, s string) error {
			return enc.WriteToken(jsontext.String(strings.ToUpper(s)))
		}),
	)
	out, err := json.Marshal(struct {
		T celsius `json:"t"`
		N string  `json:"n"`
	}{T: 21.5, N: "ok"}, json.WithMarshalers(ms))
	fmt.Println(string(out), err)

	us := json.JoinUnmarshalers(
		json.UnmarshalFunc(func(b []byte, c *celsius) error {
			var f float64
			if _, e := fmt.Sscanf(strings.Trim(string(b), `"`), "%fC", &f); e != nil {
				return e
			}
			*c = celsius(f)
			return nil
		}),
	)
	var got struct {
		T celsius `json:"t"`
	}
	err = json.Unmarshal([]byte(`{"t":"30.5C"}`), &got, json.WithUnmarshalers(us))
	fmt.Println(got.T, err)

	// The indent option reads the same option set through jsontext.
	pretty, err := json.Marshal(point{X: 1, Y: 2}, jsontext.WithIndent("  "))
	fmt.Println(string(pretty), err)
}
