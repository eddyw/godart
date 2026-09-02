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
140. A make with no capacity reads its length expression 1 time, through
     `$makeSliceN`. Before this rule, the emitted text held the expression 2
     times, and `make([]byte, IntN(100))` drew 2 random numbers.
     `testdata/programs/makeonce` proves the rule, and `TestChaCha8Read` of
     `math/rand/v2` found it.
141. The hoist of a call whose argument blocks keeps the location of the
     receiver and of a field write. The address of the receiver of a pointer
     method lands in the frame, and the write of a field walks to its
     location, because a struct variable of the frame holds a copy.
     `testdata/programs/hoistrecv` and `fieldwrite` prove the 2 rules. The
     parser of the Go repository writes the fields of a named result under a
     blocking call, and each parameter list came back empty.
142. A parallel assignment stages a copy of a struct value. A write to a
     location goes into its object, thus a staged reference lost the value
     before the read. `testdata/programs/swapfields` proves the rule. The
     level 6 encoder of `compress/flate` swaps its hash chain this way, and
     the loss gave a ratio above the limit of `TestDeflateInflateString`.
143. An element of a composite literal of a container of pointers omits the
     address operator, and the compiler builds the element and wraps it,
     exactly as the written form does. `go/printer` holds `[]*ast.Comment{{...}}`.
144. A package variable pair from 1 call holds a hidden record variable, and
     each name reads 1 field of it. The hidden variable takes the rank of the
     first name in the init order. `rand_test.go` declares
     `var rn, kn, wn, fn = GetNormalDistributionParameters()`.
145. A Go identifier outside ASCII takes a hex escape in its Dart name, for
     example `χ2` becomes `$u3c72`. Dart holds only ASCII letters, ASCII
     digits, `_`, and `$` in a name.
146. A call of a generic function whose declared result erases to Object?
     takes a cast to the Dart type of the instance. A pointer to a type
     parameter erases, because the form of a pointer follows the type
     argument. `go/token` reads `atomic.Pointer[File].Load` this way.
147. The strings overlay holds `SplitSeq` and `FieldsSeq`, thus it imports
     `iter`. With the value flow of decision 137, the import adds only the
     text of the iter package: 12860 bytes on `testdata/programs/args`, which
     is 3.7 percent, and no other package changes its form. Before the value
     flow, the same import also marked closures across the program.
148. The surface holds the go/format subtree from the real Go source:
     go/token, go/scanner, go/ast, go/build/constraint, go/parser,
     go/doc/comment, go/printer, go/format, with testing/iotest and
     text/tabwriter. 4 patches remove a deferred recover: go/ast/print.go,
     go/parser/interface.go, go/build/constraint/expr.go, and
     text/tabwriter/tabwriter.go. A source with a panic in those paths stops
     the program. `testdata/programs/gofmtuse` proves the subtree.
149. A named type that reaches itself through a pointer, a slice, or a map
     takes the Dart type Object?, and the emitter caps a cycle through a
     function type at a depth of 64. Before this rule, `type Number *Number`
     of peano.go stopped the compiler with a stack overflow.
150. `TestRunFilesOfGoTest` runs each file of the test directory of the Go
     repository whose header is `// run`, and it compares the Dart output
     with the output of go run. 54 files match, a file outside the subset
     skips, and `goTestSkips` names the 18 files whose run differs, with the
     cause of each one. A fix removes its entry.
151. A case of a switch with no tag compares with true. A case of an
     interface type needs that comparison in the output, and a plain bool is
     its own condition. Before this rule, `case i:` with an interface i wrote
     the interface into the condition, and the Dart build stopped.
152. The length and the capacity of a pointer to an array come from the type,
     and the read follows no pointer. `len(f())` with a function that gives a
     nil `*[40]int` gives 40, and a closure keeps the side effect.
153. The name of a descriptor writes each basic type by its kind, thus byte
     gives uint8 and rune gives int32. A type test compares the name, and
     `x.(uint8)` must accept a value that a byte expression boxed.
154. `new(expr)` of an array type copies the array with `$copyArray`. A Dart
     list holds no `$copy` method.
155. `writeNilChans` runs again after the init routine, and a second call
     writes only the new keys. The initializer of a package variable
     registers its nil channels after the first call, and the output held an
     undefined name.
156. A parallel assignment reads the operands of each location before the
     first write. The key of an index and the pointer of a deref each land in
     a variable, the range clause `for i, x[i] = range y` writes the element
     of the old key, and a receive case of a select takes a fresh box for its
     variable on each pass.
157. A map with a key of an array type takes its own equality and its own
     hash, because a Dart list compares by identity. `mapNew` writes them at
     each creation site, and an array inside the key compares the same way.
158. A pointer to a struct compares by identity. The == operator of the class
     reads the fields, thus 2 allocations with equal fields compared equal.
     Each other pointer keeps the == of its cell, because a field pointer is
     a fresh `$Ref` at each address operation.
159. A method expression walks to its method, exactly as the adapter of a
     method table does. An interface receiver dispatches through `$call`, a
     promoted method reads its embedded fields, a value method through a
     pointer parameter reads the value behind it, and a value receiver takes
     a copy at the call.

160. The value and the expression of a generic method bind the dictionary of
     each type argument in a closure. A Dart tear-off binds no argument, thus
     the tear-off form stays only for a method without type arguments.
     `ir.Method` and `ir.MethodExpr` carry the type arguments, normalize
     fills them from the instance record of the type checker, and the type
     becomes the instantiated signature. `genmeth1` of the go test directory
     and `testdata/programs/genmethval` prove the 6 forms.

161. The frame of a top level method takes the type parameters of the
     receiver type and of the method, in that order. The frame class already
     named the joined list, and the save site named the method list only,
     thus a resume read an empty dictionary. The body emit of
     `topLevelMethod` now carries the joined list, and the frame saves each
     dictionary. The method `wait` of `testdata/programs/genmethval` proves a
     block operation inside a generic method through a method value.

162. A builtin whose only argument is a call with 2 results spreads them,
     exactly as a function call does. `ir.Builtin` carries `Results` and
     `ResultsVar`, and the emitter writes the same record closure.
     `copy(f(a))` and `delete(h(m))` of `fixedbugs/issue15992` prove it.

