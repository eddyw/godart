# Suspension and the Go scheduler

Gart uses resumable frames for Go calls that can block.
Dart functions retain their ordinary return types. A scheduler flag indicates
suspension, and the immediate return value is not the completed Go result.

## Compiler contract

1. Block analysis propagates possible suspension through the call graph.
2. Normalization creates state boundaries around calls that can block.
3. Emitted callers observe the scheduler flag after such calls.
4. A suspended caller saves its live values and continuation state.
5. The stack unwinds to the scheduler without a logical Go return.
6. A wake resumes the saved chain and supplies the completed result.

A call that completes synchronously does not need a suspended frame.
State transitions must preserve evaluation order and avoid repeated side effects.
A resumed result comes from the completed frame, not its initial placeholder.

## Deferred calls and termination

An unwind is distinct from a return, panic, or `runtime.Goexit`.
Deferred calls run at the logical exit. A deferred call can itself block.
Returns inside nested control flow must reach the same deferred-call path.

Panic and recover retain the appropriate deferred-call context across suspension.
`GoStop` ends a goroutine through its exit path. The scheduler distinguishes
normal completion, panic, and goroutine termination when it reports completion
to a Dart host.

## Host event loop

Go goroutines share 1 Dart isolate. Channels, timers, and asynchronous host
operations wake suspended goroutines through the scheduler.
A ready queue must not indefinitely prevent a required Dart event-loop turn.
A busy Go loop without a yield can still monopolize the isolate.

The host can observe goroutine completion without treating all Go functions
as Dart futures. `iter.Pull` uses the same coroutine machinery.

## Synchronous callbacks

A host callback can require an immediate result or a valid temporary resource.
Named `//gart:sync` function types express that contract.
The compiler rejects a possibly blocking value in those positions.
Flutter builders and paint callbacks require this guarantee.

## Maintenance

Keep changes to block analysis, normalization, emission, and scheduler behavior
consistent. Relevant tests cover channels, select, nested returns, deferred calls,
panic/recover, Goexit, reflection calls, and synchronous callback rejection.

See [architecture](design.md) for the pipeline and
[phase measurements](compilation-time.md) for performance methodology.
