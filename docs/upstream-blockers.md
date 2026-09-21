# Upstream suite blockers

The [coverage manifest](../internal/parity/testdata/coverage-manifest.json)
records current package status, exact diagnostics, platform differences,
blocker causes, and reasons. Use that manifest for the package inventory.
This document defines the classification rules without a second inventory.

A first diagnostic can hide further dependencies. A blocked package stays
in the coverage denominator. Portable assertions can execute separately.
A successful compile does not establish a behavioral pass.

## Causes

- `public-api-gap`: an exported API or its behavior is absent.
- `private-test-dependency`: an upstream test needs private implementation
  state or a test-only export that the adapter does not provide.
- `runtime-boundary`: an assertion depends on native runtime or host behavior.
  This cause does not exclude all tests in the package or prove that a fix is impossible.

Keep implementation gaps separate from agreed target exclusions.
Do not add empty private helpers or remove original assertions to obtain a pass.

## Validation

Run the diagnostic comparison against the installed Go SDK:

```sh
go test ./internal/driver -run '^TestUpstreamCompileBlockers$' -count=1
```

This test compares the first compile diagnostic for each compile-blocked package.
It also requires runtime-blocked packages to compile. It does not execute their tests.
A new diagnostic or a newly successful compile requires classification review.

Run the original tests before a status change to a behavioral pass.
Preserve source identities, platform details, SDK versions, and explicit exclusions.
Read [Upstream parity](upstream-parity.md) for the result requirements.