163. The zero text of a map or of a channel whose element names the same type
     ends with null. The Dart type of such a type erases to Object?, thus
     null is the zero of the element. Read `cycleZero`. `typeparam/issue47901`
     crashed the compiler before.

164. A member call on a channel whose Dart type erased to Object? takes a
     cast back to the channel type. Read `chanExpr`. The cast covers receive,
     send, and close.

165. A blank struct field takes the name `$b<index>`, because Dart forbids 2
     parameters of 1 name. A positional literal evaluates the value of a
     blank field and discards it, thus the field keeps its zero value.
     `fixedbugs/issue21048` and `fixedbugs/issue31546` prove the 2 rules.

166. The initializer of a blank package variable runs inside `$init`, in the
     order of the type checker. Only the value falls away. The old output
     dropped the whole initializer, and `fixedbugs/issue51913` and
     `fixedbugs/issue22326` caught the loss.

167. A package variable pair also comes from a comma form: a type assertion,
     a map read, or a channel receive. The pair reads `multiValue`, and each
     name converts the record field to its own type. `fixedbugs/issue53619`
     proves the assertion form.

168. A spread call reads its function operand first, and the inner call
     second, exactly as Go orders them. An operand with an effect becomes the
     first parameter of the record closure. `fixedbugs/issue50672` proves the
     order for a call, a method, and an index operand.

169. A parallel assignment stages the base of a slice or of a map location as
     a value, thus a write of the variable in the same statement misses it.
     An array location writes through the variable, and the name stays.
     `fixedbugs/issue23188` proves the slice rule.

170. A chain of else-if branches emits flat, because the Dart front end caps
     the nesting of a statement near 250. `fixedbugs/issue2615` holds a chain
     of 256 branches.

171. A method call on a type parameter with no core type dispatches through
     the method table of its dictionary: the receiver boxes into an Iface
     with the descriptor of the dictionary, and `$call` finds the method.
     `fixedbugs/issue54348` proves the promoted case.

172. The description of a ground instance of a generic type stands in a
     package variable, exactly as a nil channel does, because 2 descriptions
     can reach each other. A description with a free type parameter stays
     inline, and its second visit gives a shallow form, thus the text ends.
     `typeparam/list2` and `typeparam/sets` crashed the emitter before.

173. A blank type parameter takes the positional name `$T<index>`, and a top
     level method takes the receiver type parameters of the method itself,
     not of the type declaration. A method can name the parameter of its
     receiver type with a blank, and the 2 object sets must give 1 text.

174. An embedded named interface of a constraint adds its methods to the type
     set, and it adds no term. The walk of the terms resolves the underlying
     interface of each embedded name. `typeparam/stringer` caught the loss.

175. A method call on a type parameter dispatches through the method table of
     its dictionary, unless the core type is a struct or a pointer. A
     parameter with no core type boxes the receiver with `$boxDesc`, an
     interface core passes the interface value through, and each other core
     boxes too, because the Dart form of the core holds no Go method. The
     same rule covers a method value and a method expression.

176. A struct type that reads a type parameter needs the name of a package
     level generic type, and stage 2 rejects each other form. The class of a
     shape stands at package level, and it cannot name the parameter. The
     target of a type test stays allowed when the switch binds no variable,
     because such a target needs the name only.

177. The name of a type that reads a type parameter interpolates the name of
     the dictionary at run time. `substParams` puts a marker into the type,
     and `interpolate` splits the printed text at each marker. The method
     keys and the reflect text follow the same rule, and each method table
     keys on the signature of the instance, not of the origin.

178. A type argument can name an interface type, and a value of such a
     parameter is an interface inside an interface. `$boxDesc` flattens the
     box, and `$call` unwraps at the `$ifaceMethod` placeholder.

179. A comparison of an interface value with a value of another type boxes
     the other side, exactly as Go compares through the interface form.

180. The frame of a class method that blocks takes the type parameters of the
     class and of the method, and the frame parameter of the signature takes
     the same joined form. The emitter clone of a function literal keeps
     `typeParams`, thus a nested frame names them too.

181. A blank type declaration drops from the emit with its methods. No code
     names a blank type, and 2 blank types give 1 Dart name.

182. Stage 2 rejects a generic struct type with a field whose name equals the
     name of a type parameter. The Dart class holds the 2 names in 1 scope.

183. `strconv.FormatFloat` with the verb f and the precision -1 gives the
     shortest fixed text with no exponent. `_expandExponent` unrolls the
     shortest form of the runtime.

184. `sort.Float64s` orders a NaN in front of each other value, exactly as
     Go does.

185. The writer of the local shape classes resumes, exactly as the nil
     channel writer does. A descriptor variable can name a shape after the
     first pass.

186. A method value whose declaration erases the Dart type of its result
     becomes a closure with a cast, exactly as `eraseCast` treats a call.

187. The builtins min and max evaluate each operand 1 time, through the
     runtime pair $minOf and $maxOf. A string compares by bytes, and a NaN
     passes through, exactly as Go orders them.

188. A write to the blank name keeps the evaluation and drops the value, in
     an assignment, in a declaration pair, and in a tuple assignment. Dart
     holds no setter of the wildcard.

189. A package variable pair whose names are all blank takes the rank of its
     first name, thus the call runs in the order of the type checker.

190. A keyed struct or array literal evaluates its values in the order of the
     source. The record form of `ir.Composite` holds the source order, and
     each slot reads 1 field of it. `fixedbugs/bug433` and
     `fixedbugs/issue17640` prove the order.

191. A compound assignment reads the operands of its location 1 time: the
     map base, the map key, and the pointer of a deref each take a variable.
     `fixedbugs/bug196` counts the calls.

192. The receiver of a method value whose body stands in a top level function
     takes the form of that function: a pointer method takes the address of a
     value, and a value method reads behind a pointer. `fixedbugs/issue7863`
     proves the address form.

193. A slice value whose Dart type erased to Object? casts back at an append
     and at a nil test, exactly as a channel does at a member call.

194. A range expression evaluates 1 time into a variable of its own, thus a
     write of the same variable inside the body misses the loop. An array or
     a pointer to an array with at most 1 iteration variable and a constant
     length evaluates 0 times, exactly as the specification orders.
     `fixedbugs/issue79909` and `fixedbugs/issue73476` prove the 2 rules.

