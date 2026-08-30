# Decisions

This file holds each decision that the work took, and each question that the
work answered. A number stays with its decision for the life of the project,
because other files name a decision by its number. A decision that a later
decision replaces keeps its number and carries a "Superseded" line.

## Decisions

The plan left these points open. The work took the decisions below.

1. The verifier accepts the address of any struct location, and not only the
   address of a struct variable. `&s[i]` and `&x.field` stay in the subset when
   the type is a struct, because the Dart reference model holds the alias.
2. An assignment to a struct location copies the fields in place with `$assign`.
   It does not replace the object. Thus a pointer to the location observes the
   new value, exactly as in Go.
3. A struct maps to a Dart class with a `$` suffix, for example `_point$`. An
   embedded field carries the name of its type, thus the suffix prevents a
   collision between the field and the class.
4. A local variable takes a name that is unique in the function. Go allows an
   inner name to shadow an outer name, and Dart does not.
5. A slice, a map, and a string hold their zero value as an object, and not as
   null. A pointer, a function value, and an interface value hold null.
6. A method of a named type that is not a struct becomes a top level function,
   for example `_Celsius$String`.
7. An interface method call reads the method table of the descriptor. The
   emitter takes a typed cast when the receiver is a simple name. It takes
   `Function.apply` in each other case, thus the receiver runs 1 time.
8. Package `main` writes `bin/main.dart`, and each other package writes
   `lib/<path>.dart`. A Go file becomes a Dart part file.
9. A map keeps the order of the writes. Go gives no order, thus a Go program
   must not depend on it. The parity file must list this gap.
10. A goto rewrite needs each label at the top level of the function body. A
    goto statement itself may stand at any depth.
11. Stage 1 loads the module 2 times. The first load finds the files of each
    standard library package, and the second load replaces them through
    `packages.Config.Overlay`. The overlay also adds `syscall/dart`, which the
    standard library does not hold.
12. A Go name that is equal to a Dart name of the emitted code takes a `$`
    suffix. `internal/name` holds the list.
13. Go 1.27 forbids a type parameter on an interface method. Thus a generic
    method never reaches a method table, and it maps directly to a Dart method.
14. A generic named type has no type descriptor, thus a value of such a type
    reaches no interface value. A value of a type parameter does reach one. The
    dictionary of the type argument holds the descriptor, and each predeclared
    type holds a dictionary of its own.
15. Each package runs the `$init` of its imports first. A flag stops a second
    run, thus the order follows the import graph.
16. Stage 2 rejects a complex number until the runtime class exists. The emitter
    wrote `Object?` and `null` for it before, and that output is a silent change
    of meaning. The rejection is temporary, and the plan keeps the runtime
    class.
17. `gart test` reuses the test packages of the go command through
    `packages.Config.Tests`. The compiler writes no build variant of its own.
18. The compiler replaces the entry point of the go command through the overlay.
    The go command fixes the import list of that package, thus the generated
    source imports only `testing` and the 2 test variants. An import of `fmt`
    from that source fails with `no metadata for fmt`.
19. A stub overlay replaces a package that the fixed import list drags in: `os`,
    `reflect`, and `testing/internal/testdeps`. A stub keeps the tree of such a
    package out of the type check. `overlay.Paths()` leaves a stub out, thus the
    surface does not hold it and stage 2 still rejects an import of it.
20. The cache key and the output path read `packages.Package.ID`, and not the
    import path. A test variant carries the import path of the package under
    test, thus the 2 collide on the path.
21. The output layout of a test build. A test variant goes to
    `lib/gart_test/<relative path>/`. The entry point goes to `bin/<relative
    path>_test.dart`. A package of the root of the module takes the name `main`,
    thus `bin/main.dart` and `bin/main_test.dart` stand next to each other.
22. The generated entry point takes the stable file name `gart_testmain.go`. The
    path of the go command lives in the build cache and holds a hash, thus that
    path breaks G5. The driver renames the file after stage 4.
23. `t.FailNow` and `t.SkipNow` throw through `dart.Stop`, and the harness holds
    the throw with `dart.Catch`. Go stops such a chain with `runtime.Goexit`.
    Each deferred function still runs, and the subset has no `recover`, thus the
    result stays equal.
24. Stage 2 rejects a `TestMain` function. Such a function replaces the entry
    point of a test binary, and the compiler writes that entry point. The rule
    reads the file name, thus it fires only for a `_test.go` file. Superseded by
    decision 46. M12 gave the surface `os`, thus the entry point can call
    `os.Exit(m.Run())`.
