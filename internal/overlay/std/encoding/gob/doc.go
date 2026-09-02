//go:build dart

// Package gob manages streams of gobs, which are binary values that an
// [Encoder] writes and a [Decoder] reads.
//
// The overlay holds the algorithm of the real package, thus it writes the same
// bytes and it reads a stream of a Go program. The type ids of the bootstrap
// keep the numbers of Go, and `init` of `type.go` checks each 1.
//
// # What the overlay leaves out
//
//   - `Debug` and the dump program. The real package keeps that code behind a
//     build tag, and it exports no name.
//   - The generated helper of an array and of a slice of a basic element. Each
//     helper is a speed path of the real package, and the general path writes
//     the same bytes.
//   - `internal/saferio`. The surface holds no such package. `recvMessage`
//     already holds the count of a message against `tooBig`, and each read
//     allocates the exact count.
//
// # What the reflect subset changes
//
//   - A `reflect.Value` of an interface holds the dynamic value directly. Thus
//     `encodeInterface` reads `IsValid` in place of `IsNil`, and it takes the
//     value itself in place of `Elem`. `encode` reads the kind of the base
//     type, and not the kind of the value.
//   - `decodeMap` takes a fresh cell for each pair. A write of the overlay
//     stores the reference of the cell, and a write of Go copies it.
//   - A decode into a pointer to a pointer whose element is not a struct, for
//     example `**int` or a struct field of the type `*int`, panics.
//     `reflect.New` of a type that is not a struct gives an untyped box, and
//     Dart rejects a write of it into a typed location.
//   - `reflect.Type.String` of the compiler holds the alias of the source, and
//     Go holds the base name. The type table of the runtime keeps 1 text for
//     each type, and the first reader of a type decides that text, thus `init`
//     of this package reads `[]uint8` and `[]int32` before any other code. The
//     2 slice types then match Go. Each other composite of an alias, for
//     example `[2]byte` or `map[byte]rune`, writes the alias where Go writes
//     the base name. Such a text stands in the name of a type definition, and
//     the decoder matches a slice, an array, and a map by the element and the
//     length, thus a stream of such a shape still reads.
package gob
