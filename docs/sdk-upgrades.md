# SDK compatibility and upgrades

`internal/sdk/compatibility.json` records the Go and Dart CI baseline,
candidate Go versions, accepted resident Dart versions, and the development
Flutter SDK. The resident compiler and CI read this manifest.

A candidate entry does not establish complete support. The application
language version remains separate from the SDK that builds the compiler.
An accepted resident version does not establish parity across all packages
or platforms. Release evidence must identify the exact SDK and host.

## Go updates

1. Add the proposed version to `go.candidates`.
2. Compare upstream source changes, overlays, package surfaces, and licenses.
3. Build Gart and Glutter with the candidate SDK.
4. Run identical native and Gart upstream suites with the same selection.
5. Cover the declared module language versions and per-file version rules.
6. Validate modules, workspaces, replacements, vendor mode, tags, embeds, and coverage.
7. Run full upstream suites before a release baseline change.
8. Record host, source fingerprints, exclusions, results, and elapsed time.
9. Promote the candidate only after acceptance.
10. Update `go.mod`, source inventories, and the pinned container image together.

Do not replace source fingerprints merely to suppress an inventory failure.

## Dart and Flutter updates

1. Select the candidate Flutter SDK through FVM, or select a standalone Dart SDK.
2. Inspect SDK layout, CLI flags, platform files, and frontend protocol changes.
3. Add the candidate resident version for local validation.
4. Run the SDK, kernel, resident lifecycle, execution, and AOT tests.
5. Validate source edits, dependency changes, compile errors, recovery, and cache invalidation.
6. Compare native Go and generated Dart outputs for representative programs.
7. Measure cold, unchanged, and edited runs, with memory usage.
8. Regenerate Flutter host code against the candidate SDK.
9. Run core widget tests and retained local application tests.
10. Validate signals, callbacks, keys, focus, scrolling, disposal, and hot reload.
11. Repeat the relevant validation on supported hosts before a release claim.
12. Update the manifest, `.fvmrc`, and affected user instructions after acceptance.

The Go tests in `internal/sdk` detect drift in the module, FVM, and container
versions. CI reads its baseline directly from the manifest.
GitHub Actions still requires manual dispatch. Local validation does not
authorize a remote run.
