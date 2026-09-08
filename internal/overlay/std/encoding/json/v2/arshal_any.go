// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

// This file patches the file of the same name of the package
// encoding/json/v2.
//
// marshalValueAny of Go descends 1 level for each level of the value, and
// marshalObjectAny and marshalArrayAny each call it again. The stack of Dart
// holds about 1,000 of those pairs, and the cycle test of Go stands at 1,001
// levels. The walk below keeps the open containers in a slice of its own, thus
// a tree of any depth costs no stack of the machine.
//
// Each other line comes from the real file.

package json

import (
	"cmp"
	"errors"
	"math"
	"reflect"
	"slices"
	"strconv"

	"encoding/json/internal"
	"encoding/json/internal/jsonflags"
	"encoding/json/internal/jsonopts"
	"encoding/json/internal/jsonwire"
	"encoding/json/jsontext"
)

// This file contains an optimized marshal and unmarshal implementation
// for the any type. This type is often used when the Go program has
// no knowledge of the JSON schema. This is a common enough occurrence
// to justify the complexity of adding logic for this.

// anyFrame holds 1 open object or 1 open array of the walk of
// marshalValueAny.
type anyFrame struct {
	// object marks an object. An array holds false.
	object bool
	// seen marks a frame whose value joined the set of the open pointers. The
	// walk leaves that set at the end of the frame, exactly as the deferred
	// call of leavePointer of Go does.
	seen  bool
	value reflect.Value
	// arr holds the elements of an array.
	arr []any
	// obj and names hold the members of an object, in the order of the write.
	obj   map[string]any
	names []string
	// i counts the elements or the members that the walk already wrote.
	i int
}

// marshalValueAny marshals a Go any as a JSON value.
// This assumes that there are no special formatting directives
// for any possible nested value.
//
// The walk holds each open object and each open array in a slice of its own.
// The recursive form of Go costs 2 frames of the machine for each level, and
// the stack of Dart holds about 1,000 of those pairs. The cycle test of Go
// stands at 1,001 levels, thus the recursive form reaches no such value.
func marshalValueAny(enc *jsontext.Encoder, val any, mo *jsonopts.Struct) error {
	xe := export.Encoder(enc)
	var stack []anyFrame

	// leaveAll drops each open frame. Go holds 1 deferred call of leavePointer
	// for each level, and this loop holds the same order.
	leaveAll := func(err error) error {
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].seen {
				leavePointer(&xe.SeenPointers, stack[i].value)
			}
		}
		return err
	}

	next := val
	for {
		if _, err := writeValueAny(enc, &stack, next, mo); err != nil {
			return leaveAll(err)
		}
		// Read the next part of the container at the top, and close each
		// container that ended.
		for {
			if len(stack) == 0 {
				return nil
			}
			top := len(stack) - 1
			if stack[top].object {
				if stack[top].i < len(stack[top].names) {
					name := stack[top].names[stack[top].i]
					stack[top].i++
					if err := enc.WriteToken(jsontext.String(name)); err != nil {
						return leaveAll(err)
					}
					next = stack[top].obj[name]
					break
				}
				if err := enc.WriteToken(jsontext.EndObject); err != nil {
					return leaveAll(err)
				}
			} else {
				if stack[top].i < len(stack[top].arr) {
					next = stack[top].arr[stack[top].i]
					stack[top].i++
					break
				}
				if err := enc.WriteToken(jsontext.EndArray); err != nil {
					return leaveAll(err)
				}
			}
			if stack[top].seen {
				leavePointer(&xe.SeenPointers, stack[top].value)
			}
			stack = stack[:top]
		}
	}
}

// writeValueAny writes 1 value. opened reports that the value is a container
// that joined the stack, thus the caller reads its parts now.
func writeValueAny(enc *jsontext.Encoder, stack *[]anyFrame, val any, mo *jsonopts.Struct) (bool, error) {
	switch val := val.(type) {
	case nil:
		return false, enc.WriteToken(jsontext.Null)
	case bool:
		return false, enc.WriteToken(jsontext.Bool(val))
	case string:
		return false, enc.WriteToken(jsontext.String(val))
	case float64:
		if math.IsNaN(val) || math.IsInf(val, 0) {
			break // use default logic below
		}
		return false, enc.WriteToken(jsontext.Float(val))
	case map[string]any:
		return openObjectAny(enc, stack, val, mo)
	case []any:
		return openArrayAny(enc, stack, val, mo)
	}
	return false, marshalOtherAny(enc, val, mo)
}

