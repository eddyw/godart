//go:build dart

// The TZif parser of zoneinfo_read.go of Go, over the file access of
// syscall/dart. The zip reader indexes the whole file in memory, because the
// Dart side gives no positioned read of a raw descriptor.
package time

import (
	"errors"
	"syscall/dart"
)

// maxFileSize bounds a read of a zone file. A larger file is not a zone file.
const maxFileSize = 10 << 20

type fileSizeError string

func (f fileSizeError) Error() string {
	return "time: file " + string(f) + " is too large"
}

// dataIO reads a binary blob from the front.
type dataIO struct {
	p     []byte
	error bool
}

func (d *dataIO) read(n int) []byte {
	if len(d.p) < n {
		d.p = nil
		d.error = true
		return nil
	}
	p := d.p[0:n]
	d.p = d.p[n:]
	return p
}

func (d *dataIO) big4() (n uint32, ok bool) {
	p := d.read(4)
	if len(p) < 4 {
		d.error = true
		return 0, false
	}
	return uint32(p[3]) | uint32(p[2])<<8 | uint32(p[1])<<16 | uint32(p[0])<<24, true
}

func (d *dataIO) big8() (n uint64, ok bool) {
	n1, ok1 := d.big4()
	n2, ok2 := d.big4()
	if !ok1 || !ok2 {
		d.error = true
		return 0, false
	}
	return (uint64(n1) << 32) | uint64(n2), true
}

func (d *dataIO) byte() (n byte, ok bool) {
	p := d.read(1)
	if len(p) < 1 {
		d.error = true
		return 0, false
	}
	return p[0], true
}

// rest gives the bytes that no read took.
func (d *dataIO) rest() []byte {
	r := d.p
	d.p = nil
	return r
}

// byteString gives the text before the first NUL.
func byteString(p []byte) string {
	for i := 0; i < len(p); i++ {
		if p[i] == 0 {
			return string(p[:i])
		}
	}
	return string(p)
}

var errBadData = errors.New("malformed time zone information")

