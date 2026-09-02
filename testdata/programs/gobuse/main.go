// Command gobuse round trips each shape that the gob overlay supports. It
// writes the hex of the stream, thus the wire format itself is under test.
package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"time"
)

// roundTrip writes a value, prints the hex of the stream, and reads the value
// again into out. It reports whether the 2 steps hold.
func roundTrip(name string, in, out any) bool {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(in); err != nil {
		fmt.Printf("%-16s encode error: %v\n", name, err)
		return false
	}
	fmt.Printf("%-16s %s\n", name, hex.EncodeToString(buf.Bytes()))
	if err := gob.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(out); err != nil {
		fmt.Printf("%-16s decode error: %v\n", name, err)
		return false
	}
	return true
}

func show(name string, v any) { fmt.Printf("%-16s %v\n", name, v) }

// decodeInto hides the argument from the vet check of a non-pointer target.
func decodeInto(dec *gob.Decoder, v any) error { return dec.Decode(v) }

type point struct {
	X, Y int
	Name string
}

type nested struct {
	Head  point
	Tail  *point
	Items []int
	Flags map[string]bool
	Grid  [2]int
	Raw   []byte
	small int
}

type shape interface {
	Area() int
}

type square struct{ Side int }

func (s square) Area() int { return s.Side * s.Side }

type circle struct{ R int }

func (c circle) Area() int { return 3 * c.R * c.R }

type box struct {
	Inside shape
	Label  string
}

// tree names itself, thus it exercises a recursive type.
type tree struct {
	Value int
	Kids  []*tree
}

// slowReader holds no ReadByte method, thus the decoder wraps it in a
// bufio.Reader.
type slowReader struct {
	data []byte
	at   int
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.at >= len(r.data) {
		return 0, io.EOF
	}
	p[0] = r.data[r.at]
	r.at++
	return 1, nil
}

// stamp writes its own bytes, thus it exercises GobEncoder and GobDecoder.
type stamp struct {
	Year  int
	Place string
}

func (s stamp) GobEncode() ([]byte, error) {
	return []byte(fmt.Sprintf("%d/%s", s.Year, s.Place)), nil
}

func (s *stamp) GobDecode(b []byte) error {
	var year int
	var place string
	if _, err := fmt.Sscanf(string(b), "%d/%s", &year, &place); err != nil {
		return err
	}
	s.Year, s.Place = year, place
	return nil
}

func numbers() {
	fmt.Println("== numbers ==")
	var i int
	if roundTrip("int", 42, &i) {
		show("int", i)
	}
	var i2 int
	if roundTrip("int negative", -7, &i2) {
		show("int negative", i2)
	}
	var i3 int
	if roundTrip("int big", 1<<40, &i3) {
		show("int big", i3)
	}
	var i8 int8
	if roundTrip("int8", int8(-5), &i8) {
		show("int8", i8)
	}
	var i16 int16
	if roundTrip("int16", int16(300), &i16) {
		show("int16", i16)
	}
	var i32 int32
	if roundTrip("int32", int32(-70000), &i32) {
		show("int32", i32)
	}
	var i64 int64
	if roundTrip("int64", int64(1)<<33, &i64) {
		show("int64", i64)
	}
	var u uint
	if roundTrip("uint", uint(42), &u) {
		show("uint", u)
	}
	var u8 uint8
	if roundTrip("uint8", uint8(200), &u8) {
		show("uint8", u8)
	}
	var u16 uint16
	if roundTrip("uint16", uint16(60000), &u16) {
		show("uint16", u16)
	}
	var u32 uint32
	if roundTrip("uint32", uint32(4000000000), &u32) {
		show("uint32", u32)
	}
	var u64 uint64
	if roundTrip("uint64", uint64(1)<<62, &u64) {
		show("uint64", u64)
	}
	var up uintptr
	if roundTrip("uintptr", uintptr(9), &up) {
		show("uintptr", up)
	}
	var f32 float32
	if roundTrip("float32", float32(2.5), &f32) {
		show("float32", f32)
	}
	var f64 float64
	if roundTrip("float64", 3.25, &f64) {
		show("float64", f64)
	}
	var fs float64
	if roundTrip("float small", 1e-9, &fs) {
		show("float small", fs)
	}
	var c64 complex64
	if roundTrip("complex64", complex64(complex(1, 2)), &c64) {
		show("complex64", c64)
	}
	var c128 complex128
	if roundTrip("complex128", complex(3, -4), &c128) {
		show("complex128", c128)
	}
}

