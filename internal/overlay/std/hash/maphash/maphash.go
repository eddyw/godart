//go:build dart

// This file patches the file of the same name of the package hash/maphash.
//
// The real file reads unsafe, internal/abi, and internal/runtime/maps. It
// calls the hash function of the map of the runtime through 2 linkname
// directives. The subset holds no such access, thus this file holds a pure Go
// hash and a walk of the type descriptor of the runtime. Each other line comes
// from the real file. Read docs/library.md.

// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package maphash provides hash functions on byte sequences and comparable values.
// It also defines [Hasher], the interface between a hash function and a hash table.
//
// These hash functions are intended to be used to implement hash
// tables, Bloom filters, and other data structures that need to map
// arbitrary strings or byte sequences to a uniform distribution on
// unsigned 64-bit integers.
//
// Each different instance of a hash table or data structure should use its own [Seed].
//
// The hash functions are not cryptographically secure.
// (See crypto/sha256 and crypto/sha512 for cryptographic use.)
package maphash

import (
	"hash"
	"syscall/dart"
)

// A Seed is a random value that selects the specific hash function
// computed by a [Hash]. If two Hashes use the same Seeds, they
// will compute the same hash values for any given input.
// If two Hashes use different Seeds, they are very likely to compute
// distinct hash values for any given input.
//
// A Seed must be initialized by calling [MakeSeed].
// The zero seed is uninitialized and not valid for use with [Hash]'s SetSeed method.
//
// Each Seed value is local to a single process and cannot be serialized
// or otherwise recreated in a different process.
type Seed struct {
	s uint64
}

// Bytes returns the hash of b with the given seed.
//
// Bytes is equivalent to, but more convenient and efficient than:
//
//	var h Hash
//	h.SetSeed(seed)
//	h.Write(b)
//	return h.Sum64()
func Bytes(seed Seed, b []byte) uint64 {
	state := seed.s
	if state == 0 {
		panic("maphash: use of uninitialized Seed")
	}

	if len(b) > bufSize {
		b = b[:len(b):len(b)] // merge len and cap calculations when reslicing
		for len(b) > bufSize {
			state = rthash(b[:bufSize], state)
			b = b[bufSize:]
		}
	}
	return rthash(b, state)
}

// String returns the hash of s with the given seed.
//
// String is equivalent to, but more convenient and efficient than:
//
//	var h Hash
//	h.SetSeed(seed)
//	h.WriteString(s)
//	return h.Sum64()
func String(seed Seed, s string) uint64 {
	state := seed.s
	if state == 0 {
		panic("maphash: use of uninitialized Seed")
	}
	for len(s) > bufSize {
		state = rthashString(s[:bufSize], state)
		s = s[bufSize:]
	}
	return rthashString(s, state)
}

// A Hash computes a seeded hash of a byte sequence.
//
// The zero Hash is a valid Hash ready to use.
// A zero Hash chooses a random seed for itself during
// the first call to a Reset, Write, Seed, Clone, or Sum64 method.
// For control over the seed, use SetSeed.
//
// The computed hash values depend only on the initial seed and
// the sequence of bytes provided to the Hash object, not on the way
// in which the bytes are provided. For example, the three sequences
//
//	h.Write([]byte{'f','o','o'})
//	h.WriteByte('f'); h.WriteByte('o'); h.WriteByte('o')
//	h.WriteString("foo")
//
// all have the same effect.
//
// Hashes are intended to be collision-resistant, even for situations
// where an adversary controls the byte sequences being hashed.
//
// A Hash is not safe for concurrent use by multiple goroutines, but a Seed is.
// If multiple goroutines must compute the same seeded hash,
// each can declare its own Hash and call SetSeed with a common Seed.
type Hash struct {
	_     [0]func()     // not comparable
	seed  Seed          // initial seed used for this hash
	state Seed          // current hash of all flushed bytes
	buf   [bufSize]byte // unflushed byte buffer
	n     int           // number of unflushed bytes
}

// bufSize is the size of the Hash write buffer.
// The buffer ensures that writes depend only on the sequence of bytes,
// not the sequence of WriteByte/Write/WriteString calls,
// by always calling rthash with a full buffer (except for the tail).
const bufSize = 128

// initSeed seeds the hash if necessary.
// initSeed is called lazily before any operation that actually uses h.seed/h.state.
// Note that this does not include Write/WriteByte/WriteString in the case
// where they only add to h.buf. (If they write too much, they call h.flush,
// which does call h.initSeed.)
func (h *Hash) initSeed() {
	if h.seed.s == 0 {
		seed := MakeSeed()
		h.seed = seed
		h.state = seed
	}
}

