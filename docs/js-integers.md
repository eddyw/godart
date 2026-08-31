# The design of the JS integer

The status: a design, with no code. The user set the order on 2026-09-01: the
JS target comes after the package coverage, the parity work, and the wasm
target. This doc holds the design, thus the work can start in a later session
with no new survey.

## The problem

A Go int holds 64 bits. A Dart int holds 64 bits on the VM, with AOT, and on
the wasm target. The check of 2026-09-01 proved the wasm case: a wrap test
with `1 << 62` doubled, `(-1) >>> 40`, and an FNV multiply gave the values of
the VM under node (read `docs/history.md`).

The JS target differs. dart2js maps an int to a JS number, which is a 64-bit
float. Such a number holds 53 bits of an integer. An overflow does not wrap,
a shift by 32 or more drops bits, and a multiply of 2 large values rounds.
Each of these differences is a silent change of meaning, and the project
rejects such a change. Thus no program of the compiler targets dart2js today,
and this design names the path.

## The reference model: GopherJS

GopherJS compiles Go to JS, and it holds a proven 64-bit model. A 64-bit
value is a pair of 32-bit halves, `{ high, low }`. Each operation reads the
halves, computes with 32-bit steps, and writes a pair. A multiply splits each
half again into 16-bit parts, because a JS multiply of 2 values over 2^26
rounds. A division goes through a float when the operands fit 53 bits, and
through a loop of shifts when they do not.

The model costs about 5 to 20 times a plain JS operation. GopherJS pays that
cost only for `int64`, `uint64`, and `uintptr`. It maps `int`, `int32`, and
the smaller widths to a plain JS number with a mask, because 32 bits fit a JS
number exactly. godart cannot split the widths this way: the subset gives
`int` 64 bits, and the emitted code holds 1 Dart int type for each width.

## The design: 1 integer type behind a conditional import

The emitted code names a type `GoInt` in place of the Dart `int`. The runtime
library holds 2 forms of it, and a conditional import selects 1:

```dart
// goruntime.dart
export 'src/goint_native.dart' if (dart.library.js) 'src/goint_js.dart';
```

### The native form

`goint_native.dart` holds an extension type over the Dart int:

```dart
extension type const GoInt(int $v) implements int {}
```

An extension type erases at the compile: the VM, AOT, and the wasm target see
a plain int, thus the 3 targets pay nothing. The clause `implements int`
gives the type each int operator, thus the emitted arithmetic stays a plain
Dart operation, and the output of the 3 targets keeps its speed and its
meaning. Goal G5 also holds: the output text is equal for each target,
because the target selects the library and not the code.

### The JS form

`goint_js.dart` holds the pair form, in the model of GopherJS:

```dart
final class GoInt {
  final int $high; // 32 bits, as a JS number
  final int $low;  // 32 bits, as a JS number
  const GoInt.pair(this.$high, this.$low);
  GoInt operator +(GoInt o) { ... }
  ...
}
```

The class holds the operators `+ - * ~/ % & | ^ << >> >>> ~ unary-`, the
comparisons, `toDouble`, `toString`, and the constructors of the runtime
helpers. The multiply and the division take the 16-bit split of GopherJS.

### What the emitter changes

- `basicType` writes `GoInt` for each integer kind, in place of `int`.
- An int literal takes a constructor form that both libraries hold, for
  example `$gi(5)`. The native form gives the value back, and the JS form
  builds a pair from 2 halves at the compile of the program.
- The helpers of int.dart (`$shl`, `$div`, `$u8`, and the others) move onto
  the type or take a GoInt parameter, thus each target holds 1 set of names.
- A Dart index, a list length, and an interop parameter need the plain int.
  The boundary takes `.toJs()` and `$gi(...)` forms, exactly where the code
  crosses into a Dart API.

The boundary work is the largest part. The emitter writes an int in many
places: an index, a slice length, a state number, a hash, a handle of the
interop. The state number and the handle stay a plain Dart int, because they
never reach Go code. Only a value of a Go integer type takes GoInt.

## The costs

| Target | The cost of the design |
|---|---|
| VM, AOT, wasm | None. The extension type erases, and the output text stays equal. |
| JS | Each 64-bit operation runs 5 to 20 JS operations, as in GopherJS. |
| The compiler | The boundary work in stage 5, and a survey of each intrinsic. |

## Open questions

- Does `implements int` on an extension type expose an operation whose JS
  form cannot hold the contract, thus the native form and the JS form drift?
  The work needs a conformance test that runs 1 operation table against both
  libraries.
- Does the JS target pay for `int8` to `int32`? GopherJS holds a plain
  number for them. A second extension type for the small widths would keep
  that speed, and it would grow the boundary work.
- How does `$f64ToBits` work on JS, where no 64-bit int exists? GopherJS
  reads a DataView. The JS library takes the same path.

## The decision

The design stands, and the work waits. The wasm target already covers the
browser with the 64-bit model, thus the JS target serves only a build that
must ship plain JS. Decision 209 holds this state.
