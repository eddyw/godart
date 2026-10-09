//go:build dart

// Package dart gives Go code access to the Dart runtime.
//
// The compiler replaces each call to this package with a call to the runtime
// library, thus no body below ever runs.
package dart

import "unsafe"

// Print writes text to standard output.
func Print(text string) {
	panic("syscall/dart needs the gart compiler")
}

// Format writes 1 value in the format of a Go verb.
//
// The verb holds the flags, the precision, and the letter, for example "v",
// "+v", or ".2f". The width belongs to the caller.
func Format(verb string, value any) string {
	panic("syscall/dart needs the gart compiler")
}

// FormatVerb writes 1 value in the parsed form of a Go verb.
//
// letter holds the code of the letter of the verb, flags holds the bit of each
// flag, width holds the least count of characters or 0, and prec holds the
// number after the point or -1.
//
// The runtime owns the width, because Go writes it on each leaf of a value. A
// slice, an array, a map, and a struct each give the width to every element,
// and the brackets take none.
//
// The package fmt reads the format text already. A bridge that takes the text
// again builds 1 Go string, decodes it, and scans it 1 time for each flag, and
// Sprintf calls the bridge 1 time for each verb.
func FormatVerb(letter, flags, width, prec int, value any) string {
	panic("syscall/dart needs the gart compiler")
}

// Stack gives 1 item for each frame of the Go stack, in the form
// "function|file|line". The count of skip drops the first frames, and frame 0
// is the function that calls Stack.
//
// A function that gives up control leaves the Dart stack, thus the stack of a
// goroutine that waits holds the frames of the current state only.
func Stack(skip int) []string {
	panic("syscall/dart needs the gart compiler")
}

// RecordPC gives the stable synthetic program counter of a source position.
func RecordPC(info string) uintptr {
	panic("syscall/dart needs the gart compiler")
}

// PCInfo gives function|file|line for a known synthetic program counter.
func PCInfo(pc uintptr) string {
	panic("syscall/dart needs the gart compiler")
}

// PCEntry gives the function entry counter for a known source position.
func PCEntry(pc uintptr) uintptr {
	panic("syscall/dart needs the gart compiler")
}

// CaptureOut starts a hold of the text of standard output, or it ends the hold
// and gives the text. The harness of an example of the package testing reads
// it.
func CaptureOut(on bool) string {
	panic("syscall/dart needs the gart compiler")
}

// GoID gives the number of the goroutine that runs now. The scheduler numbers
// each goroutine from 1, exactly as Go does.
func GoID() int {
	panic("syscall/dart needs the gart compiler")
}

// Creator gives the current goroutine creator in the frame format of Parked.
// An empty string identifies a goroutine without a Go creator.
func Creator() string {
	panic("syscall/dart needs the gart compiler")
}

// Parked gives the state of each goroutine that gave up control, except the
// goroutine that runs now.
//
// 1 item holds 1 goroutine. The first line holds the number and the reason,
// with a bar between them. Each line after it holds 1 frame in the form
// "function|file|line".
func Parked() []string {
	panic("syscall/dart needs the gart compiler")
}

// Gosched gives up control and puts the goroutine on the run queue again.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func Gosched() {
	panic("syscall/dart needs the gart compiler")
}

// Goexit ends the goroutine that runs now. Each deferred call of the chain
// runs first.
func Goexit() {
	panic("syscall/dart needs the gart compiler")
}

// Fatal ends the process without a recoverable panic or deferred calls.
func Fatal(message string) {
	panic("syscall/dart needs the gart compiler")
}

// SetPanicNil installs the builder of the value that `panic(nil)` gives. The
// initializer of the package runtime calls it, because only the emitted Go
// type holds the class of that value.
func SetPanicNil(build func() any) {
	panic("syscall/dart needs the gart compiler")
}

// SetTypeAssertionError supplies the emitted runtime type for assertion panics.
func SetTypeAssertionError(build func(string) any) {
	panic("syscall/dart needs the gart compiler")
}

// DialTLS opens a TLS connection to a host and a port, and gives its handle.
// The handle joins the table of the connections, thus ConnRead, ConnWrite,
// ConnDeadline, and ConnClose each work on it.
//
// insecure drops the check of the certificate. The name of the server holds the
// host of the check, and an empty name reads the host of the address. The list
// of ALPN holds each protocol that the client offers, and the second result
// names the protocol that the handshake took.
func DialTLS(host string, port int, insecure bool, name string, protos []string) (int, string, string) {
	panic("syscall/dart needs the gart compiler")
}

// DialTLSProxy opens a TLS connection through an HTTP proxy and gives its
// handle. head holds the whole CONNECT request, thus the caller names the
// target and the authorization.
//
// The other 4 arguments and the 3 results match DialTLS.
func DialTLSProxy(host string, port int, head string, insecure bool, name string, protos []string) (int, string, string) {
	panic("syscall/dart needs the gart compiler")
}

// ConnOption writes a socket option of a connection. The name holds
// `nodelay`, `keepalive`, `rcvbuf`, or `sndbuf`. A flag reads the value 0 or 1.
// The results retain the error text and an available native error code.
//
// The number of an option differs between the platforms, thus a platform that
// the table of the runtime holds not gives an error. A wrong number would
// write another option with no report.
func ConnOption(handle int, name string, value int) (string, int) {
	panic("syscall/dart needs the gart compiler")
}

// PacketOption writes a socket option of a datagram socket. The name holds
// `rcvbuf` or `sndbuf`. The results follow ConnOption.
func PacketOption(handle int, name string, value int) (string, int) {
	panic("syscall/dart needs the gart compiler")
}

// GC requests collection and yields to cleanup callbacks.
func GC() {
	panic("syscall/dart needs the gart compiler")
}

// AddCleanup attaches a function that runs after the collector of Dart frees
// the value, and it gives the handle of the cleanup.
func AddCleanup(target, arg any, pointerArg bool, run func()) int {
	panic("syscall/dart needs the gart compiler")
}

// StopCleanup drops a cleanup that no collection ran.
func StopCleanup(handle int) {
	panic("syscall/dart needs the gart compiler")
}

// IntVerb gives the parts of an integer value that the verb U and the verb q
// of the package fmt read.
//
// hex holds the 64-bit pattern in upper case with no padding, and r holds the
// rune of the value or the replacement rune when the value stands outside the
// range of a rune. ok is false when the value holds no integer.
//
// A type switch of Go reads the named type of a value not, thus fmt reads the
// value here and not with such a switch.
func IntVerb(value any) (hex string, r rune, ok bool) {
	panic("syscall/dart needs the gart compiler")
}

// SetFormatHooks gives the runtime the 2 calls of the package fmt that the
// Dart side cannot hold.
//
// format writes 1 leaf whose type holds a Format method, with the letter of
// the verb, the bits of the flags, the width, and the precision. The result is
// the whole text of that leaf.
//
// isPrint reports whether Go prints a rune as itself. The table of the graphic
// categories of Unicode lives in the package strconv.
func SetFormatHooks(format func(v any, letter, flags, width, prec int) string, isPrint func(r rune) bool) {
	panic("syscall/dart needs the gart compiler")
}

// SliceOverlap reports whether 2 slices share a range of 1 backing array.
//
// Go reads the address of the first element and of the last element of each
// slice, and the subset holds no such read. The Dart form of a slice holds the
// backing list itself, thus the runtime compares the identity of the 2 lists
// and the 2 ranges.
func SliceOverlap(a, b any) bool {
	panic("syscall/dart needs the gart compiler")
}

// SetValueHook gives the runtime the read of a reflect.Value. The package fmt
// writes the value that such a value holds, and not the text of its String
// method, thus the runtime must reach the value inside it.
//
// The overlay of reflect installs the hook, because only that package names
// the type.
func SetValueHook(read func(v any) (any, bool)) {
	panic("syscall/dart needs the gart compiler")
}

// Kind gives the kind of a value, in the words of the package reflect.
func Kind(value any) string {
	panic("syscall/dart needs the gart compiler")
}

// TypeInfo gives the text of the type of a value and the comparable flag. The
// package internal/reflectlite reads it.
func TypeInfo(value any) (string, bool) {
	panic("syscall/dart needs the gart compiler")
}

// DeepEqual reports whether 2 values are deeply equal. The package reflect
// reads it, and the walk follows the type descriptor of the runtime.
func DeepEqual(x, y any) bool {
	panic("syscall/dart needs the gart compiler")
}

// DescOf gives a handle of the type of a value. A nil value gives 0.
//
// The package reflect names a type that no value holds, for example the element
// type of a pointer. The runtime holds a table of the descriptors for it.
func DescOf(value any) int {
	panic("syscall/dart needs the gart compiler")
}

// DescInfo gives the text, the kind, and the comparable flag of a type handle.
func DescInfo(handle int) (string, string, bool) {
	panic("syscall/dart needs the gart compiler")
}

// DescNumIn gives the count of the parameters of a function type.
func DescNumIn(handle int) int {
	panic("syscall/dart needs the gart compiler")
}

// DescNumOut gives the count of the results of a function type.
func DescNumOut(handle int) int {
	panic("syscall/dart needs the gart compiler")
}

