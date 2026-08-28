// Package driver runs the compiler over a Go module and writes a Dart package.
package driver

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"golang.org/x/tools/go/gcexportdata"
	"golang.org/x/tools/go/packages"

	"github.com/eddyw/godart/internal/analyze"
	"github.com/eddyw/godart/internal/emit"
	"github.com/eddyw/godart/internal/load"
	"github.com/eddyw/godart/internal/meta"
	"github.com/eddyw/godart/internal/normalize"
	"github.com/eddyw/godart/internal/overlay"
	"github.com/eddyw/godart/internal/verify"
)

// Options holds the input of a build.
type Options struct {
	Dir      string
	Patterns []string
	Out      string
	Runtime  string
	Format   bool
	// NoCache compiles each package again, also when the cache holds it.
	NoCache bool
	// Tests compiles the test variants of each package, and 1 entry point for
	// each test binary.
	Tests bool
}

// Stats reports what the build did. A cached package needs no second compile.
type Stats struct {
	Compiled []string
	Cached   []string
	// Binaries lists the test binary of each package that holds a test. It is
	// empty when the build asked for no test.
	Binaries []TestBinary
	// NoTests lists each package of the module that holds no test file.
	NoTests []string
}

// TestBinary names the Dart entry point of 1 test binary.
type TestBinary struct {
	// Package is the import path of the package under test.
	Package string
	// Entry is the path of the Dart file, relative to the output directory.
	Entry string
	// Tests is the count of test functions of the binary.
	Tests int
	// Skips lists each function of a test file that gart does not run.
	Skips []string
	// Benchmarks is the count of benchmark functions of the binary.
	Benchmarks int
}

// Build compiles the packages and writes the Dart package into Out.
func Build(opt Options) error {
	_, err := BuildWithStats(opt)
	return err
}

// BuildWithStats compiles the packages and reports the state of each one.
func BuildWithStats(opt Options) (*Stats, error) {
	result, err := loadModule(opt)
	if err != nil {
		return nil, err
	}
	roots := result.Packages
	module := moduleOf(roots)
	if module == "" {
		return nil, fmt.Errorf("the packages belong to no module")
	}
	pkgs := order(roots, module)
	for _, p := range pkgs {
		if err := verify.Package(p.Fset, p.Syntax, p.TypesInfo); err != nil {
			return nil, err
		}
	}
	blocks := analyze.Module(pkgs)
	dartPkg := dartPackageName(module)
	store := meta.Open(filepath.Join(opt.Out, ".gart"))
	stats := &Stats{}
	exports := map[string]string{}
	files := map[string]string{}
	for _, p := range pkgs {
		bin := result.Binary(p.ID)
		sources, err := readSources(p, result.Sources, bin)
		if err != nil {
			return nil, err
		}
		sourceHash := meta.SourceHash(sources, dependencyHashes(p, module, exports), dartPkg)
		if !opt.NoCache {
			if old, ok := store.Read(p.ID); ok && old.SourceHash == sourceHash &&
				outputExists(opt.Out, old.Files) {
				exports[p.ID] = old.ExportHash
				stats.Cached = append(stats.Cached, p.ID)
				continue
			}
		}
		out, err := compile(p, module, dartPkg, blocks, bin)
		if err != nil {
			return nil, err
		}
		dir := outputDir(p, module)
		written := make([]string, 0, len(out.Files))
		for path, text := range out.Files {
			full := filepath.Join(dir, path)
			files[full] = text
			written = append(written, full)
		}
		sort.Strings(written)
		export, err := exportData(p)
		if err != nil {
			return nil, err
		}
		blocking := blockingExports(p, blocks)
		archive := &meta.Archive{
			Path:       p.ID,
			Name:       p.Name,
			SourceHash: sourceHash,
			ExportHash: meta.ExportHash(export, blocking),
			Export:     base64.StdEncoding.EncodeToString(export),
			Blocking:   blocking,
			Files:      written,
		}
		exports[p.ID] = archive.ExportHash
		if err := store.Write(archive); err != nil {
			return nil, err
		}
		stats.Compiled = append(stats.Compiled, p.ID)
	}
	runtime, err := runtimePath(opt.Runtime, opt.Out)
	if err != nil {
		return nil, err
	}
	files["pubspec.yaml"] = pubspec(dartPkg, module, runtime)
	if err := writeFiles(opt.Out, files); err != nil {
		return nil, err
	}
	if opt.Format && len(stats.Compiled) > 0 {
		if err := format(opt.Out); err != nil {
			return nil, err
		}
	}
	stats.Binaries = testBinaries(result, module)
	stats.NoTests = withoutTests(result, module)
	return stats, nil
}

func loadModule(opt Options) (*load.Result, error) {
	if opt.Tests {
		return load.LoadTests(opt.Dir, opt.Patterns...)
	}
	pkgs, err := load.Load(opt.Dir, opt.Patterns...)
	if err != nil {
		return nil, err
	}
	return &load.Result{Packages: pkgs}, nil
}

