// Command nodescan reports each named type of a node package that a Go
// application names.
//
//	nodescan [-pkg gart.dev/ui] <directory of the application module>
//
// It writes 1 name for each line, in the order of the alphabet. nodegen reads
// that list, and it writes 1 host case for each node type of the list. Read
// flutter/nodegen/README.md.
//
// After the names it writes 1 line for each string constant of the module,
// with a quotation mark in front. nodegen keeps each 1 that names an icon of
// Material, and it writes that icon into the host. A Go program thus names an
// icon with no change of a Dart file.
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
	"go/constant"
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
	names, texts, err := scan(flag.Arg(0), *pkg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nodescan:", err)
		os.Exit(1)
	}
	for _, n := range names {
		fmt.Println(n)
	}
	for _, t := range texts {
		fmt.Println(`"` + t)
	}
}

// iconName reports whether a string constant can name an icon of Material.
// Such a name holds a lower case letter, a digit, or an underscore, and it
// starts with a letter.
func iconName(s string) bool {
	if len(s) == 0 || len(s) > 64 || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return false
	}
	return true
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
func scan(dir, pkg string) ([]string, []string, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir: dir,
	}
	roots, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, nil, err
	}
	if len(roots) == 0 {
		return nil, nil, fmt.Errorf("the directory %s holds no package", dir)
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
		return nil, nil, bad
	}

	found := map[string]bool{}
	texts := map[string]bool{}
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
		for _, tv := range p.TypesInfo.Types {
			if tv.Value == nil || tv.Value.Kind() != constant.String {
				continue
			}
			v := constant.StringVal(tv.Value)
			if iconName(v) {
				texts[v] = true
			}
		}
	})
	out := make([]string, 0, len(found))
	for n := range found {
		out = append(out, n)
	}
	sort.Strings(out)
	list := make([]string, 0, len(texts))
	for n := range texts {
		list = append(list, n)
	}
	sort.Strings(list)
	return out, list, nil
}
