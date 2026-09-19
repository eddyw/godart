# Application parity fixtures

Run the pinned application comparisons from the repository root:

```sh
python3 scripts/parity.py applications --output build/application-parity
```

The command builds the current Gart compiler and compares all manifest entries
with native Go. Use a new output directory for each run. FVM users can prefix
the command with `fvm exec`.

The Fantasy fixture exercises JSON repair and schema generation without API
credentials or a service request. Its tests come from the original module.
The schema tests use the dependency's `safe` tag on both runtimes. That tag
selects the portable reflection implementation in Testify.

The module files pin all dependencies and checksums. Dependency source remains
in the Go module cache. No upstream implementation or assertion is copied here.
Fantasy uses Apache-2.0. JSON Schema, JSON Pointer, Go YAML, and Testify use MIT.
Testify also includes Go Spew material with its own permissive notice.
Generated output must retain these notices through the normal license collector.

A pass covers these packages and inputs. It does not establish parity for the
complete Fantasy SDK, an HTTP provider, or a terminal application.
