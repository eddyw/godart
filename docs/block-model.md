# The block model

This file holds the unwind model of a function that can block, its 4
measurements, and the benchmark support that measures it. Read it before a
change of the scheduler, of the flat state machine, or of a channel operation.

`docs/design.md` holds the rest of the architecture.

## The GopherJS block model

The user asked for this research on 2026-08-28, and the user took the model on
2026-08-29. The work is done, and `make ci` passes. GopherJS compiles Go to
JavaScript, and JavaScript also gives no blocked call.

The model of GopherJS:

1. A block analysis marks each function that can block, with a fixed point over
   the call graph. It stays conservative for an interface method and for a
   function value. Stage 3 of gart already does the same work.
2. A function that can block loses its structured control flow. The body becomes
   a flat state machine: `s: while (true) { switch ($s) { case 0: ... } }`. A
   loop, an `if`, a `switch`, and a labelled branch each become `$s = N;
   continue;`. GopherJS shares this flatten pass with `goto`.
3. Each call site that can block is a state boundary. The callee returns a frame
   object with a `$blk` field when it blocked. The caller tests that field and
   leaves the machine with `break s`. Then the caller writes its own locals and
   its state number into a frame object, and it returns that object.
4. The stack unwinds frame by frame up to the scheduler. The scheduler keeps the
   `$blk` thunk of the deepest frame. When the channel is ready, it calls that
   thunk, each frame re-enters at its saved case, and the stack rewinds.
5. `defer` pushes an entry to a `$deferred` array on the frame. `$callDeferred`
   runs at the real return, and not at an unwind.

The fast path allocates nothing. A call that does not block is a plain
synchronous call, plus 1 state assignment and 1 test.

A measurement of 4 Dart forms of the chain of 20 frames, on an ahead of time
build. The chain reads a buffered channel that already holds a value. Thus no
frame really blocks:

| Dart form | Cost |
|---|---|
| Synchronous, with no channel | 26 ns/op |
| `Future<int> f() async` with `await` at each frame | 2570 ns/op |
| `FutureOr<int> f()` with a test of `is Future` and `.then` | 90 ns/op |
| The GopherJS model with a flag and a frame class | 59 ns/op |

The GopherJS form is a factor of 44 faster than the `async` form, and a factor
of 1.5 faster than the `FutureOr` form. The extra cost over the synchronous
chain is about 1.6 nanoseconds for each frame. The GopherJS form also holds the
channel work that the synchronous chain does not hold.

Dart is statically typed, thus a function that returns `int` cannot return a
frame object. The prototype sets a flag on the scheduler and returns the zero
value. The caller tests that flag after each such call. This form keeps each
static type, and it needs no cast.

What such a change touches:

- Stage 4 needs a general flatten pass. `internal/normalize/goto.go` holds 147
  lines, and it splits only the top level labels of a body.
- Stage 5 writes the machine and a frame class for each function that can block.
  It also writes the restore preamble and the test after each such call site.
- `defer` and `panic` need the frame. They use `try` and `finally` today, and an
  unwind is a plain return.
- The channel, `select`, the mutex, the wait group, and `time.Sleep` each lose
  their `Future`. The scheduler holds the frame of a blocked goroutine.
- The isolate worker keeps a real `Future`. The scheduler wakes the goroutine
  when that future completes, thus the model still fits.
- The network access lands after the model change. `dart:io` gives `Socket`,
  `ServerSocket`, and `HttpClient`, and each one is async. A frame of the
  runtime waits for the future, exactly as `$worker` does.
  `RawSynchronousSocket` also exists, and it stops each goroutine of the
  isolate, thus it stays out.
- Each golden file of a function that can block changes 1 time.
- A reader can compare the emitted Dart of a function that can block with its Go
  source today. A flat state machine ends that property.

The model also opens a path to `iter.Pull`. A saved frame is a coroutine, and
decision 40 keeps `Pull` out because Dart gives no such primitive.

What the work gave, against the plan above:

| Chain of 20 frames, ahead of time build | Cost |
|---|---|
| Synchronous, with no channel | 26 ns/op |
| The `async` form of Dart, the model before | 3900 ns/op |
| The unwind model, with a channel of its own | 189 ns/op |
| The unwind model, with a channel of the package | 143 ns/op |

The unwind model is a factor of 21 faster than the `async` form. The prototype
gave 59 ns/op. The real output pays more for the real channel class and for the
zero value of each variable of a frame.

The 4 points that the prototype did not cover, and their answers:

1. `defer` and `panic`. The frame keeps the defer stack. A `return` runs the
   deferred calls in front of it, thus a deferred call can still write a named
   result. A `catch` block runs them again on a panic, and the unwind path
   stands outside that block. A deferred call must not block, thus stage 4
   rejects such a call. `TestRejectBlockingDefer` proves the rule.
2. A closure that captures a local. A resume reads each local from the frame
   into a new call, thus the closure of the first call reads the old value. The
   normalizer gives such a variable a box, and the frame keeps the box. A struct
   and an array need no box, because an assignment to such a variable writes the
   fields of the same object. `internal/normalize/box.go` holds the pass.
3. A function literal that can block. Such a literal becomes a local function,
   because the frame must name it. The frame keeps the function in a field, and
   `resume` calls it.
4. The zero value of a variable of a frame. The machine gives each variable its
   zero value at the start of the call, because the unwind can stand in front of
   the first write. A nil channel then costs 1 object for each such variable,
   thus the package holds 1 nil channel for each element type.

## The benchmark support

The user asked for `gart test -bench` on 2026-08-28. The value is a regression
check of the Dart output over time.

