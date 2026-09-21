package linker

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Variables struct {
	Targets map[types.Object]bool
	Imports map[*types.Package][]*packages.Package
	Digest  string
}

func Resolve(roots []*packages.Package) (*Variables, error) {
	var pkgs []*packages.Package
	packages.Visit(roots, nil, func(p *packages.Package) { pkgs = append(pkgs, p) })
	referenced := map[types.Object]bool{}
	for _, p := range pkgs {
		for _, obj := range p.TypesInfo.Uses {
			referenced[obj] = true
		}
	}
	redirected := map[types.Object]bool{}
	aliases := map[types.Object]types.Object{}
	for _, p := range pkgs {
		if p.Module == nil {
			continue
		}
		reachable := map[string]*packages.Package{}
		packages.Visit([]*packages.Package{p}, nil, func(d *packages.Package) { reachable[d.PkgPath] = d })
		for _, file := range p.Syntax {
			for _, group := range file.Comments {
				for _, comment := range group.List {
					fields := strings.Fields(comment.Text)
					if len(fields) != 3 || (fields[0] != "//go:linkname" && fields[0] != "//go:linknamestd") {
						continue
					}
					alias, ok := p.Types.Scope().Lookup(fields[1]).(*types.Var)
					if !ok {
						continue
					}
					if fields[2] != alias.Pkg().Path()+"."+alias.Name() {
						redirected[alias] = true
					}
					if !referenced[alias] {
						continue
					}
					at := strings.LastIndexByte(fields[2], '.')
					if at < 0 {
						return nil, fmt.Errorf("%s: linked variable %s has no package-qualified target", p.Fset.Position(comment.Pos()), fields[1])
					}
					targetPkg := reachable[fields[2][:at]]
					if targetPkg == nil {
						return nil, fmt.Errorf("%s: linked variable %s requires an imported target package %s", p.Fset.Position(comment.Pos()), fields[1], fields[2][:at])
					}
					target, ok := targetPkg.Types.Scope().Lookup(fields[2][at+1:]).(*types.Var)
					if !ok || !types.Identical(alias.Type(), target.Type()) {
						return nil, fmt.Errorf("%s: linked variable %s requires a target variable with the same type", p.Fset.Position(comment.Pos()), fields[1])
					}
					if alias != target {
						aliases[alias] = target
					}
				}
			}
		}
	}
	links := &Variables{Targets: map[types.Object]bool{}, Imports: map[*types.Package][]*packages.Package{}}
	owners := map[*types.Package]*packages.Package{}
	for _, pkg := range pkgs {
		owners[pkg.Types] = pkg
	}
	var digest []string
	for alias, target := range aliases {
		if redirected[target] {
			return nil, fmt.Errorf("linked variable %s.%s targets another linkname declaration", alias.Pkg().Path(), alias.Name())
		}
		links.Targets[target] = true
		digest = append(digest, alias.Pkg().Path()+"."+alias.Name()+"="+target.Pkg().Path()+"."+target.Name())
	}
	for _, p := range pkgs {
		for id, obj := range p.TypesInfo.Uses {
			if target := aliases[obj]; target != nil {
				p.TypesInfo.Uses[id] = target
			}
		}
		for _, file := range p.Syntax {
			var decls []ast.Decl
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.VAR {
					decls = append(decls, decl)
					continue
				}
				var specs []ast.Spec
				for _, spec := range gen.Specs {
					value := spec.(*ast.ValueSpec)
					var names []*ast.Ident
					for _, id := range value.Names {
						alias := p.TypesInfo.Defs[id]
						if target := aliases[alias]; target != nil {
							if len(value.Values) != 0 {
								return nil, fmt.Errorf("%s: linked variable initializers are not supported", p.Fset.Position(id.Pos()))
							}
							links.Imports[p.Types] = append(links.Imports[p.Types], owners[target.Pkg()])
						} else {
							names = append(names, id)
						}
					}
					if len(names) != 0 {
						value.Names = names
						specs = append(specs, value)
					}
				}
				if len(specs) != 0 {
					gen.Specs = specs
					decls = append(decls, gen)
				}
			}
			file.Decls = decls
		}
	}
	sort.Strings(digest)
	links.Digest = strings.Join(digest, "\x00")
	return links, nil
}
