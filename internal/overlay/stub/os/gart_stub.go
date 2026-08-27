//go:build dart

// Package os is a stub. The go command puts os in the import list of each test
// binary, and gart writes its own entry point. The curated surface does not
// hold os yet, thus the verifier rejects an import of it.
package os