func scalars() {
	fmt.Println("== other basics ==")
	var b bool
	if roundTrip("bool true", true, &b) {
		show("bool true", b)
	}
	var s string
	if roundTrip("string", "hello, gob", &s) {
		show("string", s)
	}
	var s2 string
	if roundTrip("string empty", "", &s2) {
		show("string empty", s2)
	}
	var raw []byte
	if roundTrip("bytes", []byte{1, 2, 3, 255}, &raw) {
		show("bytes", raw)
	}
	var raw2 []byte
	if roundTrip("bytes empty", []byte{}, &raw2) {
		show("bytes empty", raw2)
	}
}

func containers() {
	fmt.Println("== containers ==")
	var xs []int
	if roundTrip("[]int", []int{1, 2, 3}, &xs) {
		show("[]int", xs)
	}
	var ss []string
	if roundTrip("[]string", []string{"a", "bb", "ccc"}, &ss) {
		show("[]string", ss)
	}
	var fs []float64
	if roundTrip("[]float64", []float64{1.5, -2.5}, &fs) {
		show("[]float64", fs)
	}
	var bs []bool
	if roundTrip("[]bool", []bool{true, false, true}, &bs) {
		show("[]bool", bs)
	}
	var a3 [3]int
	if roundTrip("[3]int", [3]int{7, 8, 9}, &a3) {
		show("[3]int", a3)
	}
	var a2 [2]string
	if roundTrip("[2]string", [2]string{"x", "y"}, &a2) {
		show("[2]string", a2)
	}
	var xss [][]int
	if roundTrip("[][]int", [][]int{{1}, {2, 3}}, &xss) {
		show("[][]int", xss)
	}
	// A map of 1 pair holds 1 order, thus the hex of it repeats.
	var m1 map[string]int
	if roundTrip("map 1 pair", map[string]int{"one": 1}, &m1) {
		show("map 1 pair", m1)
	}
	var m2 map[int]string
	if roundTrip("map int key", map[int]string{7: "seven"}, &m2) {
		show("map int key", m2)
	}
	// A map of 3 pairs holds no fixed order on the wire, thus the test reads
	// the decoded pairs in the order of the key.
	var m3 map[string]int
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(map[string]int{"a": 1, "b": 2, "c": 3}); err != nil {
		fmt.Println("map 3 pairs encode error:", err)
		return
	}
	if err := gob.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&m3); err != nil {
		fmt.Println("map 3 pairs decode error:", err)
		return
	}
	show("map 3 pairs", sortedPairs(m3))
	// A pointer flattens to its element, thus the stream holds no pointer and
	// a plain int reads it back.
	v := 11
	var flat int
	if roundTrip("*int", &v, &flat) {
		show("*int", flat)
	}
}

func sortedPairs(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s=%d", k, v))
	}
	sort.Strings(out)
	return out
}

func sortedFlags(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s=%t", k, v))
	}
	sort.Strings(out)
	return out
}

func structs() {
	fmt.Println("== structs ==")
	var p1 point
	if roundTrip("point", point{X: 1, Y: 2, Name: "p"}, &p1) {
		show("point", p1)
	}
	var p2 point
	if roundTrip("point zero", point{}, &p2) {
		show("point zero", p2)
	}
	var p3 point
	if roundTrip("point partial", point{Y: 5}, &p3) {
		show("point partial", p3)
	}
	var p4 *point
	if roundTrip("*point", &point{X: 3, Y: 4, Name: "q"}, &p4) {
		show("*point", *p4)
	}
	var ps []point
	if roundTrip("[]point", []point{{X: 1}, {Y: 2}}, &ps) {
		show("[]point", ps)
	}

	in := nested{
		Head:  point{X: 1, Y: 2, Name: "head"},
		Tail:  &point{X: 3, Y: 4, Name: "tail"},
		Items: []int{5, 6},
		Flags: map[string]bool{"on": true},
		Grid:  [2]int{8, 9},
		Raw:   []byte("raw"),
		small: 3,
	}
	var out nested
	if roundTrip("nested", in, &out) {
		fmt.Printf("%-16s head=%v tail=%v items=%v grid=%v raw=%q small=%d\n",
			"nested", out.Head, *out.Tail, out.Items, out.Grid, out.Raw, out.small)
		fmt.Printf("%-16s flags=%v\n", "nested", sortedFlags(out.Flags))
	}
}