195. The name of a local type carries the position of its declaration, thus 2
     local types of 2 functions keep 2 identities. The display text stays
     short, and the reflect handle table keys on the name.
     `fixedbugs/gcc65755` and `typeparam/issue54456` prove the split.

196. `strconv.FormatFloat` with the verb f and the precision -1 unrolls the
     shortest text in the overlay, and the bridge verb f keeps 6 digits for
     fmt. The first form of decision 183 broke `%+f`, and this form holds
     both.

197. A reflect Value carries the static type of a nil value. Elem of a
     pointer stores the element type of the pointer, thus Kind and IsNil of a
     nil interface element answer, exactly as Go does.
     `fixedbugs/issue70156` proves it.

198. An append that spreads a value of a union of a string and a byte slice
     branches at run time through `$spreadBytes`.
     `typeparam/issue376214` proves both arms.

199. A field whose name equals a type parameter of its own generic type takes
     the Dart suffix $f, at the class, at each read, at each write, and in
     the descriptor. Decision 182 falls, and stage 2 accepts the form.
     `typeparam/combine` runs now.
200. A struct shape that reads a type parameter takes 1 Dart class. The class
     takes 1 Dart type parameter and 1 dictionary for each Go parameter of
     the shape, and it stands in the library of the package, in the way of a
     named generic type. A ground shape whose open twin exists resolves to
     the class of the twin through a structural match that fills each
     parameter, and 2 uses of 1 parameter must agree. The emit stage runs 2
     passes, thus the pass that finds the shapes cannot change the output.
     A type parameter whose constraint names a struct shape as its core
     erases to Object? and reads each operation from its dictionary, and its
     method calls go through the method table. A struct term of a constraint
     gives the parameter no Dart bound. The dictionary holds a copy and an
     assign for a struct and an array argument, thus a copy point of a type
     parameter keeps value semantics. `equal`, `typeswitch2`,
     `issue50690b`, `issue50690c`, and `issue54535` run now.
201. `recover` reads the chain of the deferred calls of its function. The
     runtime holds the pending panic, the chain panic, and the recovered
     flag, and the chain of each function saves and restores that state, so
     a chain inside a deferred call gives the state of the outer chain back.
     Each run of a deferred call sets the pending panic around the call, and
     a recover outside such a call gives nil. A panic inside a deferred call
     replaces the chain panic, and the calls that remain run for the new
     panic. A runtime error reaches recover as an interface value of the
     description `runtime.errorString`, whose Error method gives the text of
     Go. Stage 2 accepts `recover`, and the 6 patches that stripped
     `recover` from go/parser, go/ast, go/build/constraint, regexp/syntax,
     bytes, and text/tabwriter are gone. A machine holds the chain across a
     blocked deferred call through its states, and the end of the chain
     reads the run time state.
202. A read that follows a pointer, and a call that goes through a function
     value, panic with the nil dereference error of Go when the operand is
     nil, and Dart gives a null check error. An operand with an effect takes
     a parameter of its own, so the wrapper runs the effect 1 time. The
     equality of 2 interface values compares the fields of a struct in the
     order of the type, and a comparison that reaches a slice, a map, or a
     function panics with the uncomparable error of Go. The `len` of a
     pointer to an array stays constant, and Go evaluates the operand for no
     result.

203. The symbolic link operations ride 3 synchronous interop calls: `Lstat`,
     `Symlink`, and `ResolveSymlinks`. `dart:io` gives no metadata of a link
     itself, thus the size of a link is the length of its target, and the
     time of a link is 0. `filepath.EvalSymlinks` asks the system for the
     resolved path, and a relative argument takes `Rel` against the resolved
     working directory. `DirEntry.Info` of `os.ReadDir` reads `Lstat`, as Go
     does. The program `symlinkuse` proves the parity against `go run`.

204. The survey of `os/exec` over `Process` of `dart:io`, taken on
     2026-09-01. The package is possible as an overlay in the pattern of
     `net`: `Process.start` gives the process, and 6 interop calls carry it.
     `ProcStart` gives the handle, `ProcWrite` feeds standard input,
     `ProcRead` and `ProcReadErr` drain the 2 outputs, `ProcWait` blocks for
     the exit code, and `ProcKill` sends a signal. The 4 stream calls and
     the wait ride the scheduler, exactly as `ConnRead` does. `Process.start`
     takes a directory and an environment per process, thus `Cmd.Dir` and
     `Cmd.Env` work with no write to the process environment. The gaps: no
     `SysProcAttr`, no process group, `LookPath` cannot read the execute
     bit because Dart names no permission bit, and `ProcessState` holds the
     exit code only. The package stays out until a program of the net needs
     it, and this survey holds the map.

205. The zone database rides the file access, and the parser is the parser of
     Go. `LoadLocation` reads the variable ZONEINFO, then each zone directory
     of the system, and then the embedded database of `time/tzdata`. The zip
     file of the Go tree stays out, because the Dart output holds no Go tree.
     The overlay of `time/tzdata` holds the generated `zzipdata.go` of the Go
     tree, and the emitter writes the zip as a byte list of about 450 KB. Go
     wires `time/tzdata` to `time` with a linkname directive, and the subset
     holds no such directive, thus the overlay exports
     `time.RegisterEmbeddedTZData`, with a found flag in place of the error
     value. A loaded location takes the exact transition rule of `Date` of Go,
     and `Local` keeps the 2-lookup guess of decision 79.

206. The source map holds the module form of a file: the package path and
     the base name, as a build with `-trimpath` gives. The form keeps goal
     G5, because it needs no path of the machine. Go without `-trimpath`
     writes the path of the build machine, thus `Llongfile` and a printed
     trace differ before the base name. A line directive passes verbatim,
     with the resolution of go/token. The wrapper of a deferred call takes
     the map entry 0, and the reader drops such a frame. The run of the
     defer chain marks the line that Go names: the return statement at a
     return, and the closing brace at a fall or on the panic path. The run
     consumes the calls of the chain, thus a second run does nothing. The
     deadlock report walks the frame chain of each live goroutine through
     the TraceFrame members of each frame class, and each wait frame of the
     runtime names its reason.

