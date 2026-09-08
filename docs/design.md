# Design

This file holds the architecture of the compiler and the design of each compiler
feature. Read it before a change of a stage, of the intermediate tree, or of the
emitted Dart.

`docs/block-model.md` holds the unwind model of a function that can block.
`docs/decisions.md` holds each decision, with a number that other files name.
`docs/library.md` holds the standard library work. `README.md` holds the state
of the project.

## Components

The project has 4 components:

1. The compiler. A Go program. It converts Go packages into Dart libraries.
2. The runtime library. A Dart package with the name `goruntime`. It holds the
   core types and the scheduler.
3. The standard library overlays. Dart files and Go files that replace the Go
   files which use the operating system.
4. The build driver. It loads a Go module, plans the compile order, and writes
   the Dart packages.

## Compiler pipeline

The compiler has 6 stages. Each stage has 1 function. Do not mix the stages.

### Stage 1: Load

- Use the package `golang.org/x/tools/go/packages`.
- Load the Go module with full syntax trees and full type information.
- Configure the build tag `dart` for the load.
- Load 2 times. The first load finds the files of each standard library package,
  and the second load replaces them through `packages.Config.Overlay`.
- The overlay also adds `syscall/dart`, which the standard library does not
  hold. The go command accepts a new directory inside GOROOT.
- The first load also reads each `//go:embed` directive of the module. It
  resolves the patterns, and it writes the files into the declaration of the
  variable. The same overlay map carries the result to the second load. Read
  the section "The embed package" of `docs/library.md`.

### Stage 2: Verify

The verifier enforces the subset. It rejects code outside the subset with a
clear error and a source position. The verifier rejects:

- An import of the `unsafe` package, and all cgo code. A blank import of
  `unsafe` stays in the subset, because such an import performs no operation.
- The address of a value that names no location. A variable, a composite
  literal, a field of a struct, and an element of a slice or of an array each
  name one, and the result of a call names none. Read the "Pointers" section.
- The address of an exported package variable. The box of such a variable does
  not cross a package boundary.
- An index, a slice expression, a range loop, `len`, and `cap` on a type
  parameter with more than 1 Dart type.
- An unnamed struct type in an exported declaration of a package.
- A `goto` that leaves the body of a range loop over a function. A `return` and
  a labelled branch each stay in the subset.
- A type declaration inside a function, except a struct type.
- An import of a library package outside the supported surface. Read the
  "Standard library" section for the rule.
- A read of `embed.New` in a file that holds no `//go:embed` directive. The
  real package `embed` of Go holds no such name.
- A read of a function declaration that holds no body. A `//gart:dart`
  directive on such a declaration lifts the rejection. Read "An external Dart
  declaration".
- A `//gart:dart` declaration that names a method, a type parameter, a variadic
  parameter, more than 1 result, or a type outside the value boundary.

`internal/verify` holds a test for each rule. Read that package for the exact
list, because it is the authority.

`goto` and `fallthrough` stay in the subset. Do not reject them.

### Stage 3: Analyze

- Compute a block summary for each function in the package. Read the
  "Concurrency" section for the block rules.
- A summary lists the direct block operations and the callees of the function.
- Resolve the final block state with a fixed point over the summaries of all
  packages.
- A call through a function value reaches only the functions that flow to that
  value. `internal/analyze/flow.go` holds the value flow: each assignment, each
  argument, each result, and each struct field. A value that leaves this net
  falls into the escape group of its signature, and a call outside the net
  reads that group. Read decision 137.
- A call through an interface reaches the group of that method. The key of a
  group holds the name, the signature, and the interface of the call site, thus
  a concrete method joins only a group whose interface its receiver implements.
  A type assertion narrows that interface further: a value of `x.(I)`
  implements `I` and the interface of `x`. `internal/analyze/iface.go` holds
  the 2 rules, and each test that cannot answer gives true, thus the group
  never drops a target that a call can reach. Read decision 250.
- The body of a range loop over a function is a function of the analysis, with
  a summary of its own. Read decision 138.
- Find all instantiations of a generic function and of a generic type.
- Reject each function that can block in a synchronous export position.
  `internal/analyze/sync.go` holds the check. A named function type whose
  declaration carries `//gart:sync` names such a position, and the check reads
  the points-to sets of the value flow. Read decision 221.
- Keep all declarations. The Dart toolchain removes each unused declaration in a
  release build.

### Stage 4: Normalize

- Convert the typed syntax tree into a small intermediate tree.
- The intermediate tree keeps the statement structure of the source.
- Make all implicit conversions explicit.
- Replace a named return value with a local variable.
- Rewrite each `range` loop for its container type.
- Insert a struct copy operation at each copy point. Read the "Structs" section
  below for the copy points.
- Hoist an expression that has multiple values into local variables.
- Give each statement the order of the evaluation of gc. A call that stands
  behind an operand which reads a value takes a temporary variable in front of
  the statement. `internal/normalize/order.go` holds the rule, and
  `docs/parity.md` holds each position that still differs. Read decision 324.
- Expand a struct literal key that is a nested field selector. Go 1.27 allows
  such a key.

### Stage 5: Emit

- Walk the intermediate tree and write Dart source code.
- Run `dart format` on the files that the build wrote. A cached package needs no
  second format run, because its files stand formatted on disk already.

### Stage 6: Write metadata

- For each compiled package, write an archive file.
- The archive holds the exported type information, the block summaries, and 2
  hashes.
