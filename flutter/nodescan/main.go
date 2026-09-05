// Command nodescan reports each named type of a node package that a Go
// application names.
//
//	nodescan [-pkg gart.dev/ui] <directory of the application module>
//
// It writes 1 name for each line, in the order of the alphabet. nodegen reads
// that list, and it writes 1 host case for each node type of the list. Read
// flutter/nodegen/README.md.
//
// The answer is exact. A regular expression over the source reads a name of a
// comment, a name of a string, and a name that no code names, and it misses
// each name that an import alias or a dot import carries. This program reads
// the type information of the Go tool chain, thus it names each type that the
// application uses, and no other 1.
package main

import (
	"flag"
	"fmt"
	"go/types"
	"os"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	pkg := flag.String("pkg", "gart.dev/ui",
		"the import path of the node package. Each pack under it counts too.")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: nodescan [-pkg path] <module directory>")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	names, err := scan(flag.Arg(0), *pkg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nodescan:", err)
		os.Exit(1)
	}
	for _, n := range names {
		fmt.Println(n)
	}
}

// inSet reports whether path names the node package or 1 pack under it. Each
// pack holds the node types of 1 Dart package, and gart.dev/ui/material is
// such a pack.
func inSet(path, pkg string) bool {
	return path == pkg || strings.HasPrefix(path, pkg+"/")
}

// scan loads the module of dir and gives each named type of pkg and of each
// pack under it that the code of the module names.
//
// The load reads each dependency from source, thus a library of another module
// that builds nodes also reports its types. The node package itself names each
// 1 of its own types, and the walk skips it for that reason.
func scan(dir, pkg string) ([]string, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir: dir,
	}
	roots, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("the directory %s holds no package", dir)
	}
	var bad error
	packages.Visit(roots, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			if bad == nil {
				bad = fmt.Errorf("%s: %v", p.PkgPath, e)
			}
		}
	})
	if bad != nil {
		return nil, bad
	}

	found := map[string]bool{}
	packages.Visit(roots, nil, func(p *packages.Package) {
		if inSet(p.PkgPath, pkg) || p.TypesInfo == nil {
			return
		}
		for _, obj := range p.TypesInfo.Uses {
			name, ok := obj.(*types.TypeName)
			if !ok || name.Pkg() == nil || !inSet(name.Pkg().Path(), pkg) {
				continue
			}
			found[name.Name()] = true
		}
	})
	out := make([]string, 0, len(found))
	for n := range found {
		out = append(out, n)
	}
	sort.Strings(out)
	return out, nil
}
