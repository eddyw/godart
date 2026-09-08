// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

// This file patches the file of the same name of the package
// encoding/json/jsontext.
//
// reformatValue of Go descends 1 level for each level of the value, and
// reformatObject and reformatArray each call it again. The stack of Dart holds
// about 1,350 levels of that pair, and jsontext holds a limit of 10,000
// levels. The walk below keeps the open containers in a slice of its own, thus
// it reaches the limit of Go and it reports the same error at the same offset.
//
// Each other line comes from the real file.

package jsontext

import (
	"bytes"
	"io"
	"math/bits"

	"encoding/json/internal/jsonflags"
	"encoding/json/internal/jsonopts"
	"encoding/json/internal/jsonwire"
)

// Encoder is a streaming encoder to raw JSON tokens and values.
// It is used to write a stream of top-level JSON values,
// each terminated with a newline character.
//
// [Encoder.WriteToken] and [Encoder.WriteValue] calls may be interleaved.
// For example, the following JSON value:
//
//	{"name":"value","array":[null,false,true,3.14159],"object":{"k":"v"}}
//
// can be composed with the following calls (ignoring errors for brevity):
//
//	e.WriteToken(BeginObject)        // {
//	e.WriteToken(String("name"))     // "name"
//	e.WriteToken(String("value"))    // "value"
//	e.WriteValue(Value(`"array"`))   // "array"
//	e.WriteToken(BeginArray)         // [
//	e.WriteToken(Null)               // null
//	e.WriteToken(False)              // false
//	e.WriteValue(Value("true"))      // true
//	e.WriteToken(Float(3.14159))     // 3.14159
//	e.WriteToken(EndArray)           // ]
//	e.WriteValue(Value(`"object"`))  // "object"
//	e.WriteValue(Value(`{"k":"v"}`)) // {"k":"v"}
//	e.WriteToken(EndObject)          // }
//
// The above is one of many possible sequences of calls and
// may not represent the most sensible method to call for any given token/value.
// For example, it is probably more common to call [Encoder.WriteToken] with a string
// for object names.
type Encoder struct {
	s encoderState
}

// encoderState is the low-level state of Encoder.
// It has exported fields and methods for use by the "json" package.
type encoderState struct {
	state
	encodeBuffer
	jsonopts.Struct

	SeenPointers map[any]struct{} // only used when marshaling; identical to json.seenPointers
}

// encodeBuffer is a buffer split into 2 segments:
//
//   - buf[0:len(buf)]        // written (but unflushed) portion of the buffer
//   - buf[len(buf):cap(buf)] // unused portion of the buffer
type encodeBuffer struct {
	Buf []byte // may alias wr if it is a bytes.Buffer

	// baseOffset is added to len(buf) to obtain the absolute offset
	// relative to the start of io.Writer stream.
	baseOffset int64

	wr io.Writer

	// maxValue is the approximate maximum Value size passed to WriteValue.
	maxValue int
	// availBuffer is the buffer returned by the AvailableBuffer method.
	availBuffer []byte // always has zero length
	// bufStats is statistics about buffer utilization.
	// It is only used with pooled encoders in pools.go.
	bufStats bufferStatistics
}

// NewEncoder constructs a new streaming encoder writing to w
// configured with the provided options.
// It flushes the internal buffer when the buffer is sufficiently full or
// when a top-level value has been written.
//
// If w is a [bytes.Buffer], then the encoder appends directly into the buffer
// without copying the contents from an intermediate buffer.
func NewEncoder(w io.Writer, opts ...Options) *Encoder {
	e := new(Encoder)
	e.Reset(w, opts...)
	return e
}

// Reset resets an encoder such that it is writing afresh to w and
// configured with the provided options. Reset must not be called on
// an Encoder passed to the [encoding/json/v2.MarshalerTo.MarshalJSONTo] method
// or the [encoding/json/v2.MarshalToFunc] function.
func (e *Encoder) Reset(w io.Writer, opts ...Options) {
	switch {
	case e == nil:
		panic("jsontext: invalid nil Encoder")
	case w == nil:
		panic("jsontext: invalid nil io.Writer")
	case e.s.Flags.Get(jsonflags.WithinArshalCall):
		panic("jsontext: cannot reset Encoder passed to json.MarshalerTo")
	}
	// Reuse the buffer if it does not alias a previous [bytes.Buffer].
	b := e.s.Buf[:0]
	if _, ok := e.s.wr.(*bytes.Buffer); ok {
		b = nil
	}
	e.s.reset(b, w, opts...)
}

