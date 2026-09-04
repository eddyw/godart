// Program elfuse reads an ELF image that the program itself builds. It shows
// the header, the program headers, and each section. It also shows the symbol
// table and the 2 compressed forms of debug/elf.
package main

import (
	"bytes"
	"compress/zlib"
	"debug/elf"
	"encoding/binary"
	"fmt"
)

// strtab builds a string table and the offset of each name.
func strtab(names []string) ([]byte, map[string]uint32) {
	out := []byte{0}
	at := map[string]uint32{"": 0}
	for _, n := range names {
		at[n] = uint32(len(out))
		out = append(out, n...)
		out = append(out, 0)
	}
	return out, at
}

// zlibBlock builds the body of a section that holds the flag SHF_COMPRESSED
// with the zlib form.
func zlibBlock(plain []byte) []byte {
	var body bytes.Buffer
	binary.Write(&body, binary.LittleEndian, elf.Chdr64{
		Type:      uint32(elf.COMPRESS_ZLIB),
		Size:      uint64(len(plain)),
		Addralign: 1,
	})
	w := zlib.NewWriter(&body)
	w.Write(plain)
	w.Close()
	return body.Bytes()
}

// zstdBlock builds the body of a section that holds the flag SHF_COMPRESSED
// with the zstd form. The frame holds 1 raw block, thus no encoder is needed.
func zstdBlock(plain []byte) []byte {
	var body bytes.Buffer
	binary.Write(&body, binary.LittleEndian, elf.Chdr64{
		Type:      uint32(elf.COMPRESS_ZSTD),
		Size:      uint64(len(plain)),
		Addralign: 1,
	})
	frame := []byte{0x28, 0xb5, 0x2f, 0xfd, 0x20, byte(len(plain))}
	head := uint32(len(plain))<<3 | 0<<1 | 1
	frame = append(frame, byte(head), byte(head>>8), byte(head>>16))
	frame = append(frame, plain...)
	body.Write(frame)
	return body.Bytes()
}

type sectionPlan struct {
	name    string
	typ     elf.SectionType
	flags   elf.SectionFlag
	addr    uint64
	link    uint32
	info    uint32
	entsize uint64
	body    []byte
}

// image builds an ELF image of the 64-bit little-endian form.
func image() []byte {
	text := []byte{0x55, 0x48, 0x89, 0xe5, 0x31, 0xc0, 0x5d, 0xc3}
	plainZlib := []byte("the zlib section of the witness program")
	plainZstd := []byte("the zstd section of the witness program")

	symNames := []string{"main", "count"}
	symStr, symAt := strtab(symNames)

	var symtab bytes.Buffer
	binary.Write(&symtab, binary.LittleEndian, elf.Sym64{})
	binary.Write(&symtab, binary.LittleEndian, elf.Sym64{
		Name:  symAt["main"],
		Info:  uint8(elf.ST_INFO(elf.STB_GLOBAL, elf.STT_FUNC)),
		Shndx: 1,
		Value: 0x401000,
		Size:  uint64(len(text)),
	})
	binary.Write(&symtab, binary.LittleEndian, elf.Sym64{
		Name:  symAt["count"],
		Info:  uint8(elf.ST_INFO(elf.STB_GLOBAL, elf.STT_OBJECT)),
		Shndx: 1,
		Value: 0x402000,
		Size:  4,
	})

	plans := []sectionPlan{
		{name: ""},
		{name: ".text", typ: elf.SHT_PROGBITS,
			flags: elf.SHF_ALLOC | elf.SHF_EXECINSTR, addr: 0x401000, body: text},
		{name: ".zdata", typ: elf.SHT_PROGBITS, flags: elf.SHF_COMPRESSED,
			body: zlibBlock(plainZlib)},
		{name: ".sdata", typ: elf.SHT_PROGBITS, flags: elf.SHF_COMPRESSED,
			body: zstdBlock(plainZstd)},
		{name: ".symtab", typ: elf.SHT_SYMTAB, link: 5, info: 1, entsize: 24,
			body: symtab.Bytes()},
		{name: ".strtab", typ: elf.SHT_STRTAB, body: symStr},
	}

	var shNames []string
	for _, p := range plans[1:] {
		shNames = append(shNames, p.name)
	}
	shNames = append(shNames, ".shstrtab")
	shStr, shAt := strtab(shNames)
	plans = append(plans, sectionPlan{name: ".shstrtab", typ: elf.SHT_STRTAB, body: shStr})

	const headerSize = 64
	const progSize = 56
	const sectionSize = 64
	off := uint64(headerSize + progSize)

	offsets := make([]uint64, len(plans))
	var body bytes.Buffer
	for i, p := range plans {
		if len(p.body) == 0 {
			continue
		}
		offsets[i] = off
		body.Write(p.body)
		off += uint64(len(p.body))
	}
	shoff := off

	var b bytes.Buffer
	b.Write([]byte{0x7f, 'E', 'L', 'F'})
	b.WriteByte(byte(elf.ELFCLASS64))
	b.WriteByte(byte(elf.ELFDATA2LSB))
	b.WriteByte(byte(elf.EV_CURRENT))
	b.WriteByte(byte(elf.ELFOSABI_LINUX))
	b.WriteByte(0)
	b.Write(make([]byte, 7))
	put := func(v any) { binary.Write(&b, binary.LittleEndian, v) }
	put(uint16(elf.ET_EXEC))
	put(uint16(elf.EM_X86_64))
	put(uint32(elf.EV_CURRENT))
	put(uint64(0x401000))
	put(uint64(headerSize))
	put(shoff)
	put(uint32(0))
	put(uint16(headerSize))
	put(uint16(progSize))
	put(uint16(1))
	put(uint16(sectionSize))
	put(uint16(len(plans)))
	put(uint16(len(plans) - 1))

	put(elf.Prog64{
		Type:   uint32(elf.PT_LOAD),
		Flags:  uint32(elf.PF_R | elf.PF_X),
		Off:    offsets[1],
		Vaddr:  0x401000,
		Paddr:  0x401000,
		Filesz: uint64(len(text)),
		Memsz:  uint64(len(text)),
		Align:  0x1000,
	})

	b.Write(body.Bytes())

	for i, p := range plans {
		put(elf.Section64{
			Name:      shAt[p.name],
			Type:      uint32(p.typ),
			Flags:     uint64(p.flags),
			Addr:      p.addr,
			Off:       offsets[i],
			Size:      uint64(len(p.body)),
			Link:      p.link,
			Info:      p.info,
			Addralign: 1,
			Entsize:   p.entsize,
		})
	}
	return b.Bytes()
}

