//go:build dart

// Package reflect is a stub. The go command puts reflect in the import list of
// each test binary, and gart writes its own entry point. The curated surface
// does not hold reflect yet, thus the verifier rejects an import of it.
package reflect