func interfaces() {
	fmt.Println("== interfaces ==")
	gob.Register(square{})
	var out box
	if roundTrip("box", box{Inside: square{Side: 4}, Label: "sq"}, &out) {
		fmt.Printf("%-16s label=%s area=%d value=%v\n",
			"box", out.Label, out.Inside.Area(), out.Inside)
	}
	var empty box
	if roundTrip("box nil", box{Label: "none"}, &empty) {
		fmt.Printf("%-16s label=%s inside=%v\n", "box nil", empty.Label, empty.Inside)
	}
	gob.RegisterName("mycircle", circle{})
	var any1 any = circle{R: 3}
	var got any
	if roundTrip("any", &any1, &got) {
		show("any", got)
	}
}

func gobcoder() {
	fmt.Println("== GobEncoder ==")
	var s stamp
	if roundTrip("stamp", stamp{Year: 1971, Place: "bonn"}, &s) {
		show("stamp", s)
	}
	var ss []stamp
	if roundTrip("[]stamp", []stamp{{Year: 1, Place: "a"}, {Year: 2, Place: "b"}}, &ss) {
		show("[]stamp", ss)
	}
}

// stdtypes proves that a standard type with a GobEncoder method travels.
func stdtypes() {
	fmt.Println("== standard types ==")
	in := time.Date(1971, time.April, 2, 3, 4, 5, 6, time.UTC)
	var out time.Time
	if roundTrip("time.Time", in, &out) {
		show("time.Time", out.UTC().Format(time.RFC3339Nano))
	}
	var d time.Duration
	if roundTrip("time.Duration", 90*time.Second, &d) {
		show("time.Duration", d)
	}
	var lists map[string][]int
	if roundTrip("map of slices", map[string][]int{"a": {1, 2}}, &lists) {
		show("map of slices", lists)
	}
}

func recursive() {
	fmt.Println("== recursive ==")
	in := tree{Value: 1, Kids: []*tree{{Value: 2}, {Value: 3, Kids: []*tree{{Value: 4}}}}}
	var out tree
	if roundTrip("tree", in, &out) {
		fmt.Printf("%-16s %d %d %d %d\n", "tree", out.Value, out.Kids[0].Value,
			out.Kids[1].Value, out.Kids[1].Kids[0].Value)
	}
}

func slice0fAny() {
	fmt.Println("== []any ==")
	var out []any
	if roundTrip("[]any", []any{square{Side: 2}, circle{R: 5}}, &out) {
		for _, v := range out {
			fmt.Printf("%-16s %v %d\n", "[]any", v, v.(shape).Area())
		}
	}
}

func plainReader() {
	fmt.Println("== plain reader ==")
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(point{X: 9, Y: 8, Name: "slow"}); err != nil {
		fmt.Println("encode error:", err)
		return
	}
	var out point
	if err := gob.NewDecoder(&slowReader{data: buf.Bytes()}).Decode(&out); err != nil {
		fmt.Println("slow decode error:", err)
		return
	}
	show("slow", out)
}

func stream() {
	fmt.Println("== stream ==")
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	for i := 0; i < 3; i++ {
		if err := enc.Encode(point{X: i, Y: i * 2, Name: fmt.Sprint("p", i)}); err != nil {
			fmt.Println("stream encode error:", err)
			return
		}
	}
	fmt.Printf("%-16s %s\n", "3 points", hex.EncodeToString(buf.Bytes()))
	dec := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
	for i := 0; i < 3; i++ {
		var p point
		if err := dec.Decode(&p); err != nil {
			fmt.Println("stream decode error:", err)
			return
		}
		show("point", p)
	}
	var p point
	show("eof", dec.Decode(&p))
}

func mismatch() {
	fmt.Println("== other targets ==")
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(point{X: 1, Y: 2, Name: "z"}); err != nil {
		fmt.Println("encode error:", err)
		return
	}
	// A struct with fewer fields drops the ones that it does not hold.
	type small struct{ Y int }
	var out small
	if err := gob.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&out); err != nil {
		fmt.Println("small decode error:", err)
	} else {
		show("small", out)
	}
	// A nil target drops the whole value.
	if err := decodeInto(gob.NewDecoder(bytes.NewReader(buf.Bytes())), nil); err != nil {
		fmt.Println("nil decode error:", err)
	} else {
		fmt.Printf("%-16s dropped\n", "nil target")
	}
	enc2 := gob.NewEncoder(&bytes.Buffer{})
	fmt.Println("encode nil:", enc2.Encode(nil))
	var target int
	fmt.Println("decode value:", decodeInto(gob.NewDecoder(bytes.NewReader(buf.Bytes())), target))
	var wrong string
	fmt.Println("wrong type:", gob.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&wrong))
}

func main() {
	numbers()
	scalars()
	containers()
	structs()
	interfaces()
	gobcoder()
	stdtypes()
	recursive()
	slice0fAny()
	plainReader()
	stream()
	mismatch()
}
