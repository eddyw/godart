# License notices

The root `LICENSE` covers project-authored code. Upstream terms still apply
to copied or adapted material. A generated package does not change the
license of its input code.

The importer input adapters retain Go `gcimporter` and `srcimporter` test setup.
The import configuration helper retains Go `internal/testenv/testenv.go`.
Their sources retain the Go BSD notices and use the overlay license.

The root filesystem adapter follows Go `src/os/root.go`, `src/os/root_openat.go`,
and `src/os/root_unix.go` with their 2024 BSD notices.
The native root tests retain the original rename and race assertions from `src/os/root_test.go`.

The signal context adapter preserves the NotifyContext declarations from
Go 1.27.1 `src/os/signal/signal.go`. Its source retains the Go BSD header.

The command overlay adapts Go `src/os/exec/lp_unix.go` and
`src/os/exec/exec.go`. Child directory validation adapts `src/os/exec_posix.go`.
The overlay retains its Go BSD header and uses the overlay license.

Generated Pub package names use the reserved-word list from Dart Pub
`lib/src/utils.dart` at commit `a9ed06f7fb180b39b950aec878d4aa0911b7675e`.
`internal/driver/LICENSE-Dart` and `AUTHORS-Dart` preserve its BSD terms and attribution.
The compiler notice bundle includes these terms.