207. A complex number maps to the runtime class GoComplex, with 2 doubles
     and the Dart operators, thus the emitter writes a plain operation. The
     division holds the algorithm of Smith with the corrections of C99, as
     complex128div of the Go runtime does. `complex64` rounds the 2 parts
     with `$c64` after each operation, and a constant of it rounds at the
     emit. The builtins map to `.re`, `.im`, and the constructor. The class
     overrides the equality and the hash, thus a complex value works as a
     map key. `fmt` writes the `(1+2i)` form for the verbs `v`, `f`, `e`,
     and `g`, and the other verbs of Go for a complex value stay out. The
     program `complexuse` proves the parity, the division edges included.
     This decision closes the plan of decision 16.

208. The minimal web target. The process access of the scheduler, of the
     print helpers, and of `$exit` goes through 1 platform shim, and a
     conditional import selects the dart:io form or the web form. The
     barrel of the runtime exports `file.dart` and `net.dart` through a
     conditional export, and the web form of each holds only `$goos`, which
     gives `js`. Thus a pure program compiles with `dart compile wasm` and
     runs under node, and a program that reaches a file or a socket fails
     in the Dart compile with the name of the missing function, per the
     web decision of 2026-09-01. The default branch of each condition is
     the dart:io form, because the analyzer reads that branch. `TestWasm`
     of the driver is the smoke test.

     **2026-09-03: decision 251 replaces the second half.** Each web form
     now holds each name of its native form, and an unsupported operation
     reports the cause of Go under `GOOS=js`. The compile of a program
     that reaches a file or a socket passes.

209. The JS integer design stands in `docs/js-integers.md`, and the work
     waits. The model: the emitted code names `GoInt` in place of the Dart
     int, a conditional import selects the form, the native form is an
     extension type over int that erases on the VM, with AOT, and on wasm,
     and only the JS library holds the pair form of GopherJS. The wasm
     target already covers the browser with the 64-bit model, thus the JS
     target serves only a build that must ship plain JS.

210. The TLS server rides `SecureServerSocket` of Dart. `ListenTLS` binds
     with a security context of 2 PEM blocks and the ALPN list, `AcceptTLS`
     gives the connection and the protocol that ALPN took, and the accepted
     connection joins the table of the plain connections, thus each read,
     each write, and each deadline works on it. A `Certificate` keeps the 2
     PEM blocks of `X509KeyPair`, because the surface holds no `crypto/x509`
     and Dart parses PEM. `tls.NewListener` stays out: the secure socket of
     Dart wraps no open listener, thus `httptest.StartTLS` closes the plain
     listener and binds a fresh port of the loopback address. The client of
     `httptest` drops the certificate check, because the surface holds no
     root pool. The listener of the runtime now holds a close function, thus
     1 listener class serves the 2 server socket classes of Dart.

211. A range loop over a map rides a lazy iterator of the runtime, and the
     key snapshot is gone. `GoMap` keeps its entries in a
     list in the order of the writes, next to a `HashMap` from the key to
     the entry. A remove marks the entry dead and unlinks it, and the dead
     entry keeps its next link, thus an iterator that stands on it still
     reaches the rest of the list. The iterator starts in constant time,
     thus n loops over a map that shrinks stay linear, and `maplinear`
     joins the net. The value reads the entry at the visit, thus the NaN
     snapshot and the clear count of decision 207 are gone too: a clear
     marks each entry dead, and a NaN entry behaves as each other entry.
     The iterator types as `ir.MapIterType`, a type of the compiler that
     go/types does not hold, in the pattern of the Dart field of a select
     variable.

212. `os/exec` is an overlay over `Process.start` of `dart:io`, on the map
     of decision 204. The runtime holds the child in a table by its process
     id, thus the id is the handle of each call, and `os.Process` holds the
     id alone. 7 interop calls carry the work: `ProcStart`, `ProcWrite`,
     `ProcCloseStdin`, `ProcRead`, `ProcReadErr`, `ProcWait`, and
     `ProcKill`. The close of the standard input is the seventh call,
     because a child that reads until the end of the input needs that end,
     and the survey held 6. A command with no `Stdin` closes the input at
     the start, and a `Stdin` reader feeds it on a goroutine. Each output
     drains on a goroutine even without a writer, because a full pipe would
     stop the child. A pipe of `StdinPipe` or `StdoutPipe` works only after
     `Start`, and `CommandContext`, `SysProcAttr`, and a separate `Args[0]`
     stay out. The program `execuse` proves the parity, the kill and the
     exit code included.

213. HTTP/2 rides the Dart package `http2 ^2.3.1`, behind ALPN, and not a
     Go stack: HPACK and the framing stand in that package, and the overlay
     of `net/http` stays small. `$takeSocket` lifts the socket of a TLS
     connection out of the table of `net.dart`, and the take works only
     while no read ran, because the first read subscribes the iterator.
     9 interop calls carry the 2 sides, and `H2Fetch`, `H2Read`, and
     `H2Next` block. The client offers h2 on an https dial with the rules
     of Go: a config with `NextProtos` wins, no config offers h2, and
     `ForceAttemptHTTP2` offers h2 next to a config. 1 session carries
     each request of a host, and a session that broke leaves the pool with
     the retry rule of a reused connection. The request body reaches the
     wire as 1 piece, because the fetch takes the bytes. The server
     answers each stream on a goroutine of its own, and the answer buffers
     as the answer of HTTP/1.1 does. `httptest.Server` takes
     `EnableHTTP2`. A push of the server, a trailer, and `Request.TLS`
     stay out. The program `h2serve` proves the parity.

214. The Go integer of the emitted code is the type GoInt, and a
     conditional export selects 1 of 2 forms: an extension type over int
     that erases on the VM, with AOT, and on wasm, and the pair of
     GopherJS on dart2js, where `dart.library.js` tells the 2 web targets
     apart. A literal takes `$gi`, a literal outside 53 bits takes the 2
     halves of `$gi2`, an index or a length crosses with `$ix`, and a Go
     integer constant emits as a lazy final. The interop boundary is 1
     uniform rule of stage 5: each integer argument takes `$ix`, and each
     integer result comes back through `$gi`, `$gils`, or a typed closure
     over the result record, thus the wrappers of the runtime keep the
     plain int inside. The division of the JS form rides BigInt, because
     it is rare next to the other operations, and a dart2js bitwise result
     comes back unsigned, thus each half normalizes. The allocators
     `$goList8` to `$goListU64` keep the dense list on the native targets.
     A mixed numeric type set keeps the Dart bound num, thus such a
     generic program fails only in the dart2js compile. `goint_test.dart`
     proves the pair against the native form, a node probe proved the
     table under dart2js, and `TestJS` proves the program `intwrap` under
     node against `go run`. `docs/js-integers.md` holds the whole build.

