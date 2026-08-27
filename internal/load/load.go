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

// Load reads the packages that match the patterns in dir. The result holds the
// root packages in the order of their package path.
//
// The load runs 2 times. The first load finds the files of each standard
// library package, and the second load replaces those files with the overlay.
func Load(dir string, patterns ...string) ([]*packages.Package, error) {
	files, err := overlayFiles(dir, patterns)
	if err != nil {
		return nil, err
	}
	cfg := &packages.Config{
		Mode:       mode,
		Dir:        dir,
		BuildFlags: []string{"-tags=" + BuildTag},
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
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].PkgPath < pkgs[j].PkgPath })
	return pkgs, nil
}

// overlayFiles maps each file of a standard library package to the file of the
// overlay. A package that the standard library does not hold takes a new
// directory inside GOROOT.
func overlayFiles(dir string, patterns []string) (map[string][]byte, error) {
	found, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps,
		Dir:        dir,
		BuildFlags: []string{"-tags=" + BuildTag},
	}, patterns...)
	if err != nil {
		return nil, err
	}
	existing := map[string]*packages.Package{}
	packages.Visit(found, nil, func(p *packages.Package) { existing[p.PkgPath] = p })
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