25. A benchmark stays out of the output, exactly as `go test` with no `-bench`
    flag. A fuzz target and an example with an output comment reach the output
    as a skipped test, because `go test` runs them. Silence for them is a silent
    change of meaning. The user asked for a `-bench` flag on 2026-08-28, thus
    the plan holds the work. Read the section "The benchmark support" of
    `docs/block-model.md`.
26. The driver writes the skip line of an unsupported function, and not the
    harness. A plain run hides the output of a binary that passes, thus a line
    from the harness would not reach the user.
27. The `-v` flag and the `-run` flag reach the binary as arguments, and not as
    a compile time constant. Thus a change of a flag needs no second compile.
    `dart.Args()` gives them, and the entry point of each program now stores the
    arguments of the process.
28. `-run` reads plain text and not a regular expression. The subset holds no
    `regexp` package. A plain pattern gives the same result as the regexp of `go
    test`, and an anchor or a character class does not. Superseded. The surface
    holds `regexp`, and the entry point gives the harness
    `testing.MatchString`. `-run` and `-bench` each read a regular expression
    with the rule of `testing/match.go` of Go. Read decision 124.
29. `testing.B` and `testing.F` exist with a body that panics. A test file
    declares a benchmark and a fuzz target, and that body must type check. No
    path reaches such a method, thus the panic never runs. The benchmark work
    gives `B` a real body.
30. An unnamed struct type reaches the output as a Dart class. The name comes
    from a hash of the text of the type, thus 2 equal Go types give 1 class.
    Stage 2 rejects such a type in an exported declaration. The compiler writes
    1 class into each package that reads it, and Dart holds the 2 apart.
31. A type parameter reads its core type. Go gives a parameter a core type when
    each term of its type set shares 1 underlying type. `internal/typeset` holds
    the walk, and emit, verify, and normalize all read it.
32. The Dart type of a type parameter with a composite core type is the Dart
    type of that core type. `S ~[]E` gives `Slice<E>`. A basic core type keeps a
    Dart type parameter, because `~int` and `~string` give 2 different Dart
    types.
33. A type parameter with more than 1 Dart type reads each operation from its
    dictionary. `Dict.order` gives the order comparison. Stage 2 rejects an
    index, a slice expression, a range loop, `len`, and `cap` on such a
    parameter.
34. The address of an array stays in the subset. The Dart form of an array is a
    List, and a List already holds a reference, thus such a pointer needs no
    box.
35. The cache key holds a hash of the compiler binary. The version string alone
    is not enough, because the compiler changes many times inside 1 version.
36. Each part file of the output ends with `.part.dart`. A package may hold a
    file whose name is the name of a package inside it. `math` holds `bits.go`,
    and the package `math/bits` also exists.
37. A native package reads the real Go source. A patch replaces 1 file of such a
    package by its base name. `math` needs 7 patches, because the go command
    selects an assembly file for each architecture.
38. A benchmark runs on an ahead of time build. `dart run` starts a just in time
    compiler, and its warm up cost hides the result.
39. A range loop over a function writes the body as a Dart closure. A `break`
    gives `return false`, and a `continue` gives `return true`. The normalizer
    holds a stack of the constructs that a branch leaves, thus a nested loop and
    a nested switch keep their own break. A `return` and a labelled branch each
    write a flag and give false to the sequence, and the code after the call
    reads that flag. Stage 4 rejects a `goto` out of the body.
40. `iter.Pull` stays out of the surface. It needs a coroutine of the Go
    runtime, and the Dart VM gives no such primitive. `Seq` and `Seq2` are
    function types, thus the overlay holds only the 2. Superseded by decision
    114. The sequence runs on a goroutine of its own, and 2 channels with no
    buffer hand the control back and forth.
41. `os` reaches the file system through `syscall/dart`, and each operation is
    synchronous. Go blocks the goroutine at the same point, thus the block
    analysis of stage 3 stays simple. `dart:io` gives a synchronous form of each
    call that the package needs.
42. Each signature of `os` holds the form of the standard library, and the
    permission argument reaches no operation. A Go source then compiles the same
    way with the go command and with gart. `ReadNames` replaces `ReadDir`,
    because a `DirEntry` holds a method that reads the metadata again.
43. A pointer to a value that is not a struct holds a `Box<T>` of 1 field. The
    normalizer collects each variable whose address the code takes, and only a
    variable holds a box. A field, an element, and a call result each live
    inside another object.
44. A top level name of a package takes the suffix `$f` when a class of the
    package holds a method of that name. The emitter derives the set from the
    type information, thus a package that imports it computes the same names.