215. The state of a Flutter application in Go stands in a signal graph. A
     `Signal[T]` holds 1 value, a read inside a rebuild boundary records a
     dependency, and a write marks that boundary alone. The reason is wall 3
     of `docs/flutter-design.md`: the host builds a fresh Dart widget at each
     pass, thus Flutter never takes its identity short cut, and a dirty flag
     then rebuilds the whole mapped subtree. `Computed`, `Effect`, `Batch`,
     and `Untracked` complete the graph, and the package also holds `Store`
     for the shape of `ChangeNotifier`, `Reducer` for the shape of Redux and
     of bloc, and `Command` for the shape of the Command pattern of the
     Flutter guide. The graph needs no compiler change, and it runs on 1
     goroutine: `Post` and `Pump` carry a result from a worker goroutine, thus
     `go test -race` gives a clean run. `docs/flutter-state.md` holds the
     whole model.

216. The Dart host reads a node through its type descriptor, and not through a
     generated class. `Iface.desc.name` picks the case, and `FieldDesc.get`
     reads each property. Thus the package `goflutter` imports no output of
     any Go module, and 1 host serves each node set, the generated set of
     milestone 3 included. The application writes 1 class that implements
     `GoBridge`, about 60 lines, which forwards the 10 calls into its own Dart
     output.

217. A package of a dependency module compiles. `compiled` of
     `internal/driver/driver.go` accepts a path whose first element holds a
     dot, which is the rule that `internal/overlay/native.go` already reads
     for the import check of stage 2, and `libPath` writes such a package
     under the directory `dep`. Before this decision stage 2 accepted the
     import and stage 5 failed with "the package X has no import prefix", thus
     no Go library outside the main module reached Dart.

218. The scheduler holds `$onIdle`, and a host outside Go sets it. The hook
     runs after the run queue empties, 1 time for each turn, thus a call of
     `$schedule` inside the hook starts no second call. A Flutter host reads
     the result of the turn there. `$hosted` turns off the deadlock report,
     because a goroutine that waits for an event of the host is not a
     deadlock. The alternative was a poll of 16 milliseconds, and such a poll
     also stops `pumpAndSettle` of a widget test from ever settling.

219. The box of an interface value writes no copy when the operand is a
     composite literal. The literal builds a value that no other expression
     names, thus the box holds the only reference. A widget tree of 200 nodes
     then allocates 200 objects in place of 400, and the counter application
     of `flutter/example/counter` went from 39 copies to 2. This is the shape
     that each further copy of stage 4 must take: a rule that the compiler
     derives, and not a tag that the user writes and the compiler trusts.

220. A reactive value holds a `noCopy` field with a `Lock` and an `Unlock`
     method. A copy of such a value splits its subscriber list, because
     `$copy` of a struct builds a new `dep` that holds the subscriber slice of
     the moment, and no error names the split. The copylocks check of
     `go vet` reports each copy, and it follows the value through an enclosing
     struct. Thus the safety needs no compiler work.

221. A synchronous export position is a named function type whose declaration
     carries `//gart:sync`. A pass after stage 3 rejects each function that can
     block and that reaches a value of such a type, and the report names the
     function, the position that holds it, and the operation that blocks. A
     named type is the only candidate that stands as a struct field type and as
     a function parameter, thus 1 marker covers the whole boundary. A struct tag
     reaches no parameter, and a directive on a declaration reaches no closure
     literal. The flow of `internal/analyze/flow.go` answers the question with
     no new analysis, and a value that the flow cannot name falls into the
     escape group of its signature, thus an unresolved value gives a rejection
     and never a silent pass. `gart.dev/ui` declares `Builder` and
     `ItemBuilder`, and `internal/analyze/sync.go` holds the pass.

222. A field of a generated node that a call may leave unset is a pointer, and
     a nil pointer means "the default of Flutter". The type descriptor already
     carries `ptr:` and `newPtr:`, thus the host reads null with no new
     mechanism, and `GoNode.point` of `flutter/goflutter/lib/src/node.dart`
     holds the whole read. The 2 alternatives were a wrapper `Opt[T]`, which
     costs a nested struct read for each field, and the zero value, which
     cannot state "not set" for a bool or for a color. `ui.P` writes a pointer,
     because Go takes no address of a literal.

223. A named constructor of Flutter takes a Go struct of its own.
     `ListView.builder` gives `ui.ListViewBuilder`. The host already switches
     on the name of the type descriptor, thus the shape needs no new mechanism.

224. The Dart host of the generated node set is per application, and not
     shared. A shared host holds a case for each of the 411 node types, thus it
     names each Flutter widget and the tree shaker of Dart keeps the whole
     framework. The measurement of 2026-09-02, on an ahead of time build for
     macOS: the hand written set of 28 nodes gave a snapshot of 13,989,280
     bytes, the generated set with a shared host gave 20,844,352 bytes, and the
     generated set with a host of the 18 cases that the counter uses gave
     13,628,480 bytes. Thus the Go node set costs nothing that the application
     does not use, and the whole cost of 7.2 megabytes comes from the switch.
     `goSetWidgets` of `flutter/goflutter/lib/src/host.dart` installs the host
     of the application, and `usedNodes` of `flutter/nodegen/bin/nodegen.dart`
     reads each `ui.<Name>` of the Go source to build the list. That list is a
     superset, and a name that the scan misses gives a clear error at run time.

225. The generated set replaces the hand written node set of milestone 1. 4
     types stay hand written in `flutter/ui/node.go`, because no Flutter widget
     carries them: `Keyed`, which holds a key that a Go value cannot hold,
     `Watch`, which is a rebuild boundary, `TextInput`, which owns its value
     because the Flutter `TextField` takes a controller, and the 2 marker types
     `Builder` and `ItemBuilder`.