- The source hash covers the source files, the dependency metadata, the compiler
  version, and the flags.
- The export hash covers only the exported interface. This includes the block
  state of each exported function.
- If the source hash of a package did not change, do not compile the package
  again.
- If only a function body of a dependency changed, the export hash of the
  dependency stays equal. Then the dependents keep their cache. This mechanism
  is the early cutoff.

## An external Dart declaration

`intrinsic` of stage 5 is a flat switch on the name of each interop function of
`syscall/dart`, thus each new Dart call costs an edit of the compiler. A
binding of a large Dart API does not fit in such a switch.

A Go function declaration with no body already parses. The directive
`//gart:dart` on such a declaration names a Dart library and a Dart symbol, and
stage 5 writes a direct call of it.

```go
//gart:dart dart:math max
func dartMax(a, b float64) float64
```

The output holds `import 'dart:math' as $d0;` and the body
`return $d0.max(a, b);`. `dart:core` takes no prefix and no import, because a
prefixed import of it hides the plain names and `double` and `String` would
leave the file.

A third field `blocking` marks a Dart symbol that gives up control. The Dart
form of such a symbol waits for a future of the event loop, thus stage 3 marks
each call of it as a state boundary and the caller becomes a machine.

```go
//gart:dart package:goruntime/goruntime.dart $lookupAddrs blocking
func lookupAddrs(host string, family int) string
```

Only the intrinsic table of `syscall/dart` held such a call before, thus each
new Dart call that waits cost an edit of `internal/emit/call.go`. A package of
the overlay declares its own now.

**The value boundary.** The call holds no type descriptor, thus each parameter
type and the result type must hold 1 Dart form.

| Go | To Dart | From Dart |
|---|---|---|
| `bool` | The value | The value |
| `float64` | The value | The value |
| `int` | The value. `GoInt` implements the Dart `int`. | `$gi(...)` |
| `string` | `.toDartString()` | `GoString.fromString(...)` |
| `dart.Object` | The value. The Dart form of the type is `Object?`. | The value |

Stage 2 rejects each other type. A wider set needs a conversion table, and a
wrong conversion gives a wrong value with no report.

**The twin file.** The go command rejects a function with no body, thus a
module that holds such a declaration needs a second file behind the tag
`!dart`. `testdata/programs/dartdecl` and `testdata/programs/dartobj` each hold
that pair, and `make compare` runs the Go form and the Dart form and diffs the
2.

`dart.Object` of `syscall/dart` carries 1 Dart object through Go. A Go program
reads no field of such a value and calls no method of it, and only a Dart
declaration gives 1. `syscall/dart` carries `//go:build dart`, thus a module
that names the type needs an alias of its own in the twin file. Read decision
239 and decision 240.

## The map of a package

- 1 Go module maps to 1 Dart pub package. Generate a `pubspec.yaml` file. Mirror
  the dependencies from `go.mod`.
- 1 Go package maps to 1 Dart library. Mirror the import path in the directory
  structure.
- 1 Go file maps to 1 Dart `part` file of that library. Each part file ends with
  `.part.dart`, because a package may hold a file whose name is the name of a
  package inside it.
- An unexported Go identifier gets a `_` prefix in Dart. Thus the identifier is
  private to the Dart library. This rule is equal to the Go visibility rule.
- If an identifier collides with a Dart keyword, change the identifier. Use 1
  rule that gives the same result at each build.

## Types and values

### Integers

- The target is the Dart VM and AOT. A Dart `int` has 64 bits.
- `int` and `int64` map to the Dart `int`.
- `uint64` maps to the Dart `int` with wrap semantics. Use a runtime function
  for division, for comparison, and for a right shift.
- For `int8`, `int16`, `int32`, `uint8`, `uint16`, and `uint32`: apply a mask
  after each arithmetic operation.

### Floats and complex numbers

- `float64` maps to the Dart `double`.
- For `float32`: round the value to 32-bit precision after each operation.
- `complex64` and `complex128` map to the runtime class `GoComplex`. The class
  holds 2 doubles and the Dart operators, and the division is the algorithm of
  Smith with the corrections of C99, as in the Go runtime. For `complex64`:
  round the 2 parts with `$c64` after each operation, exactly as `float32`
  rounds with `$f32`. Read decision 207.

### Strings

- A Go string is an immutable byte sequence. Do not map it to a Dart `String`.
- Make a class `GoString` in the runtime library. Store the bytes in a
  `Uint8List`.
- Convert between `GoString` and a Dart `String` only at an interop boundary.

### Slices

- A slice maps to a runtime class with 4 fields: a backing array, an offset, a
  length, and a capacity.
- For a numeric element type, use a typed data list, for example `Int32List`.
- For all other element types, use `List<T>`.

### Arrays

- A Go array has a fixed length and value semantics.
- Copy the array at each copy point.

### Maps

- A Go map uses Go equality and a Go hash function.
- Use a Dart `HashMap` with a custom `equals` function and a custom `hashCode`
  function.
- Support a struct key and a `NaN` key.

### Structs

A struct maps to a Dart class. A struct has value semantics. Generate a `$copy`
method for each struct class. The normalize stage inserts a copy at each copy
point. The copy points are:

- An assignment.
- An argument in a function call.
- A return value.
- A channel send.
- A store into an array element, a slice element, or a map value.

### Pointers