// DescIn gives a handle of the type of 1 parameter of a function type.
func DescIn(handle, index int) int {
	panic("syscall/dart needs the gart compiler")
}

// DescOut gives a handle of the type of 1 result of a function type.
func DescOut(handle, index int) int {
	panic("syscall/dart needs the gart compiler")
}

// DescKey gives a handle of the key type of a map type.
func DescKey(handle int) int {
	panic("syscall/dart needs the gart compiler")
}

// CallValue calls a function value with the arguments and gives each result.
func CallValue(fn any, args []any) []any {
	panic("syscall/dart needs the gart compiler")
}

// DescNumMethod gives the method count for reflection.
func DescNumMethod(handle int) int {
	panic("syscall/dart needs the gart compiler")
}

// DescMethodName gives the method name at index in reflection order.
func DescMethodName(handle, index int) string {
	panic("syscall/dart needs the gart compiler")
}

// DescMethodPkgPath gives the package path of an unexported method.
func DescMethodPkgPath(handle, index int) string {
	panic("syscall/dart needs the gart compiler")
}

// MethodValue binds the method at index in the static type to the value.
func MethodValue(value any, handle, index int) any {
	panic("syscall/dart needs the gart compiler")
}

// ValueIsNil reports whether the value of a kind that can hold nil holds nil.
func ValueIsNil(value any) bool {
	panic("syscall/dart needs the gart compiler")
}

// MapKeys gives the keys of a map value. The order follows the insertion.
func MapKeys(target any) []any {
	panic("syscall/dart needs the gart compiler")
}

// MapIndex gives the value of 1 key of a map value, or nil.
func MapIndex(target, key any) any {
	panic("syscall/dart needs the gart compiler")
}

// MapHas reports whether a map value holds the key. An entry that holds a nil
// interface reads as nil, thus MapIndex alone cannot answer it.
func MapHas(target, key any) bool {
	panic("syscall/dart needs the gart compiler")
}

// MapIterOf gives an iterator over the pairs of a map value. The iterator
// reads the map itself, thus a clear inside the loop ends the loop.
func MapIterOf(target any) any {
	panic("syscall/dart needs the gart compiler")
}

// MapIterNext steps the iterator, and it reports whether a pair stands there.
func MapIterNext(it any) bool {
	panic("syscall/dart needs the gart compiler")
}

// MapIterKey gives the key of the pair that the iterator holds.
func MapIterKey(target, it any) any {
	panic("syscall/dart needs the gart compiler")
}

// MapIterValue gives the value of the pair that the iterator holds.
func MapIterValue(target, it any) any {
	panic("syscall/dart needs the gart compiler")
}

// MakeMap gives a map of the type.
// ValueRecv reads 1 value of a channel that a Value holds. The flag is false
// when the channel closed.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ValueRecv(value any) (any, bool) {
	panic("syscall/dart needs the gart compiler")
}

// ValueAddr gives a pointer to the location that a Value names. The loc names
// the kind of the location: 1 for a pointer, 2 for a field of a struct, and 3
// for an element of a slice or of an array.
func ValueAddr(value, owner any, index, loc int) any {
	panic("syscall/dart needs the gart compiler")
}

// ValueEqual reports whether 2 values hold the same type and the same value.
func ValueEqual(a, b any) bool {
	panic("syscall/dart needs the gart compiler")
}

// ValueConvert gives the value of a type conversion.
func ValueConvert(value any, handle int) any {
	panic("syscall/dart needs the gart compiler")
}

// ValueSlice gives the slice of a value, with the low, the high, and the
// maximum.
func ValueSlice(value any, low, high, max int) any {
	panic("syscall/dart needs the gart compiler")
}

// AsComplex gives the complex value inside an interface value.
func AsComplex(value any) complex128 {
	panic("syscall/dart needs the gart compiler")
}

// WithComplex gives a complex value of the type of a handle.
func WithComplex(handle int, x complex128) any {
	panic("syscall/dart needs the gart compiler")
}

// ValuePointer gives a number that names the object of a value.
func ValuePointer(value any) uintptr {
	panic("syscall/dart needs the gart compiler")
}

// ValueUnsafePointer returns the opaque pointer of a reflected value.
func ValueUnsafePointer(value any) unsafe.Pointer {
	panic("syscall/dart needs the gart compiler")
}

// DescImplements reports whether a type holds each method of an interface
// type.
func DescImplements(handle, iface int) bool {
	panic("syscall/dart needs the gart compiler")
}

// DescAssignable reports whether a value of the first type fits a location of
// the second one.
func DescAssignable(handle, other int) bool {
	panic("syscall/dart needs the gart compiler")
}

// DescConvertible reports whether a conversion of the first type to the second
// one holds.
func DescConvertible(handle, other int) bool {
	panic("syscall/dart needs the gart compiler")
}

// FieldEmbedded reports whether a field of a struct type is embedded.
func FieldEmbedded(handle, index int) bool {
	panic("syscall/dart needs the gart compiler")
}

// DescPkgPath gives the import path of the package of a type.
func DescPkgPath(handle int) string {
	panic("syscall/dart needs the gart compiler")
}

// DescVariadic reports whether the last parameter of a function type is
// variadic.
func DescVariadic(handle int) bool {
	panic("syscall/dart needs the gart compiler")
}

// DescChanDir gives the direction of a channel type.
func DescChanDir(handle int) int {
	panic("syscall/dart needs the gart compiler")
}

// SliceOf gives the handle of the slice type of an element type.
func SliceOf(elem int) int {
	panic("syscall/dart needs the gart compiler")
}

// FuncOf gives the handle of the function type of a parameter list, a result
// list, and the variadic flag.
func FuncOf(params, results []int, variadic bool) int {
	panic("syscall/dart needs the gart compiler")
}

// ArrayOf gives the handle of the array type of a length and an element type.
func ArrayOf(length, elem int) int {
	panic("syscall/dart needs the gart compiler")
}

// MapOf gives the handle of the map type of a key type and an element type.
func MapOf(key, elem int) int {
	panic("syscall/dart needs the gart compiler")
}

// PointerTo gives the handle of the pointer type of an element type.
func PointerTo(elem int) int {
	panic("syscall/dart needs the gart compiler")
}

// ChanOf gives the handle of the channel type of a direction and an element
// type. The direction is 3 for both, 2 for send, and 1 for receive.
func ChanOf(dir, elem int) int {
	panic("syscall/dart needs the gart compiler")
}

// MakeChan gives a channel of the type, with the size of the buffer.
func MakeChan(handle, buffer int) any {
	panic("syscall/dart needs the gart compiler")
}

// ValueSend writes 1 value into a channel that a Value holds. The call gives up
// control, exactly as a send of Go does.
func ValueSend(value, x any) {
	panic("syscall/dart needs the gart compiler")
}

// ValueClose closes a channel that a Value holds.
func ValueClose(value any) {
	panic("syscall/dart needs the gart compiler")
}

// ValueTryRecv reads 1 value of a channel and gives up no control. The 3
// results are the value, the flag of the receive, and the flag of a valid
// Value.
func ValueTryRecv(value any) (any, bool, bool) {
	panic("syscall/dart needs the gart compiler")
}

// ValueTrySend writes 1 value into a channel and gives up no control. The flag
// is false when the send cannot end at this moment.
func ValueTrySend(value, x any) bool {
	panic("syscall/dart needs the gart compiler")
}

// SelectValue runs a select statement of the package reflect. The direction of
// each case is 1 for a receive, 2 for a send, and 3 for the default. The 3
// results are the number of the case, the value of a receive, and the flag of
// that receive.
func SelectValue(dirs []int, chans, sends []any) (int, any, bool) {
	panic("syscall/dart needs the gart compiler")
}

// ValueClear drops each entry of a map, or writes the zero value into each
// element of a slice.
func ValueClear(value any) {
	panic("syscall/dart needs the gart compiler")
}

// DescMethodType gives a handle of the function type of 1 reflected method of a
// type. The order follows the name, and the type holds no receiver.
func DescMethodType(handle, index int) int {
	panic("syscall/dart needs the gart compiler")
}

// DescMethodFunc gives the func value of 1 exported method of a type, with the
// receiver in front of each parameter. A type of an interface gives nil.
func DescMethodFunc(handle, index int) any {
	panic("syscall/dart needs the gart compiler")
}

// EmbeddedFieldError gives the error for an invalid reflected embedded field.
func EmbeddedFieldError(handle, index, count int) string {
	panic("syscall/dart needs the gart compiler")
}

// StructOf gives a handle of the struct type that a list of fields builds.
// quotedTags holds a quoted Go literal for each nonempty tag.
func StructOf(names, packages, tags, quotedTags []string, types []int, anon []bool) int {
	panic("syscall/dart needs the gart compiler")
}

// MakeFunc gives a func value of a function type. Each call of it runs fn with
// the arguments, and the results of fn become the results of the call.
func MakeFunc(handle int, fn func(args []any) []any) any {
	panic("syscall/dart needs the gart compiler")
}

// CallSliceValue calls a variadic function value and takes the last argument as
// the slice of the variadic parameter.
func CallSliceValue(fn any, args []any) []any {
	panic("syscall/dart needs the gart compiler")
}