226. `reflect.Type` is an interface, and the overlay holds 1 unexported
     implementation. Go gives an interface, and the overlay gave a struct
     before 2026-09-02. `text/template` compares a `Type` with nil, and a
     struct holds no such comparison. A pointer to an unexported struct
     compares with nil and holds each method, but a method of an unexported
     type emits as a private Dart name, and a Dart library holds each private
     name of its own, thus no other package can call it. An interface
     dispatches through the method table, and each package then reads it.

227. A function type that names itself erases to `Object?`, and each type that
     holds such a type erases with it. `type stateFn func(*lexer) stateFn` of
     the lexer of `text/template/parse` has no finite Dart form. The depth
     guard of `dartType` wrote 30 levels of the expansion, and the Dart parser
     rejected that text. `cyclic` of `internal/emit/type.go` now walks a
     signature, and `mentionsCyclic` erases each signature that reaches such a
     type, because `func(*lexer) stateFn` is identical to the underlying type
     of `stateFn` and the 2 must take 1 Dart form. A call through such a value
     goes through a dynamic call.

228. A method value of the package reflect is a plain Dart closure of `Object?`
     parameters, and no pair of a function and a receiver. A Go program asserts
     such a value to its function type, and a pair holds no such form. Dart
     reads a parameter type against the other direction, thus a closure of
     `Object?` parameters fits each function type of the same arity. The result
     stays `dynamic`, thus an assertion to a function type with a result fails.
     `fixedbugs/issue27695` of the test directory of Go witnesses that limit.

229. A generic function that gives a pointer needs a read at the call site,
     and not a cast. The Dart form of a pointer follows the type argument: a
     pointer to a struct or to an array is the object itself, and each other
     pointer holds a cell. A generic body holds no type argument, thus `&v`
     there always writes a `Box`. `eraseCast` of `internal/emit/call.go` wrote
     `as Insets$?` on the result of `ui.P(ui.Sides(12, 6))`, and Dart reported
     that a `Box<Insets$>` is no `Insets$`. The call site now writes
     `$ptrTo<Insets$>(...)`, which reads the cell out and keeps a nil pointer
     nil. Proof program 2 of the Flutter path found the bug.

230. A type assertion to a pointer to a type parameter writes no cast.
     `$assertType` gives `Object?`, and the Dart form of such a pointer is
     `Object?` as well, thus the cast was a no-op that `flutter analyze`
     reported as `unnecessary_cast`. `typeAssert` of `internal/emit/iface.go`
     now writes the cast only for a target of another Dart type. The package
     `unique` holds the 1 site that reaches this shape today.

231. A Dart value type of a widget parameter takes a Go struct, and not an
     opaque handle. The reading of 2026-09-02 dropped 273 widget constructors,
     and a count of the blocking parameter of each 1 gave 176 distinct Dart
     types with a largest of 11. Thus no opaque handle unlocks a group. The
     parameters that a mapping does reach stand on the 411 nodes that already
     map: `FocusNode` in 74, `WidgetStateProperty<Color?>` in 54, `MouseCursor`
     in 49, `ShapeBorder` in 40, `ScrollController` in 31, and `ScrollPhysics`
     in 30. Of these, 6 of 9 hold a value and no identity, thus a Go struct or
     a name carries them. `ui.BorderSide`, `ui.Radius`, `ui.Shape`,
     `ui.Density`, `ui.IconStyle`, and `ui.StateColor` joined `ui.Insets`,
     `ui.TextStyle`, and `ui.Constraints`, and `MouseCursor`, `ScrollPhysics`,
     and `Locale` each take a name. The set went to 414 nodes and 3851 fields,
     thus 275 more parameters reach Go. `FocusNode`, `ScrollController`, and
     `WidgetStatesController` each hold an identity, and the host owns such an
     object, as it owns the controller of `ui.TextInput`.

232. A `//go:embed` directive takes its value at the declaration, and stage 1
     writes it. The compiler of Go holds a directive of the linker for the file
     list of an `embed.FS`, and this compiler holds none. An `init` of the
     generated code was the other candidate, and it breaks the initialization
     order: `var count = len(greeting)` reads the value before an `init` runs.
     Thus stage 1 rewrites the source of the declaration itself, and the go
     command sees `// go:embed` with a space, which is no directive. The
     overlay of `embed` takes the constructor `New`, because a package cannot
     reach an unexported field of another package, and stage 2 rejects a read
     of that name in a file that holds no directive.
     `testdata/programs/embeduse` and 4 tests of `internal/driver` cover it.

233. The Lookup family of `net` takes a resolver of the overlay, and not a
     call of `InternetAddress.lookup` of `dart:io`. That call gives an A record
     and an AAAA record alone, thus `LookupMX`, `LookupNS`, `LookupSRV`,
     `LookupTXT`, and `LookupCNAME` would each stay out. The socket of the
     overlay already carries UDP and TCP, thus the resolver needs no new
     interop function. `dnsmsg.go` holds the message format for the 8 record
     types of the family, and `lookup.go` holds the `Resolver`. The resolver
     reads /etc/hosts, /etc/resolv.conf, and /etc/services through
     `dart.ReadFile`, thus the overlay imports no `os`. `Resolver.Dial` is the
     seam of a test, and `testdata/programs/dnslookup` runs a server of its own
     on the loopback interface. `docs/parity.md` holds the 5 differences.

234. A function that carries the blank name takes a Dart name of its own. A Go
     package may hold many such functions, and `stringer` writes 1 for each
     type that it generates. Each 1 took the Dart name `__`, thus the output of
     `html/template` held 6 declarations of that name. Go calls none of them,
     thus the name alone matters and the body stays. `initNames` of the
     emitter already gave a name to each `init` function, and a blank function
     now joins that map with the form `_blank$<n>`.

235. `archive/tar` takes a patch of 3 files, and `sysStat` stays nil. The real
     `stat_unix.go` reads `syscall.Stat_t` and `os/user`, and
     `stat_actime1.go` and `stat_actime2.go` each read `syscall.Stat_t`. The
     surface holds no such type. `FileInfoHeader` tests `sysStat` for nil
     already, thus a nil value is the behavior that Go itself takes on a
     platform with no such file. The header then carries no owner name and no
     group name, and each other field reads as Go writes it.

