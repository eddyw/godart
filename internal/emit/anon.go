package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"
	"sort"
)

// anonClass gives the name of the Dart class of an unnamed struct type, and it
// records the type for the library file.
//
// Go gives an unnamed struct type a structural identity, and a Dart class holds
// a nominal identity. The name comes from the shape of the struct, thus 2 equal
// Go types give 1 Dart class.
func (e *emitter) anonClass(st *types.Struct) string {
	key := anonName(st)
	if _, ok := e.anon[key]; !ok {
		e.anon[key] = st
	}
	return key
}

// anonName gives the Dart class name of an unnamed struct type. The name comes
// from the text of the type, thus the same shape always gives the same name.
func anonName(st *types.Struct) string {
	sum := sha256.Sum256([]byte(anonText(st)))
	return "$Anon" + hex.EncodeToString(sum[:4])
}

// anonText writes the Go text of a type. A named type takes its import path,
// thus 2 packages of the same build give the same text.
func anonText(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Path() })
}

// writeAnonClasses writes the class of each unnamed struct type that the
// package uses. The order comes from the name, thus equal input gives an equal
// file.
func (e *emitter) writeAnonClasses() {
	names := make([]string, 0, len(e.anon))
	for key := range e.anon {
		names = append(names, key)
	}
	sort.Strings(names)
	for _, key := range names {
		st := e.anon[key]
		e.line("/// %s", anonText(st))
		e.structClass(key, nil, st, types.Comparable(st), nil)
	}
}