func (e *encoderState) reset(b []byte, w io.Writer, opts ...Options) {
	e.state.reset()
	e.encodeBuffer = encodeBuffer{Buf: b, wr: w, availBuffer: e.availBuffer, bufStats: e.bufStats}
	if bb, ok := w.(*bytes.Buffer); ok && bb != nil {
		e.Buf = bb.AvailableBuffer() // alias the unused buffer of bb
	}
	opts2 := jsonopts.Struct{} // avoid mutating e.Struct in case it is part of opts
	opts2.Join(opts...)
	e.Struct = opts2
	if e.Struct.Flags.Get(jsonflags.Multiline) {
		e.Struct.InitializeMultiline()
	}
}

// Options returns the options used to construct the encoder and
// may additionally contain semantic options passed to a
// [encoding/json/v2.MarshalEncode] call.
//
// If operating within
// a [encoding/json/v2.MarshalerTo.MarshalJSONTo] method call or
// a [encoding/json/v2.MarshalToFunc] function call,
// then the returned options are only valid within the call.
func (e *Encoder) Options() Options {
	return &e.s.Struct
}

func (e *encoderState) options() *jsonopts.Struct { return &e.Struct }

// NeedFlush determines whether to flush at this point.
func (e *encoderState) NeedFlush() bool {
	// NOTE: This function is carefully written to be inlinable.

	// Avoid flushing if e.wr is nil since there is no underlying writer.
	// Flush if less than 25% of the capacity remains.
	// Flushing at some constant fraction ensures that the buffer stops growing
	// so long as the largest Token or Value fits within that unused capacity.
	return e.wr != nil && (e.Tokens.Depth() == 1 || len(e.Buf) > 3*cap(e.Buf)/4)
}

// Flush flushes the buffer to the underlying io.Writer.
// It may append a trailing newline after the top-level value.
func (e *encoderState) Flush() error {
	if e.wr == nil || e.avoidFlush() {
		return nil
	}

	// In streaming mode, always emit a newline after the top-level value.
	if e.Tokens.Depth() == 1 && !e.Flags.Get(jsonflags.OmitTopLevelNewline) {
		e.Buf = append(e.Buf, '\n')
	}

	// Inform objectNameStack that we are about to flush the buffer content.
	e.Names.copyQuotedBuffer(e.Buf)

	// Specialize bytes.Buffer for better performance.
	if bb, ok := e.wr.(*bytes.Buffer); ok {
		// If e.Buf already aliases the internal buffer of bb,
		// then the Write call simply increments the internal offset,
		// otherwise Write operates as expected.
		// See https://go.dev/issue/42986.
		n, _ := bb.Write(e.Buf) // never fails unless bb is nil
		e.baseOffset += int64(n)

		// If the internal buffer of bytes.Buffer is too small,
		// append operations elsewhere in the Encoder may grow the buffer.
		// This would be semantically correct, but hurts performance.
		// As such, ensure 25% of the current length is always available
		// to reduce the probability that other appends must allocate.
		if avail := bb.Available(); avail < bb.Len()/4 {
			bb.Grow(avail + 1)
		}

		e.Buf = bb.AvailableBuffer()
		return nil
	}

	// Flush the internal buffer to the underlying io.Writer.
	n, err := e.wr.Write(e.Buf)
	e.baseOffset += int64(n)
	if err != nil {
		// In the event of an error, preserve the unflushed portion.
		// Thus, write errors aren't fatal so long as the io.Writer
		// maintains consistent state after errors.
		if n > 0 {
			e.Buf = e.Buf[:copy(e.Buf, e.Buf[n:])]
		}
		return &ioError{action: "write", err: err}
	}
	e.Buf = e.Buf[:0]

	// Check whether to grow the buffer.
	// Note that cap(e.Buf) may already exceed maxBufferSize since
	// an append elsewhere already grew it to store a large token.
	const maxBufferSize = 4 << 10
	const growthSizeFactor = 2 // higher value is faster
	const growthRateFactor = 2 // higher value is slower
	// By default, grow if below the maximum buffer size.
	grow := cap(e.Buf) <= maxBufferSize/growthSizeFactor
	// Growing can be expensive, so only grow
	// if a sufficient number of bytes have been processed.
	grow = grow && int64(cap(e.Buf)) < e.previousOffsetEnd()/growthRateFactor
	if grow {
		e.Buf = make([]byte, 0, cap(e.Buf)*growthSizeFactor)
	}

	return nil
}
func (e *encodeBuffer) offsetAt(pos int) int64   { return e.baseOffset + int64(pos) }
func (e *encodeBuffer) previousOffsetEnd() int64 { return e.baseOffset + int64(len(e.Buf)) }
func (e *encodeBuffer) unflushedBuffer() []byte  { return e.Buf }