236. `math/big` takes the pure Go arithmetic, and not an interop function. The
     real `arith_decl.go` declares 6 functions with no body, because an
     assembly file of each architecture holds the code. Go itself ships the
     pure Go form behind the tag `math_big_pure_go`, and `arith.go` holds each
     `_g` function that the form calls. The patch writes those 6 calls, thus
     the package needs no new interop and no assembly. A Dart form of the same
     word operations would be a second implementation to keep correct, and the
     Go form is already correct.

237. `encoding/xml` takes the real Go source, and the 6 reflect names that it
     needs enter the overlay. `Type.Bits`, `Type.FieldByIndex`, `Value.Bytes`,
     `Value.SetBytes`, `Value.Grow`, and `Value.SetLen` each build on the
     primitives that the overlay already holds, thus none needs a new interop
     function. `Value.Bytes` of an array of bytes gives a copy, because the
     runtime holds no address of an element. `encoding/gob` needs the same 6
     and 1 more thing: it indexes a table by `reflect.Kind`, and the overlay
     models a `Kind` as a text. That package thus stays out.

238. `crypto/tls` holds the value surface alone. `dart:io` owns the handshake,
     thus a `Config` field that would change it can only report support and
     change nothing. Such a field is a silent lie, and the project rejects one.
     A field that Go itself documents as ignored is a different case, and
     `Config.PreferServerCipherSuites` is the 1 field that passed that test.
     The constants, the enumerations with their `String` methods, and the 3
     cipher suite functions are pure data, thus each 1 enters. The package went
     from 32 names to 115.

239. An external Dart declaration takes the directive `//gart:dart`, and the
     value boundary of it carries 5 types. `intrinsic` of stage 5 is a flat
     switch on the name of each interop function, thus each Dart call cost an
     edit of the compiler. A Go function declaration with no body already
     parses, and the directive names the Dart library and the Dart symbol.
     Stage 2 accepted no read of a bodyless function, and it now accepts a read
     of a marked 1.

     The boundary carries `bool`, `int`, `float64`, `string`, and
     `dart.Object`. A `bool` and a `float64` hold the Dart form already, and a
     Go `int` is an extension type over the Dart `int`. A `string` needs
     `toDartString` in 1 direction and `GoString.fromString` in the other. Each
     other type needs a conversion table, and a wrong conversion gives a wrong
     value with no report, thus stage 2 rejects it. A method, a type parameter,
     a variadic parameter, and more than 1 result each reach a rejection too.

     `dart:core` takes no prefix and no import. A prefixed import of it hides
     the plain names, thus `double` and `String` would leave the file.

240. `dart.Object` enters `syscall/dart`, and the emitter gives it the Dart
     form `Object?`. Decision of 2026-09-02 held this type, and it named 2
     conditions. Decision 239 answers the first 1: a `//gart:dart` declaration
     is the producer, and a Go program still makes no such value. The second 1
     holds as written. `syscall/dart` carries `//go:build dart`, thus a module
     that names the type needs a twin file and an alias of its own.
     `testdata/programs/dartobj` holds that pair.

241. The flag 0 of `fmt` counts the sign against the width and the base prefix
     not. The pad of the overlay wrote its zeros in front of the whole text,
     thus `%#04x` of 5 gave `00x5` and Go gives `0x0005`. 3 more faults stood
     with it: `%#o` of 0 doubled the zero, the verb `O` reached no case at all,
     and the flag space reached no code. `testdata/programs/fmthex` runs 21
     verbs against 7 values, thus a change of this area now shows itself.

242. A generic struct holds the `==` operator and the `hashCode` getter, and
     `types.Comparable` is the wrong test for it. That function gives false for
     a struct that holds a field of a type parameter whose constraint is `any`,
     because such a parameter is not strictly comparable. Go still accepts `==`
     on an instance whose type arguments are comparable, and it rejects each
     other instance at the use site. The class thus held neither member, and
     Dart compared 2 instances by identity. A map with such a key found
     nothing. The dictionary of the parameter already gives `equal` and `hash`,
     thus the code that a `comparable` constraint takes works here too.

243. `os` reaches a permission bit, an owner, and a hard link through a command
     of the platform. `dart:io` names none of the 3. `Chmod`, `Chown`, and
     `Link` each run `Process.runSync`, and `Chtimes` reads the real
     `setLastAccessedSync` and `setLastModifiedSync`. A name that needs a file
     descriptor stays out, because the runtime names a file by an object alone.
     `File.Fd`, `File.SyscallConn`, `NewFile`, `StartProcess`, and `ProcAttr`
     thus stay out, and so does each name that needs a wait status or a
     resource usage.

244. `net/http/httptest` tracks each accepted connection itself, and the
     overlay of `net/http` stays unchanged. That server holds no `ConnState`
     hook, thus `CloseClientConnections` would need a change of it. A wrapper
     of the listener and of the connection costs less and it reaches no other
     caller. The wrapper keys a connection by an `int`, because the runtime
     hashes a map key by its structure and a `*tls.Conn` changes its byte
     counters while it stands in the map.

245. `strconv.IsPrint` and `strconv.IsGraphic` read `unicode`, and the
     generated tables of Go stay out. Go documents the 2 as equal to
     `unicode.IsPrint` and `unicode.IsGraphic`, and it carries its own tables
     for speed alone. `unicode` reads the real Go source in the surface, thus
     the answer is the same. A sweep of the whole rune range in
     `testdata/programs/strconvquote` proves it: 159,613 printable runes and
     159,629 graphic runes, equal for Go and for the Dart form.

246. 6 more Dart types take a Go value type: `ButtonStyle`, `ImageProvider`,
     `Decoration`, `Size`, `ShapeBorder` of a shadow, and `Gradient`. The set
     went from 414 node types to 423, and 261 constructors stay out against
     270. `ButtonStyle` carries a per state value for a color alone, because
     `StateColor` stands already and a per state padding belongs in a theme.
     `ImageProvider` takes a kind and a source, and not 1 field for each kind:
     3 of the kinds hold 1 text, and 2 fields set at once would name no image.
     `FileImage` stays out, because it reads a `File` of `dart:io` and
     `goflutter` must build for the web too.