45. `encoding/json` walks the type descriptor. The plan named a code generator,
    and the descriptor already holds the struct tag, a read, and a write of each
    field. Read `docs/history.md`.
46. A `TestMain` function replaces the entry point of a test binary. The
    compiler writes a call of `testing.MainWith`, and Go passes the result of
    `m.Run` to `os.Exit` when `TestMain` returns.
47. A `return` inside the body of a range loop over a function writes each value
    into a temporary variable. It sets a flag, and it gives false to the
    sequence. A nested sequence carries the flag out 1 level for each call.

48. A function that can block keeps the Go result type. It takes 1 more
    parameter, and that parameter holds the frame of a resume. Thus the Dart
    type of a function value and of an interface method no longer depends on the
    block state.
49. Dart is statically typed, thus a function that returns `int` gives no frame
    back. The runtime holds the frame in `$blocked`, and each caller tests that
    variable after a call that can block.
50. The flatten pass of stage 4 runs for a function that can block and for a
    function that holds a goto statement. Only a statement that holds a block
    operation, a goto, or a branch that leaves it becomes states of its own.
    Each other statement keeps its Dart form.
51. A deferred call that can block is outside the subset. The deferred calls run
    at the return of a function, and the machine keeps no state at that point.
    Stage 4 holds the rule, because stage 2 reads no block analysis. Superseded
    by decision 72. The chain of the deferred calls holds 3 states of its own.
52. A local of a machine that a closure reads and that the code writes takes a
    box. A resume reads each local from the frame into a new call, thus the
    closure and the frame must share 1 value. A struct and an array need no box,
    because an assignment to such a variable writes the fields of the same
    object.
53. A function literal that can block becomes a local function. The frame of
    such a literal keeps the function in a field, because a literal holds no
    name at package level.
54. Package main runs as the first goroutine. The program ends when that
    goroutine returns, and the runtime reports a deadlock when no goroutine can
    run and no timer waits.
55. The package holds 1 nil channel for each element type. The zero value of a
    channel type reaches each variable of a frame, and a nil channel holds no
    state that a program can change.
56. The network access takes a socket, and not an HTTP client. `syscall/dart`
    gives `DialTCP`, `ConnRead`, `ConnWrite`, and `ConnClose`, and a handle is
    an index of a list of the runtime. An `HttpClient` of `dart:io` would hold
    the whole protocol, and the Go code would then read no stream. A `net`
    overlay and an HTTP client each reach the same 4 functions later.
57. `$await` is the 1 primitive that waits for a Dart future. It writes an
    `EventWait` frame, it holds 1 wake source of the scheduler, and it takes a
    function for the cause of a failure. Each operation of `dart:io` that gives
    a future reads it, thus no such operation writes a frame of its own.
58. A read of a connection holds the rest of a packet in the connection. A Dart
    socket gives a stream of packets, and a Go read takes only the count that
    the slice holds.
59. A listener and a connection hold 2 separate handle tables. A Dart
    `ServerSocket` and a Dart `Socket` share no type, and Go also holds 2 types.
    A close of a listener cancels the iterator of the server socket, and each
    connection of the listener stays open.
60. The first load asks for each overlay package by its path. An overlay holds
    imports of its own, and the real source of the module reaches none of them.
    The load then held the overlay file of such a package next to its real
    files. The list depends only on GOROOT and on the build tag, thus 1 load for
    each process is enough.
61. The `net` overlay exports no `TCPAddr` and no `IP`. A type assertion of an
    `Addr` then gives a compile error, and not a wrong value. A partial copy of
    such a type would give a silent change of meaning.
62. `time.AfterFunc` reads a Dart timer of the runtime. The timer holds 1 wake
    source, and the run puts the Go function on the run queue as a new
    goroutine. Go also runs such a function on a new goroutine, thus the
    function can block.
63. Each operation of `sync/atomic` is a plain read or a plain write. All Go
    code runs in 1 isolate, and a goroutine gives up control only at a block
    operation.
64. Each pass of a loop holds its own copy of the variable that the first clause
    declares. A Dart for loop gives that copy when the declaration stands in its
    header. A boxed variable takes a new box in front of the third clause. Read
    the section "The loop variable of Go 1.22" of `docs/design.md`.
65. A literal inside a machine reads a copy of each box that it holds. The
    machine keeps 1 local for a variable, and it writes a new box at each pass.
    Thus the literal of a pass must hold the box of that pass. The copy stands
    in a block in front of the statement.