// WriteByte adds b to the sequence of bytes hashed by h.
// It never fails; the error result is for implementing [io.ByteWriter].
func (h *Hash) WriteByte(b byte) error {
	if h.n == len(h.buf) {
		h.flush()
	}
	h.buf[h.n] = b
	h.n++
	return nil
}

// Write adds b to the sequence of bytes hashed by h.
// It always writes all of b and never fails; the count and error result are for implementing [io.Writer].
func (h *Hash) Write(b []byte) (int, error) {
	size := len(b)
	// Deal with bytes left over in h.buf.
	// h.n <= bufSize is always true.
	// Checking it is ~free and it lets the compiler eliminate a bounds check.
	if h.n > 0 && h.n <= bufSize {
		k := copy(h.buf[h.n:], b)
		h.n += k
		if h.n < bufSize {
			// Copied the entirety of b to h.buf.
			return size, nil
		}
		b = b[k:]
		h.flush()
		// No need to set h.n = 0 here; it happens just before exit.
	}
	// Process as many full buffers as possible, without copying, and calling initSeed only once.
	if len(b) > bufSize {
		h.initSeed()
		for len(b) > bufSize {
			h.state.s = rthash(b[:bufSize], h.state.s)
			b = b[bufSize:]
		}
	}
	// Copy the tail.
	copy(h.buf[:], b)
	h.n = len(b)
	return size, nil
}

// WriteString adds the bytes of s to the sequence of bytes hashed by h.
// It always writes all of s and never fails; the count and error result are for implementing [io.StringWriter].
func (h *Hash) WriteString(s string) (int, error) {
	// WriteString mirrors Write. See Write for comments.
	size := len(s)
	if h.n > 0 && h.n <= bufSize {
		k := copy(h.buf[h.n:], s)
		h.n += k
		if h.n < bufSize {
			return size, nil
		}
		s = s[k:]
		h.flush()
	}
	if len(s) > bufSize {
		h.initSeed()
		for len(s) > bufSize {
			h.state.s = rthashString(s[:bufSize], h.state.s)
			s = s[bufSize:]
		}
	}
	copy(h.buf[:], s)
	h.n = len(s)
	return size, nil
}

// Seed returns h's seed value.
func (h *Hash) Seed() Seed {
	h.initSeed()
	return h.seed
}

// SetSeed sets h to use seed, which must have been returned by [MakeSeed]
// or by another [Hash.Seed] method.
// Two [Hash] objects with the same seed behave identically.
// Two [Hash] objects with different seeds will very likely behave differently.
// Any bytes added to h before this call will be discarded.
func (h *Hash) SetSeed(seed Seed) {
	if seed.s == 0 {
		panic("maphash: use of uninitialized Seed")
	}
	h.seed = seed
	h.state = seed
	h.n = 0
}

// Reset discards all bytes added to h.
// (The seed remains the same.)
func (h *Hash) Reset() {
	h.initSeed()
	h.state = h.seed
	h.n = 0
}

// precondition: buffer is full.
func (h *Hash) flush() {
	if h.n != len(h.buf) {
		panic("maphash: flush of partially full buffer")
	}
	h.initSeed()
	h.state.s = rthash(h.buf[:h.n], h.state.s)
	h.n = 0
}

// Sum64 returns h's current 64-bit value, which depends on
// h's seed and the sequence of bytes added to h since the
// last call to [Hash.Reset] or [Hash.SetSeed].
//
// All bits of the Sum64 result are close to uniformly and
// independently distributed, so it can be safely reduced
// by using bit masking, shifting, or modular arithmetic.
func (h *Hash) Sum64() uint64 {
	h.initSeed()
	return rthash(h.buf[:h.n], h.state.s)
}

// MakeSeed returns a new random seed.
func MakeSeed() Seed {
	var s uint64
	for {
		s = randUint64()
		// We use seed 0 to indicate an uninitialized seed/hash,
		// so keep trying until we get a non-zero seed.
		if s != 0 {
			break
		}
	}
	return Seed{s: s}
}

// Sum appends the hash's current 64-bit value to b.
// It exists for implementing [hash.Hash].
// For direct calls, it is more efficient to use [Hash.Sum64].
func (h *Hash) Sum(b []byte) []byte {
	x := h.Sum64()
	return append(b,
		byte(x>>0),
		byte(x>>8),
		byte(x>>16),
		byte(x>>24),
		byte(x>>32),
		byte(x>>40),
		byte(x>>48),
		byte(x>>56))
}

// Size returns h's hash value size, 8 bytes.
func (h *Hash) Size() int { return 8 }

// BlockSize returns h's block size.
func (h *Hash) BlockSize() int { return len(h.buf) }

// Clone implements [hash.Cloner].
func (h *Hash) Clone() (hash.Cloner, error) {
	h.initSeed()
	r := *h
	return &r, nil
}