// LoadLocationFromTZData gives a Location of a name and the bytes of a file of
// the IANA form, such as the content of /etc/localtime.
func LoadLocationFromTZData(name string, data []byte) (*Location, error) {
	d := dataIO{data, false}

	// The 4 magic bytes TZif.
	if magic := d.read(4); string(magic) != "TZif" {
		return nil, errBadData
	}

	// 1 version byte, and 15 bytes of padding.
	var version int
	var p []byte
	if p = d.read(16); len(p) != 16 {
		return nil, errBadData
	}
	switch p[0] {
	case 0:
		version = 1
	case '2':
		version = 2
	case '3':
		version = 3
	default:
		return nil, errBadData
	}

	// 6 big-endian counts: the UTC/local indicators, the standard/wall
	// indicators, the leap seconds, the transitions, the zones, and the
	// characters of the zone names.
	const (
		NUTCLocal = iota
		NStdWall
		NLeap
		NTime
		NZone
		NChar
	)
	var n [6]int
	for i := 0; i < 6; i++ {
		nn, ok := d.big4()
		if !ok {
			return nil, errBadData
		}
		if uint32(int(nn)) != nn {
			return nil, errBadData
		}
		n[i] = int(nn)
	}

	// Version 2 and 3 write the data in the 32-bit form first, and then in
	// the 64-bit form. The read skips the 32-bit form.
	is64 := false
	if version > 1 {
		skip := n[NTime]*4 +
			n[NTime] +
			n[NZone]*6 +
			n[NChar] +
			n[NLeap]*8 +
			n[NStdWall] +
			n[NUTCLocal]
		skip += 4 + 16
		d.read(skip)

		is64 = true

		// The counts of the 64-bit form can differ.
		for i := 0; i < 6; i++ {
			nn, ok := d.big4()
			if !ok {
				return nil, errBadData
			}
			if uint32(int(nn)) != nn {
				return nil, errBadData
			}
			n[i] = int(nn)
		}
	}

	size := 4
	if is64 {
		size = 8
	}

	// The transition times.
	txtimes := dataIO{d.read(n[NTime] * size), false}

	// The zone index of each transition.
	txzones := d.read(n[NTime])

	// The zone records.
	zonedata := dataIO{d.read(n[NZone] * 6), false}

	// The zone names.
	abbrev := d.read(n[NChar])

	// The leap second pairs take no use.
	d.read(n[NLeap] * (size + 4))

	// The standard/wall indicators.
	isstd := d.read(n[NStdWall])

	// The UTC/local indicators.
	isutc := d.read(n[NUTCLocal])

	if d.error {
		return nil, errBadData
	}

	var extend string
	rest := d.rest()
	if len(rest) > 2 && rest[0] == '\n' && rest[len(rest)-1] == '\n' {
		extend = string(rest[1 : len(rest)-1])
	}

	// Each zone record holds utcoff[4] isdst[1] nameindex[1].
	nzone := n[NZone]
	if nzone == 0 {
		// A file with no zone holds nothing of use.
		return nil, errBadData
	}
	zones := make([]zone, nzone)
	for i := range zones {
		var ok bool
		var n uint32
		if n, ok = zonedata.big4(); !ok {
			return nil, errBadData
		}
		if uint32(int(n)) != n {
			return nil, errBadData
		}
		zones[i].offset = int(int32(n))
		var b byte
		if b, ok = zonedata.byte(); !ok {
			return nil, errBadData
		}
		zones[i].isDST = b != 0
		if b, ok = zonedata.byte(); !ok || int(b) >= len(abbrev) {
			return nil, errBadData
		}
		zones[i].name = byteString(abbrev[b:])
	}

	tx := make([]zoneTrans, n[NTime])
	for i := range tx {
		var n int64
		if !is64 {
			n4, ok := txtimes.big4()
			if !ok {
				return nil, errBadData
			}
			n = int64(int32(n4))
		} else {
			n8, ok := txtimes.big8()
			if !ok {
				return nil, errBadData
			}
			n = int64(n8)
		}
		tx[i].when = n
		if int(txzones[i]) >= len(zones) {
			return nil, errBadData
		}
		tx[i].index = txzones[i]
		if i < len(isstd) {
			tx[i].isstd = isstd[i] != 0
		}
		if i < len(isutc) {
			tx[i].isutc = isutc[i] != 0
		}
	}

	if len(tx) == 0 {
		// A location of a fixed offset, such as Etc/GMT0, holds no
		// transition. A false transition covers all time.
		tx = append(tx, zoneTrans{when: alpha, index: 0})
	}

	l := &Location{zone: zones, tx: tx, name: name, extend: extend}

	// The cache holds the zone of the current instant, because most lookups
	// ask for it.
	sec := floorDiv(dart.NowNanos(), 1000000000)
	for i := range tx {
		if tx[i].when <= sec && (i+1 == len(tx) || sec < tx[i+1].when) {
			l.cacheStart = tx[i].when
			l.cacheEnd = omega
			l.cacheZone = &l.zone[tx[i].index]
			if i+1 < len(tx) {
				l.cacheEnd = tx[i+1].when
			} else if l.extend != "" {
				if name, offset, estart, eend, isDST, ok := tzset(l.extend, l.cacheStart, sec); ok {
					l.cacheStart = estart
					l.cacheEnd = eend
					if zoneIdx := findZone(l.zone, name, offset, isDST); zoneIdx != -1 {
						l.cacheZone = &l.zone[zoneIdx]
					} else {
						l.cacheZone = &zone{
							name:   name,
							offset: offset,
							isDST:  isDST,
						}
					}
				}
			}
			break
		}
	}

	return l, nil
}

func findZone(zones []zone, name string, offset int, isDST bool) int {
	for i, z := range zones {
		if z.name == name && z.offset == offset && z.isDST == isDST {
			return i
		}
	}
	return -1
}

// loadTzinfo reads the bytes of a zone of a source. The source is a directory,
// or an uncompressed zip file with the suffix .zip.
func loadTzinfo(name string, source string) ([]byte, error) {
	if len(source) > 4 && source[len(source)-4:] == ".zip" {
		return loadTzinfoFromZip(source, name)
	}
	if source != "" {
		name = source + "/" + name
	}
	return readFile(name)
}

// get4 gives the little-endian 32-bit value at the start of b.
func get4(b []byte) int {
	if len(b) < 4 {
		return 0
	}
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
}

// get2 gives the little-endian 16-bit value at the start of b.
func get2(b []byte) int {
	if len(b) < 2 {
		return 0
	}
	return int(b[0]) | int(b[1])<<8
}

// errZoneNotFound marks a zone that a source does not hold. loadLocation reads
// it as a miss and not as a failure.
var errZoneNotFound = errors.New("zone not found in source")