- A pointer to a struct is the Dart object reference. A pointer to an array is
  the Dart List. Such a pointer needs no box, and it has no extra cost.
- A pointer to a value of each other type holds a `Box<T>` of 1 field. The
  compiler writes a box only for a variable whose address the code takes.
- A variable and a composite literal each hold a box. A field of a struct and
  an element of a slice or of an array each hold a `$Ref` of the object and the
  index, thus the code takes the address of each 1. Stage 2 rejects the address
  of a form that names no location, and it rejects the address of an exported
  package variable.

### Interfaces

- Generate a runtime type descriptor for each named type.
- The descriptor holds a method table.
- An interface value is a pair of a descriptor and a value.
- A type assertion and a type switch compare the name of the descriptor. A cache
  of the results stays open.

### Generics

- A Go type parameter maps to a Dart type parameter.
- For each type argument, pass a dictionary object at run time.
- The dictionary gives the zero value, the equality function, the hash function,
  and the allocator of a backing array.
- Go 1.27 allows a type parameter on a method declaration. A Dart method also
  takes a type parameter, thus the direct map works for a concrete receiver.

### Multiple return values

- A function with multiple return values returns a Dart record.
- Use a record pattern to read the values at the call site.

## Statements

- `for`, `if`, and `switch` map 1:1 to a Dart statement.
- Use a Dart label for a labeled `break` and for a labeled `continue`.
- Rewrite a `switch` case that has `fallthrough` into explicit control flow.
- `defer`: use `try` and `finally` with a defer stack in the runtime.
- `panic`: throw a `GoPanic` exception. The deferred functions run through
  `finally`. Then the panic stops the program. The subset has no `recover`.
- `goto`: rewrite the function body into a loop with a `switch` on a state
  variable. Apply this rewrite only to a function that holds a `goto`.

## Concurrency

All Go code runs in 1 Dart isolate. Goroutines are cooperative.

The block analysis in stage 3 marks each function that can block. A function can
block if it does 1 of these operations:

- A channel send or a channel receive.
- A `select` statement.
- A wait on a mutex or on a wait group.
- A sleep.
- A call to an isolate worker.
- A call to a function that can block.

Rules for the emitted code:

- A marked function keeps its Go result type. It takes 1 more parameter: the
  frame of a call that gives up control.
- The body of a marked function becomes a flat state machine. Stage 4 writes it,
  and `internal/normalize/flatten.go` holds the pass.
- A channel operation, a select statement, a sleep, and an isolate worker each
  write a frame into `$blocked`. Each one gives a value that no code reads.
- Each caller tests `$blocked` after such a call. It writes its own locals into
  a frame, it writes that frame into `$blocked`, and it returns. Thus the stack
  unwinds up to the scheduler.
- The scheduler keeps the frame of the last caller. A wake calls `resume` on it,
  and each frame re-enters at its saved state. Thus the stack rewinds.
- For a call through an interface or through a function value: if 1 possible
  target can block, mark the call. Use the call graph for this decision.
- A `go` statement puts a new goroutine on the run queue.
- `sync/atomic` operations map to a plain operation. The single isolate makes
  this safe.
- A mutex is a flag with a waiter queue in the runtime library.
- Keep a synchronous path for a function that cannot block.

### Isolate workers

Goroutines do not map to a Dart isolate, because 2 isolates share no memory. An
isolate gives a different benefit: parallel work that does not block the main
isolate. Integrate this benefit from the start with these rules:

1. Each isolate runs a separate instance of the compiled program. The package
   variables, the scheduler, and the channels exist per isolate. Dart gives this
   separation for free, because all top-level state in Dart is per isolate.
2. The runtime makes no reference to state in another isolate. Do not put shared
   state into FFI memory or into other native storage.
3. The interop package gives a worker function. The worker function starts a Go
   function in a new isolate through `Isolate.run`, and it returns the result. A
   worker call maps to an `await` expression. Thus the block analysis marks the
   caller.
4. The worker contract: the worker function must be a pure function of its
   input. It must not read a package variable, and it must not write a package
   variable. In the Go build, the same API runs the function on a goroutine. The
   contract keeps the results equal in Go and in Dart.
5. Restrict the worker input and the worker output to these types: a byte slice,
   a string, an integer, a float, and a boolean. These types cross the isolate
   boundary at a low cost.
6. Make the `GoString` byte storage unmodifiable. A measurement of 2026-08-29
   proves that Dart copies such a list, thus the rule gives no gain. Read the
   facts of 2026-08-29.

## Runtime library

The `goruntime` package holds:

- `GoString`, the slice class, the map class, and the channel class.
- The scheduler, the frame, the run queue, and the waiter queues. A deadlock and
  the end of package main each stop the program.
- The type descriptors and the interface support.
- The `reflect` subset. A parity file lists the gaps.
- The defer stack and the `GoPanic` class.
- The integer functions for `uint64`, for a mask, and for a shift.
- The worker support for an isolate worker.
- The value formatter that `fmt` reads.
- The type dictionary of a generic type argument. `Dict.order` holds the order
  of a type parameter with more than 1 Dart type.
- `Box<T>`, a class of 1 field. A pointer to a value that is not a struct holds
  it.
- The file access through `dart:io`, in `src/file.dart`. Each operation is
  synchronous.
- The network access through `dart:io`, in `src/net.dart`. A dial, a read, a
  write, a listen, and an accept each wait for a future, thus each one gives up
  control. A listener and a connection hold 2 separate handle tables.
