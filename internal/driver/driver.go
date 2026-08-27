// Package driver runs the compiler over a Go module and writes a Dart package.
package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/eddyw/godart/internal/analyze"
	"github.com/eddyw/godart/internal/emit"
	"github.com/eddyw/godart/internal/load"
	"github.com/eddyw/godart/internal/normalize"
	"github.com/eddyw/godart/internal/verify"
)

// Options holds the input of a build.
type Options struct {
	Dir      string
	Patterns []string
	Out      string
	Runtime  string
	Format   bool
}

// Build compiles the packages and writes the Dart package into Out.
func Build(opt Options) error {
	roots, err := load.Load(opt.Dir, opt.Patterns...)
	if err != nil {
		return err
	}
	module := moduleOf(roots)
	if module == "" {
		return fmt.Errorf("the packages belong to no module")
	}
	pkgs := order(roots, module)
	for _, p := range pkgs {
		if err := verify.Package(p.Fset, p.Syntax, p.TypesInfo); err != nil {
			return err
		}
	}
	blocks := analyze.Module(pkgs)
	dartPkg := dartPackageName(module)
	files := map[string]string{}
	for _, p := range pkgs {
		out, err := compile(p, module, dartPkg, blocks)
		if err != nil {
			return err
		}
		dir := outputDir(p, module)
		for path, text := range out.Files {
			files[filepath.Join(dir, path)] = text
		}
	}
	runtime, err := runtimePath(opt.Runtime, opt.Out)
	if err != nil {
		return err
	}
	files["pubspec.yaml"] = pubspec(dartPkg, module, runtime)
	if err := writeFiles(opt.Out, files); err != nil {
		return err
	}
	if opt.Format {
		return format(opt.Out)
	}
	return nil
}

func compile(p *packages.Package, module, dartPkg string, blocks *analyze.Result) (*emit.Result, error) {
	tree, err := normalize.Package(p.Fset, p.Syntax, p.TypesInfo, p.Types, blocks)
	if err != nil {
		return nil, err
	}
	var imports []emit.Import
	for path, dep := range p.Imports {
		if !strings.HasPrefix(path, module) {
			continue
		}
		imports = append(imports, emit.Import{
			Path: path,
			URI:  "package:" + dartPkg + "/" + libPath(dep, module) + ".dart",
		})
	}
	sort.Slice(imports, func(i, j int) bool { return imports[i].Path < imports[j].Path })
	return emit.Package(tree, p.Types, baseName(p, module), imports, blocks)
}

// order sorts the packages of the module in dependency order. A package
// compiles after each package that it imports.
func order(roots []*packages.Package, module string) []*packages.Package {
	seen := map[string]bool{}
	var out []*packages.Package
	var visit func(p *packages.Package)
	visit = func(p *packages.Package) {
		if seen[p.PkgPath] || !strings.HasPrefix(p.PkgPath, module) {
			return
		}
		seen[p.PkgPath] = true
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

// libPath mirrors the Go import path in the Dart directory structure.
func libPath(p *packages.Package, module string) string {
	rel := strings.TrimPrefix(p.PkgPath, module)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return p.Name
	}
	return rel
}

func baseName(p *packages.Package, module string) string {
	if p.Name == "main" {
		return "main"
	}
	return filepath.Base(libPath(p, module))
}

func outputDir(p *packages.Package, module string) string {
	if p.Name == "main" {
		return "bin"
	}
	rel := libPath(p, module)
	return filepath.Join("lib", filepath.Dir(rel))
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
