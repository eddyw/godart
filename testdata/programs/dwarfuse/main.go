// Program dwarfuse reads a DWARF unit that the program itself builds. It
// shows the reader, the entry, and the attribute. It also shows the type and
// the line table of debug/dwarf.
package main

import (
	"debug/dwarf"
	"encoding/binary"
	"fmt"
)

// abbrev holds the abbreviation table of 1 unit. Code 1 names the compilation
// unit, and code 2 names a base type.
func abbrev() []byte {
	var b []byte
	b = append(b, 1, 0x11, 1) // code 1, DW_TAG_compile_unit, children
	b = append(b, 0x03, 0x08) // DW_AT_name, DW_FORM_string
	b = append(b, 0x25, 0x08) // DW_AT_producer, DW_FORM_string
	b = append(b, 0, 0)
	b = append(b, 2, 0x24, 0) // code 2, DW_TAG_base_type, no child
	b = append(b, 0x03, 0x08) // DW_AT_name, DW_FORM_string
	b = append(b, 0x3e, 0x0b) // DW_AT_encoding, DW_FORM_data1
	b = append(b, 0x0b, 0x0b) // DW_AT_byte_size, DW_FORM_data1
	b = append(b, 0, 0)
	b = append(b, 0)
	return b
}

// info holds 1 compilation unit of DWARF version 4.
func info() []byte {
	var body []byte
	body = append(body, 4, 0)       // version 4
	body = append(body, 0, 0, 0, 0) // the offset of the abbreviation table
	body = append(body, 8)          // the size of an address
	body = append(body, 1)          // the compilation unit, abbreviation 1
	body = append(body, "unit.go\x00"...)
	body = append(body, "gart witness\x00"...)
	body = append(body, 2) // the base type, abbreviation 2
	body = append(body, "int\x00"...)
	body = append(body, 5) // DW_ATE_signed
	body = append(body, 8) // 8 bytes
	body = append(body, 0) // the end of the children

	out := make([]byte, 4, 4+len(body))
	binary.LittleEndian.PutUint32(out, uint32(len(body)))
	return append(out, body...)
}

func main() {
	fmt.Println("tag:", dwarf.TagCompileUnit, dwarf.TagBaseType)
	fmt.Println("attr:", dwarf.AttrName, dwarf.AttrEncoding, dwarf.AttrByteSize)
	fmt.Println("class:", dwarf.ClassAddress, dwarf.ClassString, dwarf.ClassConstant)

	data, err := dwarf.New(abbrev(), nil, nil, info(), nil, nil, nil, nil)
	if err != nil {
		fmt.Println("new:", err)
		return
	}

	r := data.Reader()
	var typeOffset dwarf.Offset
	for {
		e, err := r.Next()
		if err != nil {
			fmt.Println("next:", err)
			return
		}
		if e == nil {
			break
		}
		fmt.Printf("entry %v children=%v offset=%d\n", e.Tag, e.Children, e.Offset)
		for _, f := range e.Field {
			fmt.Printf("  field %v class=%v value=%v\n", f.Attr, f.Class, f.Val)
		}
		if e.Tag == dwarf.TagBaseType {
			typeOffset = e.Offset
			fmt.Println("  val of name:", e.Val(dwarf.AttrName))
			ok, err := e.AttrField(dwarf.AttrByteSize), error(nil)
			_ = err
			fmt.Println("  byte size field:", ok.Val)
		}
	}

	typ, err := data.Type(typeOffset)
	if err != nil {
		fmt.Println("type:", err)
		return
	}
	fmt.Println("type:", typ, "size:", typ.Size(), "string:", typ.String())
	if bt, ok := typ.(*dwarf.IntType); ok {
		fmt.Println("int type:", bt.Name, bt.ByteSize)
	}

	r.Seek(0)
	first, err := r.Next()
	fmt.Println("seek again:", first.Tag, err)
	units, err := data.Ranges(first)
	fmt.Println("ranges:", units, err)

	lr, err := data.LineReader(first)
	fmt.Println("line reader:", lr == nil, err)

	var de dwarf.DecodeError
	de = dwarf.DecodeError{Name: "info", Offset: 3, Err: "bad form"}
	fmt.Println("decode error:", de.Error())
	var ut dwarf.Type = &dwarf.UnsupportedType{CommonType: dwarf.CommonType{ByteSize: 2, Name: "odd"}}
	fmt.Println("unsupported:", ut.String(), ut.Size())
}
