// Package driver runs the compiler over a Go module and writes a Dart package.
package driver

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"go/types"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"golang.org/x/tools/go/gcexportdata"
	"golang.org/x/tools/go/packages"

	"github.com/eddyw/godart/internal/analyze"
	"github.com/eddyw/godart/internal/emit"
	"github.com/eddyw/godart/internal/load"
	"github.com/eddyw/godart/internal/meta"
	"github.com/eddyw/godart/internal/normalize"
	"github.com/eddyw/godart/internal/overlay"
	"github.com/eddyw/godart/internal/verify"
	runtimelib "github.com/eddyw/godart/runtime"
)

// Options holds the input of a build.
type Options struct {
	Dir      string
	Patterns []string
	Out      string
	Runtime  string
	Format   bool
	TrimPath bool
	// Trace receives the elapsed duration of each build or execution phase.
	Trace func(string, time.Duration)
	// NoCache compiles each package again, also when the cache holds it.
	NoCache bool
	// Tests compiles the test variants of each package, and 1 entry point for
	// each test binary.
	Tests bool
	// Cover writes a counter into each statement of each package under test.
	// It needs Tests.
	Cover bool
	// Filter gives a source that replaces a file of the build. The driver
	// calls it in front of each load, thus a watch cycle reads the state of
	// that moment and a new declaration reaches the build at once. A nil field
	// replaces no file, and a build with Tests reads it not.
	Filter func() (load.Filter, error)
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
	// Dir is the directory of the package under test. The binary runs in that
	// directory.
	Dir string
	// Tests is the count of test functions of the binary.
	Tests int
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
	phase := time.Now()
	result, err := loadModule(opt)
	opt.trace("load", phase)
	if err != nil {
		return nil, err
	}
	if opt.Trace != nil {
		for _, timing := range result.Timings {
			opt.Trace(timing.Name, timing.Duration)
		}
	}
	roots := result.Packages
	module := moduleOf(roots)
	pkgs := order(roots, module)
	phase = time.Now()
	for _, p := range pkgs {
		if err := verify.Package(p.Fset, p.Syntax, p.TypesInfo, result.Embeds); err != nil {
			return nil, err
		}
	}
	opt.trace("verify", phase)
	phase = time.Now()
	blocks := analyze.Module(pkgs)
	opt.trace("analyze", phase)
	if err := blocks.SyncErrors(); err != nil {
		return nil, err
	}
	phase = time.Now()
	dartPkg := dartPackageName(module)
	store := meta.Open(filepath.Join(opt.Out, ".gart"))
	stats := &Stats{}
	exports := map[string]string{}
	files := map[string]string{}
	retained := map[string]bool{}
	var maps []srcPackage
	for _, p := range pkgs {
		bin := result.Binary(p.ID)
		sources, err := readSources(p, result.Sources, bin)
		if err != nil {
			return nil, err
		}
		flags := fmt.Sprintf("%s\x00trimpath=%t", dartPkg, opt.TrimPath)
		if !opt.TrimPath {
			var paths []string
			for _, file := range p.Syntax {
				paths = append(paths, p.Fset.PositionFor(file.Pos(), false).Filename)
			}
			slices.Sort(paths)
			flags += "\x00" + strings.Join(paths, "\x00")
		}
		sourceHash := meta.SourceHash(sources, dependencyHashes(p, module, exports),
			flags, blocks.Digest(p))
		if !opt.NoCache {
			if old, ok := store.Read(p.ID); ok && old.SourceHash == sourceHash &&
				outputExists(opt.Out, old.Files) {
				for _, path := range old.Files {
					retained[path] = true
				}
				exports[p.ID] = old.ExportHash
				stats.Cached = append(stats.Cached, p.ID)
				continue
			}
		}
		out, err := compile(p, module, dartPkg, blocks, bin, opt.TrimPath)
		if err != nil {
			return nil, err
		}
		dir := outputDir(p, module)
		written := make([]string, 0, len(out.Files)+len(out.Shared))
		for path, text := range out.Files {
			full := filepath.Join(dir, path)
			files[full] = text
			written = append(written, full)
		}
		// A shared file stands outside the directory of the package, and each
		// package that holds the shape writes it with equal bytes. The archive
		// lists it, thus a cached package still needs the file on disk.
		for path, text := range out.Shared {
			files[path] = text
			written = append(written, path)
		}
		// The order of a map over the files is random, thus a sort gives the
		// source map and the archive an equal order at each build.
		sort.Strings(written)
		mapAt := filepath.Join(dir, out.MapFile)
		maps = append(maps, srcPackage{base: baseName(p, module), mapAt: mapAt,
			files: append([]string{}, written...)})
		written = append(written, mapAt)
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
	opt.trace("emit", phase)
	phase = time.Now()
	runtime, err := runtimePath(opt.Runtime, opt.Out)
	if err != nil {
		return nil, err
	}
	files["pubspec.yaml"] = pubspec(dartPkg, module, runtime)
	if err := writeFiles(opt.Out, files); err != nil {
		return nil, err
	}
	if err := pruneParts(opt.Out, files, retained); err != nil {
		return nil, err
	}
	opt.trace("write", phase)
	phase = time.Now()
	if opt.Format && len(stats.Compiled) > 0 {
		if err := format(opt.Out, dartFiles(files)); err != nil {
			return nil, err
		}
	}
	opt.trace("format", phase)
	phase = time.Now()
	if err := writeSourceMaps(opt.Out, dartPkg, maps); err != nil {
		return nil, err
	}
	opt.trace("source-maps", phase)
	stats.Binaries = testBinaries(result, module)
	stats.NoTests = withoutTests(result, module)
	return stats, nil
}

func (opt Options) trace(phase string, start time.Time) {
	if opt.Trace != nil {
		opt.Trace(phase, time.Since(start))
	}
}

func loadModule(opt Options) (*load.Result, error) {
	if opt.Tests && opt.Cover {
		return load.LoadTestsCover(opt.Dir, opt.Patterns...)
	}
	if opt.Tests {
		return load.LoadTests(opt.Dir, opt.Patterns...)
	}
	if opt.Filter == nil {
		return load.Load(opt.Dir, opt.Patterns...)
	}
	filter, err := opt.Filter()
	if err != nil {
		return nil, err
	}
	return load.LoadFiltered(opt.Dir, filter, opt.Patterns...)
}

// testBinaries names the Dart entry point of each test binary of the build.
func testBinaries(result *load.Result, module string) []TestBinary {
	out := make([]TestBinary, 0, len(result.Binaries))
	for _, b := range result.Binaries {
		out = append(out, TestBinary{
			Package:    b.Under,
			Entry:      filepath.Join("bin", testDir(b.Under, module)+"_test.dart"),
			Dir:        b.Dir,
			Tests:      len(b.Tests),
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
	bin *load.Binary, trimpath bool) (*emit.Result, error) {
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
	return emit.Package(tree, p.Types, emit.Config{
		Base:        baseName(p, module),
		Imports:     imports,
		DartPackage: dartPkg,
		Fset:        p.Fset,
		TrimPath:    trimpath,
		URIs:        libraryURIs(p, module, dartPkg),
	})
}

// libraryURIs gives the Dart URI of each package that this package can name. A
// file of an unnamed struct type reads it, because such a file stands outside
// each package and still imports the type of a field.
func libraryURIs(p *packages.Package, module, dartPkg string) map[string]string {
	out := map[string]string{}
	packages.Visit([]*packages.Package{p}, nil, func(d *packages.Package) {
		if !compiled(d.PkgPath, module) {
			return
		}
		if uri := libraryURI(d, module, dartPkg); uri != "" {
			out[d.PkgPath] = uri
		}
	})
	return out
}

// compiled reports whether the build writes Dart code for a package. The
// module belongs to the build, and so does each package of the curated
// standard library surface and each package of a dependency module.
func compiled(pkgPath, module string) bool {
	if overlay.Intrinsic(pkgPath) {
		return false
	}
	if strings.HasPrefix(pkgPath, module) {
		return true
	}
	if module == stdModule && testVariant(pkgPath) {
		return true
	}
	if slices.Contains(overlay.Surface(), pkgPath) {
		return true
	}
	return external(pkgPath)
}

// external reports whether a path names a package of a dependency module. The
// first element of a module path holds a dot, and no path of the standard
// library holds one. The verifier reads the same rule at
// internal/overlay/native.go, thus stage 2 accepts the import that this
// function accepts.
func external(pkgPath string) bool {
	first, _, _ := strings.Cut(pkgPath, "/")
	return strings.Contains(first, ".")
}

// stdModule is the module of a build of a GOROOT package. No import path of
// the standard library holds it as a prefix, thus each package of the surface
// reaches the build through the surface list.
const stdModule = "std"

// testVariant reports whether a path names the external test package of a
// GOROOT package, or the entry point of its test binary. The go command
// reserves both suffixes, thus no other package carries them.
func testVariant(pkgPath string) bool {
	return strings.HasSuffix(pkgPath, "_test") || strings.HasSuffix(pkgPath, ".test")
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

// moduleOf gives the module path of the roots.
//
// The go command reports no module for a package of GOROOT, and the file
// GOROOT/src/go.mod names that module std. Thus a build of such a package takes
// stdModule.
func moduleOf(roots []*packages.Package) string {
	for _, p := range roots {
		if p.Module != nil {
			return p.Module.Path
		}
	}
	return stdModule
}

// libPath mirrors the Go import path in the Dart directory structure. A
// standard library package goes under the directory std, a package of a
// dependency module goes under the directory dep, and a test variant goes under
// the directory of its test binary.
func libPath(p *packages.Package, module string) string {
	if under := underTest(p.ID); under != "" {
		return "gart_test/" + testDir(under, module) + "/" + modulePath(p, module)
	}
	if !strings.HasPrefix(p.PkgPath, module) {
		if external(p.PkgPath) {
			return "dep/" + p.PkgPath
		}
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

// testMainPath gives the path of the entry point of a test binary, relative to
// the directory bin. The path mirrors the import path of the package under
// test.
func testMainPath(p *packages.Package, module string) string {
	return testDir(strings.TrimSuffix(p.PkgPath, ".test"), module) + "_test"
}

func baseName(p *packages.Package, module string) string {
	if isTestMain(p) {
		return filepath.Base(testMainPath(p, module))
	}
	if p.Name == "main" && underTest(p.ID) == "" {
		return "main"
	}
	return filepath.Base(libPath(p, module))
}

func outputDir(p *packages.Package, module string) string {
	if isTestMain(p) {
		return filepath.Join("bin", filepath.Dir(testMainPath(p, module)))
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
		files := map[string]string{}
		if err := fs.WalkDir(runtimelib.Files, "goruntime", func(name string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			data, err := runtimelib.Files.ReadFile(name)
			if err != nil {
				return err
			}
			files[name] = string(data)
			return nil
		}); err != nil {
			return "", err
		}
		if err := writeFiles(out, files); err != nil {
			return "", err
		}
		return "goruntime", nil
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

// Removed Go files must not leave orphan parts for the Dart analyzer.
// Each directory belongs to an emitted package. Preserve files without the
// compiler header, including manually authored Dart parts.
func pruneParts(out string, files map[string]string, retained map[string]bool) error {
	dirs := map[string]bool{}
	for path := range files {
		if strings.HasSuffix(path, ".part.dart") {
			dirs[filepath.Dir(path)] = true
		}
	}
	for dir := range dirs {
		entries, err := os.ReadDir(filepath.Join(out, dir))
		if err != nil {
			return err
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			if entry.IsDir() || !strings.HasSuffix(path, ".part.dart") {
				continue
			}
			if _, keep := files[path]; keep || retained[path] {
				continue
			}
			full := filepath.Join(out, path)
			data, err := os.ReadFile(full)
			if err != nil {
				return err
			}
			if bytes.HasPrefix(data, []byte("// Generated by gart. Do not edit.\n")) {
				if err := os.Remove(full); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// dartFiles names each Dart file of the map, in a stable order. The map holds
// the files that the build wrote, thus a cached package gives no path.
func dartFiles(files map[string]string) []string {
	out := make([]string, 0, len(files))
	for path := range files {
		if filepath.Ext(path) == ".dart" {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

// argBudget limits the bytes of 1 argument list. The limit of the operating
// system is near 1 megabyte, thus 128 kilobytes keeps a wide margin.
const argBudget = 128 << 10

// format runs dart format on the given files, which stand under out. A build
// that wrote no Dart file runs no command. A long list needs more than 1 run,
// because the argument list of a process has a limit.
func format(out string, paths []string) error {
	for len(paths) > 0 {
		n, size := 0, 0
		for n < len(paths) && (n == 0 || size+len(paths[n])+1 <= argBudget) {
			size += len(paths[n]) + 1
			n++
		}
		args := append([]string{"format"}, paths[:n]...)
		cmd := exec.Command("dart", args...)
		cmd.Dir = out
		if text, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("dart format: %v\n%s", err, text)
		}
		paths = paths[n:]
	}
	return nil
}