// marshalOtherAny marshals a value that no fast case of writeValueAny holds.
// It reads the arshaler of the type of the value.
func marshalOtherAny(enc *jsontext.Encoder, val any, mo *jsonopts.Struct) error {
	v := newAddressableValue(reflect.TypeOf(val))
	v.Set(reflect.ValueOf(val))
	marshal := lookupArshaler(v.Type()).marshal
	if mo.Marshalers != nil {
		marshal, _ = mo.Marshalers.(*Marshalers).lookup(marshal, v.Type())
	}
	return marshal(enc, v, mo)
}

// unmarshalValueAny unmarshals a JSON value as a Go any.
// This assumes that there are no special formatting directives
// for any possible nested value.
// Duplicate names must be rejected since this does not implement merging.
func unmarshalValueAny(dec *jsontext.Decoder, uo *jsonopts.Struct) (any, error) {
	switch k := dec.PeekKind(); k {
	case '{':
		return unmarshalObjectAny(dec, uo)
	case '[':
		return unmarshalArrayAny(dec, uo)
	default:
		xd := export.Decoder(dec)
		var flags jsonwire.ValueFlags
		val, err := xd.ReadValue(&flags)
		if err != nil {
			return nil, err
		}
		switch val.Kind() {
		case 'n':
			return nil, nil
		case 'f':
			return false, nil
		case 't':
			return true, nil
		case '"':
			val = jsonwire.UnquoteMayCopy(val, flags.IsVerbatim())
			if xd.StringCache == nil {
				xd.StringCache = new(stringCache)
			}
			return makeString(xd.StringCache, val), nil
		case '0':
			if uo.Flags.Get(jsonflags.UnmarshalAnyWithRawNumber) {
				return internal.RawNumberOf(val), nil
			}
			fv, err := strconv.ParseFloat(string(val), 64)
			if err != nil {
				return fv, newUnmarshalErrorAfterWithValue(dec, float64Type, errors.Unwrap(err))
			}
			return fv, nil
		default:
			panic("BUG: invalid kind: " + k.String())
		}
	}
}

// openObjectAny writes the head of a JSON object for a Go map[string]any
// (or a JSON null if nil and [jsonflags.FormatNilMapAsNull]).
//
// The result reports whether the object joined the stack of the walk.
func openObjectAny(enc *jsontext.Encoder, stack *[]anyFrame, obj map[string]any, mo *jsonopts.Struct) (bool, error) {
	xe := export.Encoder(enc)
	f := anyFrame{object: true, obj: obj}

	// Check for cycles.
	if xe.Tokens.Depth() > startDetectingCyclesAfter {
		v := reflect.ValueOf(obj)
		if err := visitPointer(&xe.SeenPointers, v); err != nil {
			return false, newMarshalErrorBefore(enc, mapStringAnyType, err)
		}
		f.seen = true
		f.value = v
	}

	// Handle empty maps.
	if len(obj) == 0 {
		if mo.Flags.Get(jsonflags.FormatNilMapAsNull) && obj == nil {
			err := enc.WriteToken(jsontext.Null)
			if f.seen {
				leavePointer(&xe.SeenPointers, f.value)
			}
			return false, err
		}
		// Optimize for marshaling an empty map without any preceding whitespace.
		if !mo.Flags.Get(jsonflags.AnyWhitespace) && !xe.Tokens.Last.NeedObjectName() {
			xe.Buf = append(xe.Tokens.MayAppendDelim(xe.Buf, '{'), "{}"...)
			xe.Tokens.Last.Increment()
			var err error
			if xe.NeedFlush() {
				err = xe.Flush()
			}
			if f.seen {
				leavePointer(&xe.SeenPointers, f.value)
			}
			return false, err
		}
	}

	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		if f.seen {
			leavePointer(&xe.SeenPointers, f.value)
		}
		return false, err
	}
	// A Go map guarantees that each entry has a unique key.
	// The only possibility of duplicates is due to invalid UTF-8.
	if !mo.Flags.Get(jsonflags.AllowInvalidUTF8) {
		xe.Tokens.Last.DisableNamespace()
	}
	// The walk reads the members by their index, thus the names of the map
	// stand in a slice of the frame. The recursive form of Go reads the map
	// itself, and it builds such a slice for the deterministic order alone.
	f.names = make([]string, 0, len(obj))
	for name := range obj {
		f.names = append(f.names, name)
	}
	if mo.Flags.Get(jsonflags.Deterministic) && len(obj) > 1 {
		slices.Sort(f.names)
	}
	*stack = append(*stack, f)
	return true, nil
}

