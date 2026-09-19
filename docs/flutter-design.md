# Flutter architecture

Glutter combines 3 layers: shared Go primitives, generated Dart bindings,
and application components written in Go.

## Shared core

`gart.dev/ui` supplies nodes, values, signals, controllers, identity keys,
and synchronous callback types. A Go component can return a subtree of these
nodes. It does not need a separate Dart widget class.

The Dart host creates real Flutter widgets. Flutter owns elements, layout,
focus, scrolling, rendering, and platform integration. Identity keys retain
host objects across Go rebuilds. The host disposes objects after their keys
leave the tree.

Go code does not receive a general `BuildContext`. Theme and media access use
host snapshots. Context-dependent APIs require an explicit host adapter.

## Application-owned libraries

`glutter bind` reads the application's resolved Dart libraries and explicit
manifest selections. It writes Go constructor and service declarations.
The generated host uses package-qualified identities, including equal class
names from different libraries.

A Go design library can compose shared primitives. A Dart design library uses
constructor bindings. Both can share Go signals and normal Flutter child slots.
This architecture permits an independent design system. The current adapter
surface does not yet guarantee all features of a complete Material replacement.

The [binding guide](../flutter/nodegen/README.md) defines manifest syntax,
generation order, supported types, output ownership, and current omissions.
Existing `gart.dev/ui/material`, `cupertino`, and `svg` packages remain available
while applications adopt their own bindings.

## State

The Go state layer works under native Go tests without Flutter.
Signals use explicit `Get`, `Set`, and `Update` operations. `Computed` derives
values, `Effect` observes them, and `Batch` combines notifications.
`Untracked` reads without a dependency record.

`ui.Watch` creates a rebuild boundary. A changed signal marks its readers,
which permits focused subtree rebuilds. Flutter retains the associated elements
and host objects when their identities remain stable.

Keep the graph on its owner goroutine. Use `Post` for worker results and
`Pump` to apply them. Reactive objects must not move through value copies
after use. Native `go vet` and runtime identity checks cover this constraint.

`Store`, reducers, and commands provide alternative application structures.
They do not require a different host or compiler representation.

## Callbacks and asynchronous services

Builders and paint callbacks use named `//gart:sync` types. The compiler rejects
blocking work in those callbacks. Event callbacks can start goroutines.

The scalar service adapter translates futures to Go results and errors.
Streams expose explicit consumption and cancellation. Runtime disposal cancels
active streams. General plugins can also use an application-defined `ui.Call`
handler. JSON service transport does not preserve arbitrary Dart object identity.

`ui.Painter` currently exposes rectangles, circles, and lines.
The canvas expires after the callback. Change `Revision` when its output changes.

## Validation

`make flutter-go` validates native state behavior.
`fvm exec make flutter-test` validates retained fields, generated hosts,
callbacks, signals, keys, focus, scrolling, disposal, reassembly, services,
and paint output. It needs no ignored example application.