- `$await`, the 1 primitive that waits for a Dart future. It writes an
  `EventWait` frame, and the value of the future reaches the caller through the
  resume.
- The timer of `time.AfterFunc`, in `src/sched.dart`. A timer holds 1 wake
  source, and the run puts the Go function on the run queue as a new goroutine.
- The JSON encoder, in `src/json.dart`. It walks the type descriptor.
- The test harness support: `$stop`, `$catch`, `$exit`, and the monotonic clock.
- The intrinsics of the type descriptor and of the platform, in
  `src/format.dart` and in `src/file.dart`. `internal/reflectlite` reads
  `$typeInfo`, `reflect.DeepEqual` reads `$deepEqual`, and `errors.As` reads
  `$assignError`. The report of `flag` reads `$zeroText`, `time` reads
  `$zoneOffset`, and `runtime.GOOS` reads `$goos`. `reflect` reads 6 more:
  `$descOf`, `$descInfo`, `$descElem`, `$descZero`, `$pointerElem`, and
  `$pointerSet`. The table of the descriptors holds a name for a type that no
  value holds.
- `$releases`, the list of releases that the scheduler runs when package main
  returns. Go ends the process at that point, and an open socket of Dart keeps
  the isolate alive. The network library adds 1 release.

All state in the runtime library is per isolate.

## The Dart form of a type set

Go gives a type parameter a core type when each term of its type set shares 1
underlying type. A set with no core type can still hold 1 Dart type.
`typeset.Describe` reads the set and gives 6 marks. Stage 2 and stage 5 both
read them.

| Type set | Dart type | godart |
|---|---|---|
| A core type that is a composite | The Dart type of the core type | Each operation of the core type |
| Each term is an integer type | `int` | Each operator. `Dict.mask` keeps the width. |
| Each term is a float type | The type parameter | An arithmetic operation is rejected, because `float32` needs a mask. |
| An integer term and a float term | The type parameter, with the bound `num` | Each operator, and `/` is rejected. |
| Any other set of more than 1 Dart type | The type parameter | An assignment, an equality, an order, a call, and a conversion to an interface |

The sign of an integer set:

- Each term reads the signed operators of Dart. `int8`, `uint16`, and `int64`
  are 3 examples. Each operator applies.
- Each term is an unsigned type of 64 bits. `uint`, `uint64`, and `uintptr` are
  the 3. The unsigned helpers apply.
- The set holds a term of each form. `/`, `%`, `>>`, a conversion to a float,
  and the builtin `print` each read the sign, thus stage 2 rejects those 5.
  `math/rand/v2` holds such a set, and `N` reads none of the 5.

## The test command

`gart test` compiles the test variants of each package and runs each test binary
with the Dart VM. `gart run` compiles the module and runs package main.

How the pipeline reaches a test binary:

1. Stage 1 loads with `packages.Config.Tests`. The load gives 4 packages for a
   package that holds a test:

   - The plain package.
   - The package with its in-package test files.
   - The external test package.
   - The entry point of the binary.
2. Stage 1 reads each `_test.go` file with `go/parser`, and it collects each
   `TestXxx` function, each `FuzzXxx` function, and each example with an output
   comment. `go/doc.Examples` gives the output comment.
3. Stage 1 writes the Go source of the entry point, and the overlay replaces the
   entry point of the go command with it. The go command fixes the import list
   of that package, thus the source imports only `testing` and the 2 test
   variants.
4. Stage 2 to stage 6 run with no change. The entry point is a normal `package
   main`.
5. The driver runs `dart pub get` 1 time, and then `dart run` for each binary.

The harness lives in `internal/overlay/std/testing/testing.go`. It is Go code
inside the subset, thus the compiler compiles it. The runtime library gives 4
new functions: `$stop`, `$catch`, `$exit`, and `$args`.

The output holds the format of `go test`, without the source position of a log
line. `docs/parity.md` lists each gap.

## The source map

The output holds a source map, and 4 readers use it.

- Stage 5 writes a marker comment in front of each new source line and in front
  of each function body. Stage 6 runs after `dart format`, removes each marker,
  and writes the map of the file. Thus the line numbers of the map name the
  lines that a Dart stack trace reports, and the output holds no marker.
- A function with an empty body holds the marker of the function alone. The
  removal of that marker leaves an empty body on 2 lines, and `dart format`
  writes such a body on 1 line. Stage 6 thus joins the 2 lines. The map file
  passes no format run, thus the writer gives an empty map the form of
  `dart format` also. Both rules keep the output equal to the output of
  `dart format`, and the map keeps the line numbers of the file on disk.
- The map of 1 package stands in the part file `<base>.map.dart`, and the init
  routine of the package registers it. A key names a Dart file the way a stack
  trace names it. The value holds each function and each line, with a delta in
  base 36. The map costs between 2.7 percent and 4.1 percent of the bytes of
  the output. The measure of 2026-09-05 read `fmtuse`, `jsonuse`, and
  `httpuse`.
- The runtime library holds the reader. `$goStack` maps a Dart stack trace to
  the Go frames, and `$stack` gives them to Go through `syscall/dart`.
- The 4 readers: a stack trace of a panic, `runtime.Caller` with each function
  that follows it, `t.Helper` of the testing harness, and `Lshortfile` of the
  log package.

