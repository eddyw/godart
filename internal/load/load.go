// Package load runs stage 1 of the compiler. It reads a Go module and returns
// the packages with their syntax trees and their type information.
package load

import (
	"errors"
	"fmt"
	"go/build"
	"path/filepath"
	"sort"

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
func Load(dir string, patterns ...string) ([]*packages.Package, error) {
	out, err := load(dir, false, patterns)
	if err != nil {
		return nil, err
	}
	return out.Packages, nil
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
	files, err := overlayFiles(found)
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
	return &Result{Packages: pkgs, Sources: files, Binaries: binaries}, nil
}

// overlayFiles maps each file of a standard library package to the file of the
// overlay. A package that the standard library does not hold takes a new
// directory inside GOROOT.
func overlayFiles(found []*packages.Package) (map[string][]byte, error) {
	existing := map[string]*packages.Package{}
	packages.Visit(found, nil, func(p *packages.Package) {
		if p.ID == p.PkgPath && existing[p.PkgPath] == nil {
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
		if len(o.Files) > len(p.CompiledGoFiles) {
			return nil, fmt.Errorf("the overlay of %s holds %d files, and the package holds %d",
				o.Path, len(o.Files), len(p.CompiledGoFiles))
		}
		empty := []byte("package " + p.Name + "\n")
		for i, name := range p.CompiledGoFiles {
			if i < len(o.Files) {
				out[name] = o.Files[i].Content
				continue
			}
			out[name] = empty
		}
	}
	return out, nil
}

// collectErrors reports the errors of the root packages and of the dependencies.
func collectErrors(roots []*packages.Package) error {
	var errs []error
	packages.Visit(roots, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			errs = append(errs, e)
		}
	})
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errors.Join(errs...)
}