// avoidFlush indicates whether to avoid flushing to ensure there is always
// enough in the buffer to unwrite the last object member if it were empty.
func (e *encoderState) avoidFlush() bool {
	switch {
	case e.Tokens.Last.Length() == 0:
		// Never flush after BeginObject or BeginArray since we don't know yet
		// if the object or array will end up being empty.
		return true
	case e.Tokens.Last.needObjectValue():
		// Never flush before the object value since we don't know yet
		// if the object value will end up being empty.
		return true
	case e.Tokens.Last.NeedObjectName() && len(e.Buf) >= 2:
		// Never flush after the object value if it does turn out to be empty.
		switch string(e.Buf[len(e.Buf)-2:]) {
		case `ll`, `""`, `{}`, `[]`: // last two bytes of every empty value
			return true
		}
	}
	return false
}

// UnwriteEmptyObjectMember unwrites the last object member if it is empty
// and reports whether it performed an unwrite operation.
func (e *encoderState) UnwriteEmptyObjectMember(prevName *string) bool {
	if last := e.Tokens.Last; !last.isObject() || !last.NeedObjectName() || last.Length() == 0 {
		panic("BUG: must be called on an object after writing a value")
	}

	// The flushing logic is modified to never flush a trailing empty value.
	// The encoder never writes trailing whitespace eagerly.
	b := e.unflushedBuffer()

	// Detect whether the last value was empty.
	var n int
	if len(b) >= 3 {
		switch string(b[len(b)-2:]) {
		case "ll": // last two bytes of `null`
			n = len(`null`)
		case `""`:
			// It is possible for a non-empty string to have `""` as a suffix
			// if the second to the last quote was escaped.
			if b[len(b)-3] == '\\' {
				return false // e.g., `"\""` is not empty
			}
			n = len(`""`)
		case `{}`:
			n = len(`{}`)
		case `[]`:
			n = len(`[]`)
		}
	}
	if n == 0 {
		return false
	}

	// Unwrite the value, whitespace, colon, name, whitespace, and comma.
	b = b[:len(b)-n]
	b = jsonwire.TrimSuffixWhitespace(b)
	b = jsonwire.TrimSuffixByte(b, ':')
	b = jsonwire.TrimSuffixString(b)
	b = jsonwire.TrimSuffixWhitespace(b)
	b = jsonwire.TrimSuffixByte(b, ',')
	e.Buf = b // store back truncated unflushed buffer

	// Undo state changes.
	e.Tokens.Last.decrement() // for object member value
	e.Tokens.Last.decrement() // for object member name
	if !e.Flags.Get(jsonflags.AllowDuplicateNames) {
		if e.Tokens.Last.isActiveNamespace() {
			e.Namespaces.Last().removeLast()
		}
	}
	e.Names.clearLast()
	if prevName != nil {
		e.Names.copyQuotedBuffer(e.Buf) // required by objectNameStack.replaceLastUnquotedName
		e.Names.replaceLastUnquotedName(*prevName)
	}
	return true
}

// UnwriteOnlyObjectMemberName unwrites the only object member name
// and returns the unquoted name.
func (e *encoderState) UnwriteOnlyObjectMemberName() string {
	if last := e.Tokens.Last; !last.isObject() || last.Length() != 1 {
		panic("BUG: must be called on an object after writing first name")
	}

	// Unwrite the name and whitespace.
	b := jsonwire.TrimSuffixString(e.Buf)
	isVerbatim := bytes.IndexByte(e.Buf[len(b):], '\\') < 0
	name := string(jsonwire.UnquoteMayCopy(e.Buf[len(b):], isVerbatim))
	e.Buf = jsonwire.TrimSuffixWhitespace(b)

	// Undo state changes.
	e.Tokens.Last.decrement()
	if !e.Flags.Get(jsonflags.AllowDuplicateNames) {
		if e.Tokens.Last.isActiveNamespace() {
			e.Namespaces.Last().removeLast()
		}
	}
	e.Names.clearLast()
	return name
}