66. An unnamed struct type takes a file of its own, `lib/gart_anon/<name>.dart`.
    The content reads the shape only, thus each package that holds the shape
    writes equal bytes and Dart gives them 1 class.
67. The name of the class of a shape covers the text of the type and the library
    of each field type. A test variant of a package holds a library of its own.
    Thus a shape that names a type of such a variant takes a class of its own.
68. A shape with an unexported field keeps its class inside the package. Go
    gives such a field the identity of its package. Dart holds a name that
    starts with an underscore private to 1 library. Stage 2 rejects only such a
    shape in an exported declaration.
69. A shape that names a type of a package that no library holds keeps its class
    inside the package. Package main is such a package.
70. A type name of a descriptor and of a type test comes from the type behind an
    alias. An alias carries the name of its own declaration, thus a test against
    an alias of a shape gave false.
71. `time.Time.String` holds a calendar. The user took the decision on
    2026-08-29. The conversion from a count of nanoseconds to a date follows the
    algorithm of Howard Hinnant. The intrinsic `dart.ZoneOffset` gives the
    offset and the name of the local zone. A Time holds no value of the
    monotonic clock, thus the text ends with the name of the zone.
72. A chain of deferred calls that can block holds 3 states of its own. The try
    block of a panic stands inside the loop of such a machine, thus a panic also
    reaches the chain.
73. Stage 4 writes the source position of each log call of the testing package
    into `testing.LogPos`. No goroutine gives up control between the statement
    and the call.
74. The description of an instance of a generic named type stands at the point
    of the use. The name holds each type argument, thus a type assertion
    compares the full name.
75. A package variable that holds a box carries the box in its declaration. The
    set covers the whole package, and stage 2 rejects the address of an exported
    package variable, because the archive holds no such field.
76. `runtime.GOOS` is a variable. Dart names the system at the start of the
    program, thus the output of 1 build runs on each system that the Dart VM
    holds. `runtime.GOARCH` is the constant `dart64`.
77. The key of a method table holds the name and the signature of the method. Go
    compares both, thus an interface with `Unwrap() error` and an interface with
    `Unwrap() []error` stay apart.
78. `errors.As` writes through the box of the target. A target of a pointer to
    an interface type panics, because the runtime library holds no method name
    of such a type.
79. The `time` overlay holds no zone database of its own. It gives 2 locations:
    `UTC` and `Local`. `Local` reads the offset and the name of the zone from
    Dart at each instant. `Date` with `Local` reads that offset 2 times, because
    its argument is a wall clock value and not an instant.
80. `Time.String` calls `Format`, thus the 2 hold 1 rule. The layout scanner
    holds `nextStdChunk` of Go byte for byte, thus each chunk of a layout gives
    the text of Go.
81. The zero value of a `Time` is the start of the Unix epoch, and not January 1
    of the year 1. A Time holds 1 count of nanoseconds, thus its range is the
    year 1678 to the year 2262.
82. `reflect.Type` holds a handle of a type descriptor of the runtime. Thus
    `Type.Elem` names a type that no value holds, and `reflect.Zero` builds the
    zero value of such a type.
83. `reflect.Value.Elem` gives a Value that keeps the pointer, thus `Set` writes
    through it. A Value that `Elem` did not read holds no location, and a write
    of such a Value panics.
84. Each predeclared type descriptor of the runtime holds a zero factory. A
    tear-off of a top level function is a constant expression of Dart, thus each
    descriptor stays `const`.
85. `os.Setenv` and `os.Unsetenv` write a map of the overlay. The Dart VM gives
    no write to the environment of the process, and the subset starts no other
    process. Thus no reader stays outside the map.
86. `t.Context` gives a context that `runOne` cancels in front of the cleanups
    of the test. Go cancels it at the same point.
87. A deadline of the network holds 2 absolute times for each handle. The
    runtime applies the limit at the start of an operation, and the source
    future stays open. Thus the next call of the same operation takes that
    future again and loses no byte and no datagram.
88. A datagram socket of a dial drops a datagram of another source. Dart gives
    no connect of a datagram socket, thus the runtime holds the remote address
    and filters each datagram.
89. `Dialer.DialContext` runs the dial on a goroutine, and a select takes the
    first of the dial and the context. A dial that ends after the context closes
    its own connection on a second goroutine.
90. The scheduler runs a list of releases when package main returns. Go ends the
    process at that point, and an open socket of Dart keeps the isolate alive.
    The network library adds 1 release.
91. A liveness analysis of a frame stays out. A hand edit of 1 frame class
    measured the gain, and the upper bound stands inside the noise of the
    baseline.
