// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

// Package md5 implements the MD5 hash algorithm as defined in RFC 1321.
//
// MD5 is cryptographically broken and should not be used for secure
// applications.
//
// The overlay drops the registration of the package crypto and the FIPS mode of
// Go. The block function is the generic one, because the subset holds no
// assembly.
package md5

import (
	"errors"
	"hash"
	"internal/byteorder"
	"math/bits"
)

// The size of an MD5 checksum in bytes.
const Size = 16

// The blocksize of MD5 in bytes.
const BlockSize = 64

const (
	init0 = 0x67452301
	init1 = 0xEFCDAB89
	init2 = 0x98BADCFE
	init3 = 0x10325476
)

// digest represents the partial evaluation of a checksum.
type digest struct {
	s   [4]uint32
	x   [BlockSize]byte
	nx  int
	len uint64
}

func (d *digest) Reset() {
	d.s[0] = init0
	d.s[1] = init1
	d.s[2] = init2
	d.s[3] = init3
	d.nx = 0
	d.len = 0
}

const (
	magic         = "md5\x01"
	marshaledSize = len(magic) + 4*4 + BlockSize + 8
)

func (d *digest) MarshalBinary() ([]byte, error) {
	return d.AppendBinary(make([]byte, 0, marshaledSize))
}

func (d *digest) AppendBinary(b []byte) ([]byte, error) {
	b = append(b, magic...)
	b = byteorder.BEAppendUint32(b, d.s[0])
	b = byteorder.BEAppendUint32(b, d.s[1])
	b = byteorder.BEAppendUint32(b, d.s[2])
	b = byteorder.BEAppendUint32(b, d.s[3])
	b = append(b, d.x[:d.nx]...)
	b = append(b, make([]byte, len(d.x)-d.nx)...)
	b = byteorder.BEAppendUint64(b, d.len)
	return b, nil
}

func (d *digest) UnmarshalBinary(b []byte) error {
	if len(b) < len(magic) || string(b[:len(magic)]) != magic {
		return errors.New("crypto/md5: invalid hash state identifier")
	}
	if len(b) != marshaledSize {
		return errors.New("crypto/md5: invalid hash state size")
	}
	b = b[len(magic):]
	b, d.s[0] = consumeUint32(b)
	b, d.s[1] = consumeUint32(b)
	b, d.s[2] = consumeUint32(b)
	b, d.s[3] = consumeUint32(b)
	b = b[copy(d.x[:], b):]
	b, d.len = consumeUint64(b)
	d.nx = int(d.len % BlockSize)
	return nil
}

func consumeUint64(b []byte) ([]byte, uint64) {
	return b[8:], byteorder.BEUint64(b[0:8])
}

func consumeUint32(b []byte) ([]byte, uint32) {
	return b[4:], byteorder.BEUint32(b[0:4])
}

func (d *digest) Clone() (hash.Cloner, error) {
	r := *d
	return &r, nil
}

// New returns a new [hash.Hash] computing the MD5 checksum. The Hash
// also implements [encoding.BinaryMarshaler], [encoding.BinaryAppender] and
// [encoding.BinaryUnmarshaler] to marshal and unmarshal the internal
// state of the hash.
func New() hash.Hash {
	d := new(digest)
	d.Reset()
	return d
}

func (d *digest) Size() int { return Size }

func (d *digest) BlockSize() int { return BlockSize }