// testBinaries names the Dart entry point of each test binary of the build.
func testBinaries(result *load.Result, module string) []TestBinary {
	out := make([]TestBinary, 0, len(result.Binaries))
	for _, b := range result.Binaries {
		out = append(out, TestBinary{
			Package:    b.Under,
			Entry:      filepath.Join("bin", testDir(b.Under, module)+"_test.dart"),
			Tests:      len(b.Tests),
			Skips:      b.Skips,
			Benchmarks: len(b.Benchmarks),
		})
	}
	return out
}

// withoutTests lists each package of the module that holds no test file. The
// test command reports such a package, exactly as go test does.
func withoutTests(result *load.Result, module string) []string {
	tested := map[string]bool{}
	for _, b := range result.Binaries {
		tested[b.Under] = true
	}
	var out []string
	for _, p := range result.Packages {
		if p.ID != p.PkgPath || !strings.HasPrefix(p.PkgPath, module) {
			continue
		}
		if tested[p.PkgPath] || strings.HasSuffix(p.PkgPath, ".test") {
			continue
		}
		out = append(out, p.PkgPath)
	}
	sort.Strings(out)
	return out
}

// readSources gives the content of each file of a package. The overlay of the
// load replaces a file of the standard library and the entry point of a test
// binary, thus the disk holds no such content.
func readSources(p *packages.Package, sources map[string][]byte, bin *load.Binary) (map[string][]byte, error) {
	out := make(map[string][]byte, len(p.CompiledGoFiles))
	for _, name := range p.CompiledGoFiles {
		key := name
		if bin != nil && name == bin.File {
			key = load.TestMainName
		}
		if text, ok := sources[name]; ok {
			out[key] = text
			continue
		}
		text, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		out[key] = text
	}
	return out, nil
}

// dependencyHashes lists the export hash of each dependency inside the module.
// A change inside a function body of a dependency keeps its export hash equal,
// thus this package keeps its cache.
func dependencyHashes(p *packages.Package, module string, exports map[string]string) []string {
	var out []string
	for _, dep := range p.Imports {
		if !compiled(dep.PkgPath, module) {
			continue
		}
		out = append(out, dep.ID+"="+exports[dep.ID])
	}
	return out
}