92. `math/rand` and `math/rand/v2` read the real Go source, and a patch replaces
    `rand.go` of each one. The algorithm stays in a file that the compiler reads
    whole, thus a seeded sequence gives the values of Go.
93. The top-level functions of each form of `math/rand` read a source that the
    clock seeds. Go 1.24 made the top-level `Seed` a no-op, and the operating
    system seeds the source of Go, thus no program can depend on the values.
94. `%v` and `%g` of a `float32` write the shortest decimal text that reads back
    as the same 32-bit value. A Dart double holds a 32-bit value with the
    precision of 64 bits, thus the shortest text of 64 bits holds more digits.
95. `os.OpenFile` replaces the content of a file only for `O_TRUNC`. Dart holds
    no mode that reads and writes and stands at the start, thus such an open
    takes `FileMode.append` and moves the position back.
96. A type parameter whose type set holds only integer terms takes the Dart type
    `int`. The Dart type parameter stays in the declaration, and no value reads
    it. `Dict` holds a mask of the width, and the emitter writes that mask after
    each arithmetic operation and after each conversion to the parameter.
97. `typeset.Describe` gives 1 record of the type set, and `Core` stays for the
    core type. The record names 6 marks: Numeric, Integer, Float, Narrow, Wide,
    and Plain. Stage 2 and stage 5 both read it, thus the 2 stages agree on the
    Dart form of each type set.
98. Stage 2 rejects an arithmetic operation, a conversion, and the builtin
    `print` of a type parameter that holds no single Dart form. A conversion to
    an interface stays inside the subset, because the dictionary holds the
    description.
99. `time.Parse` is a port of the parse of Go, with the same `ParseError` and
    the same text of each message. `FixedZone` holds an offset that the local
    zone does not carry, thus a text that names an offset reads back to the same
    instant.
100. The zone of a name, as in `MST`, reads the name that Dart gives for the
     local zone at the instant. A name that does not match gives a zone of a
     fixed offset, and `GMT` names its own offset. Go reads its zone database. A
     name of another zone gives an offset of 0 in both.
101. A method with a pointer receiver on a struct type holds its body in a top
     level function. The class holds a member that calls it. The Dart form of a
     pointer to a struct accepts null, and Dart calls no member of null. Go runs
     such a method with a nil receiver.
102. A method of a generic type stays a member of its class. A top level
     function would need the type parameters and the dictionaries of the
     receiver type.
103. A function declaration with no body writes a Dart body that panics, and
     stage 2 rejects a read of the name. An assembly file holds the code of such
     a function, or a go:linkname directive only exports the name.
104. An unexported field of a type of another package reaches no accessor of its
     descriptor. A Dart library holds each name that starts with an underscore.
105. `unique` holds each value for the life of the program. Go frees a handle
     through a cleanup of its runtime, and the subset holds no such cleanup.
106. Each setting of `internal/godebug` holds its default. The Dart runtime
     reads no GODEBUG variable.
107. An instance of a generic function that reaches a value takes a closure. The
     closure gives the dictionary of each type argument, and it passes each
     parameter through. `ir.FuncInst` holds the node, and a call unwraps it
     again. The block analysis keys such a value by the instantiated signature.
108. A request with no `Content-Length` and no `Transfer-Encoding` holds no
     body. The overlay read such a body to the end of the connection before, and
     a connection that stands open then blocked the next request.
109. The client gives a connection back to the pool only after the body reaches
     its end. A Close that stands before the end ends the connection. Go decides
     the same, and the race of its read loop can still reuse such a connection.
110. `compress/flate`, `compress/gzip`, and `compress/zlib` are native. The real
     source of each one needs no patch, and the Dart output writes the same
     bytes as Go for each level.
111. The Dart name of an unexported type holds no `_` prefix. An exported
     variable can hold a type that Go does not export, and a second package then
     names that type. A field keeps its prefix.
112. The emitter writes the type argument of `$arrayIndex` and of `$arraySet`.
     Dart reads the context of a call for a type argument, and an operand of `-`
     gives `num`.
113. The recorder of `httptest` names the type `text/plain; charset=utf-8` for
     each body. Go reads the first 512 bytes of the body for it, exactly as the
     server of the overlay does.
114. `iter.Pull` runs the sequence on a goroutine of its own. 2 channels with no
     buffer hand the control back and forth, and the 2 goroutines never run at
     the same time.
115. The signature key of stage 3 holds a wildcard for each type parameter, and
     it keeps each other part of the type. A call of a generic function joins
     the key of the generic signature with the key of the instance. Thus a
     closure of a generic function meets the call site of it.