The work is done. `gart test -bench <text>` runs each `BenchmarkXxx` function.
`-benchtime <duration>` sets the target, and `-benchcount <count>` fixes the
count of iterations.

- `testing.B` holds `N`, the timer, `SetBytes`, `ReportMetric`, and `Run` for a
  subbenchmark. `b.RunParallel` stops the benchmark with an error.
- The search follows `go test`. The first run takes 1 iteration, and each next
  count comes from the time of the last run.
- The driver runs `dart compile exe` first. The warm up cost of the Dart VM
  hides the result of a run on the VM.
- The runtime gives `$monoNanos` through `dart.MonoNanos`. The `Stopwatch` of
  Dart ticks each nanosecond, and the wall clock of `DateTime` holds only
  microseconds.
- The result line holds the format of `go test`, thus `benchstat` reads it. The
  suffix of the name is the count of isolates, thus it is always 1.
- The number measures the Dart output, and not Go. A comparison of 2 gart runs
  finds a regression, and that is the purpose.
- `-benchmem` stays out. The Dart VM gives no count of allocations for 1
  operation.
- The Dart compiler removes a body when no code reads the result. The benchmark
  must write the result to a package variable. `testdata/tests/basic` and
  `testdata/tests/async` hold that pattern.

The measurement of the async model, from `testdata/tests/async`:

| Chain of 20 frames | Go | godart |
|---|---|---|
| Synchronous | 8.5 ns/op | 28.9 ns/op |
| Asynchronous | 51.5 ns/op | 3900 ns/op |

An async frame of Dart costs about 200 nanoseconds. Each `await` schedules a
microtask. The block analysis keeps a synchronous path for a function that
cannot block, thus only a chain that can block pays the cost.

## Facts of the block model

Verified on 2026-08-29:

- `make ci` passes with the unwind model. The end to end test compiles 39 Go
  programs and runs the tests of 7 modules.
- The benchmark of `testdata/tests/async` gives 26 ns/op for the synchronous
  chain and 189 ns/op for the chain that can block. The value before the change
  was 3900 ns/op.
- A chain with a channel of the package costs 143 ns/op, thus the channel of
  each call costs about 46 ns/op for 20 frames.
- `late final` on a field of the channel class gives the field its value on the
  first read. A nil channel then reads no queue and allocates none.
- The type of `<-ch` in a context of 2 values is already a tuple in
  `types.Info`. The element type must come from the channel type.
- A Dart switch statement needs no default case, and each case clause must end
  with a transfer. Each state ends with `continue`, with `return`, or with
  `break` of the machine label.
- A Dart record type fits an `as` expression: `$r!.resume() as (int, bool)`.
- `Chan<Never>` is a subtype of `Chan<int>`, because Dart generics are
  covariant. A shared nil channel of 1 element type needs no such cast.

Verified on 2026-08-29, for the package bytes:

- `bytes` needs 1 patch of `buffer.go` and no other change of the real source.
  The general flatten pass holds the `goto` label, and the method value of
  `unicode.SpecialCase` now writes a closure.
- `bytealg.LastIndexRabinKarp` reads `any(s).(type)` over a type parameter. The
  descriptor of the dictionary makes that switch give the right case.
- A dictionary of a predeclared type is 1 value for each type, and not 1 for
  each Dart type. `int8` and `int32` share each operation, and the description
  tells them apart.

Verified on 2026-08-29, for the last 3 tasks:

- `json.Unmarshal` needs `TypeDesc.make` for a slice and for a map. The Dart
  type of the element lives only in the emitted code, thus the descriptor must
  build the container.
- A value of the type `any` takes the types of Go: a bool, a float64, a string,
  a `[]interface{}`, or a `map[string]interface{}`. The runtime holds 1
  descriptor for each of the 3 shapes.
- A labelled branch that leaves the body of a range loop over a function needs 1
  flag for each target. A nested sequence carries the flag out 1 level for each
  call, exactly as a return does.
- The flatten pass of the machine reads the label of a loop from `ir.For.Label`.
  A labelled statement reaches the pass only for a goto target.


Verified on 2026-09-03:

- `$schedule` runs the run queue until it is empty, thus a goroutine that puts
  itself back on the queue starves the Dart event loop. `runtime.Gosched` did
  exactly that, and a program that spins on it while a `time.Sleep` stands hung
  for ever. The call now gives the event loop a turn when the run queue holds
  no other goroutine and a wait on the event loop stands. Read decision 310.
- `runtime.Goexit` throws the signal `GoStop` that `testing.T.FailNow` already
  used. The class moved into `panic.dart`, thus `$schedule` catches it: the
  goroutine ends and each deferred call of its chain runs first. Go keeps the
  program alive when the goroutine of main ends this way, thus a flag replaces
  the end of the program. Read decision 280.
- A `return` inside an `if`, a `switch`, or a `for` needs its own state when
  the deferred calls of the function can block. Stage 4 gave such a statement
  no state, thus stage 5 wrote a plain Dart return that left the defer stack
  untouched and no deferred call ran. Read decision 290.

Verified on 2026-09-04:

- `$go` gives the `Goroutine` that it starts, and a Dart host reads the end of
  that goroutine there. The 3 ends of `$schedule` each report it: the normal
  return, the catch of `GoPanic`, and the catch of `GoStop`. Read decision 316.
- A body that blocks gives its result through `Frame.resume` of the last wake,
  and not through the call of the body. The scheduler keeps that value, thus a
  host reads the result of a Go function of the type `func() bool`.
- The parameter of `$go` keeps the type `void Function()`. The type
  `Object? Function()` makes the analyzer report
  `body_might_complete_normally_nullable` for each closure that stage 5 writes,
  because the context type reaches a closure with no result.
