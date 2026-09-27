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

type Symbols struct {
	Targets map[types.Object]bool
	Imports map[*types.Package][]*packages.Package
	Digest  string
}

// Resolve links symbols only within the packages that the driver emits.
func Resolve(pkgs []*packages.Package) (*Symbols, error) {
	emitted := map[*packages.Package]bool{}
	for _, p := range pkgs {
		emitted[p] = true
	}
	available := map[string]*packages.Package{}
	for _, p := range pkgs {
		available[p.PkgPath] = p
	}
	referenced := map[types.Object]bool{}
	for _, p := range pkgs {
		for _, obj := range p.TypesInfo.Uses {
			referenced[obj] = true
		}
	}
	redirected := map[types.Object]bool{}
	aliases := map[types.Object]types.Object{}
	for _, p := range pkgs {
		reachable := map[string]*packages.Package{}
		packages.Visit([]*packages.Package{p}, func(d *packages.Package) bool {
			if !emitted[d] {
				return false
			}
			reachable[d.PkgPath] = d
			return true
		}, nil)
		for _, file := range p.Syntax {
			for _, group := range file.Comments {
				for _, comment := range group.List {
					fields := strings.Fields(comment.Text)
					if len(fields) != 3 || (fields[0] != "//go:linkname" && fields[0] != "//go:linknamestd") {
						continue
					}
					alias := p.Types.Scope().Lookup(fields[1])
					kind := "variable"
					switch alias.(type) {
					case *types.Var:
					case *types.Func:
						kind = "function"
					default:
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
						return nil, fmt.Errorf("%s: linked %s %s has no package-qualified target", p.Fset.Position(comment.Pos()), kind, fields[1])
					}
					targetPkg := reachable[fields[2][:at]]
					if targetPkg == nil && kind == "function" {
						targetPkg = available[fields[2][:at]]
					}
					if targetPkg == nil {
						return nil, fmt.Errorf("%s: linked %s %s requires an imported target package %s", p.Fset.Position(comment.Pos()), kind, fields[1], fields[2][:at])
					}
					target := targetPkg.Types.Scope().Lookup(fields[2][at+1:])
					sameKind := false
					switch alias.(type) {
					case *types.Var:
						_, sameKind = target.(*types.Var)
					case *types.Func:
						_, sameKind = target.(*types.Func)
					}
					if !sameKind || !types.Identical(alias.Type(), target.Type()) {
						return nil, fmt.Errorf("%s: linked %s %s requires a target %s with the same type", p.Fset.Position(comment.Pos()), kind, fields[1], kind)
					}
					if alias != target {
						aliases[alias] = target
					}
				}
			}
		}
	}
	links := &Symbols{Targets: map[types.Object]bool{}, Imports: map[*types.Package][]*packages.Package{}}
	owners := map[*types.Package]*packages.Package{}
	for _, pkg := range pkgs {
		owners[pkg.Types] = pkg
	}
	var digest []string
	for alias, target := range aliases {
		if redirected[target] {
			return nil, fmt.Errorf("linked symbol %s.%s targets another linkname declaration", alias.Pkg().Path(), alias.Name())
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
				if fn, ok := decl.(*ast.FuncDecl); ok {
					if target := aliases[p.TypesInfo.Defs[fn.Name]]; target != nil {
						if fn.Body != nil {
							return nil, fmt.Errorf("%s: linked function aliases must have no body", p.Fset.Position(fn.Pos()))
						}
						if target.Pkg() != p.Types {
							links.Imports[p.Types] = append(links.Imports[p.Types], owners[target.Pkg()])
						}
						continue
					}
				}
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