116. `resume` is a reserved name. The frame class of a function that can block
     holds a member of that name.
117. `reflect.Type.Size` of a struct follows the rules of a 64-bit machine. Each
     field stands at an offset that its alignment allows. The size rounds up to
     the alignment of the struct. A struct whose last field holds 0 bytes takes
     1 more byte.
118. `Type.Name` gives an empty text for a composite type. The scan of the text
     read the name of the element type before, as in `[4]main.mixed`.
119. `GoString.toDartString` gives a byte that holds no rune of UTF-8 1 code
     unit of the range U+DC80 to U+DCFF, and `fromString` gives the byte again.
     The whole format path holds a Dart String, thus 1 escape covers each verb,
     each `Stringer`, and each container.
120. `$stdout` and `$stderr` write the bytes of the string. A step through the
     encoder of Dart replaces a byte that holds no rune.
121. The text of a type descriptor follows `reflect`, and not `go/types`. An
     unnamed struct type and an unnamed interface type each hold 1 space after
     the keyword. `dartText` escapes the text for the Dart literal, because a
     struct tag holds a quotation mark and a backslash.
122. `runtime.GC` allocates a block of garbage and gives up control. The
     collector of the Dart VM runs for the garbage, and a cleanup runs on a
     later turn of the event loop. Dart holds no call that forces a collection,
     thus a cleanup can stay for a later call.
123. The source hash of a package holds a digest of the block analysis of that
     package. The analysis reads the whole module. Thus a new function of a
     package that this package does not import can change the form of a call
     through a function value. `TestCacheBlockDigest` proves the rule.
124. 1 file holds the rule of `-run` and of `-bench`, and the compiler and the
     harness both read it. `internal/overlay/std/testing/match.go` carries no
     build tag. Thus the go command builds it as a package of the module, and
     the overlay gives the same file to a test binary. `internal/driver` passes
     `regexp.MatchString` to it, and the entry point of a binary passes the
     function of the surface. The rule comes from `splitRegexp` of
     `testing/match.go` of Go: the split reads each `|` outside a bracket group
     and outside a parenthesis group, and then each `/` outside such a group.
     `Run` of `T` and `Run` of `B` each apply the filter to the full name of the
     subtest, exactly as Go does.
125. `Run` of `M` writes `testing: warning: no tests to run` when the filter
     keeps no test and the arguments hold no `-test.bench`. The driver reads
     that line and writes ` [no tests to run]` after the line of the package.
     The go command reads the same line of a test binary and writes the same
     mark.
126. Stage 3 holds a method that reaches a function value in the group of the
     signature of that value. A method value binds the receiver, and a method
     expression takes the receiver as its first parameter, thus the key comes
     from the type of the selector and not from the type of the method. Before
     this rule, `valueSignature` dropped each method, thus a call through
     `d.step` of `compress/flate` never gave up control and the bit state of the
     writer took a stale value. `testdata/programs/methodvalue` proves the rule.
127. A method value of an interface value reaches the method table of the
     descriptor. The closure holds the value, and each call of it reads the
     table with `$method`. Before this rule, stage 5 wrote the name of a method
     function of the interface type, and the Dart build stopped with "Method not
     found".
128. A test binary runs in the directory of the package under test, exactly as
     the binary of `go test` does. Thus a test file reads a relative path such
     as `testdata/x.txt`. The driver gives the entry point an absolute path for
     that reason, because the Dart VM reads the package configuration next to
     that path.
129. The surface holds `internal/testenv` with 2 functions. `Builder` gives an
     empty name, and `SkipIfOptimizationOff` does nothing. The real package
     starts a process of the go command, and gart holds no such access. The
     tests of `bytes`, of `compress/flate`, and of `compress/zlib` each read
     only those 2 functions.
130. `net.Error` holds `Temporary`, exactly as Go declares it. Go marks the
     method as deprecated and keeps it in the interface, thus a program that
     calls it must compile.
131. The rune decoder of the runtime library rejects each sequence that Go
     rejects. An overlong form, a surrogate, a value above U+10FFFF, and a start
     byte above 0xF7 each give U+FFFD with a size of 1. Before this rule, a
     range loop over `"\xE0\x9F\xBF"` gave U+07FF, and `TestDecodeInvalidSequence`
     of the Go repository found it. `need` of the `unicode/utf8` overlay held
     the same gap for 0xC0, 0xC1, and 0xF5 to 0xF7.
