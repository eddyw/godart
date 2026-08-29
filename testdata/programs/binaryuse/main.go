package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type point struct {
	X   int32
	Y   int32
	Tag [4]byte
	On  bool
	F   float64
}

func main() {
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b, 0x01020304)
	fmt.Println(b[:4], binary.BigEndian.Uint32(b))
	binary.LittleEndian.PutUint64(b, 0x0102030405060708)
	fmt.Println(b, binary.LittleEndian.Uint64(b))
	fmt.Println(binary.BigEndian.Uint16([]byte{1, 2}), binary.LittleEndian.Uint16([]byte{1, 2}))

	out := binary.BigEndian.AppendUint32(nil, 0x0a0b0c0d)
	out = binary.LittleEndian.AppendUint16(out, 0x0102)
	fmt.Println(out)

	buf := make([]byte, 16)
	n := binary.PutVarint(buf, -1234567)
	fmt.Println("varint", buf[:n])
	v, m := binary.Varint(buf)
	fmt.Println("read", v, m)
	n = binary.PutUvarint(buf, 987654321)
	u, k := binary.Uvarint(buf[:n])
	fmt.Println("uvarint", u, k)

	r := bytes.NewReader(buf[:n])
	uv, err := binary.ReadUvarint(r)
	fmt.Println("reader", uv, err)

	p := point{X: -5, Y: 7, Tag: [4]byte{'a', 'b', 'c', 'd'}, On: true, F: 2.5}
	fmt.Println("size", binary.Size(p))

	var w bytes.Buffer
	if err := binary.Write(&w, binary.BigEndian, p); err != nil {
		fmt.Println("write", err)
		return
	}
	fmt.Println("bytes", w.Bytes())

	var back point
	if err := binary.Read(bytes.NewReader(w.Bytes()), binary.BigEndian, &back); err != nil {
		fmt.Println("read", err)
		return
	}
	fmt.Println("back", back, back == p)

	values := []int32{1, -2, 3}
	var vw bytes.Buffer
	if err := binary.Write(&vw, binary.LittleEndian, values); err != nil {
		fmt.Println("write slice", err)
		return
	}
	fmt.Println("slice", vw.Bytes(), binary.Size(values))

	got := make([]int32, 3)
	if err := binary.Read(bytes.NewReader(vw.Bytes()), binary.LittleEndian, got); err != nil {
		fmt.Println("read slice", err)
		return
	}
	fmt.Println("got", got)

	var one uint16
	if err := binary.Read(bytes.NewReader([]byte{0x12, 0x34}), binary.BigEndian, &one); err != nil {
		fmt.Println("read one", err)
		return
	}
	fmt.Println("one", one)
}