// WriteToken writes the next token and advances the internal write offset.
//
// The provided token kind must be consistent with the JSON grammar.
// For example, it is an error to provide a number when the encoder
// is expecting an object name (which is always a string), or
// to provide an end object delimiter when the encoder is finishing an array.
// If the provided token is invalid, then WriteToken reports a [SyntacticError] and
// the internal state remains unchanged. The offset reported
// in [SyntacticError] will be the [Encoder.OutputOffset] plus any delimiter
// or whitespace characters that would have preceded the provided token.
func (e *Encoder) WriteToken(t Token) error {
	return e.s.WriteToken(t)
}
func (e *encoderState) WriteToken(t Token) error {
	k := t.Kind()
	b := e.Buf // use local variable to avoid mutating e in case of error

	// Append any delimiters or optional whitespace.
	b = e.Tokens.MayAppendDelim(b, k)
	if e.Flags.Get(jsonflags.AnyWhitespace) {
		b = e.appendWhitespace(b, k)
	}
	pos := len(b) // offset before the token

	// Append the token to the output and to the state machine.
	var err error
	switch k {
	case 'n':
		b = append(b, "null"...)
		err = e.Tokens.appendLiteral()
	case 'f':
		b = append(b, "false"...)
		err = e.Tokens.appendLiteral()
	case 't':
		b = append(b, "true"...)
		err = e.Tokens.appendLiteral()
	case '"':
		if b, err = t.appendString(b, &e.Flags); err != nil {
			break
		}
		if e.Tokens.Last.NeedObjectName() {
			if !e.Flags.Get(jsonflags.AllowDuplicateNames) {
				if !e.Tokens.Last.isValidNamespace() {
					err = errInvalidNamespace
					break
				}
				if e.Tokens.Last.isActiveNamespace() && !e.Namespaces.Last().insertQuoted(b[pos:], false) {
					err = wrapWithObjectName(ErrDuplicateName, b[pos:])
					break
				}
			}
			e.Names.ReplaceLastQuotedOffset(pos) // only replace if insertQuoted succeeds
		}
		err = e.Tokens.appendString()
	case '0':
		if b, err = t.appendNumber(b, &e.Flags); err != nil {
			break
		}
		err = e.Tokens.appendNumber()
	case '{':
		b = append(b, '{')
		if err = e.Tokens.pushObject(); err != nil {
			break
		}
		e.Names.push()
		if !e.Flags.Get(jsonflags.AllowDuplicateNames) {
			e.Namespaces.push()
		}
		e.Flags.Clear(jsonflags.TagFlags) // tags only apply to current depth
	case '}':
		b = append(b, '}')
		if err = e.Tokens.popObject(); err != nil {
			break
		}
		e.Names.pop()
		if !e.Flags.Get(jsonflags.AllowDuplicateNames) {
			e.Namespaces.pop()
		}
	case '[':
		b = append(b, '[')
		err = e.Tokens.pushArray()
		e.Flags.Clear(jsonflags.TagFlags) // tags only apply to current depth
	case ']':
		b = append(b, ']')
		err = e.Tokens.popArray()
	default:
		err = errInvalidToken
	}
	if err != nil {
		return wrapSyntacticError(e, err, pos, +1)
	}

	// Finish off the buffer and store it back into e.
	e.Buf = b
	if e.NeedFlush() {
		return e.Flush()
	}
	return nil
}

// AppendRaw appends either a raw string (without double quotes) or number.
// Specify safeASCII if the string output is guaranteed to be ASCII
// without any characters (including '<', '>', and '&') that need escaping,
// otherwise this will validate whether the string needs escaping.
// The appended bytes for a JSON number must be valid.
//
// This is a specialized implementation of Encoder.WriteValue
// that allows appending directly into the buffer.
// It is only called from marshal logic in the "json" package.
func (e *encoderState) AppendRaw(k Kind, safeASCII bool, appendFn func([]byte) ([]byte, error)) error {
	b := e.Buf // use local variable to avoid mutating e in case of error

	// Append any delimiters or optional whitespace.
	b = e.Tokens.MayAppendDelim(b, k)
	if e.Flags.Get(jsonflags.AnyWhitespace) {
		b = e.appendWhitespace(b, k)
	}
	pos := len(b) // offset before the token

	var err error
	switch k {
	case '"':
		// Append directly into the encoder buffer by assuming that
		// most of the time none of the characters need escaping.
		b = append(b, '"')
		if b, err = appendFn(b); err != nil {
			return err
		}
		b = append(b, '"')

		// Check whether we need to escape the string and if necessary
		// copy it to a scratch buffer and then escape it back.
		isVerbatim := safeASCII || !jsonwire.NeedEscape(b[pos+len(`"`):len(b)-len(`"`)])
		if !isVerbatim {
			var err error
			b2 := append(e.availBuffer, b[pos+len(`"`):len(b)-len(`"`)]...)
			b, err = jsonwire.AppendQuote(b[:pos], b2, &e.Flags)
			e.availBuffer = b2[:0]
			if err != nil {
				return wrapSyntacticError(e, err, pos, +1)
			}
		}

		// Update the state machine.
		if e.Tokens.Last.NeedObjectName() {
			if !e.Flags.Get(jsonflags.AllowDuplicateNames) {
				if !e.Tokens.Last.isValidNamespace() {
					return wrapSyntacticError(e, errInvalidNamespace, pos, +1)
				}
				if e.Tokens.Last.isActiveNamespace() && !e.Namespaces.Last().insertQuoted(b[pos:], isVerbatim) {
					err = wrapWithObjectName(ErrDuplicateName, b[pos:])
					return wrapSyntacticError(e, err, pos, +1)
				}
			}
			e.Names.ReplaceLastQuotedOffset(pos) // only replace if insertQuoted succeeds
		}
		if err := e.Tokens.appendString(); err != nil {
			return wrapSyntacticError(e, err, pos, +1)
		}
	case '0':
		if b, err = appendFn(b); err != nil {
			return err
		}
		if err := e.Tokens.appendNumber(); err != nil {
			return wrapSyntacticError(e, err, pos, +1)
		}
	default:
		panic("BUG: invalid kind")
	}

	// Finish off the buffer and store it back into e.
	e.Buf = b
	if e.NeedFlush() {
		return e.Flush()
	}
	return nil
}