247. A `FocusNode`, a `ScrollController`, and a `WidgetStatesController` each
     take a Go key, and the host owns the object. A Go tree is data and it
     rebuilds on each change, thus a Go value cannot be the object itself: a
     new tree would give a new controller and the widget would lose its focus,
     its offset, or its pressed state. `ui.TextInput` already held that pattern
     for a text field.

     The lifetime follows the element tree, and not a build pass. `GoKeep`
     retains the object on `initState`, rebinds on `didUpdateWidget`, and
     releases on `dispose`. A post frame sweep disposes each entry with no
     user. A sweep of the keys that the last build read is wrong, because a row
     of a lazy list builds long after the list and the sweep would drop a live
     object.

     1 key that reaches 2 widgets gives a report through `FlutterError`, and
     the object stays. A dispose would break the 2 live widgets. The check runs
     after the frame, because Flutter mounts the new element before it unmounts
     the old and the count is 2 during a move.

     The read back direction is carried, and the write direction is not. A Go
     program reads whether a field holds the focus and what the offset is,
     through an optional callback of the key. It requests no focus and no
     scroll: a plain field would re-apply on each rebuild and fight the user,
     and a one shot command channel has no form in a tree of data.

248. A text property costs nothing at 60 frames each second, and the claim of
     `docs/flutter-first.md` is false. A whole tree rebuild of 200 nodes with a
     text of 20 characters costs 61.4 microseconds, which is 0.37% of the
     16,700 microsecond frame. The `GoString` conversions are 17.8 of those
     microseconds. The normal case, 1 signal and 1 boundary, costs 0.58
     microseconds.

     The measurement found 2 things that matter more than the text. The encode
     dominates the decode by a factor of 3 to 6, and
     `GoString.take(utf8.encode(s))` gives the same value 27% to 32% cheaper
     than `GoString.fromString`. The read of the optional fields costs more
     than the text: 37.4 microseconds against 8.04 for 200 nodes, because a
     `ui.Text` holds 12 optional fields and each read is a lookup of a
     descriptor map at about 15 nanoseconds.

     **2026-09-03, later the same day: the 2 numbers moved.** `fromString`
     adopts the buffer of the encoder now, and the host reads a field by its
     position in place of the map. The optional fields cost 20.3 microseconds
     against 6.85 for the text, and 1 optional read costs 8.5 nanoseconds. The
     whole frame falls from 56.9 to 40.0.

     A screen needs about 20,000 text properties before the conversions take
     10% of a frame, and Flutter lays out no such screen. `flutter/bench` holds
     the harness, and `flutter/bench/report.md` holds each number.

249. The Dart reconciler of A10 stays out, and the plain walk of A5 with the
     signal graph stands. A10 is 1.82 times the cost of A5 in the case that it
     exists for, and 150 times the cost of the signal graph.

     A build allocates a fresh node object for each node, thus no identity test
     can stop the walk and the diff must read each field. A5 reads 13 fields of
     1 tree for each node, and A10 reads 13 fields of 2 trees before it decides
     anything. The cost order proves it: A10 costs the most when nothing
     changed, and the least when everything changed, because the first field
     that differs ends the compare.

     A `ui.Watch` rebuilds 1 boundary in 0.588 microseconds at any tree size,
     and the floor of A10 is 88.2 microseconds at 200 nodes. The signal graph
     answers the same question in Go before the tree exists, and A10 answers it
     in Dart after Go rebuilt the whole tree.

     The generated `==` is no shortcut. It reads each pointer field with
     `identical`, which is correct for Go, and a build allocates a fresh
     pointer for each optional field. Each such field thus reports a change
     that nobody can see, and a fresh closure makes each button report one.

     1 condition stays open: A10 earns its cost only when the Flutter work that
     it skips for each unchanged node exceeds 202 nanoseconds. That number
     needs a Flutter harness, and `flutter/bench` holds none.

250. `crypto/rand.Int` and `crypto/rand.Prime` stay out, and `math/big` stays
     in. The 2 names read `math/big`, thus each program that reads
     `crypto/tls` would carry that package. `math/big` holds a `fmt.Formatter`,
     and the flow of stage 3 then reaches `(*io.PipeWriter).Write` through the
     group of `Write([]byte) (int, error)`. A `//gart:sync` position thus
     rejects a build method that only formats an error, and each of the 3
     Flutter examples that reads `net/http` stopped building.

     The other choice was to hold `math/big` out of the surface as well. That
     costs more: a program that wants a big integer imports the package
     itself, and it pays the cost only when it also holds a `//gart:sync`
     position. `crypto/rand` gives no such choice, because `crypto/tls` reads
     it.

     `TestSyncAcceptsAnErrorFormatWithHTTP` of `internal/driver/sync_test.go`
     holds the shape, thus the fault cannot return without a report. The real
     fix is a key of the method group that carries the interface type of the
     call site, and `docs/parity.md` holds it.

     **2026-09-03, later the same day: the 2 names return.** The fix of
     decision 251 landed. The key of a method group now carries the interface
     of the call site, a type assertion narrows that interface further, and 3
     patched files of `math/big` close the last 2 paths.
     `testdata/programs/randbig` proves the 2 names against the go command.

251. The web target declares each name of the io shim, and each unsupported
     operation gives an honest cause at run time. The decision of 2026-09-01
     said the opposite: a program that reaches a file or a socket fails in the
     Dart compile, with the name of the missing function.

     That rule made the whole web target unreachable. The Go package `os`
     names each function of the io shim, thus a program that only reads
     `os.Getenv` failed in the Dart compile with 36 missing names. `net/http`
     added 30 more. No real program compiled for the web.

     `file_web.dart`, `process_web.dart`, `net_web.dart`, and `http2_web.dart`
     now declare each name of their native form. A file operation gives the
     cause `function not implemented`, and a socket operation gives
     `operation not supported`. The Go side wraps each text in a `PathError`
     or an `OpError`, thus a program reads the same error that Go gives under
     `GOOS=js`. `os.Getenv`, `os.Args`, `os.Exit`, `os.Getwd`, `os.Getpid`,
     and `os.TempDir` each give the answer of Go on that target.

     The other choice was to keep the compile-time failure. It reports the
     fault earlier, and it costs the whole target. Go itself takes the run
     time answer on `js/wasm`, thus this choice also matches Go.

     `TestWebIO` of `internal/driver/e2e_test.go` runs the program of
     `testdata/web/webio` under `dart compile js` and under
     `dart compile wasm`, and it reads the exact text of each cause.

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