132. A type descriptor holds the function type of each exported method, under
     the key of the method. `reflect.Value.Method` gives a func value from it,
     and `Value.Call` of that value reads the parameter types and the result
     types. The table costs 10337 bytes on `testdata/programs/args`, which is
     3.1 percent. A bound method travels as a `BoundMethod` pair, because a
     Dart closure holds a fixed arity, and `$callValue` unwraps the pair.
     `testdata/programs/reflectmethod` proves the API.
133. The order of the methods of `reflect` follows the name, exactly as in Go.
     The key of a method table holds the name and then a parenthesis, and the
     parenthesis stands below each letter and each digit, thus a sort of the
     keys gives the order of the names.
134. The address of an exported package variable stays allowed inside a main
     package and inside an external test package. No import can name such a
     package, thus the box of the variable never crosses a package boundary.
     `rand_test` of the Go repository takes `&Sink`, and the old rule rejected
     it.
135. `os.Pipe` gives an error. The Dart side holds no pipe of `dart:io` that a
     `*File` of the overlay can wrap. `TestUpdateExample` of `math/rand/v2`
     compiles with it and skips at run time.
136. `reflect.Value.MapKeys` gives the keys in the order of the insertion, and
     Go gives a random order. `docs/parity.md` holds the gap.
137. Stage 3 holds a value flow in place of the signature groups.
     `internal/analyze/flow.go` follows each function value through each
     assignment, each argument, each result, and each struct field, thus a
     call through a value reaches only the functions that flow to it. A value
     that leaves this net falls into the escape group of its signature, and a
     call outside the net reads that group, thus an untracked construct keeps
     the old behavior. Before this rule, `iter.Pull` in a program marked each
     `iter.Seq` closure of the program. The output of
     `testdata/programs/iteruse` fell 19 percent with this rule, and the
     output of `slicesuse` fell 2.3 percent. 2 measurements of 2026-08-31
     first closed the 2 cheap paths: a reachability pass and an owner key each
     closed nothing.
138. The body of a range loop over a function is a function of stage 3, and it
     becomes a local state machine of stage 5 when it blocks. The box rules of
     stage 4 treat the body as a function literal. Before this rule, a block
     operation inside such a body was lost: the plain closure ignored the park
     of a channel, ran on with a junk value, and `testdata/programs/rangeblock`
     gave 6 in place of 66.
139. The name of the descriptor of a function type holds no parameter name.
     A type assertion compares the name, the text of a declared signature
     holds the parameter names, and the text of an assertion target holds
     none. Before this rule, `x.(func(int) int)` failed on a value whose
     declaration named the parameter.

## Answered questions of 2026-08-31

- Does Dart hold a coroutine for `iter.Pull`? No. The sequence runs on a
  goroutine of its own, and 2 channels with no buffer hand the control back and
  forth. Read decision 114.
- Can `runtime.GC` force a collection? No. Dart holds no such call. An
  allocation of a block of garbage provokes one, and a cleanup runs on a later
  turn of the event loop. Read decision 122.
- Does a reachability pass cut the cost of `iter.Pull`? No. Such a pass reads
  the same groups of the signature. Read the workstream doc.
- Can the compiler and a test binary share 1 source file? Yes. A file of the
  overlay with no build tag reaches both. Read decision 124.
- Does a call through a function value always give up control? No. A method
  that reaches such a value stood in no group of stage 3 before 2026-08-31.
  Read decision 126.
- Does the real source of `compress/flate` stay inside the subset? Yes. The
  package needs no patch, and the Dart output writes the same bytes as Go.

## Answered questions

- Does the compiler need a source map of the emitted Dart? Yes. The user took
  the decision on 2026-08-30. The map costs 7 percent of the bytes of the
  output. Read the section "The source map" of `docs/design.md`.
- Does `net/http` reach the surface with the real source of Go? No. That source
  holds 53000 lines and reads a vendored HTTP/2 stack, `mime`, `net/textproto`,
  and `compress/gzip`. The overlay holds an HTTP/1.1 client and an HTTP/1.1
  server of 1500 lines.
- Can 2 select statements meet each other? Yes. Each channel now holds a waiter
  of a select statement, and the first channel that reaches it writes the number
  of the case. `io.Pipe` needs that shape.
- Which form does a pointer to a type parameter take? 1 form that holds both
  forms of a pointer, and `$deref` reads it. The Dart form of a pointer follows
  the type argument: a pointer to a struct is the object itself.

- How does a method run with a nil pointer receiver? The body stands in a top
  level function, and the class holds a member that calls it. Read decision 101.
- Which packages of the standard library reach the surface with no new compiler
  feature? 35 of the 66 read the real Go source. Read
  `internal/overlay/native.go`.