The map holds the module form of a Go file: the package path and the base
name, as a build with `-trimpath` gives. Each file of a package lives in the
directory of the package, thus the form needs no path of the machine, and 2
checkouts of 1 tree give equal output for goal G5. The text of a line
directive passes verbatim, and go/token resolves a relative directive against
the directory of the source, exactly as the go compiler does. A line marker
names its file only when a directive changed it, and the map entry then
carries the file behind a `|`. The marker `//#g0` unmaps the line that
follows: the wrapper of a deferred call holds no Go frame, and the reader
drops such a line from a stack.

Stage 4 gives each statement the position of its Go source. The flat state
machine gives each statement that it builds the position of the source
statement. The return of a fall off the end keeps no position, and the
emitter marks the closing brace at each run of the defer chain outside a
return, exactly as Go names such an exit.

A function that gives up control leaves the Dart stack, and its frame chain
holds the state. Each frame class implements TraceFrame: the Go name, the
file, the line of the state that resumes, and the frame below. The deadlock
report of the scheduler walks the chain of each live goroutine and writes the
Go frames. A wait frame of the runtime names the reason of the goroutine
header, such as `chan receive`.

`testdata/programs/callers` proves `runtime.Caller`, `runtime.Callers`,
`runtime.CallersFrames`, and `runtime.FuncForPC` against Go.
`testdata/programs/logfile` proves `Lshortfile`, and `testdata/tests/basic/calc`
proves the position of a helper.

## M15, the shared class of an unnamed struct type

The user took option 2 on 2026-08-29. The work is done, and `make ci` passes.

An unnamed struct type takes a file of its own, `lib/gart_anon/<name>.dart`. The
file reads the shape only, thus each package that holds the shape writes equal
bytes, and Dart gives the 2 packages 1 class. `emit.Result.Shared` holds each
such file, and the driver writes it at the root of the output. The `Files` field
of the archive lists it, thus `outputExists` keeps a cached package correct.

The parts of the work:

- The library of a package writes its body first. The body can name a shape, and
  the import of a shape stands at the top of the file.
- The name of the class covers the text of the type and the library of each
  field type. A test variant of a package holds a library of its own. Thus a
  shape that names a type of such a variant takes a class of its own.
- A Dart URI is a string literal, and a dollar sign in such a literal starts an
  interpolation. Thus the file name drops the dollar sign of the class name.
- A shared file holds its own nil channel of an element type. The name carries
  the class, thus 2 files of 1 library give no ambiguous import.

2 shapes keep the class inside the package:

1. A shape with an unexported field. Go gives such a field the identity of its
   package. Dart holds a name that starts with an underscore private to 1
   library. Stage 2 still rejects such a shape in an exported declaration of a
   package that an import can name. No import names a main package or an
   external test package, thus the exported tables of the test files of Go
   pass.
2. A shape that names a type of a package that no library holds. Package main is
   such a package, and no other package names a type of it.

`testdata/programs/anonstructs` holds a second package. It gives an alias of a
shape, and package main passes its own literal of the shape to a function of
that package.

## The loop variable of Go 1.22

Go 1.22 gives each pass of a loop its own copy of the variable that the loop
declares. The compiler wrote 1 variable for the whole loop, thus a closure of
each pass read the value of the last pass. `testdata/programs/loopvar` gave `3 3
3` against `0 1 2`. It now covers 6 forms. The 6 are a 3 clause loop, a range
loop, a variable of the body, a `go` statement, the address of the variable, and
a `continue` statement.

The 4 parts of the fix:

1. The declaration of the first clause stays inside the loop. `ir.For.Init`
   holds it, and a Dart for loop gives the per pass copy when the declaration
   stands in its header.
2. A boxed variable of the first clause takes a new box in front of the third
   clause. The copy of Dart holds a reference, thus 2 passes would share 1 box.
3. A machine writes a new box at each pass. Thus a literal takes a copy of the
   box in a block in front of its statement. `ir.FuncLit.Captured` holds each
   such variable, and the body of the literal reads the copy. A local function
   keeps its name for the statements that follow it, thus the block of such a
   function ends with the statement list.
4. `assignedVars` holds each name that a declaration binds. A range clause and a
   short declaration each bind a new variable at each pass, thus such a variable
   needs a box inside a machine.

A record pattern of Dart takes no boxed variable. Thus a multiple assignment and
a multiple declaration read the record from a temporary variable. The
declaration form writes a new box for each name.

## The descriptor of a generic named type

Decision 14 kept a value of a generic named type out of an interface. The work
of 2026-08-29 took option 1: the description of an instance stands at the point
of the use.

The package holds a variable for the description of each other named type, and
an instance reaches the output only through its use. The name of the description
holds each type argument, thus a type assertion of the wrong instance gives
false. `internal/emit/iface.go` holds `instanceDesc`.

`testdata/programs/genericiface` proves an interface method call, a type
assertion, a type switch, the verb `%T`, and a pointer to an instance.

## A defined type over a struct of another package

`type Time3339 time.Time` gives the 2 names 1 struct object of go/types, thus a
conversion between them names 1 value. The emitter gave each name a class of
its own before, and stage 2 rejected the conversion.

The class must stand in the library of `time`. Go gives the unexported fields
`wall`, `ext`, and `loc` the identity of that package, and a Dart library holds
each name that starts with an underscore, thus a class of another library could
never read them.

