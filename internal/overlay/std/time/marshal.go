// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package time

import "errors"

const (
	timeBinaryVersionV1 byte = iota + 1
	timeBinaryVersionV2
)

// unixToInternal is the count of seconds between the start of the year 1 and
// the start of the Unix epoch. A Time counts from the year 1, and each call of
// the Unix family counts from the epoch.
const unixToInternal int64 = (1969*365 + 1969/4 - 1969/100 + 1969/400) * 86400

// AppendBinary writes the binary form of t after b.
func (t Time) AppendBinary(b []byte) ([]byte, error) {
	var offsetMin int16
	var offsetSec int8
	version := timeBinaryVersionV1
	if t.Location() == UTC {
		offsetMin = -1
	} else {
		offset, _ := t.Location().lookup(t.unixSec())
		if offset%60 != 0 {
			version = timeBinaryVersionV2
			offsetSec = int8(offset % 60)
		}
		offset /= 60
		if offset < -32768 || offset == -1 || offset > 32767 {
			return b, errors.New("Time.MarshalBinary: unexpected zone offset")
		}
		offsetMin = int16(offset)
	}
	sec := t.sec()
	nsec := t.nanosecond()
	b = append(b,
		version,
		byte(sec>>56),
		byte(sec>>48),
		byte(sec>>40),
		byte(sec>>32),
		byte(sec>>24),
		byte(sec>>16),
		byte(sec>>8),
		byte(sec),
		byte(nsec>>24),
		byte(nsec>>16),
		byte(nsec>>8),
		byte(nsec),
		byte(offsetMin>>8),
		byte(offsetMin),
	)
	if version == timeBinaryVersionV2 {
		b = append(b, byte(offsetSec))
	}
	return b, nil
}

// MarshalBinary gives the binary form of t.
func (t Time) MarshalBinary() ([]byte, error) {
	b, err := t.AppendBinary(make([]byte, 0, 16))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// UnmarshalBinary reads the binary form of a time into t.
func (t *Time) UnmarshalBinary(data []byte) error {
	buf := data
	if len(buf) == 0 {
		return errors.New("Time.UnmarshalBinary: no data")
	}
	version := buf[0]
	if version != timeBinaryVersionV1 && version != timeBinaryVersionV2 {
		return errors.New("Time.UnmarshalBinary: unsupported version")
	}
	wantLen := 1 + 8 + 4 + 2
	if version == timeBinaryVersionV2 {
		wantLen++
	}
	if len(buf) != wantLen {
		return errors.New("Time.UnmarshalBinary: invalid length")
	}
	buf = buf[1:]
	sec := int64(buf[7]) | int64(buf[6])<<8 | int64(buf[5])<<16 |
		int64(buf[4])<<24 | int64(buf[3])<<32 | int64(buf[2])<<40 |
		int64(buf[1])<<48 | int64(buf[0])<<56
	buf = buf[8:]
	nsec := int32(buf[3]) | int32(buf[2])<<8 | int32(buf[1])<<16 |
		int32(buf[0])<<24
	buf = buf[4:]
	offset := int(int16(buf[1])|int16(buf[0])<<8) * 60
	if version == timeBinaryVersionV2 {
		offset += int(int8(buf[2]))
	}
	loc := UTC
	if offset != -1*60 {
		if o, _ := Local.lookup(sec - unixToInternal); o == offset {
			loc = Local
		} else {
			loc = FixedZone("", offset)
		}
	}
	*t = Time{ext: sec, wall: uint64(nsec)}.In(loc)
	return nil
}

// appendStrictRFC3339 writes the RFC 3339 text of t after b, and it reports
// each value that the standard cannot hold.
//
// The year must take 4 digits, thus a year outside the range 0 to 9999 gives an
// error. The hour of a zone must stand in the range 0 to 23.
func (t Time) appendStrictRFC3339(b []byte) ([]byte, error) {
	n0 := len(b)
	b = t.appendFormat(b, RFC3339Nano)
	if len(b) < n0+len("9999") || b[n0+len("9999")] != '-' {
		return b, errors.New("year outside of range [0,9999]")
	}
	if b[len(b)-1] != 'Z' {
		c := b[len(b)-len("Z07:00")]
		hour := 10*(b[len(b)-len("07:00")]-'0') + (b[len(b)-len("7:00")] - '0')
		if ('0' <= c && c <= '9') || hour >= 24 {
			return b, errors.New("timezone hour outside of range [0,23]")
		}
	}
	return b, nil
}

// MarshalJSON gives the RFC 3339 text of t between 2 quotation marks.
//
// Go reports an error when the year stands outside the range 0 to 9999.
func (t Time) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(RFC3339Nano)+2)
	b = append(b, '"')
	b, err := t.appendStrictRFC3339(b)
	if err != nil {
		return nil, errors.New("Time.MarshalJSON: " + err.Error())
	}
	return append(b, '"'), nil
}

// UnmarshalJSON reads an RFC 3339 text between 2 quotation marks into t.
func (t *Time) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return errors.New("Time.UnmarshalJSON: input is not a JSON string")
	}
	out, err := Parse(RFC3339, string(data[1:len(data)-1]))
	if err != nil {
		return err
	}
	*t = out
	return nil
}

func (t Time) appendTo(b []byte, errPrefix string) ([]byte, error) {
	b, err := t.appendStrictRFC3339(b)
	if err != nil {
		return nil, errors.New(errPrefix + err.Error())
	}
	return b, nil
}

// AppendText writes the RFC 3339 text of t after b.
//
// Go reports an error when the year stands outside the range 0 to 9999.
func (t Time) AppendText(b []byte) ([]byte, error) {
	return t.appendTo(b, "Time.AppendText: ")
}

// MarshalText gives the RFC 3339 text of t.
func (t Time) MarshalText() ([]byte, error) {
	return t.appendTo(make([]byte, 0, len(RFC3339Nano)), "Time.MarshalText: ")
}

// UnmarshalText reads an RFC 3339 text into t.
func (t *Time) UnmarshalText(data []byte) error {
	out, err := Parse(RFC3339, string(data))
	if err != nil {
		return err
	}
	*t = out
	return nil
}
