package main

import (
	"encoding/asn1"
	"fmt"
	"math/big"
	"time"
)

// wide holds 1 field of each type that parseField reads through the address of
// the field. The real file writes that read as a type switch over
// v.Addr().Interface().
type wide struct {
	Raw   asn1.RawValue
	OID   asn1.ObjectIdentifier
	Bits  asn1.BitString
	When  time.Time
	Count asn1.Enumerated
	Big   *big.Int
}

func main() {
	source := wide{
		Raw:   asn1.RawValue{Tag: asn1.TagPrintableString, Bytes: []byte("go")},
		OID:   asn1.ObjectIdentifier{1, 2, 840, 113549},
		Bits:  asn1.BitString{Bytes: []byte{0xA0}, BitLength: 3},
		When:  time.Date(2026, 9, 3, 10, 20, 30, 0, time.UTC),
		Count: 7,
		Big:   big.NewInt(1 << 40),
	}
	text, err := asn1.Marshal(source)
	if err != nil {
		fmt.Println("marshal:", err)
		return
	}
	var back wide
	if _, err := asn1.Unmarshal(text, &back); err != nil {
		fmt.Println("unmarshal:", err)
		return
	}
	fmt.Println(string(back.Raw.Bytes))
	fmt.Println(back.OID)
	fmt.Println(back.Bits.Bytes, back.Bits.BitLength)
	fmt.Println(back.When.Format(time.RFC3339))
	fmt.Println(back.Count)
	fmt.Println(back.Big)

	// A Flag reads true when the field is present. The bytes hold a SEQUENCE
	// that carries 1 BOOLEAN.
	type flagged struct {
		Set asn1.Flag `asn1:"optional"`
	}
	var seen flagged
	if _, err := asn1.Unmarshal([]byte{0x30, 0x03, 0x01, 0x01, 0xFF}, &seen); err != nil {
		fmt.Println("flag:", err)
		return
	}
	fmt.Println(bool(seen.Set))
}