`Time3339` thus declares no class. `structOwner` of `internal/emit/type.go`
gives the owner of the class: the instance of a generic type of decision 347,
or the named type of the package that declares the struct object. `structRef`
reads it, and each reference to `Time3339` writes the form of `time.Time`. The
descriptor of `Time3339` keeps the name and the method set of `Time3339`, thus
a type assertion and the verb `%T` each answer as Go answers.

The owner comes from the fields. `st.Field(0).Pkg()` names the package that
wrote the struct type, and the first named struct type of the scope of that
package owns the class. The answer thus reads the type alone, and 2 packages
that both compile the reference give 1 answer. A struct with no field names no
package, and such a type keeps its own class.

A method with a value receiver stands in the Dart class of its type, and the
class of `time.Time` cannot hold a method of another package. A pack of named
types of 1 package that holds such a method thus keeps its own class, and the
conversion keeps its rejection with a report that names that cause.
`internal/verify/verify.go` holds the same rule, because stage 2 and stage 5
must agree.

`testdata/programs/structpkgconv` proves the conversion in the 2 directions, a
pointer conversion, a method with a pointer receiver, an interface, the verb
`%T`, a type assertion, the equality, and the copy point.

## The size of the Dart output

The emitter writes each declaration of each package that the program imports,
and it removes no unreachable one. The session of 2026-09-05 measured the
output of 4 programs and cut the largest contributor.

Each byte count below sums the bytes of every `.dart` file of the output, the
source map parts included. `make run` and `make compare` write that tree.

| program | files | bytes before | bytes after | cut |
| --- | --- | --- | --- | --- |
| `hello` | 3 | 2,527 | 2,527 | 0.0 percent |
| `fmtuse` | 64 | 1,182,810 | 1,109,564 | 6.2 percent |
| `jsonuse` | 255 | 4,721,053 | 4,133,345 | 12.4 percent |
| `httpuse` | 1,054 | 14,593,511 | 13,071,001 | 10.4 percent |

The 10 largest packages of `jsonuse` held these bytes. A package holds its
library file, its part files, and its source map part.

| package | bytes before | bytes after |
| --- | --- | --- |
| `encoding/json/v2` | 1,000,076 | 879,900 |
| `encoding/json/jsontext` | 607,380 | 520,650 |
| `unicode` | 528,283 | 525,901 |
| `reflect` | 318,052 | 214,046 |
| `time` | 274,038 | 218,577 |
| `fmt` | 193,735 | 184,955 |
| `math` | 191,970 | 191,970 |
| `encoding/binary` | 177,341 | 119,675 |
| `io` | 171,676 | 137,373 |
| `encoding/json` | 146,860 | 126,940 |

### Where the bytes stood

The type descriptions held 25.7 percent of `jsonuse` before the cut. The
description of a pointer type held 14.4 percent, the description of a value
type held 9.9 percent, and the rest held 1.4 percent.

The `methodTypes` table alone held 14.3 percent. The description of a type and
the description of a pointer to it each wrote 1 function type for each exported
method, thus the text stood twice.

Decision 352 gives each description with no name of its own 1 variable of the
library. The type descriptions hold 16.1 percent of `jsonuse` now, and
`methodTypes` holds 1.4 percent.

### What a tree shake would still give

A name walk of the emitted Dart, from `main` and from each `$init` routine,
marks about 25 percent of the bytes of `jsonuse` as dead. Of the whole output,
dead functions and dead classes hold 21 percent. The description of a pointer
that no code names holds 1.9 percent.

Stage 5 takes none of it, because 4 rules break a plain reachability walk:

1. `reflect` reads a description by name. `TypeOf`, `StructOf`, `FuncOf`, a
   method table, and `iface.dart` each read a description that no call site
   names.
2. An `init` function of a package runs for its effect.
3. A method of a type reaches a call through an interface, and no direct call
   names it.
4. A linkname hook and an intrinsic of stage 5 reach code that the Go source
   does not name.

A rule that drops such a declaration gives a wrong answer with no report, and
that is worse than a large file.

### A shake at the level of the package

The session of 2026-09-05 asked a narrower question: does a shake that drops a
whole package give the bytes, and is such a shake safe? The answer is no to
both, and the measure names 3 causes.

**1. The output already holds the import closure alone.** `order` of
`internal/driver/driver.go` walks the imports from the root package, and it
keeps each package that the Dart runtime library does not hold. The Go tool
chain names 73 packages for `jsonuse`, and gart writes 32 of them. A shake that
drops a package outside the closure thus finds nothing.

**2. The entry point reaches each package.** The `$init` routine of a package
calls the `$init` routine of each package that it imports, because an `init`
function and a package variable of that package must run first. A walk from
`bin/main.dart` over those calls reaches each of the 32 packages of `jsonuse`
and each of the 117 packages of `httpuse`. A shake at the level of the package
thus drops 0 bytes.

**3. It reaches no table of `unicode`.** 5 packages of `jsonuse` import
`unicode`, and each 1 calls a function of it: `bytes`, `strings`, `reflect`,
`encoding/json/v2`, and `encoding/json/internal/jsonwire`.

To drop a package the compiler must first prove 2 things: no live declaration
names a declaration of that package, and the `$init` routine of that package
writes nothing that the program reads. The first proof is the declaration walk
that the 4 rules above break. The second proof holds a 5th rule: a package that
writes a field of `internal/linkhook` at the start reaches a reader that names
no declaration of it.

gart thus ships no shake at the level of the package.

### Where the tables of `unicode` stand

