# License notices

The root `LICENSE` covers project-authored code. Upstream terms still apply
to copied or adapted material. A generated package does not change the
license of its input code.

The signal context adapter preserves the NotifyContext declarations from
Go 1.27.1 `src/os/signal/signal.go`. Its source retains the Go BSD header.

Go-derived overlays and upstream test excerpts use the notice in
[`internal/overlay/std/LICENSE`](../internal/overlay/std/LICENSE).
The anonymous-interface cycle detector and compiler directive validation
adapt `irgen.go`, `lex.go`, `noder.go`, and `writer.go` from Go compiler
`src/cmd/compile/internal/noder/`. Their BSD notice resides in
[`internal/load/LICENSE-Go`](../internal/load/LICENSE-Go).
Syntax diagnostics use the unchanged Go compiler parser in
`internal/gosyntax`. Its `ORIGIN.json` records the SDK and source hashes.
Its `LICENSE` retains the Go BSD terms. The diagnostic adapter follows
`src/cmd/compile/internal/base/print.go` and retains `internal/load/LICENSE-Go`.
Malformed literals retain the behavior of `src/cmd/compile/internal/types2/expr.go`.
The process identity adapter adapts group-count validation from Go
`src/syscall/syscall_bsd.go` and `src/syscall/syscall_linux.go`.
The runtime license includes their BSD terms.
The certificate trust adapter adapts Go 1.27.1 `crypto/x509/root.go`,
`root_linux.go`, `root_darwin.go`, and `internal/macos` wrappers.
The trust fixture retains certificate constants from Go `crypto/x509/verify_test.go`.
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
