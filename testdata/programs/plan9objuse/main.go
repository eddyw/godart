// Program plan9objuse reads a Plan 9 a.out image that the program itself
// builds. It shows the header, each section, and the symbol table of
// debug/plan9obj.
package main

import (
	"bytes"
	"debug/plan9obj"
	"encoding/binary"
	"fmt"
)

// image builds a Plan 9 a.out image of the AMD64 form.
func image() []byte {
	text := []byte("TEXT")
	data := []byte("DATA!!")

	var syms bytes.Buffer
	writeSym := func(value uint64, typ byte, name string) {
		binary.Write(&syms, binary.BigEndian, value)
		syms.WriteByte(typ)
		syms.WriteString(name)
		syms.WriteByte(0)
	}
	writeSym(0x200000, 'T', "main.main")
	writeSym(0x200040, 'D', "main.count")

	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, uint32(plan9obj.MagicAMD64))
	binary.Write(&b, binary.BigEndian, uint32(len(text)))
	binary.Write(&b, binary.BigEndian, uint32(len(data)))
	binary.Write(&b, binary.BigEndian, uint32(16))
	binary.Write(&b, binary.BigEndian, uint32(syms.Len()))
	binary.Write(&b, binary.BigEndian, uint32(0x200010))
	binary.Write(&b, binary.BigEndian, uint32(0))
	binary.Write(&b, binary.BigEndian, uint32(0))
	binary.Write(&b, binary.BigEndian, uint64(0x2000f0))
	b.Write(text)
	b.Write(data)
	b.Write(syms.Bytes())
	return b.Bytes()
}

func main() {
	fmt.Printf("magic: %d %d %d %d\n", plan9obj.Magic386, plan9obj.MagicAMD64,
		plan9obj.MagicARM, plan9obj.Magic64)

	raw := image()
	f, err := plan9obj.NewFile(bytes.NewReader(raw))
	if err != nil {
		fmt.Println("new file:", err)
		return
	}
	fmt.Printf("header: magic=%d bss=%d entry=%#x ptr=%d load=%#x hdr=%d\n",
		f.Magic, f.Bss, f.Entry, f.PtrSize, f.LoadAddress, f.HdrSize)

	for _, s := range f.Sections {
		fmt.Printf("section %-4s size=%d offset=%d\n", s.Name, s.Size, s.Offset)
	}

	text := f.Section("text")
	body, err := text.Data()
	fmt.Printf("text: %q err=%v\n", body, err)

	buf := make([]byte, 4)
	n, err := text.ReadAt(buf, 0)
	fmt.Printf("read at: %d %q %v\n", n, buf, err)

	r := text.Open()
	one := make([]byte, 2)
	n, err = r.Read(one)
	fmt.Printf("open: %d %q %v\n", n, one[:n], err)

	fmt.Println("missing section:", f.Section("none") == nil)

	syms, err := f.Symbols()
	fmt.Println("symbols err:", err)
	for _, s := range syms {
		fmt.Printf("sym %#x %c %s\n", s.Value, s.Type, s.Name)
	}

	fmt.Println("close:", f.Close())

	bad := make([]byte, 64)
	_, err = plan9obj.NewFile(bytes.NewReader(bad))
	fmt.Println("bad magic:", err)

	_, err = plan9obj.Open("no-such-file")
	fmt.Println("open missing:", err != nil)

	fmt.Println("err no symbols:", plan9obj.ErrNoSymbols)
}