The tables hold 527,984 bytes of the 4,134,163 bytes of `jsonuse`, and 481,816
of those bytes stand inside the `$init` routine of the package, in 538
statements. Go writes each table as a package variable with a composite
literal, thus stage 4 makes each 1 a statement of `init`, and stage 5 writes
that statement into `$init`. A shake at the level of the package keeps the
whole routine, because the routine always runs.

The rest of the program names 14 declarations of `unicode`: `CaseRange`,
`IsDigit`, `IsGraphic`, `IsLetter`, `IsNumber`, `IsPrint`, `IsSpace`,
`SimpleFold`, `ToLower`, `ToTitle`, `ToUpper`, and the 3 case methods of
`SpecialCase`.

A name walk at the level of the package variable, from those 14 names, keeps
119,302 bytes of the routine and drops 362,514. That is 68.7 percent of the
package and 8.8 percent of the output of `jsonuse`. The largest tables that
stay: `L`, `CaseRanges`, `M`, `S`, `P`, and `N`. The largest tables that go:
`C`, `Cn`, `Lo`, `Mn`, the map `Scripts`, and each script table under it. A
script and a category reach the program through the 3 maps `Categories`,
`Scripts`, and `Properties` alone, and no live declaration names those 3.

A program of the measure read the emitted Dart, thus the number names an upper
bound and no shipped gain.

Such a shake stands at the level of the package variable, and it is not the
shake of this section. gart holds neither proof that a name walk needs: the
initializer of a variable must hold no side effect, and no `go:linkname`
directive may bind the variable to another package.

**A lazy declaration ships, and it saves 574,560 bytes.** A package variable
whose initializer cannot panic and writes nothing takes the form
`late T x = <value>;`. Dart runs such an initializer at the first read, thus
the tree shaker drops the variable and the whole value when no code reads it.
The measure of this session, over a program that reads 1 table of `unicode`:
an ahead of time binary of 7,684,640 bytes before the change and 7,110,080
after it, thus 574,560 bytes and 7.5 percent. A map literal takes the lazy
form too.

**dart2wasm rejects a record of exactly 20 fields, and not the lazy form.** The
type flow analysis stopped with `type 'Call' is not a subtype of type
'Parameter' in type cast`. Decision 373 named 2 map variables of `net/http`,
and decision 387 named 1 declaration of that package. Neither is the cause. A
delta debug ended at the ordering record of 20 fields of `Transport$Clone`:
dart2wasm writes 1 class for each shape of a record and gives it a `hashCode`
getter that calls `Object.hash` with the shape id in front of each field.
`Object.hash` holds 20 positional parameters, thus 20 fields give 21 arguments.
The lazy form only changed the order in which the analysis reached that call.
`padRecord` of stage 4 adds 1 element of the type `bool` to such a record, and
no code reads it. Read decision 395.
`testdata/dartbug/record20/` holds the witness of 3 lines and the report.

## The start time of a large program on the Dart VM

The output of a large program holds many megabytes of Dart, and the Dart VM
reads all of it at each start. The session of 2026-09-05 measured that cost.

The program is `flutter/example/compass/server`, the server of the sample with
9 routes over `net/http` and `encoding/json`. gart wrote 1,122 `.dart` files
with 15,029,679 bytes, and the runtime library adds 40 files. The server reads
2 JSON files of 2,076,698 bytes at the start, and then it listens.

The measure counts the time from the start of the process to the first answer
of the route `/login`. A poll loop sends a request each 2 milliseconds. Each
row gives the median of 5 runs, with a warm file cache.

| Step | Time | Size |
| --- | --- | --- |
| `dart run bin/main.dart` to the first answer | 5.41 seconds | — |
| `dart compile kernel` | 2.87 seconds | 47,881,256 bytes |
| `dart run` of that kernel file to the first answer | 1.33 seconds | — |
| `dart compile exe` | 18.3 to 18.7 seconds | 35,444,096 bytes |
| The binary to the first answer | 0.41 seconds | — |
| `go build`, and the binary to the first answer | 0.012 seconds | 9,400,690 bytes |

The numbers name 3 costs. The front end holds the largest part of the 5.41
seconds: a run of the kernel file alone takes 1.33 seconds. The Dart VM itself
starts fast, because `dart run` of a 3 line file takes 0.17 seconds. The load
and the just in time compile of the kernel hold the other 1.16 seconds.

The ahead of time build removes both costs. The binary answers after 0.41
seconds, which is 13 times faster than `dart run`. It costs 18.5 seconds of
build time, and the file holds 35.4 megabytes against 9.4 megabytes for the Go
binary.

A server thus wants `dart compile exe`. A short program that runs 1 time wants
`dart run`, because the build costs more than the 5 seconds of the start.

The machine: an Apple M4 Max with 16 cores and 128 gigabytes, macOS 27.0, Dart
3.13.2, and Go 1.27.0.

## A deferred call that can block

The deferred calls ran at the return of a function, and the machine kept no
state at that point. Thus stage 4 rejected a deferred call that can block, and
`defer cancel()` of the `context` package stayed outside the subset.

A machine whose deferred calls can block now holds 3 states for the chain of
those calls:

1. The first state tests the stack. An empty stack jumps to the last state, and
   each other pass runs 1 call.
2. The second state holds the resume and the test of `$blocked`, exactly as each
   other call site does.
3. The last state throws a held panic again, or it returns each result.

