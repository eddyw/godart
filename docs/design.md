# Compiler architecture

Gart translates a verified Go subset into Dart packages. The Go SDK supplies
package discovery, type information, and standard library source.
The Dart SDK supplies compilation, execution, and garbage collection.

## Pipeline

| Stage | Responsibility | Code |
|---|---|---|
| Load | Discover packages, select files, apply overlays, and retain language versions | `internal/load` |
| Verify | Reject unsupported constructs with source positions | `internal/verify` |
| Analyze | Determine which calls can block and enforce synchronous callbacks | `internal/analyze` |
| Normalize | Preserve evaluation order and form resumable control flow | `internal/normalize` |
| Emit | Write Dart code and type descriptors | `internal/emit` |
| Drive | Resolve resources, cache packages, write source maps, and execute | `internal/driver` |

The load uses the `dart` build tag. A package can use original Go source,
patched source, or a replacement overlay. `internal/overlay` defines those
choices. Changes to overlays require native/target parity and notice validation.

## Value representation

- Strings contain Go bytes, including invalid UTF-8. Dart string conversions are explicit.
- Slices share backing storage and retain separate length and capacity.
- Arrays and structs preserve Go value copies. Reference fields retain shared storage.
- Scalar addresses use address cells. Pointer identities are not machine addresses.
- Interfaces pair values with type descriptors and method tables.
- Generic code receives type information where Dart representation alone is insufficient.
- Integer operations use target-specific runtime support. See [JavaScript integers](js-integers.md).

Type descriptors support interfaces, reflection, copies, and host adapters.
Dynamic reflection still has limits. See [compatibility](parity.md).

## Control flow

Go goroutines share 1 Dart isolate. A call that blocks saves its continuation
and returns control to the scheduler. The implementation does not require
an `async` Dart function for each Go function.
See the [block model](block-model.md) for its frame and unwind invariants.

A named function type with `//gart:sync` defines a synchronous host callback.
The compiler rejects a callback that can block in that position.
Flutter builders and paint callbacks depend on this guarantee.

The selected Go language version comes from package and file metadata.
Before Go 1.22, loop variables share storage. Later language versions use
per-iteration variables where Go requires them.

## Caches and execution

Package caches include source, dependencies, language versions, and compiler
inputs. Unchanged output files retain their contents. Source maps use absolute
paths by default, or portable paths with `-trimpath`.

The resident Dart frontend retains compiler state across edits. Disk kernels
also require matching SDK and dependency identities. Applications always run
in fresh processes. Source mode uses the public Dart command directly.
See [SDK compatibility](sdk-upgrades.md) and [phase measurements](compilation-time.md).

## Standard library maintenance

Keep the package surface and overlay decisions in code, not duplicate prose
inventories. `make surface` compares exported names against the selected SDK.
`make surface ARGS=-absent` lists absent packages.
An exported-name match does not establish behavioral parity.

Upstream tests retain their original assertions and fixture bytes.
The [parity procedure](upstream-parity.md) defines source fingerprints,
exclusions, target coverage, and result validation.
The [license procedure](licenses.md) covers reused and generated material.