// ValueCap gives the capacity of a slice, of an array, or of a channel.
func ValueCap(value any) int {
	panic("syscall/dart needs the gart compiler")
}

// AppendValue appends each value to a slice and gives the new slice.
func AppendValue(slice any, items []any) any {
	panic("syscall/dart needs the gart compiler")
}

// AppendSliceValue appends each element of the second slice to the first one.
func AppendSliceValue(slice, add any) any {
	panic("syscall/dart needs the gart compiler")
}

// CopyValue copies each element of the source into the destination.
func CopyValue(dst, src any) int {
	panic("syscall/dart needs the gart compiler")
}

func MakeMap(handle int) any {
	panic("syscall/dart needs the gart compiler")
}

// MapSet writes 1 entry of a map. The flag drops the entry.
func MapSet(target, key, value any, drop bool) {
	panic("syscall/dart needs the gart compiler")
}

// DescLength gives the count of the elements of an array type.
func DescLength(handle int) int {
	panic("syscall/dart needs the gart compiler")
}

func DescSize(handle int) uintptr {
	panic("syscall/dart needs the gart compiler")
}

func DescAlign(handle int) int {
	panic("syscall/dart needs the gart compiler")
}

func DescOffset(handle, index int) uintptr {
	panic("syscall/dart needs the gart compiler")
}

// FieldCount gives the count of the fields of a struct type.
func FieldCount(handle int) int {
	panic("syscall/dart needs the gart compiler")
}

// FieldPkgPath gives the import path of the package that declares an
// unexported field. An exported field gives an empty text.
//
// An unnamed struct type holds no package of its own, thus the path of the
// type of the struct gives no answer for such a field.
func FieldPkgPath(handle, index int) string {
	panic("syscall/dart needs the gart compiler")
}

// FieldInfo gives the name, the tag, the exported flag, and the type handle of
// 1 field of a struct type.
func FieldInfo(handle, index int) (string, string, bool, int) {
	panic("syscall/dart needs the gart compiler")
}

// CopyInterface detaches a struct or array from a reflected location.
func CopyInterface(value any) any {
	panic("syscall/dart needs the gart compiler")
}

// FieldRead gives the value of 1 field of a struct value.
func FieldRead(value any, index int) any {
	panic("syscall/dart needs the gart compiler")
}

// FieldWrite writes 1 field of a struct value.
func FieldWrite(value any, index int, x any) {
	panic("syscall/dart needs the gart compiler")
}

// ValueLen gives the length of a slice, an array, a string, or a map.
func ValueLen(value any) int {
	panic("syscall/dart needs the gart compiler")
}

// ValueIndex gives 1 element of a slice, an array, or a string.
func ValueIndex(value any, index int) any {
	panic("syscall/dart needs the gart compiler")
}

// ValueSetIndex writes 1 element of a slice or an array.
func ValueSetIndex(value any, index int, x any) {
	panic("syscall/dart needs the gart compiler")
}

// AsInt gives the whole number inside a value of an integer kind.
func AsInt(value any) int64 {
	panic("syscall/dart needs the gart compiler")
}

// AsFloat gives the float inside a value of a float kind.
func AsFloat(value any) float64 {
	panic("syscall/dart needs the gart compiler")
}

// AsBool gives the flag inside a value of the kind bool.
func AsBool(value any) bool {
	panic("syscall/dart needs the gart compiler")
}

// AsString gives the text inside a value of the kind string.
func AsString(value any) string {
	panic("syscall/dart needs the gart compiler")
}

// WithInt gives a value of the type, with the whole number inside it.
func WithInt(handle int, x int64) any {
	panic("syscall/dart needs the gart compiler")
}

// WithFloat gives a value of the type, with the float inside it.
func WithFloat(handle int, x float64) any {
	panic("syscall/dart needs the gart compiler")
}

// WithBool gives a value of the type, with the flag inside it.
func WithBool(handle int, x bool) any {
	panic("syscall/dart needs the gart compiler")
}

// WithString gives a value of the type, with the text inside it.
func WithString(handle int, x string) any {
	panic("syscall/dart needs the gart compiler")
}

// NewValue gives a pointer to a fresh zero value of the type.
func NewValue(handle int) any {
	panic("syscall/dart needs the gart compiler")
}

// MakeValue gives a slice of the type, with the length and the capacity.
func MakeValue(handle, length, capacity int) any {
	panic("syscall/dart needs the gart compiler")
}

// DescElem gives a handle of the element type of a type. A type that holds no
// element type gives 0.
func DescElem(handle int) int {
	panic("syscall/dart needs the gart compiler")
}

// DescZero gives the zero value of a type, inside an interface value.
func DescZero(handle int) any {
	panic("syscall/dart needs the gart compiler")
}

// PointerElem gives the value behind a pointer, inside an interface value.
func PointerElem(pointer any) any {
	panic("syscall/dart needs the gart compiler")
}

// PointerSet writes a value behind a pointer.
func PointerSet(pointer, value any) {
	panic("syscall/dart needs the gart compiler")
}

// AssignError writes a value into the target of errors.As. It reports false
// when the type of the value does not hold the element type of the target.
//
// The target holds a pointer, and Go writes such a target through reflect.
func AssignError(value any, target any) bool {
	panic("syscall/dart needs the gart compiler")
}

// CheckErrorTarget stops the program when the target of errors.As holds no
// pointer, holds a nil pointer, or points to a type that is neither an
// interface type nor a type of the error interface.
//
// Go reads the type of the target with reflect, and it writes the same 2
// texts.
func CheckErrorTarget(target any) {
	panic("syscall/dart needs the gart compiler")
}

// ZeroText gives the text of the String method of the zero value of the type of
// a value. An empty text means that the type holds no such method.
//
// The package flag reads it. Go builds the zero value through reflect.New, and
// the type descriptor of the runtime holds the same factory.
func ZeroText(value any) string {
	panic("syscall/dart needs the gart compiler")
}

// OS names the operating system that runs the program, in the words of the Go
// constant GOOS.
func OS() string {
	panic("syscall/dart needs the gart compiler")
}

// Upper gives the text in upper case, with the rules of Unicode.
func Upper(text string) string {
	panic("syscall/dart needs the gart compiler")
}

// Lower gives the text in lower case, with the rules of Unicode.
func Lower(text string) string {
	panic("syscall/dart needs the gart compiler")
}

// ParseFloat reads a float from text. The second result is false when the text
// holds no float.
func ParseFloat(text string) (float64, bool) {
	panic("syscall/dart needs the gart compiler")
}

// ParseFloat32 reads a 32-bit float from text, with the single rounding of Go.
// The second result is false when the text holds no float.
//
// A read that gives a float64 first and narrows second rounds 2 times, thus a
// decimal that stands exactly between 2 values of float32 takes the wrong
// neighbour.
func ParseFloat32(text string) (float32, bool) {
	panic("syscall/dart needs the gart compiler")
}

// Sleep gives up control for a count of nanoseconds. The call blocks the
// goroutine, thus the caller becomes an async function.
func Sleep(nanoseconds int64) {
	panic("syscall/dart needs the gart compiler")
}

// NowNanos gives the count of nanoseconds since the start of the Unix epoch.
func NowNanos() int64 {
	panic("syscall/dart needs the gart compiler")
}

// ZoneOffset gives the local zone at an instant of the Unix epoch.
//
// The argument is a count of seconds. The 2 values are the offset in seconds
// east of UTC and the name of the zone.
//
// The argument holds seconds, because a Time of the package time reaches the
// year 1. The count of nanoseconds of that instant does not fit an int64.
func ZoneOffset(sec int64) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// IntText gives the decimal text of a signed value.
//
// The portable Go form builds the text 1 digit at a time and it joins each
// digit to the front, thus it allocates 1 string for each digit.
func IntText(v int64) string {
	panic("syscall/dart needs the gart compiler")
}

// UintText gives the decimal text of an unsigned value.
func UintText(v uint64) string {
	panic("syscall/dart needs the gart compiler")
}

// IntRadix gives the text of a signed value in a base of 2 to 36. The digits
// are 0 to 9 and a to z, exactly as strconv writes them.
func IntRadix(v int64, base int) string {
	panic("syscall/dart needs the gart compiler")
}

// UintRadix gives the text of an unsigned value in a base of 2 to 36.
func UintRadix(v uint64, base int) string {
	panic("syscall/dart needs the gart compiler")
}

// Sqrt gives the square root of a value, with the rounding of IEEE 754.
//
// The portable Go form builds the result bit by bit in a loop of 54 steps.
// The compiler of Go replaces the call with 1 machine instruction, and gart
// holds no such rule, thus the interop gives the same instruction.
func Sqrt(v float64) float64 {
	panic("syscall/dart needs the gart compiler")
}

// Floor gives the largest whole value that is not above the argument.
func Floor(v float64) float64 {
	panic("syscall/dart needs the gart compiler")
}

// Ceil gives the smallest whole value that is not below the argument.
func Ceil(v float64) float64 {
	panic("syscall/dart needs the gart compiler")
}

// Trunc gives the whole part of a value, toward zero.
func Trunc(v float64) float64 {
	panic("syscall/dart needs the gart compiler")
}