// Comparable returns the hash of comparable value v with the given seed
// such that Comparable(s, v1) == Comparable(s, v2) if v1 == v2.
// If v != v, then the resulting hash is randomly distributed.
func Comparable[T comparable](seed Seed, v T) uint64 {
	return comparableHash(v, seed)
}

// WriteComparable adds x to the data hashed by h.
func WriteComparable[T comparable](h *Hash, x T) {
	// writeComparable directly operates on h.state
	// without using h.buf. Mix in the buffer length so it won't
	// commute with a buffered write, which either changes h.n or changes
	// h.state.
	if h.n != 0 {
		writeComparable(h, h.n)
	}
	writeComparable(h, x)
}

// rthash mixes a block of bytes into a state. The real file calls the hash
// function of the map of the runtime through a linkname, and that function
// reads unsafe. This form holds the same contract: equal bytes with an equal
// state give an equal result, and the result of 1 seed says nothing about the
// result of another seed.
func rthash(buf []byte, seed uint64) uint64 {
	if len(buf) == 0 {
		return seed
	}
	state := mix(seed ^ uint64(len(buf)))
	i := 0
	for ; i+8 <= len(buf); i += 8 {
		state = mix(state ^ word(buf[i:i+8]))
	}
	var tail uint64
	for ; i < len(buf); i++ {
		tail = tail<<8 | uint64(buf[i])
	}
	return mix(state ^ tail)
}

// rthashString holds the rule of rthash for a string. The 2 must agree, thus
// this form reads the same core over the bytes of the text.
func rthashString(s string, state uint64) uint64 {
	return rthash([]byte(s), state)
}

// word reads 8 bytes as 1 number, with the low byte first.
func word(b []byte) uint64 {
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

// mix spreads each bit of a number over the whole word. It holds the finalizer
// of the 64-bit MurmurHash 3.
func mix(x uint64) uint64 {
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 33
	x *= 0xc4ceb9fe1a85ec53
	x ^= x >> 33
	return x
}

// randUint64 gives a random number of the platform. Go reads the random source
// of its runtime, and this form reads the random source of Dart.
func randUint64() uint64 {
	b := dart.RandomBytes(8)
	return word(b)
}

func comparableHash[T comparable](v T, seed Seed) uint64 {
	return hashValue(mix(seed.s), any(v))
}

func writeComparable[T comparable](h *Hash, v T) {
	h.state.s = comparableHash(v, h.state)
}

// hashValue mixes 1 comparable value into a state. Go reads the hash function
// of the map of the runtime for it. The subset holds no such access, thus this
// form walks the value through the type descriptor of the runtime.
//
// The walk holds the contract of Go: 2 values that Go reports as equal give an
// equal result. A pointer, a channel, and an unsafe pointer each hash by their
// identity. A float that is not a number gives a random result, exactly as Go
// documents.
func hashValue(state uint64, v any) uint64 {
	switch dart.Kind(v) {
	case "nil", "invalid":
		return mix(state ^ 0x9e3779b97f4a7c15)
	case "bool":
		if dart.AsBool(v) {
			return mix(state ^ 0x2545f4914f6cdd1d)
		}
		return mix(state ^ 0x1d8e4e27c47d124f)
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return mix(state ^ uint64(dart.AsInt(v)))
	case "float32", "float64":
		return hashFloat(state, dart.AsFloat(v))
	case "complex64", "complex128":
		c := dart.AsComplex(v)
		return hashFloat(hashFloat(state, real(c)), imag(c))
	case "string":
		s := dart.AsString(v)
		return rthashString(s, mix(state^uint64(len(s))))
	case "ptr", "chan", "unsafe.Pointer":
		return mix(state ^ uint64(dart.ValuePointer(v)))
	case "array":
		n := dart.ValueLen(v)
		state = mix(state ^ uint64(n))
		for i := 0; i < n; i++ {
			state = hashValue(state, dart.ValueIndex(v, i))
		}
		return state
	case "struct":
		desc := dart.DescOf(v)
		n := dart.FieldCount(desc)
		state = mix(state ^ uint64(n))
		for i := 0; i < n; i++ {
			state = hashValue(state, dart.FieldRead(v, i))
		}
		return state
	}
	panic("maphash: hash of unhashable type")
}

// hashFloat mixes 1 float into a state. The value 0 and the value -0 are equal
// in Go, thus the 2 must give 1 result. A value that is not a number is equal
// to no value, thus each read gives a fresh random result.
func hashFloat(state uint64, f float64) uint64 {
	if f != f {
		return mix(state ^ randUint64())
	}
	if f == 0 {
		return mix(state ^ 0x8ebc6af09c88c6e3)
	}
	return mix(state ^ dart.Float64Bits(f))
}
