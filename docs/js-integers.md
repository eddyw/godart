# The Go integer of the JS target

The 2026-09-16 suite covers 10 VM/JS parity programs. It includes integer boundaries, reflection, generic operations, clocks, pointers, and identity.

## The problem

A Go int holds 64 bits. A Dart int holds 64 bits on the VM, with AOT, and on
the wasm target. dart2js maps an int to a JS number, which is a 64-bit float
with 53 bits of an integer. An overflow does not wrap, a shift by 32 or more
drops bits, and a multiply of 2 large values rounds. Each difference is a
silent change of meaning, and the project rejects such a change.

## The model

The emitted code names the type `GoInt` in place of the Dart `int`. The
runtime holds 2 forms of 1 surface, and a conditional export selects 1:

```dart
// goint.dart
export 'goint_native.dart' if (dart.library.js) 'goint_js.dart';
```

`dart.library.js` separates dart2js from the wasm target: dart2js holds the
old dart:js library, and dart2wasm does not.

### The native form

`goint_native.dart` holds an extension type over the Dart int:

```dart
extension type const GoInt(int $v) implements int { ... }
```

The extension type erases at the compile, thus the VM, AOT, and the wasm
target see a plain int, and the emitted arithmetic costs nothing. The form
redeclares `+ - * & | ^ ~` and the unary minus, thus the result type stays
GoInt. Goal G5 holds: the output text is equal for each target, because the
target selects the library and not the code.

### The JS form

`goint_js.dart` holds the pair of GopherJS: a signed high half and an
unsigned low half, each of 32 bits. Each step stays under 53 bits, thus each
step is exact in a JS number. The multiply splits each half into 16-bit
parts. The division, the remainder, and the decimal text ride BigInt,
because they are rare next to the other operations, and the BigInt of Dart
is exact on dart2js. A bitwise operation of dart2js gives an unsigned 32-bit
result, thus each half normalizes after the operation.

## The surface

The 2 forms hold 1 surface, and the emitted code uses only it:

- `$gi(int)` builds a Go integer of a value inside 53 bits, and `$gi2(high,
  low)` builds one of 2 halves. The emitter writes `$gi` for a small literal
  and `$gi2` for a large one.
- `$ix(GoInt)` gives the plain Dart int back, for an index, a length, or a
  handle. `$fromF64(double)` truncates, as a Go conversion does.
- The helpers of the old int.dart moved in: the masks `$i8` to `$u32`, the
  division forms, the shifts, the unsigned order, the unsigned text, and the
  float bit reads. Each takes and gives GoInt.
- `$goList8` to `$goListU64` allocate the backing list of an integer slice.
  The native form keeps the dense typed-data list, because the list type
  erases there, and the JS form gives a plain list of pairs.
- `$gils(Slice<int>)` lifts a byte slice of the runtime into a Go slice.

## The boundary

The interop boundary is 1 uniform rule of stage 5, in `intrinsic` of
`internal/emit/call.go`: each Go integer argument of an interop call takes
`$ix`, and each integer result comes back through `$gi`, `$gils`, or a small
typed closure over the result record. Thus the runtime library keeps the
plain int inside, and no wrapper of `net.dart` or `file.dart` changed for
the cutover. Float-bit operations, reflection integers, and clock timestamps use raw
`GoInt` boundaries because their values can exceed the exact JS integer range.

A Go constant of an integer type emits as a lazy `final`, because `$gi` is
no Dart constant.

## The costs and the gaps

| Target | The cost |
|---|---|
| VM, AOT, wasm | None. The extension type erases, and the output text stays equal. |
| JS | Each 64-bit operation runs several JS operations, and a division rides BigInt. |

The 2026-09-16 changes preserve `GoInt` across reflection integer access and
JSON paths. Mixed numeric generics use dictionaries without a `num` bound.
Wall-clock and scheduler timestamps retain nanosecond precision.

Remaining limits:

- Timer durations and some network deadlines still pass through Dart `int`.
  Values outside the exact JS integer range can lose precision there.
- Browser file and raw socket operations remain unavailable.
- `$ix` narrows to a JS number. Interop callers must use it only where the
  value fits the exact range or the target API requires that conversion.

## The proof

- `runtime/goruntime/test/goint_test.dart` runs an operation table of the
  JS form against the native form on the VM: 30 values crossed, the shifts,
  the masks, the texts, the float bits, and the division edges.
- A node probe ran the same table compiled with dart2js, and the output was
  byte-equal with the VM run.
- `TestJS` compiles `testdata/programs/intwrap` with `dart compile js` and
  runs it under node. The output is byte-equal with `go run`.
- `TestWebIO` compiles `testdata/web/webio` with `dart compile js` and with
  `dart compile wasm`. That program reads `os` and `net/http`, thus the whole
  io surface passes the JS compile.