// Abs gives the value with no sign.
func Abs(v float64) float64 {
	panic("syscall/dart needs the gart compiler")
}

// Float64Bits gives the IEEE 754 bit pattern of a 64-bit value.
func Float64Bits(v float64) uint64 {
	panic("syscall/dart needs the gart compiler")
}

// Float64FromBits gives the 64-bit value of an IEEE 754 bit pattern.
func Float64FromBits(b uint64) float64 {
	panic("syscall/dart needs the gart compiler")
}

// Float32Bits gives the IEEE 754 bit pattern of a 32-bit value.
func Float32Bits(v float32) uint32 {
	panic("syscall/dart needs the gart compiler")
}

// Float32FromBits gives the 32-bit value of an IEEE 754 bit pattern.
func Float32FromBits(b uint32) float32 {
	panic("syscall/dart needs the gart compiler")
}

// JSONValid reports whether the bytes hold 1 valid JSON value. The scanner of
// the runtime holds each rule of encoding/json/jsontext, the depth limit of
// 10000 levels included.
func JSONValid(data []byte) bool {
	panic("syscall/dart needs the gart compiler")
}

// JSONUnmarshal decodes 1 JSON document into a target that the scanner of the
// runtime serves. The results are a code and 2 numbers.
//
// The code jsonFastDone reports that the runtime wrote the target.
// jsonFastGoPath asks the Go code of this package to run from the start, and
// the runtime then wrote nothing. jsonFastMethod names the bounds of the value
// in the 2 numbers, thus the caller calls UnmarshalJSON with those bytes.
// jsonFastMaxDepth names the offset of the bracket that broke the depth limit.
//
// The scanner walks the bytes with a stack of its own. The decoder of the Go
// code descends, and the stack of Dart holds about 1,400 levels of it.
func JSONUnmarshal(data []byte, target any) (int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

// RandomBytes gives n bytes of the secure source of the platform.
func RandomBytes(n int) []byte {
	panic("syscall/dart needs the gart compiler")
}

// MonoNanos gives the count of nanoseconds since the start of the program.
//
// The value comes from a monotonic clock. A change of the system time does not
// move it, and the value holds a finer step than the wall clock. A benchmark
// reads it.
func MonoNanos() int64 {
	panic("syscall/dart needs the gart compiler")
}

// Worker runs f in a new isolate and gives the result.
//
// The function must be a pure function of its input. It must not read a
// package variable, and it must not write one. In the Go build, the same API
// runs the function on a goroutine, thus the result stays equal.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func Worker(f func([]byte) []byte, input []byte) []byte {
	panic("syscall/dart needs the gart compiler")
}

// SliceLen gives the length of a slice of an unknown element type.
func SliceLen(slice any) int {
	panic("syscall/dart needs the gart compiler")
}

// SliceSwap exchanges 2 elements of a slice of an unknown element type.
func SliceSwap(slice any, i, j int) {
	panic("syscall/dart needs the gart compiler")
}

// Stop stops the call chain of the goroutine. Each deferred function of the
// chain still runs.
//
// The package testing uses it for FailNow. Go stops such a chain with
// runtime.Goexit.
func Stop() {
	panic("syscall/dart needs the gart compiler")
}

// Catch calls f and holds a stop signal.
//
// It reports false when f called Stop, and true in each other case. A panic
// passes through. The call gives up control, thus the caller becomes an async
// function.
func Catch(f func()) bool {
	panic("syscall/dart needs the gart compiler")
}

// CatchArg calls f with arg and holds a stop signal, as Catch does.
func CatchArg[T any](f func(T), arg T) bool {
	panic("syscall/dart needs the gart compiler")
}

// Exit ends the program with a status code.
func Exit(code int) {
	panic("syscall/dart needs the gart compiler")
}

// Args gives the arguments of the program, without the name of the program.
func Args() []string {
	panic("syscall/dart needs the gart compiler")
}

// The mode of FileOpen.
const (
	// OpenRead opens a file for a read.
	OpenRead = 0
	// OpenWrite opens a file for a read and a write, and it replaces the
	// content. os.Create reads it, and Go gives such a file each mode.
	OpenWrite = 1
	// OpenAppend opens a file for a write at the end.
	OpenAppend = 2
	// OpenReadWrite opens a file for a read and a write, and it keeps the
	// content. The position stands at the start.
	OpenReadWrite = 3
)

// The whence of FileSeek.
const (
	SeekStart   = 0
	SeekCurrent = 1
	SeekEnd     = 2
)

// FileOpen opens a file and gives its handle. The second result holds the cause
// of a failure, and it is empty after a success.
//
// Each function below gives such a cause. An empty text means a success.
func FileOpen(path string, mode int) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

