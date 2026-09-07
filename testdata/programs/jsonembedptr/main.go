// Command jsonembedptr proves the 9 cases of TestUnmarshalEmbeddedUnexported
// of encoding/json. Go refuses a write through an embedded pointer to an
// unexported struct type, and it allows a write to a field that an embedded
// value of such a type forwards.
package main

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type embed1 struct{ Q int }

type embed2 struct{ Q int }

type embed3 struct {
	Q int64 `json:",string"`
}

type withMethod struct{}

func (withMethod) F() {}

// S1 holds an embedded pointer to an unexported struct type. Go cannot
// allocate it, thus the write of Q fails and the write of R holds.
type S1 struct {
	*embed1
	R int
}

// S2 holds a field Q of its own, which takes precedence over the field of the
// embedded type.
type S2 struct {
	*embed1
	Q int
}

// S3 holds the embedded value, and no allocation stands in the way.
type S3 struct {
	embed1
	R int
}

// S4 holds 2 embedded types that each declare Q, thus the 2 fields annihilate
// and no write of the pointer runs.
type S4 struct {
	*embed1
	embed2
}

// S5 repeats S1 with a tag on the field of the embedded type.
type S5 struct {
	*embed3
	R int
}

// S6 gives the embedded type a name, thus the member holds an object.
type S6 struct {
	embed1 `json:"embed1"`
}

// S7 holds a named embedded type next to an embedded type that forwards Q.
type S7 struct {
	embed1 `json:"embed1"`
	embed2
}

// S8 names both embedded types and holds a field Q of its own.
type S8 struct {
	embed1 `json:"embed1"`
	embed2 `json:"embed2"`
	Q      int
}

// S9 holds an embedded unexported type that declares a method.
type S9 struct {
	withMethod `json:"embed"`
}

func main() {
	run("S1", `{"R":2,"Q":1}`, new(S1))
	run("S2", `{"Q":1}`, new(S2))
	run("S3", `{"R":2,"Q":1}`, new(S3))
	run("S4", `{"R":2}`, new(S4))
	run("S5", `{"R":2,"Q":1}`, new(S5))
	run("S6", `{"embed1": {"Q": 1}}`, new(S6))
	run("S7", `{"embed1": {"Q": 1}, "Q": 2}`, new(S7))
	run("S8", `{"embed1": {"Q": 1}, "embed2": {"Q": 2}, "Q": 3}`, new(S8))
	run("S9", `{"embed": {}}`, new(S9))
	fields()
}

// run decodes the input into the target and it writes the error and the value.
func run(name, in string, ptr any) {
	err := json.Unmarshal([]byte(in), ptr)
	fmt.Printf("%s: err=%v\n", name, errText(err))
	fmt.Printf("%s: value=%s\n", name, show(ptr))
}

// errText writes the type, the text, and the parts of an error.
func errText(err error) string {
	if err == nil {
		return "none"
	}
	if e, ok := err.(*json.UnmarshalTypeError); ok {
		return fmt.Sprintf("%T Value=%q Struct=%q Field=%q Offset=%d text=%v",
			e, e.Value, e.Struct, e.Field, e.Offset, e)
	}
	return fmt.Sprintf("%T %v", err, err)
}

// show writes each field of a decoded value. A nil embedded pointer writes
// the word nil, because the address itself differs between the 2 runtimes.
//
// The walk of the package reflect stays out of this function. `%v` of a
// reflect.Value that names an unexported field writes the text of the method
// String under gart, and Go writes the value itself.
func show(ptr any) string {
	switch v := ptr.(type) {
	case *S1:
		return "S1{embed1:" + ptrText(v.embed1) + fmt.Sprintf(" R:%d}", v.R)
	case *S2:
		return "S2{embed1:" + ptrText(v.embed1) + fmt.Sprintf(" Q:%d}", v.Q)
	case *S3:
		return fmt.Sprintf("S3{embed1:{%d} R:%d}", v.embed1.Q, v.R)
	case *S4:
		return "S4{embed1:" + ptrText(v.embed1) +
			fmt.Sprintf(" embed2:{%d}}", v.embed2.Q)
	case *S5:
		return "S5{embed3:" + ptr3Text(v.embed3) + fmt.Sprintf(" R:%d}", v.R)
	case *S6:
		return fmt.Sprintf("S6{embed1:{%d}}", v.embed1.Q)
	case *S7:
		return fmt.Sprintf("S7{embed1:{%d} embed2:{%d}}", v.embed1.Q, v.embed2.Q)
	case *S8:
		return fmt.Sprintf("S8{embed1:{%d} embed2:{%d} Q:%d}",
			v.embed1.Q, v.embed2.Q, v.Q)
	case *S9:
		return "S9{withMethod:{}}"
	}
	return "unknown"
}

func ptrText(p *embed1) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("&{%d}", p.Q)
}

func ptr3Text(p *embed3) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("&{%d}", p.Q)
}

// fields proves the rule of the read only mark of the package reflect.
func fields() {
	v := reflect.ValueOf(&S1{}).Elem()
	fmt.Printf("S1.embed1: exported=%t canSet=%t canAddr=%t canInterface=%t\n",
		v.Type().Field(0).IsExported(), v.Field(0).CanSet(),
		v.Field(0).CanAddr(), v.Field(0).CanInterface())
	fmt.Printf("S1.R: canSet=%t canInterface=%t\n",
		v.Field(1).CanSet(), v.Field(1).CanInterface())
	w := reflect.ValueOf(&S3{}).Elem()
	fmt.Printf("S3.embed1: canSet=%t\n", w.Field(0).CanSet())
	fmt.Printf("S3.embed1.Q: canSet=%t canInterface=%t\n",
		w.Field(0).Field(0).CanSet(), w.Field(0).Field(0).CanInterface())
	fmt.Printf("S3.embed1 hasPkgPath=%t anonymous=%t\n",
		w.Type().Field(0).PkgPath != "", w.Type().Field(0).Anonymous)
}
