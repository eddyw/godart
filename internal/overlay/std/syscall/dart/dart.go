//go:build dart

// Package dart gives Go code access to the Dart runtime.
//
// The compiler replaces each call to this package with a call to the runtime
// library, thus no body below ever runs.
package dart

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
// The result is empty on success, and it names the cause on a failure.
//
// The number of an option differs between the platforms, thus a platform that
// the table of the runtime holds not gives an error. A wrong number would
// write another option with no report.
func ConnOption(handle int, name string, value int) string {
	panic("syscall/dart needs the gart compiler")
}

// PacketOption writes a socket option of a datagram socket. The name holds
// `rcvbuf` or `sndbuf`. The result follows ConnOption.
func PacketOption(handle int, name string, value int) string {
	panic("syscall/dart needs the gart compiler")
}

// GC provokes a collection and gives up control. Each cleanup of a value that
// no code reaches then runs.
func GC() {
	panic("syscall/dart needs the gart compiler")
}

// AddCleanup attaches a function that runs after the collector of Dart frees
// the value, and it gives the handle of the cleanup.
func AddCleanup(target any, run func()) int {
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

// StructOf gives a handle of the struct type that a list of fields builds. The
// 4 lists hold the name, the tag, the type handle, and the embedded mark of
// each field, in the order of the fields.
func StructOf(names, tags []string, types []int, anon []bool) int {
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

func FileFd(handle, std int) int {
	panic("syscall/dart needs the gart compiler")
}

func DirectoryOpen(path string) (int, int) {
	panic("syscall/dart needs the gart compiler")
}

func DirectoryFd(fd int) int {
	panic("syscall/dart needs the gart compiler")
}

func DirectoryClose(fd int) int {
	panic("syscall/dart needs the gart compiler")
}

// FileRead reads bytes into the slice and gives the count. A count of 0 with no
// cause means the end of the file.
func FileRead(handle int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// FileWrite writes the bytes of the slice and gives the count.
func FileWrite(handle int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// FileSeek moves the read position and gives the new offset from the start.
func FileSeek(handle int, offset int64, whence int) (int64, string) {
	panic("syscall/dart needs the gart compiler")
}

func FileClose(handle int) string {
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

// WriteFile writes a whole file. It replaces the content of an existing file.
func WriteFile(path string, data []byte) string {
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

func Remove(path string) string {
	panic("syscall/dart needs the gart compiler")
}

// RemoveAll removes a path and each path inside it.
func RemoveAll(path string) (string, string, string) {
	panic("syscall/dart needs the gart compiler")
}

// Mkdir makes a directory. It also makes each parent directory when deep is
// true.
func Mkdir(path string, deep bool) string {
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

// DialTCP opens a TCP connection to a host and a port, and gives its handle.
// The second result holds the cause of a failure, and it is empty after a
// success.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func DialTCP(host string, port int) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ConnAddrs gives the local address, the local port, the peer address, and
// the peer port of a connection.
func ConnAddrs(handle int) (string, int, string, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ConnRead reads bytes of a connection into the slice and gives the count. A
// count of 0 with no cause means the end of the stream.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ConnRead(handle int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ConnWrite writes the bytes of the slice to a connection and gives the count.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ConnWrite(handle int, p []byte) (int, string) {
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
// handle, the port that the system took, and the cause of a failure.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ListenUDP(host string, port int) (int, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ListenMulticastUDP opens a datagram socket that shares its port with another
// socket of the system. The results are the handle, the port that the system
// took, and the cause of a failure.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ListenMulticastUDP(host string, port int) (int, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// JoinMulticast joins a multicast group on a datagram socket. An empty
// interface name reads each interface of the system. The result is the cause of
// a failure.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func JoinMulticast(handle int, group, iface string) string {
	panic("syscall/dart needs the gart compiler")
}

// DialUDP opens a datagram socket that holds a remote address. Such a socket
// drops a datagram of another source. The results are the handle, the local
// port that the system took, and the cause of a failure.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func DialUDP(host string, port int) (int, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// PacketRead reads 1 datagram into the slice. The results are the count, the
// host of the source, the port of the source, and the cause of a failure. A
// datagram that holds more bytes than the slice loses the rest.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func PacketRead(handle int, p []byte) (int, string, int, string) {
	panic("syscall/dart needs the gart compiler")
}

// PacketWrite writes the bytes of the slice to a host and a port, and gives
// the count.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func PacketWrite(handle int, p []byte, host string, port int) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// PacketClose closes a datagram socket.
func PacketClose(handle int) string {
	panic("syscall/dart needs the gart compiler")
}

// PacketDeadline writes the 2 deadlines of a datagram socket.
func PacketDeadline(handle int, read, write int64) string {
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

// FetchStart sends a request through the fetch API of the host, and it waits
// for the head of the answer. Each pair of items of header holds the name of a
// field and its value, and the third result holds the same shape.
//
// The results are the handle of the body, the status, the header, the address
// of the answer, and the cause of a failure. The address is empty unless a
// redirect moved the answer.
//
// deadline ends the wait, in nanoseconds after the start of the Unix epoch. A
// value of 0 holds no limit, and the same limit covers each read of the body.
//
// A target that holds no fetch API gives the cause "operation not supported".
//
// The call blocks the goroutine, thus the caller becomes an async function.
func FetchStart(method, url string, header []string, body []byte, deadline int64) (int, int, []string, string, string) {
	panic("syscall/dart needs the gart compiler")
}

// FetchRead reads bytes of the body of an answer into the slice and gives the
// count. A count of 0 with no cause means the end of the body.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func FetchRead(handle int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// FetchClose ends the read of the body of an answer and frees its handle.
func FetchClose(handle int) string {
	panic("syscall/dart needs the gart compiler")
}

// StartTimer starts a timer that runs f after a count of nanoseconds, and
// gives the handle of the timer. Go runs f on a new goroutine, thus f can
// block.
func StartTimer(nanoseconds int64, f func()) int {
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

// ProcStart starts a process. argv holds the program and each argument, dir
// holds the working directory, and env holds each variable in the form
// name=value. An empty dir keeps the directory of the program. useEnv tells
// whether env replaces the environment, because a nil list and an empty list
// differ in Go. The results are the process id and the cause of a failure.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ProcStart(argv []string, dir string, env []string, useEnv bool) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ProcWrite writes the bytes of the slice to the standard input of a process.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ProcWrite(pid int, p []byte) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ProcCloseStdin closes the standard input of a process. A child that reads
// until the end of the input then sees that end.
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
//
// The call blocks the goroutine, thus the caller becomes an async function.
func ProcWait(pid int) (int, string) {
	panic("syscall/dart needs the gart compiler")
}

// ProcKill sends a signal to a process. The signal is the number of the
// system, for example 9 for SIGKILL and 15 for SIGTERM.
func ProcKill(pid int, signal int) string {
	panic("syscall/dart needs the gart compiler")
}

// KillPid sends a signal to any process by its id. ProcKill reaches a child
// of this program only.
func KillPid(pid int, number int) string {
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

func Getuid() int      { panic("syscall/dart needs the gart compiler") }
func Geteuid() int     { panic("syscall/dart needs the gart compiler") }
func Getgid() int      { panic("syscall/dart needs the gart compiler") }
func Getegid() int     { panic("syscall/dart needs the gart compiler") }
func Getppid() int     { panic("syscall/dart needs the gart compiler") }
func Getpagesize() int { panic("syscall/dart needs the gart compiler") }

func Getgroups() ([]int, int) { panic("syscall/dart needs the gart compiler") }

func SendSignal(pid, signal int) int { panic("syscall/dart needs the gart compiler") }

// VerifySystemCertificates uses the platform trust policy for DER certificates.
// The date uses seconds from 2001-01-01 when hasDate is true.
func VerifySystemCertificates(certs []string, name string, date float64, hasDate bool) ([]string, int, string) {
	panic("syscall/dart needs the gart compiler")
}
