# Godart

> [!NOTE]
> Godart is my personal experiment. I share it as it stands, with no release
> schedule or support commitment.

Godart compiles Go to Dart. `gart` builds and runs Go programs.
`glutter` builds Flutter applications from Go structs, composition, and signals.
The generated Dart can target the VM, AOT, JavaScript, WebAssembly, and Flutter.

APIs and generated output can change. The project has no independent security
audit. Read the [compatibility limits](docs/parity.md) and
[security policy](SECURITY.md).

## Go compatibility

Use Go 1.27.0 to build Godart. Gart supports a subset of Go 1.27.
See the [compatibility guide](docs/parity.md) for package and target limits.

## Start

Install Go and Dart, then run these commands from this checkout:

```sh
make get
make build
export PATH="$PWD/build:$PATH"
cd /path/to/your/go/module
gart run -o dartout ./...
gart test -o dartout ./...
```

The compiler includes its Dart runtime. `-runtime` is an optional override.
Use `-trimpath` for portable source paths and `-timings` for phase durations.
The [SDK manifest](internal/sdk/compatibility.json) separates validated baselines
from candidates. See [SDK compatibility](docs/sdk-upgrades.md) before an upgrade.

## Flutter

From this checkout, select the development SDK through FVM:

```sh
fvm install
fvm exec ./build/glutter new hello
cd hello
fvm exec ../build/glutter run -d macos
```

Go code can compose shared core widgets and use generated bindings to Dart
libraries. Application-owned bindings remain experimental.
See the [Glutter guide](docs/glutter.md) and [binding guide](flutter/nodegen/README.md).

## More information

- [Gart commands](docs/gart.md) and [Glutter commands](docs/glutter.md)
- [Compatibility](docs/parity.md) and [upstream parity tests](docs/upstream-parity.md)
- [Architecture and developer guides](docs/README.md)
- [Project purpose](ABOUT.md), [contribution policy](CONTRIBUTING.md), and [security](SECURITY.md)

## License

Project-authored code uses the [BSD 3-Clause License](LICENSE).
The license covers project-authored work in this repository, including work
that predates the commit that introduced the license file.
Upstream material retains its own terms. See [license notices](docs/licenses.md).
