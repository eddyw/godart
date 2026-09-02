// Package load runs stage 1 of the compiler. It reads a Go module and returns
// the packages with their syntax trees and their type information.
package load

import (
	"errors"
	"fmt"
	"go/build"
	"path/filepath"
	"sort"
	"strings"
	"sync"

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
	Embeds map[string]bool
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

// Load reads the packages that match the patterns in dir.
//
// The load runs 2 times. The first load finds the files of each standard
// library package, and the second load replaces those files with the overlay.
func Load(dir string, patterns ...string) (*Result, error) {
	return load(dir, false, patterns)
}

// LoadTests reads the packages and also each test variant of them. It replaces
// the test entry point of the go command with the entry point of the compiler.
func LoadTests(dir string, patterns ...string) (*Result, error) {
	return load(dir, true, patterns)
}

func load(dir string, tests bool, patterns []string) (*Result, error) {
	found, err := packages.Load(&packages.Config{
		Mode:       findMode,
		Dir:        dir,
		BuildFlags: []string{"-tags=" + BuildTag},
		Tests:      tests,
	}, patterns...)
	if err != nil {
		return nil, err
	}
	surface, err := surfacePackages(dir)
	if err != nil {
		return nil, err
	}
	files, err := overlayFiles(found, surface)
	if err != nil {
		return nil, err
	}
	embeds, err := embedRewrites(found, files)
	if err != nil {
		return nil, err
	}
	var binaries []*Binary
	if tests {
		binaries, err = testBinaries(found)
		if err != nil {
			return nil, err
		}
		for _, b := range binaries {
			files[b.File] = b.Source
		}
	}
	cfg := &packages.Config{
		Mode:       mode,
		Dir:        dir,
		BuildFlags: []string{"-tags=" + BuildTag},
		Tests:      tests,
		Overlay:    files,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package matches %v in %s", patterns, dir)
	}
	if err := collectErrors(pkgs); err != nil {
		return nil, err
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].ID < pkgs[j].ID })
	return &Result{Packages: pkgs, Sources: files, Binaries: binaries, Embeds: embeds}, nil
}

// overlayFiles maps each file of a standard library package to the file of the
// overlay. A package that the standard library does not hold takes a new
// directory inside GOROOT.
//
// A test build can hold a package of the surface only as a test variant,
// because the package under test imports it. Such a variant reads the files of
// the real package, thus the overlay replaces them. The variant of the package
// under test stays out, because its file list also holds each test file.
func overlayFiles(found, surface []*packages.Package) (map[string][]byte, error) {
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
				name := filepath.Join(build.Default.GOROOT, "src", filepath.FromSlash(o.Path), f.Name)
				out[name] = f.Content
			}
			continue
		}
		if o.Patch {
			patchFiles(out, o, p)
			continue
		}
		if len(o.Files) > len(p.CompiledGoFiles) {
			return nil, fmt.Errorf("the overlay of %s holds %d files, and the package holds %d",
				o.Path, len(o.Files), len(p.CompiledGoFiles))
		}
		replaceFiles(out, o, p)
	}
	return out, nil
}

// surfacePackages gives the real file list of each overlay package.
//
// The first load reads only the imports of the module. An overlay may import a
// package that the real source of the module does not reach, and the second
// load then holds the overlay file of that package next to its real files. Thus
// the load asks for each overlay package by its path.
//
// The list depends only on GOROOT and on the build tag, thus 1 load for each
// process is enough. A watch build and the end to end test each run many
// builds in 1 process.
func surfacePackages(dir string) ([]*packages.Package, error) {
	surfaceOnce.Do(func() {
		var paths []string
		for _, o := range overlay.Packages() {
			if o.Path == overlay.IntrinsicPath {
				continue
			}
			paths = append(paths, o.Path)
		}
		surfaceFound, surfaceErr = packages.Load(&packages.Config{
			Mode:       findMode,
			Dir:        dir,
			BuildFlags: []string{"-tags=" + BuildTag},
		}, paths...)
	})
	return surfaceFound, surfaceErr
}

var (
	surfaceOnce  sync.Once
	surfaceFound []*packages.Package
	surfaceErr   error
)

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
// keeps the 2 paths equal. An overlay file with a new name takes the first free
// file of the package.
func replaceFiles(out map[string][]byte, o overlay.Package, p *packages.Package) {
	byBase := map[string]string{}
	for _, name := range p.CompiledGoFiles {
		byBase[filepath.Base(name)] = name
	}
	taken := map[string]bool{}
	rest := make([]overlay.File, 0, len(o.Files))
	for _, f := range o.Files {
		name, ok := byBase[f.Name]
		if !ok {
			rest = append(rest, f)
			continue
		}
		out[name] = f.Content
		taken[name] = true
	}
	free := make([]string, 0, len(p.CompiledGoFiles))
	for _, name := range p.CompiledGoFiles {
		if !taken[name] {
			free = append(free, name)
		}
	}
	empty := []byte("package " + p.Name + "\n")
	for i, name := range free {
		if i < len(rest) {
			out[name] = rest[i].Content
			continue
		}
		out[name] = empty
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
func collectErrors(roots []*packages.Package) error {
	root := map[string]bool{}
	for _, p := range roots {
		root[p.ID] = true
	}
	var errs []error
	packages.Visit(roots, nil, func(p *packages.Package) {
		if !root[p.ID] && overlay.Outside(p.PkgPath) {
			return
		}
		for _, e := range p.Errors {
			errs = append(errs, e)
		}
	})
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errors.Join(errs...)
}