// WriteValue writes the next raw value and advances the internal write offset.
// The Encoder does not simply copy the provided value verbatim, but
// parses it to ensure that it is syntactically valid and reformats it
// according to how the Encoder is configured to format whitespace and strings.
// If [AllowInvalidUTF8] is specified, then any invalid UTF-8 is mangled
// as the Unicode replacement character, U+FFFD.
//
// The provided value kind must be consistent with the JSON grammar
// (see examples on [Encoder.WriteToken]). If the provided value is invalid,
// then WriteValue reports a [SyntacticError] and the internal state remains unchanged.
// The offset reported in [SyntacticError] will be the [Encoder.OutputOffset]
// plus the offset into v of any encountered syntax error.
func (e *Encoder) WriteValue(v Value) error {
	return e.s.WriteValue(v)
}
func (e *encoderState) WriteValue(v Value) error {
	e.maxValue |= len(v) // bitwise OR is a fast approximation of max

	k := v.Kind()
	b := e.Buf // use local variable to avoid mutating e in case of error

	// Append any delimiters or optional whitespace.
	b = e.Tokens.MayAppendDelim(b, k)
	if e.Flags.Get(jsonflags.AnyWhitespace) {
		b = e.appendWhitespace(b, k)
	}
	pos := len(b) // offset before the value

	// Append the value to the output.
	var n int
	n += jsonwire.ConsumeWhitespace(v[n:])
	b, m, err := e.reformatValue(b, v[n:], e.Tokens.Depth())
	if err != nil {
		return wrapSyntacticError(e, err, pos+n+m, +1)
	}
	n += m
	n += jsonwire.ConsumeWhitespace(v[n:])
	if len(v) > n {
		err = jsonwire.NewInvalidCharacterError(v[n:], "after top-level value")
		return wrapSyntacticError(e, err, pos+n, 0)
	}

	// Append the kind to the state machine.
	switch k {
	case 'n', 'f', 't':
		err = e.Tokens.appendLiteral()
	case '"':
		if e.Tokens.Last.NeedObjectName() {
			if !e.Flags.Get(jsonflags.AllowDuplicateNames) {
				if !e.Tokens.Last.isValidNamespace() {
					err = errInvalidNamespace
					break
				}
				if e.Tokens.Last.isActiveNamespace() && !e.Namespaces.Last().insertQuoted(b[pos:], false) {
					err = wrapWithObjectName(ErrDuplicateName, b[pos:])
					break
				}
			}
			e.Names.ReplaceLastQuotedOffset(pos) // only replace if insertQuoted succeeds
		}
		err = e.Tokens.appendString()
	case '0':
		err = e.Tokens.appendNumber()
	case '{':
		if err = e.Tokens.pushObject(); err != nil {
			break
		}
		if err = e.Tokens.popObject(); err != nil {
			panic("BUG: popObject should never fail immediately after pushObject: " + err.Error())
		}
		if e.Flags.Get(jsonflags.ReorderRawObjects) {
			mustReorderObjects(b[pos:])
		}
	case '[':
		if err = e.Tokens.pushArray(); err != nil {
			break
		}
		if err = e.Tokens.popArray(); err != nil {
			panic("BUG: popArray should never fail immediately after pushArray: " + err.Error())
		}
		if e.Flags.Get(jsonflags.ReorderRawObjects) {
			mustReorderObjects(b[pos:])
		}
	}
	if err != nil {
		return wrapSyntacticError(e, err, pos, +1)
	}

	// Finish off the buffer and store it back into e.
	e.Buf = b
	if e.NeedFlush() {
		return e.Flush()
	}
	return nil
}

// CountNextDelimWhitespace counts the number of bytes of delimiter and
// whitespace bytes assuming the upcoming token is a JSON value.
// This method is used for error reporting at the semantic layer.
func (e *encoderState) CountNextDelimWhitespace() (n int) {
	const next = Kind('"') // arbitrary kind as next JSON value
	delim := e.Tokens.needDelim(next)
	if delim > 0 {
		n += len(",") | len(":")
	}
	if delim == ':' {
		if e.Flags.Get(jsonflags.SpaceAfterColon) {
			n += len(" ")
		}
	} else {
		if delim == ',' && e.Flags.Get(jsonflags.SpaceAfterComma) {
			n += len(" ")
		}
		if e.Flags.Get(jsonflags.Multiline) {
			if m := e.Tokens.NeedIndent(next); m > 0 {
				n += len("\n") + len(e.IndentPrefix) + (m-1)*len(e.Indent)
			}
		}
	}
	return n
}