func (d *digest) Write(p []byte) (nn int, err error) {
	nn = len(p)
	d.len += uint64(nn)
	if d.nx > 0 {
		n := copy(d.x[d.nx:], p)
		d.nx += n
		if d.nx == BlockSize {
			block(d, d.x[:])
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= BlockSize {
		n := len(p) &^ (BlockSize - 1)
		block(d, p[:n])
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

func (d *digest) Sum(in []byte) []byte {
	// Make a copy of d so that caller can keep writing and summing.
	d0 := *d
	hash := d0.checkSum()
	return append(in, hash[:]...)
}

func (d *digest) checkSum() [Size]byte {
	// Append 0x80 to the end of the message and then append zeros
	// until the length is a multiple of 56 bytes. Finally append
	// 8 bytes representing the message length in bits.
	//
	// 1 byte end marker :: 0-63 padding bytes :: 8 byte length
	tmp := [1 + 63 + 8]byte{0x80}
	pad := (55 - d.len) % 64                     // calculate number of padding bytes
	byteorder.LEPutUint64(tmp[1+pad:], d.len<<3) // append length in bits
	d.Write(tmp[:1+pad+8])

	// The previous write ensures that a whole number of
	// blocks (i.e. a multiple of 64 bytes) have been hashed.
	if d.nx != 0 {
		panic("d.nx != 0")
	}

	var digest [Size]byte
	byteorder.LEPutUint32(digest[0:], d.s[0])
	byteorder.LEPutUint32(digest[4:], d.s[1])
	byteorder.LEPutUint32(digest[8:], d.s[2])
	byteorder.LEPutUint32(digest[12:], d.s[3])
	return digest
}

// Sum returns the MD5 checksum of the data.
func Sum(data []byte) [Size]byte {
	var d digest
	d.Reset()
	d.Write(data)
	return d.checkSum()
}

func blockGeneric(dig *digest, p []byte) {
	// load state
	a, b, c, d := dig.s[0], dig.s[1], dig.s[2], dig.s[3]

	for i := 0; i <= len(p)-BlockSize; i += BlockSize {
		// eliminate bounds checks on p
		q := p[i:]
		q = q[:BlockSize:BlockSize]

		// save current state
		aa, bb, cc, dd := a, b, c, d

		// load input block
		x0 := byteorder.LEUint32(q[4*0x0:])
		x1 := byteorder.LEUint32(q[4*0x1:])
		x2 := byteorder.LEUint32(q[4*0x2:])
		x3 := byteorder.LEUint32(q[4*0x3:])
		x4 := byteorder.LEUint32(q[4*0x4:])
		x5 := byteorder.LEUint32(q[4*0x5:])
		x6 := byteorder.LEUint32(q[4*0x6:])
		x7 := byteorder.LEUint32(q[4*0x7:])
		x8 := byteorder.LEUint32(q[4*0x8:])
		x9 := byteorder.LEUint32(q[4*0x9:])
		xa := byteorder.LEUint32(q[4*0xa:])
		xb := byteorder.LEUint32(q[4*0xb:])
		xc := byteorder.LEUint32(q[4*0xc:])
		xd := byteorder.LEUint32(q[4*0xd:])
		xe := byteorder.LEUint32(q[4*0xe:])
		xf := byteorder.LEUint32(q[4*0xf:])

		// round 1
		a = b + bits.RotateLeft32((((c^d)&b)^d)+a+x0+0xd76aa478, 7)
		d = a + bits.RotateLeft32((((b^c)&a)^c)+d+x1+0xe8c7b756, 12)
		c = d + bits.RotateLeft32((((a^b)&d)^b)+c+x2+0x242070db, 17)
		b = c + bits.RotateLeft32((((d^a)&c)^a)+b+x3+0xc1bdceee, 22)
		a = b + bits.RotateLeft32((((c^d)&b)^d)+a+x4+0xf57c0faf, 7)
		d = a + bits.RotateLeft32((((b^c)&a)^c)+d+x5+0x4787c62a, 12)
		c = d + bits.RotateLeft32((((a^b)&d)^b)+c+x6+0xa8304613, 17)
		b = c + bits.RotateLeft32((((d^a)&c)^a)+b+x7+0xfd469501, 22)
		a = b + bits.RotateLeft32((((c^d)&b)^d)+a+x8+0x698098d8, 7)
		d = a + bits.RotateLeft32((((b^c)&a)^c)+d+x9+0x8b44f7af, 12)
		c = d + bits.RotateLeft32((((a^b)&d)^b)+c+xa+0xffff5bb1, 17)
		b = c + bits.RotateLeft32((((d^a)&c)^a)+b+xb+0x895cd7be, 22)
		a = b + bits.RotateLeft32((((c^d)&b)^d)+a+xc+0x6b901122, 7)
		d = a + bits.RotateLeft32((((b^c)&a)^c)+d+xd+0xfd987193, 12)
		c = d + bits.RotateLeft32((((a^b)&d)^b)+c+xe+0xa679438e, 17)
		b = c + bits.RotateLeft32((((d^a)&c)^a)+b+xf+0x49b40821, 22)

		// round 2
		a = b + bits.RotateLeft32((((b^c)&d)^c)+a+x1+0xf61e2562, 5)
		d = a + bits.RotateLeft32((((a^b)&c)^b)+d+x6+0xc040b340, 9)
		c = d + bits.RotateLeft32((((d^a)&b)^a)+c+xb+0x265e5a51, 14)
		b = c + bits.RotateLeft32((((c^d)&a)^d)+b+x0+0xe9b6c7aa, 20)
		a = b + bits.RotateLeft32((((b^c)&d)^c)+a+x5+0xd62f105d, 5)
		d = a + bits.RotateLeft32((((a^b)&c)^b)+d+xa+0x02441453, 9)
		c = d + bits.RotateLeft32((((d^a)&b)^a)+c+xf+0xd8a1e681, 14)
		b = c + bits.RotateLeft32((((c^d)&a)^d)+b+x4+0xe7d3fbc8, 20)
		a = b + bits.RotateLeft32((((b^c)&d)^c)+a+x9+0x21e1cde6, 5)
		d = a + bits.RotateLeft32((((a^b)&c)^b)+d+xe+0xc33707d6, 9)
		c = d + bits.RotateLeft32((((d^a)&b)^a)+c+x3+0xf4d50d87, 14)
		b = c + bits.RotateLeft32((((c^d)&a)^d)+b+x8+0x455a14ed, 20)
		a = b + bits.RotateLeft32((((b^c)&d)^c)+a+xd+0xa9e3e905, 5)
		d = a + bits.RotateLeft32((((a^b)&c)^b)+d+x2+0xfcefa3f8, 9)
		c = d + bits.RotateLeft32((((d^a)&b)^a)+c+x7+0x676f02d9, 14)
		b = c + bits.RotateLeft32((((c^d)&a)^d)+b+xc+0x8d2a4c8a, 20)

		// round 3
		a = b + bits.RotateLeft32((b^c^d)+a+x5+0xfffa3942, 4)
		d = a + bits.RotateLeft32((a^b^c)+d+x8+0x8771f681, 11)
		c = d + bits.RotateLeft32((d^a^b)+c+xb+0x6d9d6122, 16)
		b = c + bits.RotateLeft32((c^d^a)+b+xe+0xfde5380c, 23)
		a = b + bits.RotateLeft32((b^c^d)+a+x1+0xa4beea44, 4)
		d = a + bits.RotateLeft32((a^b^c)+d+x4+0x4bdecfa9, 11)
		c = d + bits.RotateLeft32((d^a^b)+c+x7+0xf6bb4b60, 16)
		b = c + bits.RotateLeft32((c^d^a)+b+xa+0xbebfbc70, 23)
		a = b + bits.RotateLeft32((b^c^d)+a+xd+0x289b7ec6, 4)
		d = a + bits.RotateLeft32((a^b^c)+d+x0+0xeaa127fa, 11)
		c = d + bits.RotateLeft32((d^a^b)+c+x3+0xd4ef3085, 16)
		b = c + bits.RotateLeft32((c^d^a)+b+x6+0x04881d05, 23)
		a = b + bits.RotateLeft32((b^c^d)+a+x9+0xd9d4d039, 4)
		d = a + bits.RotateLeft32((a^b^c)+d+xc+0xe6db99e5, 11)
		c = d + bits.RotateLeft32((d^a^b)+c+xf+0x1fa27cf8, 16)
		b = c + bits.RotateLeft32((c^d^a)+b+x2+0xc4ac5665, 23)

		// round 4
		a = b + bits.RotateLeft32((c^(b|^d))+a+x0+0xf4292244, 6)
		d = a + bits.RotateLeft32((b^(a|^c))+d+x7+0x432aff97, 10)
		c = d + bits.RotateLeft32((a^(d|^b))+c+xe+0xab9423a7, 15)
		b = c + bits.RotateLeft32((d^(c|^a))+b+x5+0xfc93a039, 21)
		a = b + bits.RotateLeft32((c^(b|^d))+a+xc+0x655b59c3, 6)
		d = a + bits.RotateLeft32((b^(a|^c))+d+x3+0x8f0ccc92, 10)
		c = d + bits.RotateLeft32((a^(d|^b))+c+xa+0xffeff47d, 15)
		b = c + bits.RotateLeft32((d^(c|^a))+b+x1+0x85845dd1, 21)
		a = b + bits.RotateLeft32((c^(b|^d))+a+x8+0x6fa87e4f, 6)
		d = a + bits.RotateLeft32((b^(a|^c))+d+xf+0xfe2ce6e0, 10)
		c = d + bits.RotateLeft32((a^(d|^b))+c+x6+0xa3014314, 15)
		b = c + bits.RotateLeft32((d^(c|^a))+b+xd+0x4e0811a1, 21)
		a = b + bits.RotateLeft32((c^(b|^d))+a+x4+0xf7537e82, 6)
		d = a + bits.RotateLeft32((b^(a|^c))+d+xb+0xbd3af235, 10)
		c = d + bits.RotateLeft32((a^(d|^b))+c+x2+0x2ad7d2bb, 15)
		b = c + bits.RotateLeft32((d^(c|^a))+b+x9+0xeb86d391, 21)

		// add saved state
		a += aa
		b += bb
		c += cc
		d += dd
	}

	// save state
	dig.s[0], dig.s[1], dig.s[2], dig.s[3] = a, b, c, d
}

func block(dig *digest, p []byte) {
	blockGeneric(dig, p)
}
