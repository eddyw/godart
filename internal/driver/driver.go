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
	"sort"
	"strings"

	"golang.org/x/tools/go/gcexportdata"
	"golang.org/x/tools/go/packages"

	"github.com/eddyw/godart/internal/analyze"
	"github.com/eddyw/godart/internal/emit"
	"github.com/eddyw/godart/internal/load"
	"github.com/eddyw/godart/internal/meta"
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
	// NoCache compiles each package again, also when the cache holds it.
	NoCache bool
}

// Stats reports what the build did. A cached package needs no second compile.
type Stats struct {
	Compiled []string
	Cached   []string
}

// Build compiles the packages and writes the Dart package into Out.
func Build(opt Options) error {
	_, err := BuildWithStats(opt)
	return err
}

// BuildWithStats compiles the packages and reports the state of each one.
func BuildWithStats(opt Options) (*Stats, error) {
	roots, err := load.Load(opt.Dir, opt.Patterns...)
	if err != nil {
		return nil, err
	}
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
		sources, err := readSources(p)
		if err != nil {
			return nil, err
		}
		sourceHash := meta.SourceHash(sources, dependencyHashes(p, module, exports), dartPkg)
		if !opt.NoCache {
			if old, ok := store.Read(p.PkgPath); ok && old.SourceHash == sourceHash &&
				outputExists(opt.Out, old.Files) {
				exports[p.PkgPath] = old.ExportHash
				stats.Cached = append(stats.Cached, p.PkgPath)
				continue
			}
		}
		out, err := compile(p, module, dartPkg, blocks)
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
			Path:       p.PkgPath,
			Name:       p.Name,
			SourceHash: sourceHash,
			ExportHash: meta.ExportHash(export, blocking),
			Export:     base64.StdEncoding.EncodeToString(export),
			Blocking:   blocking,
			Files:      written,
		}
		exports[p.PkgPath] = archive.ExportHash
		if err := store.Write(archive); err != nil {
			return nil, err
		}
		stats.Compiled = append(stats.Compiled, p.PkgPath)
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
	return stats, nil
}

func readSources(p *packages.Package) (map[string][]byte, error) {
	out := make(map[string][]byte, len(p.CompiledGoFiles))
	for _, name := range p.CompiledGoFiles {
		text, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		out[name] = text
	}
	return out, nil
}

// dependencyHashes lists the export hash of each dependency inside the module.
// A change inside a function body of a dependency keeps its export hash equal,
// thus this package keeps its cache.
func dependencyHashes(p *packages.Package, module string, exports map[string]string) []string {
	var out []string
	for path := range p.Imports {
		if !strings.HasPrefix(path, module) {
			continue
		}
		out = append(out, path+"="+exports[path])
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