A return statement writes each result into a variable of the frame and jumps to
the chain. The try block of a panic now stands inside the loop of the machine.
Thus a panic writes the frame and jumps to the chain too. The frame keeps the
panic in the field `$p` until the last call ends.

`testdata/programs/deferblock` proves the order of the calls, a named result, a
value result, a loop, and the panic path.

## The source position of a test log line

`go test` writes `file.go:12: text` for each log line, and the harness of the
overlay reads no stack trace.

Stage 4 now writes the position of each call of `Log`, `Logf`, `Error`,
`Errorf`, `Fatal`, `Fatalf`, `Skip`, and `Skipf` into the variable
`testing.LogPos`. The statement stands in front of the call, and no goroutine
gives up control between the 2. The harness reads the variable and clears it.

2 positions still differ:

- A helper that calls `t.Helper` gives the position of the call inside the
  helper. Go gives the position of the caller, and such a rule needs a stack of
  positions.
- A deferred log line after `Fatal` holds no position. Go writes the position of
  its own runtime, for example `panic.go:694`. The comparison of `go test` and
  `gart test` cuts a position that names a file outside the test files.

## Incremental compilation and watch mode

These rules make watch mode possible with no large refactor. Obey them from the
first commit.

### Rules for the compiler

- Compile each package as a pure function. The inputs are the source files, the
  dependency metadata, the compiler version, and the flags. The outputs are the
  Dart files and the metadata archive.
- Do not use global state in the compile function.
- Make the output deterministic. Sort the keys before each map iteration. Do not
  write a timestamp or an absolute path into the output. Equal inputs must give
  byte-equal output.
- Derive each generated identifier from a stable source, for example the source
  position. Do not use a global counter.
- Use the 2 hashes from stage 6 for the early cutoff.
- The block state of a function is part of the exported interface. A change from
  synchronous to asynchronous changes the export hash. Thus the dependents
  compile again, and that behavior is correct.
- The source hash of a package holds a digest of the block analysis of that
  package. The analysis reads the whole module. Thus a new function of a package
  that this package does not import can change the form of a call through a
  function value. `Result.Digest` gives the text.
- Write a Dart output file only when its content hash changed. This rule
  protects the incremental compiler of Dart from a false invalidation.

### The watch mode layer

Watch mode is a thin layer on the rules above. It has 4 parts:

1. A daemon process. It keeps the parsed packages, the type information, and the
   summaries in memory.
2. A file watcher, for example `fsnotify`.
3. A dirty set. When a file changes, put the package of the file into the set.
   Compile the packages in the set. Then apply the early cutoff to the
   dependents.
4. An in-memory cache in front of the disk cache.

## Go 1.27 impact

The project targets Go 1.27.0. The language version of the spec is go1.27, dated
May 26, 2026. Go 1.27 makes 3 language changes and 1 tool change that the
compiler must handle:

1. A method declaration may declare a type parameter. A Dart method also takes a
   type parameter. A generic method on a concrete receiver maps directly. A
   generic method and the interface method table need a decision. Read the open
   questions.
2. A key in a struct composite literal may be any valid field selector, and not
   only a top-level field name. The normalize stage must expand such a key into
   a nested store, with a copy at each level.
3. Function type inference applies in all assignment contexts that involve a
   function. The package `go/types` resolves this inference, thus the compiler
   needs no new code.
4. Go 1.27 removed the GODEBUG setting `gotypesalias`. Thus `go/types` always
   makes a `*types.Alias` node. Each type switch over a `types.Type` must handle
   `*types.Alias`. The compiler calls `types.Unalias` at each such point.

The work handled each of the 4 items. Point 1 needed no method table, because Go
1.27 forbids a type parameter on an interface method.

The builtin `new` accepts a type or an expression. `new(x)` with an expression
of a type that is not a struct gives a rejected pointer. Stage 2 must reject it.


## The freshness rule of a copy point

Go gives a struct and an array value semantics, thus each copy point needs a
fresh value. A copy of a value that no other expression names gives nothing,
and the compiler holds 1 rule for that test.

`ir.Fresh` of `internal/ir/ir.go` names 5 shapes: a composite literal, a call, a
copy, a zero value, and a conversion. Stage 4 reads it at each copy point, and
stage 5 reads it at the box of an interface value. The 2 must agree: a copy that
only 1 of them writes gives either a wasted object or an alias.

A builtin is not fresh. `iterkey` and `iterval` each read the object that the
map holds, thus a `for k, v := range m` over a map of struct values gave the
loop variables an alias of the entry. Read decision 255.

`append` needs a copy of its own. The path that grows writes each element into a
new array, and Go copies each 1 there. `elemAssign` of `internal/emit/call.go`
writes that argument, and it gives an empty text for an element type that needs
no copy. The spread form copies each element of the source, because it holds no
call site where stage 4 could copy the argument. Read decision 256.

## The shadow set of a package name

A Dart class holds each method and each field of a Go struct type as a member.
A package level name of the same Dart name is invisible inside such a class,
thus `topName` of the emitter adds the suffix `$f` to it.

The set holds each method name and each field name of each struct type of the
package. `ccc[p.ccc]` of `vendor/golang.org/x/text/unicode/norm` reads the
package variable `ccc` and the field `ccc`, and Dart read the field 2 times
before the field names joined the set. The set comes from the type information,
thus the emitter computes the same set for the package that it compiles and for
a package that it imports. Read decision 305.