// exportData writes the exported type information of a package.
func exportData(p *packages.Package) ([]byte, error) {
	var buf bytes.Buffer
	if err := gcexportdata.Write(&buf, p.Fset, p.Types); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// blockingExports lists each exported function of the package that can block.
// The block state belongs to the exported interface, because it changes each
// call site.
func blockingExports(p *packages.Package, blocks *analyze.Result) []string {
	var out []string
	scope := p.Types.Scope()
	for _, n := range scope.Names() {
		switch obj := scope.Lookup(n).(type) {
		case *types.Func:
			if blocks.Blocks(obj) {
				out = append(out, obj.Name())
			}
		case *types.TypeName:
			named, ok := types.Unalias(obj.Type()).(*types.Named)
			if !ok {
				continue
			}
			for i := 0; i < named.NumMethods(); i++ {
				m := named.Method(i)
				if blocks.Blocks(m) {
					out = append(out, obj.Name()+"."+m.Name())
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

func outputExists(out string, files []string) bool {
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			return false
		}
	}
	return true
}

func compile(p *packages.Package, module, dartPkg string, blocks *analyze.Result,
	bin *load.Binary) (*emit.Result, error) {
	tree, err := normalize.Package(p.Fset, p.Syntax, p.TypesInfo, p.Types, blocks)
	if err != nil {
		return nil, err
	}
	if bin != nil {
		for _, f := range tree.Files {
			f.Name = load.TestMainName
		}
	}
	var imports []emit.Import
	for path, dep := range p.Imports {
		if !compiled(dep.PkgPath, module) {
			continue
		}
		uri := libraryURI(dep, module, dartPkg)
		if uri == "" {
			continue
		}
		imports = append(imports, emit.Import{Path: path, URI: uri})
	}
	sort.Slice(imports, func(i, j int) bool { return imports[i].Path < imports[j].Path })
	return emit.Package(tree, p.Types, baseName(p, module), imports, blocks)
}

// compiled reports whether the build writes Dart code for a package. The
// module belongs to the build, and so does each package of the curated
// standard library surface.
func compiled(pkgPath, module string) bool {
	if overlay.Intrinsic(pkgPath) {
		return false
	}
	if strings.HasPrefix(pkgPath, module) {
		return true
	}
	return slices.Contains(overlay.Paths(), pkgPath)
}

// order sorts the packages of the module in dependency order. A package
// compiles after each package that it imports.
//
// The key is the identifier and not the import path, because a test variant
// carries the import path of the package under test.
func order(roots []*packages.Package, module string) []*packages.Package {
	seen := map[string]bool{}
	var out []*packages.Package
	var visit func(p *packages.Package)
	visit = func(p *packages.Package) {
		if seen[p.ID] || !compiled(p.PkgPath, module) {
			return
		}
		seen[p.ID] = true
		paths := make([]string, 0, len(p.Imports))
		for path := range p.Imports {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			visit(p.Imports[path])
		}
		out = append(out, p)
	}
	for _, p := range roots {
		visit(p)
	}
	return out
}

func moduleOf(roots []*packages.Package) string {
	for _, p := range roots {
		if p.Module != nil {
			return p.Module.Path
		}
	}
	return ""
}

// libPath mirrors the Go import path in the Dart directory structure. A
// standard library package goes under the directory std, and a test variant
// goes under the directory of its test binary.
func libPath(p *packages.Package, module string) string {
	if under := underTest(p.ID); under != "" {
		return "gart_test/" + testDir(under, module) + "/" + modulePath(p, module)
	}
	if !strings.HasPrefix(p.PkgPath, module) {
		return "std/" + p.PkgPath
	}
	return modulePath(p, module)
}

func modulePath(p *packages.Package, module string) string {
	rel := strings.TrimPrefix(strings.TrimPrefix(p.PkgPath, module), "/")
	if rel == "" {
		return p.Name
	}
	return rel
}

// testDir names the directory of 1 test binary. It mirrors the path of the
// package under test inside the module.
func testDir(under, module string) string {
	rel := strings.TrimPrefix(strings.TrimPrefix(under, module), "/")
	if rel == "" {
		return "main"
	}
	return rel
}

// underTest reads the package under test from the identifier of a test
// variant. The result is empty when the identifier names no variant.
func underTest(id string) string {
	open := strings.Index(id, " [")
	if open < 0 || !strings.HasSuffix(id, ".test]") {
		return ""
	}
	return strings.TrimSuffix(id[open+2:len(id)-1], ".test")
}

// isTestMain reports whether a package is the entry point of a test binary.
func isTestMain(p *packages.Package) bool {
	return p.Name == "main" && strings.HasSuffix(p.PkgPath, ".test")
}

func baseName(p *packages.Package, module string) string {
	if isTestMain(p) {
		return testDir(strings.TrimSuffix(p.PkgPath, ".test"), module) + "_test"
	}
	if p.Name == "main" && underTest(p.ID) == "" {
		return "main"
	}
	return filepath.Base(libPath(p, module))
}

func outputDir(p *packages.Package, module string) string {
	if isTestMain(p) {
		return filepath.Join("bin", filepath.Dir(baseName(p, module)))
	}
	if p.Name == "main" && underTest(p.ID) == "" {
		return "bin"
	}
	return filepath.Join("lib", filepath.Dir(libPath(p, module)))
}

// libraryURI gives the Dart URI of a package. A main package has no URI,
// because no library imports it.
func libraryURI(p *packages.Package, module, dartPkg string) string {
	if isTestMain(p) || (p.Name == "main" && underTest(p.ID) == "") {
		return ""
	}
	return "package:" + dartPkg + "/" + libPath(p, module) + ".dart"
}

// dartPackageName maps a Go module path to a pub package name. A pub name
// holds lower case letters, digits, and the underscore.
func dartPackageName(module string) string {
	base := module
	if k := strings.LastIndex(base, "/"); k >= 0 {
		base = base[k+1:]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(base) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" || out[0] >= '0' && out[0] <= '9' {
		out = "go_" + out
	}
	return out
}

func pubspec(dartPkg, module, runtime string) string {
	return fmt.Sprintf(`name: %s
description: The Dart output of the Go module %s.
publish_to: none

environment:
  sdk: ^3.13.0

dependencies:
  goruntime:
    path: %s
`, dartPkg, module, runtime)
}

func runtimePath(runtime, out string) (string, error) {
	if runtime == "" {
		return "", fmt.Errorf("the build needs the path of the goruntime package")
	}
	abs, err := filepath.Abs(runtime)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(abs, "pubspec.yaml")); err != nil {
		return "", fmt.Errorf("%s holds no Dart package: %w", abs, err)
	}
	return abs, nil
}

// writeFiles writes a file only when its content changed. An equal file keeps
// its time stamp, thus the incremental compiler of Dart keeps its cache.
func writeFiles(out string, files map[string]string) error {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		full := filepath.Join(out, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o777); err != nil {
			return err
		}
		if old, err := os.ReadFile(full); err == nil && string(old) == files[path] {
			continue
		}
		if err := os.WriteFile(full, []byte(files[path]), 0o666); err != nil {
			return err
		}
	}
	return nil
}

func format(out string) error {
	cmd := exec.Command("dart", "format", ".")
	cmd.Dir = out
	if text, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("dart format: %v\n%s", err, text)
	}
	return nil
}
