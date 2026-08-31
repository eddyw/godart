//go:build dart

// Package tzdata holds an embedded copy of the IANA time zone database. An
// import of this package anywhere in a program gives time.LoadLocation the
// database when the system holds no zone files.
//
// The import grows the program by about 450 KB.
package tzdata

import "time"

func init() {
	time.RegisterEmbeddedTZData(loadFromEmbeddedTZData)
}

// get4s gives the little-endian 32-bit value at the start of s.
func get4s(s string) int {
	if len(s) < 4 {
		return 0
	}
	return int(s[0]) | int(s[1])<<8 | int(s[2])<<16 | int(s[3])<<24
}

// get2s gives the little-endian 16-bit value at the start of s.
func get2s(s string) int {
	if len(s) < 2 {
		return 0
	}
	return int(s[0]) | int(s[1])<<8
}

// loadFromEmbeddedTZData gives the content of a zone file inside the
// uncompressed zip text of zzipdata.go. The flag is false when the zip does
// not hold the name. The code is loadFromEmbeddedTZData of Go, with the flag
// in place of an error value.
func loadFromEmbeddedTZData(name string) (string, bool) {
	const (
		zecheader = 0x06054b50
		zcheader  = 0x02014b50
		ztailsize = 22

		zheadersize = 30
		zheader     = 0x04034b50
	)

	z := zipdata

	idx := len(z) - ztailsize
	n := get2s(z[idx+10:])
	idx = get4s(z[idx+16:])

	for i := 0; i < n; i++ {
		if get4s(z[idx:]) != zcheader {
			break
		}
		meth := get2s(z[idx+10:])
		size := get4s(z[idx+24:])
		namelen := get2s(z[idx+28:])
		xlen := get2s(z[idx+30:])
		fclen := get2s(z[idx+32:])
		off := get4s(z[idx+42:])
		zname := z[idx+46 : idx+46+namelen]
		idx += 46 + namelen + xlen + fclen
		if zname != name {
			continue
		}
		if meth != 0 {
			// The generated zip holds no compressed entry.
			return "", false
		}

		idx = off
		if get4s(z[idx:]) != zheader ||
			get2s(z[idx+8:]) != meth ||
			get2s(z[idx+26:]) != namelen ||
			z[idx+30:idx+30+namelen] != name {
			return "", false
		}
		xlen = get2s(z[idx+28:])
		idx += 30 + namelen + xlen
		return z[idx : idx+size], true
	}

	return "", false
}
