// Program peuse reads a COFF object of the PE format that the program itself
// builds. It shows the header, each section, and the symbol table. It also
// shows the string table and the aux symbol of debug/pe.
package main

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"strings"
)

func name8(s string) [8]uint8 {
	var out [8]uint8
	copy(out[:], s)
	return out
}

// image builds a COFF object of the AMD64 machine with 2 sections and 3
// symbols. The second symbol is the aux record of the first 1.
func image() []byte {
	const headerSize = 20
	const sectionSize = 40
	const dataOff = headerSize + 2*sectionSize

	text := []byte{0x48, 0x31, 0xc0, 0xc3, 0x90, 0x90, 0x90, 0x90}
	data := []byte("gart")
	symOff := dataOff + len(text) + len(data)

	var b bytes.Buffer
	put := func(v any) { binary.Write(&b, binary.LittleEndian, v) }

	put(pe.FileHeader{
		Machine:              pe.IMAGE_FILE_MACHINE_AMD64,
		NumberOfSections:     2,
		TimeDateStamp:        0x5f000000,
		PointerToSymbolTable: uint32(symOff),
		NumberOfSymbols:      3,
		SizeOfOptionalHeader: 0,
		Characteristics:      pe.IMAGE_FILE_LINE_NUMS_STRIPPED,
	})
	put(pe.SectionHeader32{
		Name:             name8(".text"),
		SizeOfRawData:    uint32(len(text)),
		PointerToRawData: uint32(dataOff),
		Characteristics:  pe.IMAGE_SCN_CNT_CODE | pe.IMAGE_SCN_MEM_EXECUTE | pe.IMAGE_SCN_MEM_READ,
	})
	put(pe.SectionHeader32{
		Name:             name8(".data"),
		SizeOfRawData:    uint32(len(data)),
		PointerToRawData: uint32(dataOff + len(text)),
		Characteristics:  pe.IMAGE_SCN_CNT_INITIALIZED_DATA | pe.IMAGE_SCN_MEM_READ,
	})
	b.Write(text)
	b.Write(data)

	// The section symbol of .text, with 1 aux record.
	put(pe.COFFSymbol{
		Name:               name8(".text"),
		Value:              0,
		SectionNumber:      1,
		Type:               0,
		StorageClass:       3, // IMAGE_SYM_CLASS_STATIC
		NumberOfAuxSymbols: 1,
	})
	// The aux record of the format 5.
	put(pe.COFFSymbolAuxFormat5{
		Size:           uint32(len(text)),
		NumRelocs:      0,
		NumLineNumbers: 0,
		Checksum:       0x12345678,
		SecNum:         1,
		Selection:      pe.IMAGE_COMDAT_SELECT_ANY,
	})
	// A symbol whose name stands in the string table.
	put(pe.COFFSymbol{
		Name:               [8]uint8{0, 0, 0, 0, 4, 0, 0, 0},
		Value:              0,
		SectionNumber:      1,
		Type:               0x20,
		StorageClass:       2, // IMAGE_SYM_CLASS_EXTERNAL
		NumberOfAuxSymbols: 0,
	})

	long := "averylongsymbolname\x00"
	put(uint32(4 + len(long)))
	b.WriteString(long)
	return b.Bytes()
}

func main() {
	fmt.Printf("machine: %#x %#x %#x\n", pe.IMAGE_FILE_MACHINE_AMD64,
		pe.IMAGE_FILE_MACHINE_ARM64, pe.IMAGE_FILE_MACHINE_I386)
	fmt.Println("symbol size:", pe.COFFSymbolSize)
	fmt.Println("comdat:", pe.IMAGE_COMDAT_SELECT_ANY, pe.IMAGE_COMDAT_SELECT_LARGEST)

	raw := image()
	f, err := pe.NewFile(bytes.NewReader(raw))
	if err != nil {
		fmt.Println("new file:", err)
		return
	}
	fmt.Printf("header: machine=%#x sections=%d symbols=%d opt=%d flags=%#x\n",
		f.Machine, f.NumberOfSections, f.NumberOfSymbols,
		f.SizeOfOptionalHeader, f.Characteristics)
	fmt.Println("optional header:", f.OptionalHeader == nil)

	for _, s := range f.Sections {
		fmt.Printf("section %-6s size=%d raw=%d flags=%#x\n",
			s.Name, s.VirtualSize, s.Size, s.Characteristics)
		body, err := s.Data()
		fmt.Printf("  data %x err=%v\n", body, err)
	}

	text := f.Section(".text")
	r := text.Open()
	head := make([]byte, 3)
	n, err := r.Read(head)
	fmt.Printf("open: %d %x %v\n", n, head[:n], err)
	fmt.Println("missing section:", f.Section(".none") == nil)

	fmt.Println("coff symbols:", len(f.COFFSymbols))
	for i, s := range f.COFFSymbols {
		full, err := s.FullName(f.StringTable)
		fmt.Printf("coff %d name=%q value=%d sect=%d type=%#x class=%d aux=%d err=%v\n",
			i, full, s.Value, s.SectionNumber, s.Type, s.StorageClass,
			s.NumberOfAuxSymbols, err)
	}
	for i, s := range f.Symbols {
		fmt.Printf("sym %d %s value=%d sect=%d type=%#x class=%d\n",
			i, s.Name, s.Value, s.SectionNumber, s.Type, s.StorageClass)
	}

	aux, err := f.COFFSymbolReadSectionDefAux(0)
	if err != nil {
		fmt.Println("aux:", err)
	} else {
		fmt.Printf("aux: size=%d relocs=%d lines=%d checksum=%#x secnum=%d selection=%d\n",
			aux.Size, aux.NumRelocs, aux.NumLineNumbers, aux.Checksum,
			aux.SecNum, aux.Selection)
	}
	_, err = f.COFFSymbolReadSectionDefAux(2)
	fmt.Println("aux of a symbol with none:", err)
	_, err = f.COFFSymbolReadSectionDefAux(99)
	fmt.Println("aux of a bad index:", err)

	name, err := f.StringTable.String(4)
	fmt.Printf("string table: %q %v\n", name, err)
	_, err = f.StringTable.String(0)
	fmt.Println("string table at 0:", err)

	names, err := f.ImportedSymbols()
	fmt.Println("imported symbols:", names, err)
	libs, err := f.ImportedLibraries()
	fmt.Println("imported libraries:", libs, err)

	_, err = f.DWARF()
	fmt.Println("dwarf:", err)
	fmt.Println("close:", f.Close())

	bad := make([]byte, 128)
	bad[0], bad[1] = 'M', 'Z'
	_, err = pe.NewFile(bytes.NewReader(bad))
	// The text of the error holds the 4 bytes under the verb "% x". The fmt
	// overlay writes no space between the 2 digits of a byte, thus the witness
	// reads the name of the error alone. Read docs/parity.md.
	fmt.Println("bad signature:", err != nil,
		strings.Contains(err.Error(), "invalid PE file signature"))
	_, err = pe.Open("no-such-file")
	fmt.Println("open missing:", err != nil)
	fe := &pe.FormatError{}
	fmt.Println("format error:", fe.Error())
}