// appendWhitespace appends whitespace that immediately precedes the next token.
func (e *encoderState) appendWhitespace(b []byte, next Kind) []byte {
	if delim := e.Tokens.needDelim(next); delim == ':' {
		if e.Flags.Get(jsonflags.SpaceAfterColon) {
			b = append(b, ' ')
		}
	} else {
		if delim == ',' && e.Flags.Get(jsonflags.SpaceAfterComma) {
			b = append(b, ' ')
		}
		if e.Flags.Get(jsonflags.Multiline) {
			b = e.AppendIndent(b, e.Tokens.NeedIndent(next))
		}
	}
	return b
}

// AppendIndent appends the appropriate number of indentation characters
// for the current nested level, n.
func (e *encoderState) AppendIndent(b []byte, n int) []byte {
	if n == 0 {
		return b
	}
	b = append(b, '\n')
	b = append(b, e.IndentPrefix...)
	for ; n > 1; n-- {
		b = append(b, e.Indent...)
	}
	return b
}

// reformatValue parses a JSON value from the start of src and
// appends it to the end of dst, reformatting whitespace and strings as needed.
// It returns the extended dst buffer and the number of consumed input bytes.
//
// The walk holds each open object and each open array in a slice of its own.
// The recursive form of Go costs 2 frames of the machine for each level, and
// the stack of Dart holds about 1,350 of those pairs. jsontext holds a limit
// of 10,000 levels, thus the recursive form reaches no such value.
//
// The count of the bytes is the count of the whole value, exactly as the sum
// of the counts of the recursive form gives.
func (e *encoderState) reformatValue(dst []byte, src Value, depth int) ([]byte, int, error) {
	// TODO: Should this update ValueFlags as input?
	var stack []reformatFrame

	// fail ends the walk. It pops the namespace of each open object, and it
	// wraps the error with each open frame. inner reports whether the frame at
	// the top wraps also: a value that ended inside it carries its name or its
	// index, and a delimiter of the frame itself carries neither.
	fail := func(dst []byte, n int, err error, inner bool) ([]byte, int, error) {
		for i := len(stack) - 1; i >= 0; i-- {
			f := stack[i]
			if i < len(stack)-1 || inner {
				if f.object {
					err = wrapWithObjectName(err, f.quotedName)
				} else {
					err = wrapWithArrayIndex(err, f.idx)
				}
			}
			if f.pushed {
				e.Namespaces.pop()
			}
		}
		return dst, n, err
	}

	n := 0
	step := stepValue
	for {
		switch step {
		case stepValue:
			if uint(len(src)) <= uint(n) {
				return fail(dst, n, io.ErrUnexpectedEOF, true)
			}
			k := Kind(src[n]).normalize()
			if k == '{' || k == '[' {
				object := k == '{'
				if depth+len(stack) == maxNestingDepth+1 {
					return fail(dst, n, errMaxDepth, true)
				}
				closer := byte(']')
				if object {
					closer = '}'
				}
				dst = append(dst, byte(k))
				n++

				// Append (possible) end of the container.
				n += jsonwire.ConsumeWhitespace(src[n:])
				if uint(len(src)) <= uint(n) {
					return fail(dst, n, io.ErrUnexpectedEOF, true)
				}
				if src[n] == closer {
					dst = append(dst, closer)
					n++
					step = stepAfter
					continue
				}
				f := reformatFrame{object: object}
				if object && !e.Flags.Get(jsonflags.AllowDuplicateNames) {
					e.Namespaces.push()
					f.pushed = true
				}
				stack = append(stack, f)
				step = stepElem
				if object {
					step = stepName
				}
				continue
			}
			var m int
			var err error
			switch k {
			case 'n':
				if jsonwire.ConsumeNull(src[n:]) == 0 {
					m, err = jsonwire.ConsumeLiteral(src[n:], "null")
					return fail(dst, n+m, err, true)
				}
				dst = append(dst, "null"...)
				n += len("null")
			case 'f':
				if jsonwire.ConsumeFalse(src[n:]) == 0 {
					m, err = jsonwire.ConsumeLiteral(src[n:], "false")
					return fail(dst, n+m, err, true)
				}
				dst = append(dst, "false"...)
				n += len("false")
			case 't':
				if jsonwire.ConsumeTrue(src[n:]) == 0 {
					m, err = jsonwire.ConsumeLiteral(src[n:], "true")
					return fail(dst, n+m, err, true)
				}
				dst = append(dst, "true"...)
				n += len("true")
			case '"':
				if m = jsonwire.ConsumeSimpleString(src[n:]); m > 0 {
					dst = append(dst, src[n:n+m]...) // copy simple strings verbatim
					n += m
					break
				}
				dst, m, err = jsonwire.ReformatString(dst, src[n:], &e.Flags)
				if err != nil {
					return fail(dst, n+m, err, true)
				}
				n += m
			case '0':
				if m = jsonwire.ConsumeSimpleNumber(src[n:]); m > 0 && !e.Flags.Get(jsonflags.CanonicalizeNumbers) {
					dst = append(dst, src[n:n+m]...) // copy simple numbers verbatim
					n += m
					break
				}
				dst, m, err = jsonwire.ReformatNumber(dst, src[n:], &e.Flags)
				if err != nil {
					return fail(dst, n+m, err, true)
				}
				n += m
			default:
				err = jsonwire.NewInvalidCharacterError(src[n:], "at start of value")
				return fail(dst, n, err, true)
			}
			step = stepAfter

		case stepName:
			top := len(stack) - 1

			// Append optional newline and indentation.
			if e.Flags.Get(jsonflags.Multiline) {
				dst = e.AppendIndent(dst, depth+len(stack))
			}

			// Append object name.
			n += jsonwire.ConsumeWhitespace(src[n:])
			if uint(len(src)) <= uint(n) {
				return fail(dst, n, io.ErrUnexpectedEOF, false)
			}
			m := jsonwire.ConsumeSimpleString(src[n:])
			isVerbatim := m > 0
			if isVerbatim {
				dst = append(dst, src[n:n+m]...)
			} else {
				var err error
				dst, m, err = jsonwire.ReformatString(dst, src[n:], &e.Flags)
				if err != nil {
					return fail(dst, n+m, err, false)
				}
			}
			stack[top].quotedName = src[n : n+m]
			if !e.Flags.Get(jsonflags.AllowDuplicateNames) &&
				!e.Namespaces.Last().insertQuoted(stack[top].quotedName, isVerbatim) {
				return fail(dst, n, ErrDuplicateName, true)
			}
			n += m

			// Append colon.
			n += jsonwire.ConsumeWhitespace(src[n:])
			if uint(len(src)) <= uint(n) {
				return fail(dst, n, io.ErrUnexpectedEOF, true)
			}
			if src[n] != ':' {
				err := jsonwire.NewInvalidCharacterError(src[n:], "after object name (expecting ':')")
				return fail(dst, n, err, true)
			}
			dst = append(dst, ':')
			n += len(":")
			if e.Flags.Get(jsonflags.SpaceAfterColon) {
				dst = append(dst, ' ')
			}

			// Append object value.
			n += jsonwire.ConsumeWhitespace(src[n:])
			if uint(len(src)) <= uint(n) {
				return fail(dst, n, io.ErrUnexpectedEOF, true)
			}
			step = stepValue

		case stepElem:
			// Append optional newline and indentation.
			if e.Flags.Get(jsonflags.Multiline) {
				dst = e.AppendIndent(dst, depth+len(stack))
			}

			// Append array value.
			n += jsonwire.ConsumeWhitespace(src[n:])
			if uint(len(src)) <= uint(n) {
				return fail(dst, n, io.ErrUnexpectedEOF, false)
			}
			step = stepValue

		case stepAfter:
			if len(stack) == 0 {
				return dst, n, nil
			}
			top := len(stack) - 1

			// Append comma or the end of the container.
			n += jsonwire.ConsumeWhitespace(src[n:])
			if uint(len(src)) <= uint(n) {
				return fail(dst, n, io.ErrUnexpectedEOF, false)
			}
			if stack[top].object {
				switch src[n] {
				case ',':
					dst = append(dst, ',')
					if e.Flags.Get(jsonflags.SpaceAfterComma) {
						dst = append(dst, ' ')
					}
					n += len(",")
					step = stepName
				case '}':
					if e.Flags.Get(jsonflags.Multiline) {
						dst = e.AppendIndent(dst, depth+len(stack)-1)
					}
					dst = append(dst, '}')
					n += len("}")
					if stack[top].pushed {
						e.Namespaces.pop()
					}
					stack = stack[:top]
					step = stepAfter
				default:
					err := jsonwire.NewInvalidCharacterError(src[n:], "after object value (expecting ',' or '}')")
					return fail(dst, n, err, false)
				}
				continue
			}
			switch src[n] {
			case ',':
				dst = append(dst, ',')
				if e.Flags.Get(jsonflags.SpaceAfterComma) {
					dst = append(dst, ' ')
				}
				n += len(",")
				stack[top].idx++
				step = stepElem
			case ']':
				if e.Flags.Get(jsonflags.Multiline) {
					dst = e.AppendIndent(dst, depth+len(stack)-1)
				}
				dst = append(dst, ']')
				n += len("]")
				stack = stack[:top]
				step = stepAfter
			default:
				err := jsonwire.NewInvalidCharacterError(src[n:], "after array value (expecting ',' or ']')")
				return fail(dst, n, err, false)
			}
		}
	}
}