func FileOpenFlags(path string, flags, mode int) (int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

func FileFd(handle, std int) int {
	panic("syscall/dart needs the gart compiler")
}

func FilePipe() (int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

func FileFromFD(fd int) (int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

func DirectoryOpen(path string) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func DirectoryFd(fd int) int {
	panic("syscall/dart needs the gart compiler")
}

func DirectoryChdir(fd int) int {
	panic("syscall/dart needs the gart compiler")
}

func DirectoryClose(fd int) int {
	panic("syscall/dart needs the gart compiler")
}

// FileRead reads bytes into the slice and gives the count. A count of 0 with no
// cause means the end of the file. Native pipe I/O can suspend the goroutine.
// The last result identifies actual deadline expiry.
func FileRead(handle int, p []byte) (int, string, int, bool) {
	panic("syscall/dart needs the gart compiler")
}

// FileWrite writes the bytes of the slice and gives the count.
// Native pipe I/O can suspend the goroutine.
// The last result identifies actual deadline expiry.
func FileWrite(handle int, p []byte) (int, string, int, bool) {
	panic("syscall/dart needs the gart compiler")
}

// FileDeadline sets a relative pipe timeout in nanoseconds. Clear cancels it.
// Mode 1 sets read, mode 2 sets write, and mode 3 sets both.
func FileDeadline(handle int, timeout int64, mode int, clear bool) bool {
	panic("syscall/dart needs the gart compiler")
}

// FileSeek moves the read position and gives the new offset from the start.
func FileSeek(handle int, offset int64, whence int) (int64, string, int) {
	panic("syscall/dart needs the gart compiler")
}

// FileRawPollable identifies native files that support readiness and deadlines.
func FileRawPollable(handle int) bool {
	panic("syscall/dart needs the gart compiler")
}

// FileRawPrepare reports closed or expired native poll state before a callback.
// Kind 1 means closed, 2 means expired, and 3 means unavailable poll support.
func FileRawPrepare(handle, mode int) int {
	panic("syscall/dart needs the gart compiler")
}

// FileRawWait suspends until actual native readiness, close, or deadline expiry.
// The results contain native errno and the FileRawPrepare kind.
func FileRawWait(handle, mode int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// FileRawCancel cancels pending operations without a descriptor close.
// The file owner closes it after all active callbacks return.
func FileRawCancel(handle int) {
	panic("syscall/dart needs the gart compiler")
}

func FileClose(handle int) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

func NativeTruncate(name string, size int64) int {
	panic("syscall/dart needs the gart compiler")
}

func NativePread(fd int, p []byte, off int64) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func NativePwrite(fd int, p []byte, off int64) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func NativeFtruncate(fd int, size int64) int {
	panic("syscall/dart needs the gart compiler")
}

func NativeFsync(fd int) int {
	panic("syscall/dart needs the gart compiler")
}

// FileSync writes each held byte of a file to the storage.
func FileSync(handle int) string {
	panic("syscall/dart needs the gart compiler")
}

// FileTruncate cuts the file of the handle to the size. A size above the
// length of the file grows it with zero bytes.
func FileTruncate(handle int, size int64) string {
	panic("syscall/dart needs the gart compiler")
}

// ReadFile reads a whole file.
func ReadFile(path string) ([]byte, string) {
	panic("syscall/dart needs the gart compiler")
}

// Root operations use native directory descriptors and return errno.
func RootOpen(path string) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func RootOpenAt(parent int, path string) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func RootOpenDir(parent int, path string) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func RootOpenFile(parent int, path string, flags, mode int) (int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

func RootMkdir(parent int, path string, mode int) int {
	panic("syscall/dart needs the gart compiler")
}

func RootReadlink(parent int, path string) (string, int) {
	panic("syscall/dart needs the gart compiler")
}

func RootChmod(parent int, path string, mode int) int {
	panic("syscall/dart needs the gart compiler")
}

func RootChown(parent int, path string, uid, gid int) int {
	panic("syscall/dart needs the gart compiler")
}

func RootChtimes(parent int, path string, atime int64, setAtime bool, mtime int64, setMtime bool) int {
	panic("syscall/dart needs the gart compiler")
}

func RootLink(oldParent int, oldname string, newParent int, newname string) int {
	panic("syscall/dart needs the gart compiler")
}

func RootRename(oldParent int, oldname string, newParent int, newname string) int {
	panic("syscall/dart needs the gart compiler")
}

func RootSymlink(target string, parent int, path string) int {
	panic("syscall/dart needs the gart compiler")
}

func RootRemove(parent int, path string) int {
	panic("syscall/dart needs the gart compiler")
}

func RootRemoveAll(parent int, path string) int {
	panic("syscall/dart needs the gart compiler")
}

func HasNativeStat() bool { panic("syscall/dart needs the gart compiler") }

// NativeStat returns metadata and errno. The metadata contains size, mode,
// mtime seconds/nanoseconds, device, inode, link count, uid, gid, rdev,
// block size/count, atime seconds/nanoseconds, ctime seconds/nanoseconds,
// birthtime seconds/nanoseconds, flags, and generation.
func NativeStat(path string, follow bool) ([]int64, int) {
	panic("syscall/dart needs the gart compiler")
}

// NativeLstatat returns the NativeStat fields relative to a directory descriptor.
func NativeLstatat(fd int, path string) ([]int64, int) {
	panic("syscall/dart needs the gart compiler")
}

// NativeFstatat returns the NativeStat fields relative to a directory descriptor.
// A false output passes a nil metadata pointer to the native call.
func NativeFstatat(fd int, path string, output bool, flags int) ([]int64, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallFcntlFlock transports all fields of the native record-lock structure.
// A false input passes a nil pointer. Other pointer commands remain unavailable.
func SyscallFcntlFlock(fd uintptr, cmd int, input bool, start, length int64, pid int32, kind, whence int16, pad0, pad1 uint32) ([]int64, int) {
	panic("syscall/dart needs the gart compiler")
}

// NativeFstat returns the NativeStat fields for an open descriptor.
func NativeFstat(fd int) ([]int64, int) {
	panic("syscall/dart needs the gart compiler")
}

// Stat reads the metadata of a path. The 3 values are the size in bytes, the
// directory flag, and the time of the last change in nanoseconds.
func Stat(path string) (int64, bool, int64, string) {
	panic("syscall/dart needs the gart compiler")
}

// Lstat reads the metadata of a path, and it follows no symbolic link. The 4
// values are the size in bytes, the directory flag, the link flag, and the time
// of the last change in nanoseconds. The size of a link is the length of its
// target, and the time of a link is 0.
func Lstat(path string) (int64, bool, bool, int64, string) {
	panic("syscall/dart needs the gart compiler")
}

// Symlink makes a symbolic link at the path, and the link points at the target.
func Symlink(target, link string) string {
	panic("syscall/dart needs the gart compiler")
}

// Link makes a hard link at newname, and it names the content of oldname.
// dart:io holds no such call, thus the Dart side runs the command ln of the
// platform.
func Link(oldname, newname string) string {
	panic("syscall/dart needs the gart compiler")
}

// Umask gives the file mode creation mask of the process. A create of Go drops
// each bit of the mask from the mode of the call.
func Umask() int

// FileChmod changes the mode through an open descriptor and returns errno.
func FileChmod(fd, mode int) int { panic("syscall/dart needs the gart compiler") }

// FileChown changes ownership through an open descriptor and returns errno.
func FileChown(fd, uid, gid int) int { panic("syscall/dart needs the gart compiler") }

// NativeChmod changes a native path's mode and returns errno.
func NativeChmod(path string, mode int) int {
	panic("syscall/dart needs the gart compiler")
}

// NativeExecutableAccess tests execute access with effective user and group IDs.
// Call only when HasNativeStat reports native support. The result is errno.
func NativeExecutableAccess(path string) int {
	panic("syscall/dart needs the gart compiler")
}

// NativeRename moves a native path and returns errno.
func NativeRename(from, to string) int {
	panic("syscall/dart needs the gart compiler")
}

// NativeChown changes native ownership and returns errno. Follow selects the symlink target.
func NativeChown(path string, uid, gid int, follow bool) int {
	panic("syscall/dart needs the gart compiler")
}

// Chmod writes the permission bits of a path. The mode holds the 12 bits of a
// POSIX mode. dart:io holds no such call, thus the Dart side runs the command
// chmod of the platform.
func Chmod(path string, mode int) string {
	panic("syscall/dart needs the gart compiler")
}

// Chown writes the owner and the group of a path. A number below 0 leaves that
// 1 of the 2 alone, and follow false takes a symbolic link itself. dart:io
// holds no such call, thus the Dart side runs the command chown of the
// platform.
func Chown(path string, uid, gid int, follow bool) string {
	panic("syscall/dart needs the gart compiler")
}

// Chtimes writes the time of the last read and the time of the last change, in
// nanoseconds since the start of the Unix epoch. A flag that is false leaves
// that 1 of the 2 alone.
//
// dart:io holds the millisecond, thus the call drops the finer part.
func Chtimes(path string, atime int64, setAtime bool, mtime int64, setMtime bool) string {
	panic("syscall/dart needs the gart compiler")
}

// NativeChtimes updates native timestamps and returns errno.
func NativeChtimes(path string, atime int64, setAtime bool, mtime int64, setMtime bool) int {
	panic("syscall/dart needs the gart compiler")
}

// NativeReadlink returns the raw link target and errno.
func NativeReadlink(path string) (string, int) {
	panic("syscall/dart needs the gart compiler")
}

// ReadLink gives the target of 1 symbolic link. A path that holds no link
// gives a cause.
func ReadLink(path string) (string, string) {
	panic("syscall/dart needs the gart compiler")
}

// ResolveSymlinks gives the path after each symbolic link, from the root. The
// path must exist.
func ResolveSymlinks(path string) (string, string) {
	panic("syscall/dart needs the gart compiler")
}

func Remove(path string) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// RemoveAll removes a path and each path inside it.
func RemoveAll(path string) (string, string, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// DirectoryCreate creates a directory with native permission bits and returns errno.
func DirectoryCreate(path string, mode int) int { panic("syscall/dart needs the gart compiler") }

// Mkdir creates a directory through the host file API.
func Mkdir(path string) string {
	panic("syscall/dart needs the gart compiler")
}

func Rename(from, to string) string {
	panic("syscall/dart needs the gart compiler")
}

func FileIdentical(first, second string) (bool, string) {
	panic("syscall/dart needs the gart compiler")
}

// ReadDir reads the names inside a directory. Each name holds no directory
// part, and the list is in the order of the name.
func ReadDir(path string) ([]string, string) {
	panic("syscall/dart needs the gart compiler")
}

// SetRuntimeGODEBUG updates the runtime controls that Go implements.
func SetRuntimeGODEBUG(settings string) {
	panic("syscall/dart needs the gart compiler")
}

func RuntimeGODEBUGVersion() int {
	panic("syscall/dart needs the gart compiler")
}

func DefaultGODEBUG() string {
	panic("syscall/dart needs the gart compiler")
}

func RegisterGODEBUG() {
	panic("syscall/dart needs the gart compiler")
}

func IncGODEBUG(name string) {
	panic("syscall/dart needs the gart compiler")
}

func GODEBUGCount(name string) uint64 {
	panic("syscall/dart needs the gart compiler")
}

// Getenv reads 1 variable of the process-start environment.
func Getenv(name string) (string, bool) {
	panic("syscall/dart needs the gart compiler")
}

// Environ reads each variable of the environment, in the form name=value.
func Environ() []string {
	panic("syscall/dart needs the gart compiler")
}

// Getpid gives the process id of the program.
func Getpid() int {
	panic("syscall/dart needs the gart compiler")
}

// Hostname gives the name of the host. The second result holds the cause of a
// failure, and it is empty after a success.
func Hostname() (string, string) {
	panic("syscall/dart needs the gart compiler")
}

// Executable gives the path of the program that runs. The second result holds
// the cause of a failure.
func Executable() (string, string) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGetwd() (string, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGetcwd(buf []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallUtimensat(dirfd int, path string, atimeSec, atimeNsec, mtimeSec, mtimeNsec int64) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallUtimes(path string, atimeSec, atimeUsec, mtimeSec, mtimeUsec int64) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFutimesat(dirfd int, path string, atimeSec, atimeUsec, mtimeSec, mtimeUsec int64) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFutimes(fd int, atimeSec, atimeUsec, mtimeSec, mtimeUsec int64) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallUtime(path string, hasTimes bool, atime, mtime int64) int {
	panic("syscall/dart needs the gart compiler")
}

func NativeGetwd() (string, int) {
	panic("syscall/dart needs the gart compiler")
}

func NativeChdir(path string) int {
	panic("syscall/dart needs the gart compiler")
}

func Getwd() (string, string) {
	panic("syscall/dart needs the gart compiler")
}

func Chdir(path string) string {
	panic("syscall/dart needs the gart compiler")
}

// TempDir gives the directory that the system holds for a temporary file.
func TempDir() string {
	panic("syscall/dart needs the gart compiler")
}

// PathSeparator gives the character that separates 2 parts of a path.
func PathSeparator() string {
	panic("syscall/dart needs the gart compiler")
}

// Stderr writes text to standard error.
func Stderr(text string) {
	panic("syscall/dart needs the gart compiler")
}

// StdinRead reads bytes from standard input. A count of 0 means the end of the
// input.
func StdinRead(p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// DialTCP returns a connection handle, error text, native error code, and syscall name.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func DialTCP(host string, port int, localHost string, localPort int) (int, string, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ConnAddrs gives the local address, the local port, the peer address, and
// the peer port of a connection.
func ConnAddrs(handle int) (string, int, string, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ConnRead reads bytes of a connection into the slice and gives the count.
// A non-empty read with no bytes and no error means the end of the stream.
// The last result preserves an available native error code.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ConnRead(handle int, p []byte) (int, string, int) {
	panic("syscall/dart needs the gart compiler")
}

// ConnWrite writes the bytes of the slice to a connection and gives the count.
// The last result preserves an available native error code.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ConnWrite(handle int, p []byte) (int, string, int) {
	panic("syscall/dart needs the gart compiler")
}

// ConnClose closes a connection in both directions.
func ConnClose(handle int) string {
	panic("syscall/dart needs the gart compiler")
}

// ListenTCP gives the handle, port, and failure cause.
// A port of 0 asks the system for a free port.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ListenTCP(host string, port int, ipv6Only bool) (int, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ListenTCPHost gives the bound IP address of an open TCP listener.
func ListenTCPHost(handle int) string {
	panic("syscall/dart needs the gart compiler")
}

// ListenRawAcquire retains the native listener descriptor for a Go callback.
// The caller must release it after the callback returns or panics.
func ListenRawAcquire(handle int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func ListenRawRelease(descriptor int) {
	panic("syscall/dart needs the gart compiler")
}

// ListenFile duplicates the native listener descriptor. The caller owns the copy.
func ListenFile(handle int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// FileListener duplicates a TCP listener descriptor and gives its handle, address,
// port, native error code and operation. The caller retains source ownership.
func FileListener(fd int) (int, string, int, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ListenTLS opens a TLS listener on a host and a port. The certificate chain
// and the key each hold the PEM form, and alpn holds each protocol that the
// server offers. The 3 results are the handle, the port that the system took,
// and the cause of a failure.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ListenTLS(host string, port int, cert, key []byte, alpn []string) (int, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// AcceptTLS waits for 1 connection of a TLS listener. The 3 results are the
// handle of the connection, the protocol that ALPN took, and the cause of a
// failure.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func AcceptTLS(handle int) (int, string, string) {
	panic("syscall/dart needs the gart compiler")
}

// Accept waits for 1 connection of a listener and gives its handle.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func Accept(handle int) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ListenDeadline sets the accept deadline in Unix nanoseconds. Zero removes it.
func ListenDeadline(handle int, nanos int64) string {
	panic("syscall/dart needs the gart compiler")
}

// ListenClose closes a listener. Each connection of it stays open. A listener
// of a Unix socket also removes the file of the socket.
// The call blocks the goroutine until the close completes.
func ListenClose(handle int) string {
	panic("syscall/dart needs the gart compiler")
}

// DialUnix opens a connection to a Unix socket and gives its handle. The
// handle belongs to the table of the connections.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func DialUnix(path string) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ListenUnix opens a listener on a Unix socket and gives its handle.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ListenUnix(path string) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ConnDeadline writes the 2 deadlines of a connection, in nanoseconds after
// the start of the Unix epoch. A value of 0 holds no limit.
func ConnDeadline(handle int, read, write int64) string {
	panic("syscall/dart needs the gart compiler")
}

// ListenUDP opens a datagram socket on a host and a port. The results are the
// handle, the port that the system took, the cause of a failure, and its errno.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ListenUDP(host string, port int) (int, int, string, int) {
	panic("syscall/dart needs the gart compiler")
}

// ListenMulticastUDP opens a datagram socket that shares its port with another
// socket of the system. The results are the handle, the port that the system
// took, the failure text, and the native error code.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ListenMulticastUDP(host string, port int) (int, int, string, int) {
	panic("syscall/dart needs the gart compiler")
}

// JoinMulticast joins a multicast group on a datagram socket.
// If hasInterface is false, the system selects the interface.
// The results are the failure text and the native error code.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func JoinMulticast(handle int, group string, index int, hasInterface bool) (string, int) {
	panic("syscall/dart needs the gart compiler")
}

// ConnectUDP connects a bound datagram socket. The results are the local
// address, remote address, failure text, and native error code.
// The caller must close the socket if this function returns an error.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ConnectUDP(handle int, host string, port int) (string, string, string, int) {
	panic("syscall/dart needs the gart compiler")
}

// PacketRead reads 1 datagram into the slice. The results are the count, the
// host of the source, the port of the source, and the cause of a failure. A
// datagram that holds more bytes than the slice loses the rest.
// The last result preserves an available native error code.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func PacketRead(handle int, p []byte) (int, string, int, string, int) {
	panic("syscall/dart needs the gart compiler")
}

// PacketWrite writes the bytes of the slice to a host and a port, and gives
// the count. The last result preserves an available native error code.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func PacketWrite(handle int, p []byte, host string, port int) (int, string, int) {
	panic("syscall/dart needs the gart compiler")
}

// PacketClose waits until the host releases the datagram socket.
func PacketClose(handle int) string {
	panic("syscall/dart needs the gart compiler")
}

// PacketDeadline writes the 2 deadlines of a datagram socket.
func PacketDeadline(handle int, read, write int64) string {
	panic("syscall/dart needs the gart compiler")
}

func InterfaceMulticastAddrs(index int) ([]string, string) {
	panic("syscall/dart needs the gart compiler")
}

func InterfaceTable() ([]string, string) {
	panic("syscall/dart needs the gart compiler")
}

// Interfaces gives the network interfaces of the host.
//
// The first result holds 1 item for each interface, in the form
// "<index>|<name>". The second result holds 1 item for each address of an
// interface, in the form "<index>|<address>/<prefix>". The index of an
// address names the interface that holds it. A name and an address each hold
// no bar, thus the first bar of an item ends the index. The third result
// holds the cause of a failure.
//
// A web target holds no such API. The web form of the runtime gives 2 empty
// lists and no cause, exactly as Go does under GOOS=js.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func Interfaces() ([]string, []string, string) {
	panic("syscall/dart needs the gart compiler")
}

// FetchOpen allocates a request handle before the request can block.
func FetchOpen() int {
	panic("syscall/dart needs the gart compiler")
}

// FetchCancel aborts a request or its response body. FetchClose frees the handle.
func FetchCancel(handle int) {
	panic("syscall/dart needs the gart compiler")
}

// FetchStart sends a request through the fetch API of the host, and it waits
// for the head of the answer. Each pair of items of header holds the name of a
// field and its value, and the third result holds the same shape.
//
// Call FetchOpen first. Call FetchClose when the request ends.
//
// The results are the status code, status text, header, response address,
// and failure cause. The address is empty unless a redirect moved the answer.
//
// deadline ends the wait, in nanoseconds after the start of the Unix epoch. A
// value of 0 holds no limit, and the same limit covers each read of the body.
//
// A target that holds no fetch API gives the cause "operation not supported".
//
// The call blocks the goroutine, thus the caller becomes an async function.
func FetchStart(handle int, method, url string, header []string, body []byte, deadline int64) (int, string, []string, string, string) {
	panic("syscall/dart needs the gart compiler")
}

// FetchRead reads bytes of the body of an answer into the slice and gives the
// count. A count of 0 with no cause means the end of the body.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func FetchRead(handle int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// FetchClose frees a request handle and cancels its response reader.
func FetchClose(handle int) string {
	panic("syscall/dart needs the gart compiler")
}

// StartTimer starts a timer that runs f after a count of nanoseconds, and
// gives the handle of the timer. Go runs f on a new goroutine, thus f can
// block.
func StartTimer(nanoseconds int64, f func()) int {
	panic("syscall/dart needs the gart compiler")
}

// MarkTimerChannel gives a timer channel zero visible length and capacity.
// The channel must have a buffer capacity of 1.
func MarkTimerChannel(channel any) {
	panic("syscall/dart needs the gart compiler")
}

// StopTimer cancels a timer. It reports false when the timer ran already.
//
// A timer of a synctest bubble that a goroutine outside each bubble reaches
// ends the program, exactly as Go does.
func StopTimer(handle int) bool {
	panic("syscall/dart needs the gart compiler")
}

// ResetTimer cancels a timer for a restart. It reports false when the timer
// ran already.
//
// The result is equal to the result of StopTimer. Go names the operation in
// the report of a timer of a synctest bubble, thus the report of this function
// names a reset.
func ResetTimer(handle int) bool {
	panic("syscall/dart needs the gart compiler")
}

// LockToken gives a token for 1 wait on a lock.
func LockToken() int {
	panic("syscall/dart needs the gart compiler")
}

// LockWait gives up control until LockWake runs with the same token.
//
// The wait is not durable. A synctest bubble that holds such a goroutine thus
// stands still, and its clock does not move. Go gives a lock the same rule.
func LockWait(token int) {
	panic("syscall/dart needs the gart compiler")
}

// LockWake ends the wait of the token. It performs no operation for a token
// that no goroutine waits on.
func LockWake(token int) {
	panic("syscall/dart needs the gart compiler")
}

// InBubble reports whether the goroutine of the caller stands inside a
// synctest bubble.
func InBubble() bool {
	panic("syscall/dart needs the gart compiler")
}

// BubbleRun runs f on the first goroutine of a new synctest bubble, and it
// waits for the end of each goroutine of the bubble.
//
// The bubble holds a clock of its own. That clock moves only when each
// goroutine of the bubble is durably blocked, thus a sleep inside the bubble
// takes no real time.
//
// The result names the cause of a deadlock of the bubble, and it is empty
// after a normal end. The call blocks the goroutine, thus the caller becomes
// an async function.
func BubbleRun(f func()) string {
	panic("syscall/dart needs the gart compiler")
}

// BubbleWait waits until each other goroutine of the bubble of the caller is
// durably blocked.
//
// It reports false when another goroutine of the bubble waits already, and it
// then gives up no control. The call blocks the goroutine, thus the caller
// becomes an async function.
func BubbleWait() bool {
	panic("syscall/dart needs the gart compiler")
}

// ProcStart starts executable. argv holds the child name and arguments, dir
// holds the working directory, and env holds each variable in the form
// name=value. An empty dir keeps the directory of the program.
// env supplies the complete environment, including when the list is empty.
// The caller must reject NUL in all strings before this call.
// The results are the process id, the OS error code, and the error text.
// An error code of 0 means that Dart supplied no OS error code.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ProcStart(executable string, argv []string, dir string, env []string) (int, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ProcWrite writes the bytes of the slice to the standard input of a process.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ProcWrite(pid int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ProcCloseStdin closes the standard input of a process. A child that reads
// until the end of the input then sees that end. Repeated calls succeed.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ProcCloseStdin(pid int) string {
	panic("syscall/dart needs the gart compiler")
}

// ProcCloseIO closes the 2 output pipes of a process. A read that waits on such
// a pipe ends at once.
//
// A caller that gave up on the child then stays behind no orphan that holds the
// pipe.
func ProcCloseIO(pid int) string {
	panic("syscall/dart needs the gart compiler")
}

// ProcCloseOutput closes stdout, or stderr when errStream is true.
func ProcCloseOutput(pid int, errStream bool) string {
	panic("syscall/dart needs the gart compiler")
}

// ProcRead reads bytes of the standard output of a process. A count of 0 with
// no cause means the end of the stream.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ProcRead(pid int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ProcReadErr reads bytes of the standard error of a process. A count of 0
// with no cause means the end of the stream.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ProcReadErr(pid int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ProcWait waits for the end of a process and gives the exit code. A process
// that a signal ended gives the negative number of the signal.
// The second result reports that another wait already consumed the status.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ProcWait(pid int) (int, bool, string) {
	panic("syscall/dart needs the gart compiler")
}

// KillPid sends a signal to a process by its id.
func KillPid(pid int, number int) string {
	panic("syscall/dart needs the gart compiler")
}

// SignalPipeIgnore records the Go SIGPIPE disposition for native children.
// It does not change the Dart process disposition.
func SignalPipeIgnore(ignored bool) {
	panic("syscall/dart needs the gart compiler")
}

// SignalWatch starts a watch of 1 operating system signal and gives its
// handle. The number is the number of the system, for example 2 for SIGINT.
// A signal that the platform cannot catch gives a cause.
func SignalWatch(number int) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// SignalWait waits for the next event of a watch and gives the number of the
// signal. A number of 0 with no cause means that SignalStop closed the watch.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func SignalWait(id int) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// SignalStop stops a watch. The goroutine that waits on it wakes with a
// number of 0.
func SignalStop(id int) string {
	panic("syscall/dart needs the gart compiler")
}

// H2Client wraps an open TLS connection whose ALPN took h2, and gives the
// handle of the session. The socket leaves the table of the connections,
// thus the old handle answers no read and no write. The wrap works only
// while no read ran on the connection.
func H2Client(conn int) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// H2Fetch runs 1 request on a session of a client. The results are the
// handle of the stream of the body, the status, the flat response headers,
// and the cause of a failure. The flat list holds each name in front of its
// value.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func H2Fetch(session int, method, path, authority string, headers []string, body []byte, hasBody bool) (int, int, []string, string) {
	panic("syscall/dart needs the gart compiler")
}

// H2Read reads bytes of the body of a stream. A count of 0 with no cause
// means the end of the stream.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func H2Read(stream int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// H2Server wraps an accepted TLS connection whose ALPN took h2, and gives
// the handle of the session of the server.
func H2Server(conn int) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// H2Next waits for the next stream of a client. The results are the handle
// of the stream, the method, the path, the authority, the flat headers, and
// the cause. The end of the connection gives the cause EOF.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func H2Next(session int) (int, string, string, string, []string, string) {
	panic("syscall/dart needs the gart compiler")
}

