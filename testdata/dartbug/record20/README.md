# A record of 20 fields stops dart2wasm

`padRecord` of stage 4 gives an ordering record of exactly 20 fields 1 more
element, and no code reads it. This directory holds the witness of the cause,
and the report below. Remove the pad when the Dart SDK closes the defect, and
`dart compile wasm` of `testdata/web/webio` proves the removal.

Run the witness:

```sh
cd testdata/dartbug/record20
dart pub get
dart compile wasm bin/main.dart -o /dev/null
```

The witness still fails on 2026-09-16 with Dart 3.13.3. The web fixture retains the padding workaround.

Read decision 395.

## The report

**Title:** `dart compile wasm`: a record of exactly 20 fields crashes the type
flow analysis with `type 'Call' is not a subtype of type 'Parameter' in type
cast`

**Dart SDK version**

```
Dart SDK version: 3.13.3 (stable) (Tue Sep 1 01:07:17 2026 -0700) on "macos_arm64"
```

**The command**

```
dart compile wasm bin/main.dart -o main.wasm
```

**The source**

`pubspec.yaml`:

```yaml
name: record20
publish_to: none
environment:
  sdk: ^3.13.0
```

`bin/main.dart`:

```dart
void main() {
  print((1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20));
}
```

**The error**

```
Unhandled exception:
type 'Call' is not a subtype of type 'Parameter' in type cast
#0      Summary.apply (package:vm/transformations/type_flow/summary.dart:1094)
#1      _DirectInvocation._processFunction (package:vm/transformations/type_flow/analysis.dart:444)
#2      _DirectInvocation.process (package:vm/transformations/type_flow/analysis.dart:309)
#3      _WorkList.processInvocation (package:vm/transformations/type_flow/analysis.dart:1956)
#4      TypeFlowAnalysis.applyCall (package:vm/transformations/type_flow/analysis.dart:2253)
#5      Call.apply (package:vm/transformations/type_flow/summary.dart:388)
...
#33     _runTfaPhase (package:dart2wasm/compile.dart:513)
<asynchronous suspension>
#34     compile (package:dart2wasm/compile.dart:256)
<asynchronous suspension>
#35     generateWasm (package:dart2wasm/generate_wasm.dart:78)
<asynchronous suspension>
#36     main (file:///.../sdk/pkg/dart2wasm/bin/dart2wasm.dart:10)
<asynchronous suspension>
```

**What I expected**

The program compiles to wasm, exactly as the same program with 19 fields or
with 21 fields does.

**What I found**

`dart compile js` and `dart compile exe` of the same program each pass. Only
the wasm target fails.

- A record of 19 fields passes.
- A record of 20 fields fails.
- A record of 21 fields passes.
- 19 positional fields and 1 named field also fail, thus the total count
  drives it.
- The record needs no use at run time. A dead declaration of the type is
  enough.

`Object.hash` of `dart:core` holds 20 positional parameters, `object1` to
`object20`. dart2wasm writes 1 implementation class for each shape of a
record and gives it a `hashCode` getter, and that getter calls `Object.hash`
with the shape id in front of each field. A record of 20 fields thus gives a
call of 21 arguments, and above 20 fields the generator reads
`Object.hashAll` instead. The guard reads the field count where it must read
the argument count.

The type flow analysis then applies the summary of `Object.hash` to 21
positional arguments, and `Summary.apply` casts statement 21 to `Parameter`
where a `Call` stands.

A large program can hide the fault. `largeSummarySize` is 300 and
`maxDirectInvocationsPerSelector` is 10, thus after 10 distinct direct
invocations of `Object.hash` the analysis reads raw arguments and the cast
never runs. An unrelated edit can therefore make the fault appear or
disappear.
