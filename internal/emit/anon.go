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

// nilChan gives the name of the shared nil channel of an element type.
//
// The zero value of a channel type reaches each variable of a frame, and such
// a channel holds no state that a program can change. Thus 1 value serves each
// variable of the type.
func (e *emitter) nilChan(elem types.Type) string {
	text := e.dartType(elem)
	sum := sha256.Sum256([]byte(text))
	key := "$nilChan" + hex.EncodeToString(sum[:4])
	if _, ok := e.nilChans[key]; !ok {
		e.nilChans[key] = elem
	}
	return key
}

// writeNilChans writes the nil channel of each element type that the package
// uses. The order comes from the name, thus equal input gives an equal file.
func (e *emitter) writeNilChans() {
	written := map[string]bool{}
	for {
		names := make([]string, 0, len(e.nilChans))
		for key := range e.nilChans {
			if !written[key] {
				names = append(names, key)
			}
		}
		if len(names) == 0 {
			return
		}
		sort.Strings(names)
		for _, key := range names {
			// The zero value of the element type can need another such
			// variable, thus the loop reads the map again.
			written[key] = true
			elem := e.nilChans[key]
			e.line("final Chan<%s> %s = Chan<%s>.nil(() => %s);",
				e.dartType(elem), key, e.dartType(elem), e.zero(elem))
		}
		e.line("")
	}
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