// H2WriteHead writes the status and the fields of an answer. endStream
// marks an answer with no body.
func H2WriteHead(stream int, status int, headers []string, endStream bool) string {
	panic("syscall/dart needs the gart compiler")
}

// H2Write writes bytes of a body onto a stream.
func H2Write(stream int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// H2End ends the send side of a stream, and drops the stream.
func H2End(stream int) string {
	panic("syscall/dart needs the gart compiler")
}

// H2Close ends a session and its connection.
func H2Close(session int) string {
	panic("syscall/dart needs the gart compiler")
}

// WeakRef gives a weak reference of the Dart runtime to the value inside v.
// The reference holds no strong link, thus the collector of Dart frees the
// value when no other name reads it.
//
// A second call with the same value gives the same reference, thus 2 weak
// pointers of 1 value compare equal.
func WeakRef(v any) *Object {
	panic("syscall/dart needs the gart compiler")
}

// WeakTarget gives the value of r. It gives nil after the collector of Dart
// freed the value.
func WeakTarget(r *Object) any {
	panic("syscall/dart needs the gart compiler")
}

// An Object holds 1 object of the Dart host.
//
// The Dart form of this type is Object?, thus a value crosses the boundary of
// a //gart:dart declaration with no conversion. A Go program reads no field of
// such a value and calls no method of it. Only a Dart declaration gives 1, and
// only a Dart declaration reads 1.
//
// This file carries //go:build dart, thus a module that names this type also
// needs a twin file and an alias of its own.
type Object struct{ _ [0]func() }

func Getuid() int                { panic("syscall/dart needs the gart compiler") }
func Geteuid() int               { panic("syscall/dart needs the gart compiler") }
func Getgid() int                { panic("syscall/dart needs the gart compiler") }
func Getegid() int               { panic("syscall/dart needs the gart compiler") }
func Getppid() int               { panic("syscall/dart needs the gart compiler") }
func Getpgrp() int               { panic("syscall/dart needs the gart compiler") }
func Getpgid(pid int) (int, int) { panic("syscall/dart needs the gart compiler") }
func Getsid(pid int) (int, int)  { panic("syscall/dart needs the gart compiler") }
func NumCPU() int                { panic("syscall/dart needs the gart compiler") }

func Getpagesize() int { panic("syscall/dart needs the gart compiler") }

func Setpgid(pid, pgid int) int { panic("syscall/dart needs the gart compiler") }
func Setsid() (int, int)        { panic("syscall/dart needs the gart compiler") }

func Getpriority(which, who int) (int, int)    { panic("syscall/dart needs the gart compiler") }
func Setpriority(which, who, priority int) int { panic("syscall/dart needs the gart compiler") }

func Getgroups() ([]int, int) { panic("syscall/dart needs the gart compiler") }

func SendSignal(pid, signal int) int { panic("syscall/dart needs the gart compiler") }

// VerifySystemCertificates uses the platform trust policy for DER certificates.
// The date uses seconds from 2001-01-01 when hasDate is true.
func VerifySystemCertificates(certs []string, name string, date float64, hasDate bool) ([]string, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// GrowValue increases slice capacity after the caller validates its length.
func GrowValue(value any, count int) any { panic("syscall/dart needs the gart compiler") }

func NativeLink(from, to string) int {
	panic("syscall/dart needs the gart compiler")
}

func NativeSymlink(from, to string) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSocket(domain, typ, proto int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallOpen(path string, flags int, mode uint32) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallOpenat(dirfd int, path string, flags int, mode uint32) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSeek(fd int, offset int64, whence int) (int64, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallClose(fd int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallAccess(path string, mode uint32) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFchmodat(dirfd int, path string, mode uint32, flags int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallDup(fd int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallDup2(oldfd, newfd int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallDup3(oldfd, newfd, flags int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFsync(fd int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFdatasync(fd int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallMkfifo(path string, mode uint32) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallChflags(path string, flags int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFchflags(fd int, flags int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallPathconf(path string, name int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFpathconf(fd int, name int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallRouteRIB(facility, param int32) ([]byte, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSysctl(name string) (string, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSysctlUint32(name string) (uint32, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallStatfs(path string, output bool) ([]int64, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFstatfs(fd int, output bool) ([]int64, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallGetfsstat returns the native count, Statfs_t fields, and errno.
// Each record has 13 scalar fields, 2064 name bytes, and 8 reserved values.
func SyscallGetfsstat(count, flags int) (int, []int64, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGettimeofday(output bool) (int64, int64, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGetrlimit(resource int, output bool) (uint64, uint64, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSetrlimit(resource int, input bool, cur, max uint64) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallFcntl accepts the integer commands F_GETFD, F_SETFD, F_GETFL, and F_SETFL.
func SyscallFcntl(fd, cmd, arg int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallReadDirent(fd int, data []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallUnlink(path string, directory bool) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallReadlink(path string, buf []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallGetrusage returns native Rusage fields and errno.
// Fields contain user and system seconds/microseconds, then the SDK counter order.
func SyscallGetrusage(who int, output bool) ([]int64, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallUmask sets the process file creation mask and returns its previous value.
func SyscallUmask(mask int) int { panic("syscall/dart needs the gart compiler") }

func SyscallFaccessat(dirfd int, path string, mode uint32, flags int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFaccessatStat(dirfd int, path string, flags int) (uint32, uint32, uint32, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallCapDacOverride() bool {
	panic("syscall/dart needs the gart compiler")
}

func SyscallExchangedata(first, second string, options int) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallRevoke requires a path without NUL bytes.
func SyscallRevoke(path string) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallUndelete requires a path without NUL bytes.
func SyscallUndelete(path string) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallKevent requires complete 32-byte events with nil Udata and a nil or 16-byte timespec.
func SyscallKevent(kq int, changes, events, timeout []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallBpfIoctl accepts only the selected Darwin BPF requests and their complete byte objects.
// BIOCSETF uses a 16-byte program with a zero pointer and complete 8-byte instructions.
func SyscallBpfIoctl(fd, request int, data, instructions []byte) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallKqueue() (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGetdtablesize() int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallIssetugid() bool {
	panic("syscall/dart needs the gart compiler")
}

func SyscallDarwinMknod(path string, mode uint32, dev int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallMknodat(dirfd int, path string, mode uint32, dev int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallMkdirat(dirfd int, path string, mode uint32) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFchownat(dirfd int, path string, uid, gid, flags int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallUnlinkat(dirfd int, path string) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallRenameat(olddirfd int, oldpath string, newdirfd int, newpath string) int {
	panic("syscall/dart needs the gart compiler")
}

// BigmodUintSlice requires an element pointer with n available uint cells.
func BigmodUintSlice(ptr *uint, n int) []uint {
	panic("syscall/dart needs the gart compiler")
}

// SyscallUname returns the 6 native Utsname arrays and errno.
// A false output passes a null pointer to the native query.
func SyscallUname(output bool) ([]int64, int) { return nil, 0 }

// UnhashableError returns a runtime error for a non-nil, unhashable value.
func UnhashableError(value any) error {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGetxattr(path, attr string, dest []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallListxattr(path string, dest []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSetxattr(path, attr string, data []byte, flags int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallRemovexattr(path, attr string) int {
	panic("syscall/dart needs the gart compiler")
}

// UnixgramOpen creates an owned native datagram descriptor and reads its actual names.
func UnixgramOpen(local, remote string, bind, connect bool) (int, string, string, string, int) {
	panic("syscall/dart needs the gart compiler")
}
func UnixgramRead(handle int, data, control []byte) (int, int, int, string, string, int, bool) {
	panic("syscall/dart needs the gart compiler")
}
func UnixgramWrite(handle int, data, control []byte, destination string, sendTo bool) (int, int, string, int, bool) {
	panic("syscall/dart needs the gart compiler")
}
func UnixgramClose(handle int) (string, int) { panic("syscall/dart needs the gart compiler") }
func UnixgramDeadline(handle int, timeout int64, mode int, clear bool) (string, int) {
	panic("syscall/dart needs the gart compiler")
}
func UnixgramBuffer(handle int, read bool, size int) (string, int) {
	panic("syscall/dart needs the gart compiler")
}
func UnixgramShutdown(handle int, direction int) (string, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSendto(fd int, data []byte, flags int, destination []byte) int {
	panic("syscall/dart needs the gart compiler")
}

func NativeRead(fd int, p []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func NativeWrite(fd int, p []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallAttachLsf requires complete 8-byte Linux SockFilter entries.
func SyscallAttachLsf(fd int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSetsockoptString(fd, level, opt int, value string) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGetsockoptInt(fd, level, opt int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSetsockoptInt(fd, level, opt, value int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSocketpair(domain, typ, proto int) (int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGetsockoptByte(fd, level, opt int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGetsockoptUcred(fd, level, opt int) (int, int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallGetsockoptIPMreq requires an 8-byte output slice.
func SyscallGetsockoptIPMreq(fd, level, opt int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallSetsockoptIPMreq accepts a nil slice for a nil pointer. Other slices must contain 8 bytes.
func SyscallSetsockoptIPMreq(fd, level, opt int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallGetsockoptIPMreqn requires a 12-byte output slice.
func SyscallGetsockoptIPMreqn(fd, level, opt int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallSetsockoptIPMreqn accepts a nil slice for a nil pointer. Other slices must contain 12 bytes.
func SyscallSetsockoptIPMreqn(fd, level, opt int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallGetsockoptIPv6Mreq requires a 20-byte output slice.
func SyscallGetsockoptIPv6Mreq(fd, level, opt int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallSetsockoptIPv6Mreq accepts a nil slice for a nil pointer. Other slices must contain 20 bytes.
func SyscallSetsockoptIPv6Mreq(fd, level, opt int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallSetsockoptTimeval accepts a nil slice for a nil pointer. Other slices must contain 16 bytes.
func SyscallSetsockoptTimeval(fd, level, opt int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallGetsockoptICMPv6Filter requires a 32-byte output slice.
func SyscallGetsockoptICMPv6Filter(fd, level, opt int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallSetsockoptICMPv6Filter accepts a nil slice for a nil pointer. Other slices must contain 32 bytes.
func SyscallSetsockoptICMPv6Filter(fd, level, opt int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallMmap requires a positive length.
func SyscallMmap(fd int, offset int64, length int, prot, flags int) ([]byte, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallMadvise(data []byte, advice int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallMlock(data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallMunlock(data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallMprotect(data []byte, prot int) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallMunmap requires a nonempty slice with equal length and capacity.
func SyscallMunmap(data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallShutdown(fd, how int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFlock(fd, how int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallListen(fd, how int) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallSysinfo requires nil or a 112-byte native sysinfo structure.
func SyscallSysinfo(data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGettid() int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallTimes requires nil or a 32-byte native tms structure.
func SyscallTimes(data []byte) (int64, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallInotifyInit() (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallInotifyInit1(flags int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallInotifyAddWatch requires a path without NUL.
func SyscallInotifyAddWatch(fd int, path string, mask uint32) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallInotifyRmWatch(fd int, watchdesc uint32) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallEpollCreate1(flag int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallEpollCtl requires nil or 1 complete native epoll event.
func SyscallEpollCtl(epfd, op, fd int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallEpollWait requires count complete native epoll events.
func SyscallEpollWait(epfd int, data []byte, count, msec int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallPipe() (int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallPipe2(flags int) (int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallSplice requires nil offsets or exactly 8 native-endian bytes.
// Shared offset storage retains the native pointer alias.
func SyscallSplice(rfd int, read []byte, wfd int, write []byte, length, flags int) (int64, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallTee(rfd, wfd, length, flags int) (int64, int) {
	panic("syscall/dart needs the gart compiler")
}

func SyscallDarwinSendfile(outfd, infd int, offset int64, count int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallSendfile requires a nil offset or exactly 8 native-endian bytes.
func SyscallSendfile(outfd, infd int, offset []byte, count int) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallNanosleep requires nil or complete native 16-byte Timespec buffers.
func SyscallNanosleep(request, leftover []byte) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSync() {
	panic("syscall/dart needs the gart compiler")
}

func SyscallSyncFileRange(fd int, off, n int64, flags int) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallFallocate(fd int, mode uint32, off, length int64) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallSelect requires nil or full native FD sets and a nil or native timeval.
// Shared FD-set backing storage retains native pointer aliases.
func SyscallSelect(nfd int, read, write, except, timeout []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallBind and SyscallConnect require a complete native socket address.
func SyscallBind(fd int, address []byte) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallConnect(fd int, address []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallGetsockname and SyscallGetpeername require SizeofSockaddrAny bytes.
func SyscallGetsockname(fd int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

func SyscallGetpeername(fd int, data []byte) int {
	panic("syscall/dart needs the gart compiler")
}

// SyscallRecvfrom requires SizeofSockaddrAny bytes in address.
func SyscallRecvfrom(fd int, data []byte, flags int, address []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallAccept requires SizeofSockaddrAny bytes in address.
func SyscallAccept(fd int, flags int, address []byte) (int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallSendmsgN requires a complete native socket address or nil.
func SyscallSendmsgN(fd int, data, oob []byte, flags int, address []byte) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

// SyscallRecvmsg requires SizeofSockaddrAny bytes in address.
func SyscallRecvmsg(fd int, data, oob []byte, flags int, address []byte) (int, int, int, int) {
	panic("syscall/dart needs the gart compiler")
}

// BigMulAddVWW requires nonnil element pointers with n available uint cells.
// It reads each source word before the corresponding destination write.
func BigMulAddVWW(z, x *uint, n int, m, a uint) uint {
	panic("syscall/dart needs the gart compiler")
}

// BigAddMulVVWW requires nonnil element pointers with n available uint cells.
// It reads both source words before the corresponding destination write.
func BigAddMulVVWW(z, x, y *uint, n int, m, a uint) uint {
	panic("syscall/dart needs the gart compiler")
}