Go-derived overlays and upstream test excerpts use the notice in
[`internal/overlay/std/LICENSE`](../internal/overlay/std/LICENSE).
The anonymous-interface cycle detector and compiler directive validation
adapt `irgen.go`, `lex.go`, `noder.go`, `unified.go`, and `writer.go` from Go compiler
`src/cmd/compile/internal/noder/`. Their BSD notice resides in
[`internal/load/LICENSE-Go`](../internal/load/LICENSE-Go).
Language-version diagnostics and embed validation follow Go `noder/irgen.go`
and `noder/noder.go` with `internal/load/LICENSE-Go`.
Import diagnostics preserve path rules from Go `cmd/compile/internal/noder/import.go`.
Syntax diagnostics use the unchanged Go compiler parser in
`internal/gosyntax`. Its `ORIGIN.json` records the SDK and source hashes.
Its `LICENSE` retains the Go BSD terms. The diagnostic adapter follows
`src/cmd/compile/internal/base/print.go` and retains `internal/load/LICENSE-Go`.
Malformed literals retain the behavior of `src/cmd/compile/internal/types2/expr.go`.
Malformed-byte recovery follows `src/cmd/compile/internal/syntax/source.go` and retains `internal/load/LICENSE-Go`.
Range diagnostics follow `src/cmd/compile/internal/types2/range.go` and
retain `internal/load/LICENSE-Go`.
Runtime error values and bounds metadata adapt Go `src/runtime/error.go` and
`src/internal/abi/bounds.go`. The runtime source retains their BSD notice.
The `go:nointerface` adapter follows Go compiler `noder/lex.go`, `types2/decl.go`, `types2/lookup.go`, and `reflectdata/reflect.go`.
Its verifier retains the BSD notice in `internal/load/LICENSE-Go`.
The Linux `syscall.Fchmodat` and `syscall.Faccessat` adapters follow Go `src/syscall/syscall_linux.go` and retain its BSD notice.
The Linux descriptor-relative path adapters follow Go syscall wrappers and retain their BSD notice.
The Linux timestamp adapters follow Go `src/syscall/syscall_linux.go` and `syscall_linux_arm64.go` and retain their 2009 and 2015 BSD notices.
The macOS timestamp adapters follow Go `src/syscall/syscall_bsd.go` and retain its 2009 BSD notice.
The multicast socket options follow Go `src/net/sockopt_posix.go`, `sockoptip4_linux.go`, `sockoptip4_bsdvar.go`, and `sockoptip6_posix.go` and retain their 2009 and 2011 BSD notices.
Interface method symbols adapt Go `src/cmd/compile/internal/types/fmt.go`.
The emitter source and `internal/emit/LICENSE-Go` preserve its BSD notice.
The compiler assertion adapter executes unchanged Go
`src/cmd/compile/internal/test/testdata/assert_test.go` and preserves its BSD notice.
Type assertion diagnostics adapt Go `src/runtime/error.go`, `type.go`, and
`src/cmd/compile/internal/reflectdata/reflect.go`. The emitter retains
`internal/emit/LICENSE-Go`. The runtime license includes the Go BSD terms.
Syscall constants and names use the active SDK `src/syscall/zerrors_*.go`
file. Native syscall data types use its selected `ztypes_*.go` file.
Error methods adapt `src/syscall/syscall_unix.go` with its BSD notice.
Syscall string helpers retain Go `src/syscall/syscall.go` and `exec_unix.go` bodies.
Their overlay source retains the Go BSD notice.
Syscall time helpers retain Go `timestruct.go`, `syscall.go`, and Linux/macOS constructors.
Their source headers preserve the Go BSD notice.
Directory record parsing retains Go `src/syscall/dirent.go` and its Linux/macOS helpers with their BSD notices.
The test-environment command adapter follows `src/internal/testenv/testenv.go`
and `exec.go` with the Go BSD notice.
Test temporary directories and directory restoration adapt Go `src/testing/testing.go` with its BSD notice.
Harness regressions retain original `testing_test.go` and `sub_test.go` assertions.
Benchmark result types and format methods adapt `src/testing/benchmark.go`
and retain its BSD notice.
The runtime caller adapter retains original declarations from
`src/runtime/symtab_test.go`, `callers_test.go`, `stack_test.go`,
`traceback_test.go`, and `runtime_test.go` with their Go BSD notices.
The panic traceback adapter preserves `src/runtime/crash_test.go` and
`src/runtime/testdata/testprog/deadlock.go` with their Go BSD notices.
The process identity adapter adapts group-count validation from Go
`src/syscall/syscall_bsd.go` and `src/syscall/syscall_linux.go`.
The process priority adapter follows the Go generated Linux and macOS syscall wrappers.
It preserves their native value and error contracts under the Go BSD terms.
The runtime license includes their BSD terms.
The time value representation and its clock helpers adapt Go `src/time/time.go`.
Directory creation, sticky-bit behavior, and user directories follow Go `src/os/file.go` with its BSD notice.
Directory enumeration follows Go `src/os/dir_unix.go` and `dir_darwin.go` with their BSD notices.
Descriptor-relative directory metadata follows Go `src/os/statat_unix.go` with its BSD notice.
Rooted entry metadata follows `newUnixDirent` in Go `src/os/file_unix.go` with its BSD notice.
Native directory records use Go `src/syscall/ztypes_*.go` layouts.
Native directory syscalls adapt Darwin `Getdirentries` cursor behavior from
`src/syscall/syscall_darwin.go` with its Go BSD notice.
Path removal and rename preserve the error selection from Go `src/os/file_unix.go` with its BSD notice.
Recursive removal follows Go `src/os/removeall_at.go` with its 2018 BSD notice.
Symbolic link reads adapt its buffer growth and EINTR retry with the Go BSD notice.
Temporary file and directory creation follow Go `src/os/tempfile.go` with its BSD notice.
The adapter uses the runtime random-byte source and the shared path join function.
Temporary directory selection follows Go `src/os/file_unix.go` with its BSD notice.
The WriteFile overlay preserves Go `src/os/file.go` behavior and its BSD notice.
Descriptor-based directory and file attribute changes follow Go `src/os/file_posix.go`,
`src/internal/poll/fd_posix.go`, and `fd_unix.go` with their BSD notices.
Path-based truncation follows Go `src/os/file_unix.go` with its BSD notice.
Positional I/O follows Go `src/os/file.go` and `src/internal/poll/fd_unix.go`
with their BSD notices.
File synchronization follows Go `src/internal/poll/fd_fsync_darwin.go` and
`fd_fsync_posix.go` with their BSD notices.
Path permission and ownership changes follow Go `src/os/file_posix.go` with its BSD notice.
Timestamp updates follow Go `src/os/file_posix.go` and `src/syscall/syscall_{bsd,linux}.go`.
The native adapter and Timespec method preserve their Go BSD notices.
Native file metadata adapts Go `src/os/stat_linux.go`, `stat_darwin.go`,
`types.go`, and `types_unix.go` with their BSD notice.
The native adapter uses `src/syscall/ztypes_{linux,darwin}_{amd64,arm64}.go` layouts.
Public stat adapters preserve the contracts of Go `src/syscall` stat wrappers and their BSD notices.
Descriptor duplication follows Go `src/syscall/zsyscall_{linux,darwin}_{amd64,arm64}.go` with its BSD notice.
Raw filesystem sync adapters follow the same generated syscall sources with their BSD notice.
The native `Umask` adapter follows those generated wrappers and preserves the Go BSD notice.
Descriptor flag controls adapt Go `src/syscall/exec_unix.go` with its BSD notice.
Resource-limit adapters follow Go `src/syscall/syscall_linux.go` and the generated Darwin syscall wrappers with their BSD notice.
Their native `Rlimit` layout follows the SDK `ztypes` declarations for Linux and Darwin on amd64 and arm64.
Resource-usage adapters follow the generated `Getrusage` wrappers and `Rusage` layouts in those Go SDK sources.
They preserve the Go BSD notice and the Darwin `Timeval` field widths.
The ownership adapter preserves original `src/os/os_unix_test.go` assertions
and the `internal/testenv` unsupported-syscall predicates with their BSD notices.
Unix `PWD` validation follows Go `src/os/getwd.go` with its BSD notice.
The syscall.Getwd adapter preserves Go `src/syscall/syscall_bsd.go` and `syscall_linux.go` behavior with their BSD notice.
The shared file state, descriptor adoption, and rename rules follow Go `src/os/file_unix.go`
with its BSD notice.
Zero-size slice allocation and reflection growth follow Go `src/runtime/slice.go`
and `src/reflect/value.go`. Slice exchange follows `src/reflect/swapper.go`.
The runtime and reflection overlay preserve their Go BSD notices.
Reflection constructor validation and embedded methods follow
Go `src/reflect/type.go`.
Direct interface values follow `src/cmd/compile/internal/types/type.go`.
Method package metadata follows `src/cmd/compile/internal/reflectdata/reflect.go`.
Method order follows `src/cmd/compile/internal/types/sym.go`.
The runtime and overlay preserve their Go BSD notices.
Reflection slice allocation validation adapts Go `src/reflect/value.go` and
`src/runtime/malloc.go`. The runtime and overlay preserve their BSD notices.
Shared type layout calculations adapt the Go-derived reflection overlay.
The runtime layout source and license preserve the Go BSD notice.
The certificate trust adapter adapts Go 1.27.1 `crypto/x509/root.go`,
`root_linux.go`, `root_darwin.go`, and `internal/macos` wrappers.
The trust fixture retains certificate constants from Go `crypto/x509/verify_test.go`.
The x509 overlay preserves Go 1.27.1 certificate source and its BSD notices.
The upstream Limbo schema uses C2SP/x509-limbo data under Apache 2.0
and go-jsonschema generator templates under MIT. Generated packages that
include this schema receive both notices from the embedded overlay license.
The CIDR parser adapts Go `src/net/ip.go` and retains its BSD notice.
TCP fallback and partial deadlines retain Go `src/net/dial.go` and
`ipsock.go`, `tcpsock_posix.go`, and `hook.go` declarations with their BSD notices.
Typed UDP context setup adapts Go `src/net/dial.go` and retains its BSD notice.
UDP dial order follows Go `src/net/sock_posix.go` and `src/net/ipsock_posix.go`.
UDP source address conversion follows Go `src/net/udpsock_posix.go` with its BSD notice.
Network error classification adapts Go `src/net/net.go` and `src/net/error_unix.go` with their BSD notices.
TCP listener address conversion adapts Go `src/net/ipsock_posix.go`.
Its overlay source retains the Go BSD notice.
Resolver fixtures retain Go `src/net/main_test.go` setup inputs from a native SDK execution.
The generated fixture preserves its BSD notice.
Internet address selection adapts Go `src/net/ipsock.go` and `lookup.go`.
RFC 6724 address order retains Go `src/net/addrselect.go` with its BSD notice.
The address overlay retains their BSD notice.
Resolver configuration retains Go `src/net/conf.go`, `nss.go`, `dnsconfig.go`,
`dnsconfig_unix.go`, and selected `dnsclient_unix.go` and `parse.go` declarations.
The resolver source preserves their BSD notices.
DNS transport and record validation retain Go `src/net/dnsclient_unix.go`,
`dnsclient.go`, `lookup.go`, and `net.go` declarations with their BSD notices.
The DNS message codec uses the SDK vendored `golang.org/x/net/dns/dnsmessage` source.
Resolver lookup coordination retains Go `src/net/lookup.go`, `ipsock.go`,
`dnsclient_unix.go`, and `hook.go` declarations with their BSD notices.
The shared lookup group uses native Go `src/internal/singleflight` source.
The host cache retains Go `src/net/hosts.go` and selected `dnsclient.go` declarations.
Its source preserves the Go BSD notice.
IP protocol validation adapts Go `src/net/dial.go`, `parse.go`,
`lookup.go`, and `lookup_unix.go` with their BSD notice.
Network interface types, flags, and native address filters adapt Go
`src/net/interface.go`, `interface_linux.go`, and `interface_bsd.go`.
The overlay, native adapter, and runtime license retain the Go BSD notice.
UDP local-address and datagram tests retain assertions from Go
`src/net/udpsock_test.go`
and its BSD notice. The datagram adapter also retains `flaky` and
`SkipFlaky` from `src/internal/testenv/testenv.go` with its BSD notice.
UDP input validation follows Go `src/net/ipsock_posix.go`
and `src/net/udpsock.go` with their BSD notices.
Socket I/O errors and zero-value socket operations follow Go `src/net/net.go`,
`tcpsock.go`, `udpsock.go`, `udpsock_posix.go`, and `unixsock.go`.
UDP peer addresses follow Go `src/net/sock_posix.go`.
The native datagram adapter preserves its BSD notice.
Native socket error wrappers follow
`src/net/fd_posix.go`, `fd_unix.go`, and `fd_windows.go`.
The overlays retain their BSD notices.
The closed-socket error retains Go `src/internal/poll/fd.go` declarations
and their BSD notice.
The close test adapter retains assertions and helpers from Go
`error_test.go`, `error_unix_test.go`, `pipe.go`, `file.go`, and
`mockserver_test.go` with their BSD notices.
The stream contract adapter copies Go-vendored `golang.org/x/net/nettest`
`conntest.go` and its deadline constants without assertion changes.
The generated fixture includes the exact x/net BSD license.
The timeout adapter also retains Go `src/net/timeout_test.go` and
`src/internal/poll/fd.go` declarations with their BSD notices.
Socket deadline precedence follows Go `src/internal/poll/fd_unix.go`.
The Dart runtime retains its Go BSD notice.
Unix listener close tests retain assertions from Go `src/net/unixsock_test.go`
and the address helper from `src/net/mockserver_test.go` with their BSD notices.
The metrics overlay adapts Go `src/runtime/metrics/value.go`, `sample.go`,
`histogram.go`, and `description.go`. Its source retains the Go BSD notice.
GODEBUG counter validation follows `src/internal/godebug/godebug.go`.
The runtime reads `src/internal/godebugs/table.go` from the active SDK.
UTC values preserve the location representation from Go `src/time/time.go`
and `zoneinfo.go`. Their source headers retain the BSD notice.
Source headers and the runtime license preserve their BSD terms.
Generated widget files retain notices for their Dart documentation sources.