func main() {
	fmt.Println("class:", elf.ELFCLASS64, elf.ELFCLASS32)
	fmt.Println("data:", elf.ELFDATA2LSB, elf.ELFDATA2MSB)
	fmt.Println("type:", elf.ET_EXEC, elf.ET_REL, elf.ET_DYN)
	fmt.Println("machine:", elf.EM_X86_64, elf.EM_AARCH64, elf.EM_386)
	fmt.Println("section type:", elf.SHT_PROGBITS, elf.SHT_SYMTAB, elf.SHT_STRTAB)
	fmt.Println("prog type:", elf.PT_LOAD, elf.PT_DYNAMIC)
	fmt.Println("info:", elf.ST_BIND(0x12), elf.ST_TYPE(0x12), elf.ST_INFO(elf.STB_GLOBAL, elf.STT_FUNC))

	raw := image()
	f, err := elf.NewFile(bytes.NewReader(raw))
	if err != nil {
		fmt.Println("new file:", err)
		return
	}
	fmt.Printf("header: %v %v %v %v %v %v abi=%d entry=%#x\n",
		f.Class, f.Data, f.Version, f.OSABI, f.Type, f.Machine, f.ABIVersion, f.Entry)

	for i, p := range f.Progs {
		fmt.Printf("prog %d %v flags=%v off=%d vaddr=%#x filesz=%d memsz=%d align=%d\n",
			i, p.Type, p.Flags, p.Off, p.Vaddr, p.Filesz, p.Memsz, p.Align)
		buf := make([]byte, p.Filesz)
		m, err := p.ReadAt(buf, 0)
		fmt.Printf("  data %d %x err=%v\n", m, buf[:m], err)
	}

	for i, s := range f.Sections {
		fmt.Printf("section %d %-10q %v flags=%v addr=%#x off=%d size=%d link=%d entsize=%d\n",
			i, s.Name, s.Type, s.Flags, s.Addr, s.Offset, s.Size, s.Link, s.Entsize)
	}

	text := f.Section(".text")
	body, err := text.Data()
	fmt.Printf("text: %x err=%v\n", body, err)

	head := make([]byte, 4)
	n, err := text.Open().Read(head)
	fmt.Printf("open: %d %x %v\n", n, head[:n], err)

	zdata, err := f.Section(".zdata").Data()
	fmt.Printf("zlib section: %q err=%v\n", zdata, err)
	sdata, err := f.Section(".sdata").Data()
	fmt.Printf("zstd section: %q err=%v\n", sdata, err)

	fmt.Println("by type:", f.SectionByType(elf.SHT_SYMTAB).Name)
	fmt.Println("missing section:", f.Section(".none") == nil)

	syms, err := f.Symbols()
	fmt.Println("symbols err:", err)
	for _, s := range syms {
		fmt.Printf("sym %s info=%#x other=%d shndx=%d value=%#x size=%d\n",
			s.Name, s.Info, s.Other, s.Section, s.Value, s.Size)
	}

	_, err = f.DynamicSymbols()
	fmt.Println("dynamic symbols:", err)
	_, err = f.ImportedSymbols()
	fmt.Println("imported symbols:", err)
	_, err = f.ImportedLibraries()
	fmt.Println("imported libraries:", err)
	_, err = f.DynString(elf.DT_NEEDED)
	fmt.Println("dyn string:", err)
	_, err = f.DWARF()
	fmt.Println("dwarf:", err)
	fmt.Println("close:", f.Close())

	_, err = elf.NewFile(bytes.NewReader(make([]byte, 64)))
	fmt.Println("bad magic:", err)
	_, err = elf.Open("no-such-file")
	fmt.Println("open missing:", err != nil)
	fmt.Println("no symbols:", elf.ErrNoSymbols)

	fe := &elf.FormatError{}
	fmt.Println("format error:", fe.Error())
	fmt.Println("relocation:", elf.R_X86_64_PC32, elf.R_AARCH64_ABS64)
}
