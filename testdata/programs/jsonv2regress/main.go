package main

import (
	"encoding"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"net/netip"
	"reflect"
	"strconv"
)

type textValue struct{ Text string }

func (v textValue) MarshalText() ([]byte, error)  { return []byte(v.Text), nil }
func (v *textValue) UnmarshalText(b []byte) error { v.Text = string(b); return nil }

type pointerValue struct{ Text string }

func (v *pointerValue) MarshalText() ([]byte, error) { return []byte(v.Text), nil }
func (v *pointerValue) UnmarshalText(b []byte) error { v.Text = string(b); return nil }

type genericValue[T any] struct{ Value T }

func (v *genericValue[T]) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprint(v.Value)), nil
}

func showGeneric[T any](value T) {
	v := struct{ genericValue[T] }{genericValue[T]{value}}
	show("generic anonymous", v)
}

func show(name string, v any, opts ...json.Options) {
	b, err := json.Marshal(v, opts...)
	fmt.Println(name, string(b), err)
}

func main() {
	for _, tag := range []reflect.StructTag{`json:"a,\""`, `json:"a\\b"`, `json:"a\tb"`, `json:"\q"`, `other:"x" json:"ok"`} {
		v, ok := tag.Lookup("json")
		fmt.Printf("tag %q %t\n", v, ok)
	}
	for _, text := range []string{"1.2.3.4", "2001:db8::1", "fe80::1%en0"} {
		ip := netip.MustParseAddr(text)
		b, err := ip.MarshalText()
		fmt.Println("ip", ip.String(), ip.IsValid(), ip.Is4(), string(b), err)
		show("ip json", ip)
	}
	v := struct{ textValue }{textValue{"hello"}}
	fmt.Println("anonymous methods", reflect.TypeOf(v).NumMethod(), reflect.TypeOf(&v).NumMethod())
	_, ok := any(v).(encoding.TextMarshaler)
	fmt.Println("anonymous assertion", ok)
	show("anonymous", v)
	err := json.Unmarshal([]byte(`"changed"`), &v)
	fmt.Println("anonymous decode", v.Text, err)
	p := struct{ pointerValue }{pointerValue{"pointer"}}
	show("anonymous pointer method", p)
	err = json.Unmarshal([]byte(`"changed pointer"`), &p)
	fmt.Println("anonymous pointer decode", p.Text, err)
	showGeneric(42)
	errValue := error(&strconv.NumError{Func: "ParseUint", Num: "-1", Err: strconv.ErrSyntax})
	show("error", []error{errValue}, json.WithMarshalers(json.JoinMarshalers(
		json.MarshalToFunc(func(e *jsontext.Encoder, v *strconv.NumError) error {
			return e.WriteToken(jsontext.String(v.Error()))
		}),
		json.MarshalFunc(func(error) ([]byte, error) { return []byte(`"fallback"`), nil }),
	)))
}
