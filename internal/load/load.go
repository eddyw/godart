// Package load runs stage 1 of the compiler. It reads a Go module and returns
// the packages with their syntax trees and their type information.
package load

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/eddyw/godart/internal/overlay"
)

// BuildTag selects the standard library overlays of the compiler.
const BuildTag = "dart"

const mode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo |
	packages.NeedModule

const findMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedModule

// Result holds the packages of 1 load.
type Result struct {
	// Timings separates metadata discovery from syntax and type loading.
	Timings []PhaseTiming
	// Packages holds the root packages in the order of their identifier.
	Packages []*packages.Package
	// Sources gives the content of each file that the overlay replaced. A key
	// is the file path that the go command reports.
	Sources map[string][]byte
	// Binaries lists 1 test binary for each package that holds a test. It is
	// empty when the load asked for no test.
	Binaries []*Binary
	// Embeds holds the path of each file of the module that carried a
	// `//go:embed` directive. The loader rewrote each 1, thus stage 2 allows
	// the constructor of the package embed there.
	Embeds         map[string]bool
	DefaultGODEBUG map[string]string
}

type PhaseTiming struct {
	Name     string
	Duration time.Duration
}

// Binary reads the test binary of 1 package.
func (r *Result) Binary(id string) *Binary {
	for _, b := range r.Binaries {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// Filter replaces a file of the build before the load. A key is the path of
// the file on disk, and the value is the source that the compiler reads. The
// file on disk never changes.
//
// A Flutter application reads it: `internal/nodefilter` drops each node type
// that the application does not name, thus the Dart output holds no class and
// no type descriptor of a widget that no code builds.
type Filter map[string][]byte

// Load reads the packages that match the patterns in dir.
//
// The load runs 2 times. The first load finds the files of each standard
// library package, and the second load replaces those files with the overlay.
func Load(dir string, patterns ...string) (*Result, error) {
	return load(dir, false, false, nil, patterns, "", nil)
}

// LoadTests reads the packages and also each test variant of them. It replaces
// the test entry point of the go command with the entry point of the compiler.
func LoadTests(dir string, patterns ...string) (*Result, error) {
	return load(dir, true, false, nil, patterns, "", nil)
}

// Options configures package loading and an optional overlay-discovery cache.
type Options struct {
	Dir       string
	Patterns  []string
	BuildTags []string
	CacheDir  string
	Tests     bool
	Cover     bool
	Filter    Filter
}

func LoadWithOptions(opt Options) (*Result, error) {
	return load(opt.Dir, opt.Tests, opt.Cover, opt.Filter, opt.Patterns, opt.CacheDir, opt.BuildTags)
}

func load(dir string, tests, cover bool, filter Filter, patterns []string, cacheDir string, tags []string) (*Result, error) {
	var timings []PhaseTiming
	phase := time.Now()
	sdk, err := prepareGoSDK(dir)
	if err != nil {
		return nil, err
	}
	found, err := packages.Load(&packages.Config{
		Env:        sdk.env,
		Mode:       findMode,
		Dir:        dir,
		BuildFlags: []string{tagFlag(tags)},
		Tests:      tests,
	}, patterns...)
	if err != nil {
		return nil, err
	}
	timings = append(timings, PhaseTiming{"load-discovery", time.Since(phase)})
	phase = time.Now()
	var surface []*packages.Package
	if cacheDir != "" {
		surface, err = cachedSurface(dir, cacheDir, sdk.env, tags...)
	} else {
		surface, err = surfacePackages(dir, sdk.env, tags)
	}
	if err != nil {
		return nil, err
	}
	timings = append(timings, PhaseTiming{"load-surface", time.Since(phase)})
	phase = time.Now()
	files, err := overlayFiles(found, surface, sdk.root)
	if err != nil {
		return nil, err
	}
	// The filter stands in front of the rewrite of the directives, because a
	// file that it replaces holds no `//go:embed` directive.
	for name, content := range filter {
		files[name] = content
	}
	embeds, err := embedRewrites(found, files)
	if err != nil {
		return nil, err
	}
	var binaries []*Binary
	if tests {
		binaries, err = testBinaries(found, cover, files, sdk.env)
		if err != nil {
			return nil, err
		}
		for _, b := range binaries {
			files[b.File] = b.Source
			for name, content := range b.Cover {
				files[name] = content
			}
		}
	}
	// Go rejects overlays beneath GOMODCACHE. Embed rewrites preserve imports
	// and file selection, thus only the compiler parser needs their content.
	goFiles := make(map[string][]byte, len(files))
	for name, content := range files {
		if !embeds[name] {
			goFiles[name] = content
		}
	}
	var rangeExtras sync.Map
	cfg := &packages.Config{
		Env:        sdk.env,
		Mode:       mode,
		Dir:        dir,
		BuildFlags: []string{tagFlag(tags)},
		Tests:      tests,
		Overlay:    goFiles,
		ParseFile: func(fset *token.FileSet, name string, src []byte) (*ast.File, error) {
			if embeds[name] {
				src = files[name]
			}
			file, err := parser.ParseFile(fset, name, src, parser.AllErrors|parser.ParseComments)
			if err != nil {
				if diagnostics := compilerSyntaxErrors(fset, name, src, file); len(diagnostics) != 0 {
					return file, diagnostics
				}
				file, err = repairRangeClauses(fset, name, src, file, err, &rangeExtras)
			}
			if err == nil {
				if directives := misplacedDirectives(fset, file, src); len(directives) != 0 {
					return file, directives
				}
			}
			return file, err
		},
	}
	timings = append(timings, PhaseTiming{"load-overlay", time.Since(phase)})
	phase = time.Now()
	pkgs, err := packages.Load(cfg, patterns...)
	timings = append(timings, PhaseTiming{"load-types", time.Since(phase)})
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package matches %v in %s", patterns, dir)
	}
	if err := collectErrors(pkgs, files, &rangeExtras); err != nil {
		return nil, err
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].ID < pkgs[j].ID })
	var defaults map[string]string
	for _, pkg := range pkgs {
		if pkg.Name == "main" {
			phase = time.Now()
			defaults, err = defaultGODEBUG(dir, patterns, tags, tests, sdk.env)
			if err != nil {
				return nil, err
			}
			timings = append(timings, PhaseTiming{"load-godebug", time.Since(phase)})
			break
		}
	}
	return &Result{Timings: timings, Packages: pkgs, Sources: files, Binaries: binaries, Embeds: embeds, DefaultGODEBUG: defaults}, nil
}