Before source reuse or asset additions, record the origin and verify the
distribution terms. Preserve required notices for translations and adaptations.
Do not distribute an asset whose terms remain unresolved.

After a generator change, validate the notices in its actual output.
Run `TestSourceNotices` for Go output and the nodegen notice tests for widget output.

Gart writes a package-root `LICENSE` with notices from the emitted Go
packages. It reads source headers and notice files between each package
directory and its module or SDK root. Cached builds refresh these notices.
The embedded Dart runtime includes its own license.

Nodegen preserves upstream notices beside generated widget declarations
and host code. It resolves the packages that supply copied documentation.
Flutter collects package-root licenses for its license registry.

Both `gart licenses` and `glutter licenses` print embedded compiler notices.
After a compiler dependency or Go SDK change, run:

```sh
go generate ./internal/notices
go test ./internal/notices
```

Binary archives must include `LICENSE` and the output of `gart licenses`
as `THIRD_PARTY_NOTICES`. The embedded command also serves installations
that have no sidecar files.

A standalone application binary also needs the generated package license,
runtime license, and applicable dependency notices in its distribution.

Notice collection does not establish permission to distribute arbitrary
source or assets. Asset terms, platform plugins, and native dependencies
still need review before distribution. Preserve licenses for files that
use unconventional notice locations.

Multicast interface enumeration adapts Go `net/interface_linux.go` and
`internal/routebsd/address.go`. The overlay and native adapter retain their Go BSD notices.

Time text marshaling retains Go `src/time/time.go`.
Its source preserves the Go BSD notice.
