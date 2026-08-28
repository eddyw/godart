// Package normalize runs stage 4 of the compiler. It converts the typed syntax
// tree into the intermediate tree of package ir.
//
// The stage makes each implicit conversion explicit, replaces a named result
// with a local variable, rewrites each range loop, and inserts a struct copy at
// each copy point.
package normalize

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"

	"github.com/eddyw/godart/internal/name"

	"github.com/eddyw/godart/internal/analyze"
	"github.com/eddyw/godart/internal/ir"
)

// Package builds the intermediate tree of 1 package.
func Package(fset *token.FileSet, files []*ast.File, info *types.Info, pkg *types.Package, blocks *analyze.Result) (*ir.Package, error) {
	b := &builder{fset: fset, info: info, pkg: pkg, blocks: blocks}
	out := &ir.Package{Path: pkg.Path(), Name: pkg.Name()}
	for _, f := range files {
		file := &ir.File{Name: filepath.Base(fset.Position(f.Pos()).Filename)}
		for _, d := range f.Decls {
			file.Decls = append(file.Decls, b.decl(d)...)
		}
		out.Files = append(out.Files, file)
	}
	for _, step := range info.InitOrder {
		out.InitOrder = append(out.InitOrder, step.Lhs...)
	}
	if len(b.errs) > 0 {
		return nil, errors.Join(b.errs...)
	}
	return out, nil
}

type builder struct {
	fset  *token.FileSet
	info  *types.Info
	pkg   *types.Package
	errs  []error
	fn    *ir.Func
	names map[types.Object]string
	used  map[string]bool

	// gotoLabels holds the label of each goto target of the current function.
	gotoLabels map[string]bool

	// blocks holds the result of stage 3.
	blocks *analyze.Result
}

// declareLocal gives a local variable a name that is unique in the function.
// Go allows an inner name to shadow an outer name, but Dart does not, thus the
// stage renames the inner variable.
func (b *builder) declareLocal(obj *types.Var, pref string) *ir.Var {
	base := name.Local(pref)
	n := base
	for i := 1; b.used[n]; i++ {
		n = fmt.Sprintf("%s$%d", base, i)
	}
	b.used[n] = true
	v := &ir.Var{Name: n, Obj: obj}
	if obj != nil {
		v.Type = obj.Type()
		b.names[obj] = n
	}
	return v
}

func (b *builder) fail(pos token.Pos, format string, args ...any) {
	b.errs = append(b.errs, fmt.Errorf("%s: %s", b.fset.Position(pos), fmt.Sprintf(format, args...)))
}

// temp builds a deterministic name for a generated variable. The name comes
// from the source position, thus it never depends on a global counter.
func (b *builder) temp(pos token.Pos, role string) string {
	p := b.fset.Position(pos)
	return fmt.Sprintf("$%s%d_%d", role, p.Line, p.Column)
}

func (b *builder) typeOf(e ast.Expr) types.Type {
	if tv, ok := b.info.Types[e]; ok {
		return tv.Type
	}
	if id, ok := e.(*ast.Ident); ok {
		if obj := b.info.ObjectOf(id); obj != nil {
			return obj.Type()
		}
	}
	return types.Typ[types.Invalid]
}

// --- declarations ---

func (b *builder) decl(d ast.Decl) []ir.Decl {
	switch d := d.(type) {
	case *ast.FuncDecl:
		if f := b.funcDecl(d); f != nil {
			return []ir.Decl{f}
		}
	case *ast.GenDecl:
		return b.genDecl(d)
	}
	return nil
}

func (b *builder) genDecl(d *ast.GenDecl) []ir.Decl {
	var out []ir.Decl
	for _, spec := range d.Specs {
		switch spec := spec.(type) {
		case *ast.ImportSpec:
		case *ast.TypeSpec:
			obj, _ := b.info.Defs[spec.Name].(*types.TypeName)
			if obj != nil {
				out = append(out, &ir.TypeDecl{Name: obj.Name(), Obj: obj})
			}
		case *ast.ValueSpec:
			out = append(out, b.valueSpec(spec, d.Tok)...)
		}
	}
	return out
}

