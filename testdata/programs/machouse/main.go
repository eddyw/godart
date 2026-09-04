// Program machouse reads a Mach-O image that the program itself builds. It
// shows the header and the load commands. It also shows each section and the
// symbol table of debug/macho.
package main

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"fmt"
)

func name16(s string) [16]byte {
	var out [16]byte
	copy(out[:], s)
	return out
}

// image builds a Mach-O image of the 64-bit form with 1 segment, 2 sections,
// and a symbol table.
func image() []byte {
	const headerSize = 32
	const segSize = 72
	const sectSize = 80
	const symtabSize = 24
	const cmdsz = segSize + 2*sectSize + symtabSize
	const dataOff = headerSize + cmdsz

	text := []byte("\x90\x90\x90\x90\xc3\x00\x00\x00")
	data := []byte("gart")
	strtab := append([]byte{0}, "_main\x00_count\x00"...)

	symOff := dataOff + len(text) + len(data)
	strOff := symOff + 2*16

	var b bytes.Buffer
	put := func(v any) { binary.Write(&b, binary.LittleEndian, v) }

	put(macho.FileHeader{
		Magic:  macho.Magic64,
		Cpu:    macho.CpuAmd64,
		SubCpu: 3,
		Type:   macho.TypeExec,
		Ncmd:   2,
		Cmdsz:  cmdsz,
		Flags:  macho.FlagNoUndefs | macho.FlagPIE,
	})
	put(uint32(0)) // the reserved word of the 64-bit header

	put(macho.Segment64{
		Cmd:     macho.LoadCmdSegment64,
		Len:     segSize + 2*sectSize,
		Name:    name16("__TEXT"),
		Addr:    0x100000000,
		Memsz:   0x1000,
		Offset:  0,
		Filesz:  uint64(dataOff + len(text) + len(data)),
		Maxprot: 7,
		Prot:    5,
		Nsect:   2,
		Flag:    0,
	})
	put(macho.Section64{
		Name:   name16("__text"),
		Seg:    name16("__TEXT"),
		Addr:   0x100001000,
		Size:   uint64(len(text)),
		Offset: uint32(dataOff),
		Align:  4,
		Flags:  0x80000400,
	})
	put(macho.Section64{
		Name:   name16("__data"),
		Seg:    name16("__TEXT"),
		Addr:   0x100002000,
		Size:   uint64(len(data)),
		Offset: uint32(dataOff + len(text)),
		Align:  0,
	})
	put(macho.SymtabCmd{
		Cmd:     macho.LoadCmdSymtab,
		Len:     symtabSize,
		Symoff:  uint32(symOff),
		Nsyms:   2,
		Stroff:  uint32(strOff),
		Strsize: uint32(len(strtab)),
	})

	b.Write(text)
	b.Write(data)
	put(macho.Nlist64{Name: 1, Type: 0x0f, Sect: 1, Desc: 0, Value: 0x100001000})
	put(macho.Nlist64{Name: 7, Type: 0x0f, Sect: 2, Desc: 0, Value: 0x100002000})
	b.Write(strtab)
	return b.Bytes()
}

func main() {
	fmt.Println("magic:", macho.Magic32, macho.Magic64, macho.MagicFat)
	fmt.Println("cpu:", macho.CpuAmd64, macho.CpuArm64, macho.Cpu386)
	fmt.Println("type:", macho.TypeObj, macho.TypeExec, macho.TypeDylib)
	fmt.Println("cmd:", macho.LoadCmdSegment64, macho.LoadCmdSymtab, macho.LoadCmdRpath)
	fmt.Printf("gostring: %s %s\n", macho.CpuArm64.GoString(), macho.TypeExec.GoString())

	raw := image()
	f, err := macho.NewFile(bytes.NewReader(raw))
	if err != nil {
		fmt.Println("new file:", err)
		return
	}
	fmt.Printf("header: %v %v sub=%d %v ncmd=%d cmdsz=%d flags=%#x\n",
		f.Magic, f.Cpu, f.SubCpu, f.Type, f.Ncmd, f.Cmdsz, f.Flags)
	fmt.Println("byte order:", f.ByteOrder == binary.LittleEndian)

	for i, l := range f.Loads {
		switch c := l.(type) {
		case *macho.Segment:
			fmt.Printf("load %d segment %s addr=%#x memsz=%d nsect=%d\n",
				i, c.Name, c.Addr, c.Memsz, c.Nsect)
		case *macho.Symtab:
			fmt.Printf("load %d symtab nsyms=%d strsize=%d\n", i, c.Nsyms, c.Strsize)
		default:
			fmt.Printf("load %d raw %d\n", i, len(l.Raw()))
		}
	}

	for _, s := range f.Sections {
		fmt.Printf("section %s/%s addr=%#x size=%d offset=%d align=%d flags=%#x\n",
			s.Seg, s.Name, s.Addr, s.Size, s.Offset, s.Align, s.Flags)
	}

	text := f.Section("__text")
	body, err := text.Data()
	fmt.Printf("text: %x err=%v\n", body, err)

	r := text.Open()
	head := make([]byte, 2)
	n, err := r.Read(head)
	fmt.Printf("open: %d %x %v\n", n, head[:n], err)

	seg := f.Segment("__TEXT")
	segBody, err := seg.Data()
	fmt.Println("segment bytes:", len(segBody), err)

	fmt.Println("missing section:", f.Section("__none") == nil)
	fmt.Println("missing segment:", f.Segment("__NONE") == nil)

	for _, s := range f.Symtab.Syms {
		fmt.Printf("sym %s type=%#x sect=%d desc=%d value=%#x\n",
			s.Name, s.Type, s.Sect, s.Desc, s.Value)
	}

	names, err := f.ImportedSymbols()
	fmt.Println("imported symbols:", names, err)
	libs, err := f.ImportedLibraries()
	fmt.Println("imported libraries:", libs, err)

	_, err = f.DWARF()
	fmt.Println("dwarf:", err)
	fmt.Println("close:", f.Close())

	_, err = macho.NewFile(bytes.NewReader(make([]byte, 64)))
	fmt.Println("bad magic:", err)
	_, err = macho.Open("no-such-file")
	fmt.Println("open missing:", err != nil)
	_, err = macho.OpenFat("no-such-file")
	fmt.Println("open fat missing:", err != nil)
	fmt.Println("fat error:", macho.ErrNotFat)

	var fe = &macho.FormatError{}
	fmt.Println("format error is not nil:", fe != nil)
}
