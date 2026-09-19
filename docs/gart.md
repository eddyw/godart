# Gart commands

Gart loads a Go module and emits a Dart package. The runtime is bundled.
Go remains an execution prerequisite for package discovery and standard sources.
See [SDK compatibility](sdk-upgrades.md) for supported SDK selection.

## Commands

| Command | Purpose |
|---|---|
| `gart build ./...` | Write Dart source. |
| `gart run ./... -- args` | Build and execute package main. |
| `gart test ./...` | Build and execute test packages. |
| `gart watch ./...` | Rebuild after Go source changes. |
| `gart load ./...` | Show loaded packages. |
| `gart version` | Show the compiler version. |
| `gart licenses` | Show bundled license notices. |

Put compiler flags before package patterns. Arguments after `--` belong to
the application. Commands default to `./...` when no pattern is present.

## Build flags

| Flag | Effect |
|---|---|
| `-o dir` | Output directory. The default is `dartout`. |
| `-runtime dir` | Override the bundled runtime. |
| `-trimpath` | Use portable module/package source paths. |
| `-format=false` | Omit the external Dart formatter. |
| `-timings` | Report phase durations on stderr. |
| `-tags list` | Add comma-separated Go build tags. The `dart` tag remains active. |

Run and test also accept `-frontend=auto|resident|source`.
`auto` uses a verified resident SDK or warns before source execution.
`resident` requires a verified SDK. `source` uses the public `dart run` command.
A compilation or application error never causes a second execution.

Source paths are absolute by default. Concurrent commands require separate
output directories. Gart rejects an output directory that another command owns.

## Test flags

| Flag | Effect |
|---|---|
| `-v` | Print individual test results. |
| `-run expression` | Select test names and subtests. |
| `-short` | Request the test's short mode. |
| `-count n` | Repeat tests. A count of 0 executes none. |
| `-cover` | Report statement coverage. |
| `-bench expression` | Select benchmarks. |
| `-benchtime duration` | Target benchmark duration. |
| `-benchcount n` | Use a fixed benchmark iteration count. |

Benchmarks use AOT output. Their measurements describe Gart output, without
native Go allocation counters. Fuzz tests execute their seed corpora only.

## Output and caches

The output contains Dart libraries, entry points, package metadata, a runtime,
source maps, licenses, and `.gart` cache data.
Run `dart pub get` in that directory before direct Dart compilation.
`gart run` and `gart test` resolve dependencies automatically.

```sh
gart build -o dartout ./...
cd dartout
dart pub get
dart compile exe bin/main.dart -o program
```

Use the public Dart compiler for JavaScript or WebAssembly output where the
[platform limits](parity.md) permit the application.

Resident kernels require matching SDK and dependency identities. A new SDK
cannot reuse an incompatible kernel. Source mode bypasses Gart's kernel cache.
Delete the selected output directory for a completely fresh Gart build.
Global Go and Pub caches usually need no deletion.
See [compilation measurements](compilation-time.md) for precise phase definitions.