// reformatFrame holds 1 open object or 1 open array of the walk of
// reformatValue.
type reformatFrame struct {
	// object marks an object. An array holds false.
	object bool
	// pushed marks a frame that pushed a namespace of the names. The pop of it
	// stands at the end of the frame, exactly as the deferred pop of Go does.
	pushed bool
	// idx counts the elements of an array that the walk already wrote. The
	// wrapper of an error reads it.
	idx int64
	// quotedName holds the name of the member that the frame writes now. The
	// wrapper of an error reads it.
	quotedName []byte
}

// OutputOffset returns the current output byte offset. It gives the location
// of the next byte immediately after the most recently written token or value.
// The number of bytes actually written to the underlying [io.Writer] may be less
// than this offset due to internal buffering.
func (e *Encoder) OutputOffset() int64 {
	return e.s.previousOffsetEnd()
}

// AvailableBuffer returns a zero-length buffer with a possible non-zero capacity.
// This buffer is intended to be used to populate a [Value]
// being passed to an immediately succeeding [Encoder.WriteValue] call.
//
// Example usage:
//
//	b := e.AvailableBuffer()
//	b = append(b, '"')
//	b = appendString(b, v) // append the string formatting of v
//	b = append(b, '"')
//	... := e.WriteValue(b)
//
// Note that WriteValue expects a valid JSON value. Constructing a value in a
// raw []byte requires more care than using constructor functions like
// [String], which always return valid [Token]s or [Value]s.
func (e *Encoder) AvailableBuffer() []byte {
	// NOTE: We don't return e.Buf[len(e.Buf):cap(e.Buf)] since WriteValue would
	// need to take special care to avoid mangling the data while reformatting.
	// WriteValue can't easily identify whether the input Value aliases e.Buf
	// without using unsafe.Pointer. Thus, we just return a different buffer.
	// Should this ever alias e.Buf, we need to consider how it operates with
	// the specialized performance optimization for bytes.Buffer.
	n := 1 << bits.Len(uint(e.s.maxValue|63)) // fast approximation for max length
	if cap(e.s.availBuffer) < n {
		e.s.availBuffer = make([]byte, 0, n)
	}
	return e.s.availBuffer
}