// unmarshalObjectAny unmarshals a JSON object as a Go map[string]any.
// It panics if not decoding a JSON object.
func unmarshalObjectAny(dec *jsontext.Decoder, uo *jsonopts.Struct) (map[string]any, error) {
	switch tok, err := dec.ReadToken(); {
	case err != nil:
		return nil, err
	case tok.Kind() != '{':
		panic("BUG: invalid kind: " + tok.Kind().String())
	}
	obj := make(map[string]any)
	// A Go map guarantees that each entry has a unique key.
	// The only possibility of duplicates is due to invalid UTF-8.
	if !uo.Flags.Get(jsonflags.AllowInvalidUTF8) {
		export.Decoder(dec).Tokens.Last.DisableNamespace()
	}
	var errUnmarshal error
	for dec.PeekKind() != '}' {
		tok, err := dec.ReadToken()
		if err != nil {
			return obj, err
		}
		name := tok.String()

		// Manually check for duplicate names.
		if _, ok := obj[name]; ok {
			// TODO: Unread the object name.
			name := export.Decoder(dec).PreviousTokenOrValue()
			err := newDuplicateNameError(dec.StackPointer(), nil, dec.InputOffset()-len64(name))
			return obj, err
		}

		val, err := unmarshalValueAny(dec, uo)
		obj[name] = val
		if err != nil {
			if isFatalError(err, uo.Flags) {
				return obj, err
			}
			errUnmarshal = cmp.Or(err, errUnmarshal)
		}
	}
	if _, err := dec.ReadToken(); err != nil {
		return obj, err
	}
	return obj, errUnmarshal
}

// openArrayAny writes the head of a JSON array for a Go []any
// (or a JSON null if nil and [jsonflags.FormatNilSliceAsNull]).
//
// The result reports whether the array joined the stack of the walk.
func openArrayAny(enc *jsontext.Encoder, stack *[]anyFrame, arr []any, mo *jsonopts.Struct) (bool, error) {
	xe := export.Encoder(enc)
	f := anyFrame{arr: arr}

	// Check for cycles.
	if xe.Tokens.Depth() > startDetectingCyclesAfter {
		v := reflect.ValueOf(arr)
		if err := visitPointer(&xe.SeenPointers, v); err != nil {
			return false, newMarshalErrorBefore(enc, sliceAnyType, err)
		}
		f.seen = true
		f.value = v
	}

	// Handle empty slices.
	if len(arr) == 0 {
		if mo.Flags.Get(jsonflags.FormatNilSliceAsNull) && arr == nil {
			err := enc.WriteToken(jsontext.Null)
			if f.seen {
				leavePointer(&xe.SeenPointers, f.value)
			}
			return false, err
		}
		// Optimize for marshaling an empty slice without any preceding whitespace.
		if !mo.Flags.Get(jsonflags.AnyWhitespace) && !xe.Tokens.Last.NeedObjectName() {
			xe.Buf = append(xe.Tokens.MayAppendDelim(xe.Buf, '['), "[]"...)
			xe.Tokens.Last.Increment()
			var err error
			if xe.NeedFlush() {
				err = xe.Flush()
			}
			if f.seen {
				leavePointer(&xe.SeenPointers, f.value)
			}
			return false, err
		}
	}

	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		if f.seen {
			leavePointer(&xe.SeenPointers, f.value)
		}
		return false, err
	}
	*stack = append(*stack, f)
	return true, nil
}

// unmarshalArrayAny unmarshals a JSON array as a Go []any.
// It panics if not decoding a JSON array.
func unmarshalArrayAny(dec *jsontext.Decoder, uo *jsonopts.Struct) ([]any, error) {
	switch tok, err := dec.ReadToken(); {
	case err != nil:
		return nil, err
	case tok.Kind() != '[':
		panic("BUG: invalid kind: " + tok.Kind().String())
	}
	arr := []any{}
	var errUnmarshal error
	for dec.PeekKind() != ']' {
		val, err := unmarshalValueAny(dec, uo)
		arr = append(arr, val)
		if err != nil {
			if isFatalError(err, uo.Flags) {
				return arr, err
			}
			errUnmarshal = cmp.Or(errUnmarshal, err)
		}
	}
	if _, err := dec.ReadToken(); err != nil {
		return arr, err
	}
	return arr, errUnmarshal
}