// loadTzinfoFromZip gives the bytes of a zone inside an uncompressed zip file.
func loadTzinfoFromZip(zipfile, name string) ([]byte, error) {
	buf, err := readFile(zipfile)
	if err != nil {
		return nil, err
	}

	const (
		zecheader = 0x06054b50
		zcheader  = 0x02014b50
		ztailsize = 22

		zheadersize = 30
		zheader     = 0x04034b50
	)

	idx := len(buf) - ztailsize
	if idx < 0 || get4(buf[idx:]) != zecheader {
		return nil, errors.New("corrupt zip file " + zipfile)
	}
	n := get2(buf[idx+10:])
	idx = get4(buf[idx+16:])

	// The central directory holds 1 record for each file:
	//	magic[4] disk[2] diskwhere[2] flags[2]
	//	method[2] mtime[2] mdate[2] crc[4]
	//	csize[4] size[4] namelen[2] xlen[2] fclen[2]
	//	disknumber[2] iattrs[2] eattrs[4]
	//	offset[4] name[namelen] extra[xlen] fcomment[fclen]
	for i := 0; i < n; i++ {
		if idx+46 > len(buf) || get4(buf[idx:]) != zcheader {
			break
		}
		meth := get2(buf[idx+10:])
		size := get4(buf[idx+24:])
		namelen := get2(buf[idx+28:])
		xlen := get2(buf[idx+30:])
		fclen := get2(buf[idx+32:])
		off := get4(buf[idx+42:])
		if idx+46+namelen > len(buf) {
			break
		}
		zname := buf[idx+46 : idx+46+namelen]
		idx += 46 + namelen + xlen + fclen
		if string(zname) != name {
			continue
		}
		if meth != 0 {
			return nil, errors.New("unsupported compression for " + name + " in " + zipfile)
		}

		// The local header holds:
		//	magic[4] version[2] flags[2] method[2]
		//	mtime[2] mdate[2] crc[4] csize[4] size[4]
		//	namelen[2] xlen[2] name[namelen] extra[xlen]
		idx = off
		if idx+30 > len(buf) ||
			get4(buf[idx:]) != zheader ||
			get2(buf[idx+8:]) != meth ||
			get2(buf[idx+26:]) != namelen ||
			string(buf[idx+30:idx+30+namelen]) != name {
			return nil, errors.New("corrupt zip file " + zipfile)
		}
		xlen = get2(buf[idx+28:])
		idx += 30 + namelen + xlen
		if idx+size > len(buf) {
			return nil, errors.New("corrupt zip file " + zipfile)
		}
		return buf[idx : idx+size], nil
	}

	return nil, errZoneNotFound
}

// loadFromEmbeddedTZData reads a zone of the database that time/tzdata embeds.
// The import of that package writes the function.
var loadFromEmbeddedTZData func(name string) (string, bool)

// RegisterEmbeddedTZData accepts the reader of the embedded database. The
// package time/tzdata calls it from its init function. Go wires the 2 packages
// with a linkname directive, and the subset holds no such directive, thus the
// overlay exports this function. A program never calls it.
func RegisterEmbeddedTZData(f func(name string) (string, bool)) {
	loadFromEmbeddedTZData = f
}

// loadLocation reads a zone of a name from the first source that holds it, and
// then from the embedded database.
func loadLocation(name string, sources []string) (z *Location, firstErr error) {
	for _, source := range sources {
		zoneData, err := loadTzinfo(name, source)
		if err == nil {
			if z, err = LoadLocationFromTZData(name, zoneData); err == nil {
				return z, nil
			}
		}
		if firstErr == nil && err != nil && !isNotFound(err) {
			firstErr = err
		}
	}
	if loadFromEmbeddedTZData != nil {
		if zoneData, found := loadFromEmbeddedTZData(name); found {
			if z, err := LoadLocationFromTZData(name, []byte(zoneData)); err == nil {
				return z, nil
			} else if firstErr == nil {
				firstErr = err
			}
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return nil, errors.New("unknown time zone " + name)
}

// isNotFound reports whether an error names a path or a zone that a source
// does not hold.
func isNotFound(err error) bool {
	if err == errZoneNotFound {
		return true
	}
	text := err.Error()
	return textContains(text, "No such file or directory") ||
		textContains(text, "no such file or directory") ||
		textContains(text, "cannot find the file")
}

func textContains(text, part string) bool {
	for i := 0; i+len(part) <= len(text); i++ {
		if text[i:i+len(part)] == part {
			return true
		}
	}
	return false
}

// readFile reads a whole file through the Dart side.
func readFile(name string) ([]byte, error) {
	data, cause := dart.ReadFile(name)
	if cause != "" {
		return nil, errors.New(cause)
	}
	if len(data) > maxFileSize {
		return nil, fileSizeError(name)
	}
	return data, nil
}