- Which form does a type parameter of an integer type set take? The Dart type
  `int`, and `Dict` holds a mask of the width. Read the section "The Dart form
  of a type set" of `docs/design.md`.
- Which form does `time.Parse` take? A port of the parse of Go, with the same
  `ParseError`. `FixedZone` holds an offset that `Local` does not carry.
- Does a frame need only the variables that live at the point of the block? No.
  A hand edit of 1 frame class measured the gain on 2026-08-30, and the upper
  bound stands inside the noise of the baseline. Read decision 91.
- Does `math/rand` need the exact algorithm of Go? Yes, and each form now reads
  the real Go source. Read the section "The 2 forms of math/rand" of
  `docs/library.md`.
- Which form does `flag` take? The overlay. The work of 2026-08-29 gave it.
- Does the calendar of the other direction reach `time.Date`? Yes. Read the
  section "The calendar of the time overlay".
- Does `reflect` reach a read and a write through a pointer? Yes, with 6
  intrinsics and a handle of a type descriptor. Read decision 82.
- Does the network reach UDP and a Unix socket? Yes, with 9 new interop
  functions. TLS gives a client, and each field that Dart cannot honor stops the
  program.
- Can a background session write into the checkout? Yes. The guard reads
  `worktree.bgIsolation` from the settings of the project directory, and that
  file now holds the key. The user holds no worktree for this project.

- Which form does a source position of a log line take? Stage 4 writes the
  position into a variable of the harness. Read the section "The source position
  of a test log line" of `docs/design.md`.
- Which form does a descriptor of a generic named type take? Option 1, at the
  point of the use.
- Does `time.Time.String` take a calendar or a panic? A calendar. The user took
  the decision on 2026-08-29.
- Can a deferred call block? Yes. The chain of the deferred calls holds 3 states
  of its own.

- Does Dart pass an unmodifiable `Uint8List` between 2 isolates by reference?
  No. Dart copies it. Read `docs/history.md`.
- Does an extension type give a cheaper `uint64` than a runtime function? No. An
  extension type erases to a plain `int`, thus it gives compile time safety
  only. Read `docs/history.md`.
- Does the compiler need a second GOROOT? No. The overlay is a map in memory,
  `x/tools` writes it into a JSON file of a temp directory, and it passes
  `-overlay=` to the go command. No code writes into GOROOT.
- Does a loop variable of Go 1.22 keep its per iteration value? It did not, and
  the work of 2026-08-29 fixed it. Read the section "The loop variable of Go
  1.22" of `docs/design.md`.
- Does the compiler take option 1 or option 2 of M15? Option 2. Each shape takes
  a file of its own.
- Does the compiler take the GopherJS block model? Yes. The work is done, and
  the cost of a chain of 20 frames fell from 3900 to 189 nanoseconds. Read the
  section "The GopherJS block model".
- How does `defer` work inside a flat state machine? The frame keeps the defer
  stack, and a return runs the deferred calls in front of it. A deferred call
  that can block is outside the subset.

- How does the compiler emit an unnamed struct type? Read decision 30. A table
  driven test now compiles.
- Does `encoding/json` need a code generator? No. The type descriptor already
  holds the struct tag, a read, and a write of each field. Read decision 45.
- Does a pointer to a value that is not a struct need a box? Yes, and only a
  variable holds one. Read decision 43.
- How does Go code reach a file? Through `syscall/dart` with a synchronous call
  of `dart:io`. Read decision 41.
- Which form does the network access take? A socket. Read decision 56. A `net`
  overlay and an HTTP client each stay open, and both read the same 4 interop
  functions.
- `gart test` is possible, and it reuses the test packages of the go command.
  Read decisions 17 to 29.
- A benchmark needs no run with no `-bench` flag. `go test` behaves the same.
  The `-bench` flag is done.
- Does the async model cost too much? Yes, for a deep chain. It costs a factor
  of 140 against the synchronous chain, and Go costs a factor of 6. Read the
  section "The benchmark support".
- The project name is `godart`, and the command name is `gart`. The user
  revisits the names later.
- A generic method never reaches a method table. Go 1.27 forbids a type
  parameter on an interface method.
- The `reflect` gaps of `fmt` are in `docs/parity.md`.
- `GoString` needs no offset. A substring shares the storage with a byte view.
- Does a `range` loop over a function stay outside the subset? No. Stage 2
  accepts it, and `TestRangeFuncBranch` of `internal/verify` proves the rule.
  Read decision 39 and decision 47.