// overlayFiles maps each file of a standard library package to the file of the
// overlay. A package that the standard library does not hold takes a new
// directory inside GOROOT.
//
// A test build can hold a package of the surface only as a test variant,
// because the package under test imports it. Such a variant reads the files of
// the real package, thus the overlay replaces them. The variant of the package
// under test stays out, because its file list also holds each test file.
func overlayFiles(found, surface []*packages.Package, goroot string) (map[string][]byte, error) {
	existing := map[string]*packages.Package{}
	packages.Visit(append(found, surface...), nil, func(p *packages.Package) {
		if p.ID == p.PkgPath {
			existing[p.PkgPath] = p
			return
		}
		if existing[p.PkgPath] == nil && !holdsTestFile(p) {
			existing[p.PkgPath] = p
		}
	})
	out := map[string][]byte{}
	for _, o := range overlay.Packages() {
		p, ok := existing[o.Path]
		if !ok || len(p.CompiledGoFiles) == 0 {
			for _, f := range o.Files {
				name := filepath.Join(goroot, "src", filepath.FromSlash(o.Path), f.Name)
				out[name] = f.Content
			}
			continue
		}
		if o.Patch {
			patchFiles(out, o, p)
			continue
		}
		replaceFiles(out, o, p, goroot)
	}
	roots := append(found, surface...)
	for _, o := range overlay.Packages() {
		if err := patchDiffs(out, o, existing[o.Path], roots); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// fileStands reports whether a path names a file on disk.
func fileStands(name string) bool {
	info, err := os.Stat(name)
	return err == nil && !info.IsDir()
}

// patchDiffs applies each diff of an overlay package. A diff `<base>.patch`
// names the file `<base>` by its base name, and 2 kinds of file carry that
// name.
//
// A test file of the package under test is the first kind. 2 test variants of
// a directory hold such a file: the variant of the package itself, and the
// external test package whose path carries the suffix _test. A build with no
// test variant of the package holds no such file.
//
// A source file of the real package is the second kind. The compiler reads a
// native package from the real Go source, thus a change of 1 constant needs no
// copy of the whole file. A package that the overlay replaces also keeps such a
// file, because replaceFiles leaves the slot of a diff alone.
//
// A diff that names no file of either kind stops the build, unless the file
// stands in the directory of the package and a build tag of the architecture
// leaves it out, or unless it is a test file and the build holds no test
// variant. A name that matches nothing at all is a mistake of the overlay.
func patchDiffs(out map[string][]byte, o overlay.Package, real *packages.Package,
	roots []*packages.Package) error {
	if len(o.Patches) == 0 {
		return nil
	}
	sources := map[string]string{}
	dir := ""
	if real != nil {
		for _, name := range real.CompiledGoFiles {
			sources[filepath.Base(name)] = name
		}
		if len(real.CompiledGoFiles) > 0 {
			dir = filepath.Dir(real.CompiledGoFiles[0])
		}
	}
	tests := map[string]string{}
	packages.Visit(roots, nil, func(p *packages.Package) {
		if !holdsTestFile(p) || (p.PkgPath != o.Path && p.PkgPath != o.Path+"_test") {
			return
		}
		for _, name := range p.CompiledGoFiles {
			tests[filepath.Base(name)] = name
		}
	})
	for _, f := range o.Patches {
		base := strings.TrimSuffix(f.Name, ".patch")
		name, ok := sources[base]
		if !ok {
			name, ok = tests[base]
		}
		if !ok {
			// A build tag of the architecture selects the file set of a
			// package, thus a diff of a file that stands in the directory and
			// outside the build applies to nothing. The overlay of nistec
			// holds the generic form of P-256, and a machine with the assembly
			// reads another file.
			if dir != "" && fileStands(filepath.Join(dir, base)) {
				continue
			}
			// A test file needs a test variant of the build, and a build with
			// no test holds none.
			if strings.HasSuffix(base, "_test.go") && len(tests) == 0 {
				continue
			}
			return fmt.Errorf("the diff %s of %s names no file of the package", f.Name, o.Path)
		}
		src, ok := out[name]
		if !ok {
			read, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			src = read
		}
		patched, err := ApplyPatch(src, f.Content, f.Name)
		if err != nil {
			return err
		}
		out[name] = patched
	}
	return nil
}

// surfacePackages gives the real file list of each overlay package.
//
// The first load reads only the imports of the module. An overlay may import a
// package that the real source of the module does not reach, and the second
// load then holds the overlay file of that package next to its real files. Thus
// the load asks for each overlay package by its path.
//
// SDK sources, tags, and environment inputs identify a discovered file set.
func surfacePackages(dir string, env, tags []string) ([]*packages.Package, error) {
	identity, _, err := surfaceInputs(dir, env, tags...)
	if err != nil {
		return nil, err
	}
	if identity == "" {
		return discoverSurface(dir, env, tags...)
	}
	value, _ := surfaces.LoadOrStore(identity, sync.OnceValues(func() ([]*packages.Package, error) {
		return discoverSurface(dir, env, tags...)
	}))
	return value.(func() ([]*packages.Package, error))()
}

var surfaces sync.Map

func tagFlag(tags []string) string {
	return "-tags=" + strings.Join(append([]string{BuildTag}, tags...), ",")
}

// holdsTestFile reports whether the file list of a package holds a test file.
// The variant of a package under test holds one, and each other variant holds
// only the files of the real package.
func holdsTestFile(p *packages.Package) bool {
	for _, name := range p.CompiledGoFiles {
		if strings.HasSuffix(name, "_test.go") {
			return true
		}
	}
	return false
}

// replaceFiles replaces each file of a standard library package with the file
// of the overlay. A file that the overlay does not hold becomes empty.
//
// A file of the overlay takes the file of the same base name. The output path
// of a Dart part file comes from the name of the Go file, thus an equal name
// keeps the 2 paths equal.
//
// An overlay file with a new base name takes a new path inside the directory
// of the package. The go command reads such a file, exactly as it reads the
// files of a package that the standard library does not hold. A file that took
// the free slot of a real file instead would carry the path of that real file,
// thus each report of the compiler would name a source that holds no such
// line.
//
// A diff of the same directory names a real file, and that file keeps its own
// content. Read patchDiffs.
func replaceFiles(out map[string][]byte, o overlay.Package, p *packages.Package, goroot string) {
	byBase := map[string]string{}
	for _, name := range p.CompiledGoFiles {
		byBase[filepath.Base(name)] = name
	}
	dir := filepath.Join(goroot, "src", filepath.FromSlash(o.Path))
	if len(p.CompiledGoFiles) > 0 {
		dir = filepath.Dir(p.CompiledGoFiles[0])
	}
	taken := map[string]bool{}
	for _, f := range o.Files {
		name, ok := byBase[f.Name]
		if !ok {
			out[filepath.Join(dir, f.Name)] = f.Content
			continue
		}
		out[name] = f.Content
		taken[name] = true
	}
	// A diff of the directory changes a real file, thus that file keeps its
	// content and no empty file replaces it.
	for _, f := range o.Patches {
		taken[filepath.Join(dir, strings.TrimSuffix(f.Name, ".patch"))] = true
	}
	empty := []byte("package " + p.Name + "\n")
	for _, name := range p.CompiledGoFiles {
		// The selected SDK files define this platform's constants and data types.
		if o.Path == "syscall" && (strings.HasPrefix(filepath.Base(name), "zerrors_") || strings.HasPrefix(filepath.Base(name), "ztypes_")) {
			continue
		}
		if !taken[name] {
			out[name] = empty
		}
	}
}

// patchFiles replaces 1 file of a package that the compiler reads from the real
// Go source. Each other file of the package keeps its content.
//
// The overlay file matches a file of the package by its base name. A patch that
// matches nothing stays out, because a build tag of the architecture selects
// the file set of a package. The package math holds 1 file for each
// architecture with assembly.
func patchFiles(out map[string][]byte, o overlay.Package, p *packages.Package) {
	byBase := map[string]string{}
	for _, name := range p.CompiledGoFiles {
		byBase[filepath.Base(name)] = name
	}
	for _, f := range o.Files {
		if name, ok := byBase[f.Name]; ok {
			out[name] = f.Content
		}
	}
}

// collectErrors reports the errors of the root packages and of the
// dependencies that the build compiles.
//
// An error of a standard library package outside the surface stays out. The
// overlay of another package gives such an error, and the verifier reports the
// real cause with a source position: a compiled package imports a package
// outside the surface.
func collectErrors(roots []*packages.Package, sources map[string][]byte, rangeExtras *sync.Map) error {
	root := map[string]bool{}
	for _, p := range roots {
		root[p.ID] = true
	}
	var errs []error
	packages.Visit(roots, nil, func(p *packages.Package) {
		if !root[p.ID] && p.Module == nil && p.PkgPath != "command-line-arguments" && overlay.Outside(p.PkgPath) {
			return
		}
		parsed, err := compilerParseErrors(p, sources)
		if err != nil {
			errs = append(errs, err)
			return
		}
		if len(parsed) != 0 {
			// The compiler reports parse errors before type analysis.
			for _, err := range parsed {
				errs = append(errs, err)
			}
			return
		}
		for _, err := range rangeClauseErrors(p, rangeExtras) {
			errs = append(errs, err)
		}
		for _, file := range p.Syntax {
			if file.Name.Name == "_" {
				errs = append(errs, fmt.Errorf("%s: invalid package name _", p.Fset.Position(file.Name.Pos())))
			}
		}
		if !p.IllTyped {
			errs = append(errs, interfaceCycles(p)...)
			errs = append(errs, directiveContracts(p)...)
		}
		syntaxError := false
		for _, e := range p.Errors {
			if e.Kind == packages.ParseError && (strings.HasPrefix(e.Msg, "syntax error") ||
				e.Msg == "expression in defer must be function call" || e.Msg == "expression in go must be function call") {
				syntaxError = true
			}
		}
		start := len(errs)
		for _, e := range p.Errors {
			if syntaxError && e.Kind != packages.ParseError {
				continue
			}
			if redundantImportError(p, e) || redundantEncodingError(p, e) {
				continue
			}
			e = compilerVersionDiagnostic(p, compilerImportDiagnostic(p, e))
			if strings.HasPrefix(e.Msg, "\t") && len(errs) > start {
				errs[len(errs)-1] = fmt.Errorf("%w\n\t%s: %s", errs[len(errs)-1], e.Pos, strings.TrimPrefix(e.Msg, "\t"))
			} else {
				errs = append(errs, e)
			}
		}
	})
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errors.Join(errs...)
}