func (b *builder) valueSpec(spec *ast.ValueSpec, tok token.Token) []ir.Decl {
	var out []ir.Decl
	if tok == token.CONST {
		for _, name := range spec.Names {
			obj, _ := b.info.Defs[name].(*types.Const)
			if obj == nil || name.Name == "_" {
				continue
			}
			out = append(out, &ir.ConstDecl{Name: obj.Name(), Type: obj.Type(), Value: obj.Val()})
		}
		return out
	}
	vars := make([]*ir.Var, 0, len(spec.Names))
	for _, name := range spec.Names {
		obj, _ := b.info.Defs[name].(*types.Var)
		if obj == nil {
			continue
		}
		vars = append(vars, &ir.Var{Name: obj.Name(), Obj: obj, Type: obj.Type()})
	}
	switch {
	case len(spec.Values) == 0:
		for _, v := range vars {
			out = append(out, &ir.Global{Var: v})
		}
	case len(spec.Values) == len(vars):
		for i, v := range vars {
			out = append(out, &ir.Global{Var: v, Init: b.copyPoint(b.exprAs(spec.Values[i], v.Type))})
		}
	default:
		b.fail(spec.Pos(), "a package variable with multiple results is not supported yet")
	}
	return out
}

func (b *builder) funcDecl(d *ast.FuncDecl) *ir.Func {
	obj, _ := b.info.Defs[d.Name].(*types.Func)
	if obj == nil {
		return nil
	}
	sig, _ := obj.Type().(*types.Signature)
	if sig == nil {
		return nil
	}
	f := &ir.Func{Name: obj.Name(), Obj: obj, Pos: d.Pos(), Async: b.blocks.Blocks(obj)}
	b.names = map[types.Object]string{}
	b.used = map[string]bool{}
	b.gotoLabels = gotoTargets(d.Body)
	b.fn = f
	if d.Recv != nil && len(d.Recv.List) > 0 {
		f.Recv = b.recvVar(d.Recv.List[0], sig)
	}
	f.TypeParams = typeParams(sig.TypeParams())
	f.Params = b.varsOf(sig.Params())
	f.Results = b.results(sig)
	if d.Body == nil {
		b.fail(d.Pos(), "a function with no body is not supported")
		return nil
	}
	f.Body = b.block(d.Body)
	if len(b.gotoLabels) > 0 {
		f.Body = b.stateMachine(f.Body, d.Pos())
	}
	b.fn = nil
	return f
}

// typeParams lists the type parameters of a signature.
func typeParams(list *types.TypeParamList) []*types.TypeParam {
	if list == nil {
		return nil
	}
	out := make([]*types.TypeParam, list.Len())
	for i := range out {
		out[i] = list.At(i)
	}
	return out
}

// gotoTargets lists the label of each goto statement of a function body.
func gotoTargets(body *ast.BlockStmt) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		if br, ok := n.(*ast.BranchStmt); ok && br.Tok == token.GOTO && br.Label != nil {
			out[br.Label.Name] = true
		}
		return true
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func (b *builder) recvVar(field *ast.Field, sig *types.Signature) *ir.Var {
	v := sig.Recv()
	n := "$recv"
	if len(field.Names) > 0 && field.Names[0].Name != "_" {
		n = field.Names[0].Name
	}
	return b.declareLocal(v, n)
}

func (b *builder) varsOf(t *types.Tuple) []*ir.Var {
	out := make([]*ir.Var, t.Len())
	for i := range out {
		v := t.At(i)
		n := v.Name()
		if n == "" || n == "_" {
			n = fmt.Sprintf("$p%d", i)
		}
		out[i] = b.declareLocal(v, n)
	}
	return out
}

// results replaces a named result with a local variable. The variable holds the
// zero value until a return statement writes it.
func (b *builder) results(sig *types.Signature) []*ir.Var {
	rs := sig.Results()
	out := make([]*ir.Var, rs.Len())
	for i := range out {
		v := rs.At(i)
		if n := v.Name(); n != "" && n != "_" {
			out[i] = b.declareLocal(v, n)
			out[i].Named = true
			continue
		}
		out[i] = &ir.Var{Name: fmt.Sprintf("$r%d", i), Type: v.Type()}
	}
	return out
}

// hasNamedResult reports whether the Go source gives a name to a result.
func hasNamedResult(f *ir.Func) bool {
	for _, r := range f.Results {
		if r.Named {
			return true
		}
	}
	return false
}