// StackDepth returns the depth of the state machine for written JSON data.
// Each level on the stack represents a nested JSON object or array.
// It is incremented whenever a [BeginObject] or [BeginArray] token is encountered
// and decremented whenever an [EndObject] or [EndArray] token is encountered.
//
// StackDepth returns 0 when not inside any object or array.
// In particular, it returns 0 before any tokens have been written,
// after any top-level value has been written, and between values
// when encoding a stream of top-level values (e.g., NDJSON).
// StackDepth returns 1 inside a top-level object or array,
// 2 inside a nested object or array, and so on.
//
// For example, consider encoding the following JSON:
//
//	{"a": [1, 2], "b": {"c": 3}}
//
// While encoding, StackDepth would report the following:
//
//   - At the start, StackDepth reports 0.
//   - After encoding the outer '{', StackDepth reports 1.
//   - After encoding the inner '[', StackDepth reports 2.
//   - After encoding the inner ']', StackDepth reports 1.
//   - After encoding the outer '}', StackDepth reports 0.
func (e *Encoder) StackDepth() int {
	// NOTE: Keep in sync with Decoder.StackDepth.
	return e.s.Tokens.Depth() - 1
}

// StackIndex returns information about the specified stack level.
// It must be a number between 0 and [Encoder.StackDepth], inclusive.
// For each level, it reports the kind:
//
//   - [KindInvalid] for a level of zero,
//   - [KindBeginObject] for a level representing a JSON object, and
//   - [KindBeginArray] for a level representing a JSON array.
//
// It also reports the length of that JSON object or array encoded so far.
// Each name and value in a JSON object is counted separately,
// so the effective number of members is half the length.
// A complete JSON object must have an even length.
func (e *Encoder) StackIndex(i int) (Kind, int64) {
	// NOTE: Keep in sync with Decoder.StackIndex.
	switch s := e.s.Tokens.index(i); {
	case i > 0 && s.isObject():
		return '{', s.Length()
	case i > 0 && s.isArray():
		return '[', s.Length()
	default:
		return 0, s.Length()
	}
}

// StackPointer returns a JSON Pointer (RFC 6901) to the most recently written value.
func (e *Encoder) StackPointer() Pointer {
	return Pointer(e.s.AppendStackPointer(nil, -1))
}

func (e *encoderState) AppendStackPointer(b []byte, where int) []byte {
	e.Names.copyQuotedBuffer(e.Buf)
	return e.state.appendStackPointer(b, where)
}
