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
	"go/version"
	"path/filepath"

	"github.com/eddyw/godart/internal/name"

	"github.com/eddyw/godart/internal/analyze"
	"github.com/eddyw/godart/internal/ir"
)

// Package builds the intermediate tree of 1 package.
func Package(fset *token.FileSet, files []*ast.File, info *types.Info, pkg *types.Package, blocks *analyze.Result) (*ir.Package, error) {
	b := &builder{fset: fset, info: info, pkg: pkg, blocks: blocks}
	b.globalBoxes = map[*types.Var]bool{}
	for _, f := range files {
		b.markAddressed(f, b.globalBoxes, true)
	}
	out := &ir.Package{Path: pkg.Path(), Name: pkg.Name()}
	for _, f := range files {
		v := info.FileVersions[f]
		if v == "" {
			v = pkg.GoVersion()
		}
		b.sharedLoopVars = v != "" && version.Compare(v, "go1.22") < 0
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

// receiverName gives the type name of a receiver field.
func receiverName(e ast.Expr) string {
	for {
		switch x := e.(type) {
		case *ast.StarExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.IndexListExpr:
			e = x.X
		case *ast.Ident:
			return x.Name
		default:
			return ""
		}
	}
}

type builder struct {
	sharedLoopVars bool
	fset           *token.FileSet
	info           *types.Info
	pkg            *types.Package
	errs           []error
	fn             *ir.Func
	names          map[types.Object]string
	used           map[string]bool

	// boxes holds each variable of the current function whose address the code
	// takes. Such a variable holds a box.
	boxes map[*types.Var]bool

	// globalBoxes holds each package variable whose address the code takes. The
	// set covers the whole package, because each function of the package reads
	// such a variable through its box.
	globalBoxes map[*types.Var]bool

	// retFlag names the variable that marks a return statement inside the body
	// of a range loop over a function. It is empty when the function holds no
	// such statement.
	retFlag string
	// retVars holds 1 variable for each result of the function. A return inside
	// such a body writes them.
	retVars []*ir.Var

	// branchFlags holds 1 variable for each construct that a branch leaves from
	// inside the body of a range loop over a function.
	branchFlags []*ir.Var

	// loops holds the construct that a break statement or a continue statement
	// leaves, from the outermost to the innermost.
	loops []loop

	// gotoLabels holds the label of each goto target of the current function.
	gotoLabels map[string]bool

	// blocks holds the result of stage 3.
	blocks *analyze.Result

	// pending holds the declaration of each function literal that can block.
	// Such a literal becomes a local function, because the frame of the literal
	// must name it.
	pending []ir.Stmt

	// noOrder stops the order pass of `order.go`. The condition and the post
	// statement of a for loop each run more than 1 time, thus no call of them
	// leaves the loop.
	noOrder int

	// orderN counts the temporary variables of the order pass. The name of
	// such a variable holds the count, thus 1 statement can hold more than 1.
	orderN int
}

// frameClass builds the name of the Dart frame class of 1 function. The name
// comes from the file and the position, thus 2 packages give no collision and
// the name never depends on a counter.
func (b *builder) frameClass(pos token.Pos) string {
	p := b.fset.Position(pos)
	var out []byte
	out = append(out, '_')
	for _, r := range filepath.Base(p.Filename) {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			out = append(out, byte(r))
			continue
		}
		out = append(out, '_')
	}
	return fmt.Sprintf("%s_%d_%d$F", out, p.Line, p.Column)
}

// litName builds the name of the local function of a literal that can block.
func (b *builder) litName(pos token.Pos) string {
	p := b.fset.Position(pos)
	return fmt.Sprintf("$lit%d_%d", p.Line, p.Column)
}

// addResults gives the frame each result of a function that names one. A
// deferred call can write such a result, thus it lives as long as the frame.
//
// A blank result of such a function also lives there, because the return path
// writes each result and reads them together.
func addResults(m *ir.Machine, results []*ir.Var) {
	if !namedResults(results) {
		return
	}
	m.Locals = append(m.Locals, results...)
}

// namedResults reports whether the Go source names 1 result or more.
func namedResults(results []*ir.Var) bool {
	for _, r := range results {
		if r.Named {
			return true
		}
	}
	return false
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
		v.Boxed = b.boxes[obj]
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
			// No code names a blank type, and 2 blank types give 1 Dart name.
			if obj != nil && obj.Name() != "_" {
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
		vars = append(vars, &ir.Var{Name: obj.Name(), Obj: obj, Type: obj.Type(),
			Boxed: b.globalBoxes[obj]})
	}
	switch {
	case len(spec.Values) == 0:
		for _, v := range vars {
			out = append(out, &ir.Global{Var: v})
		}
	case len(spec.Values) == len(vars):
		for i, v := range vars {
			b.startInit(spec.Values[i])
			init := b.copyPoint(b.exprAs(spec.Values[i], v.Type))
			pending := b.pending
			b.pending = nil
			pre := b.orderRun(spec.Pos(), func(o *orderer) { init = o.expr(init) })
			out = append(out, &ir.Global{Var: v, Init: init,
				Pending: append(pending, pre...)})
		}
	case len(spec.Values) == 1:
		// The call runs 1 time. A hidden variable holds the record of the
		// results, and each name reads 1 field of it. The hidden variable
		// takes the place of the first name in the order of the init.
		b.startInit(spec.Values[0])
		call := b.multiValue(spec.Values[0])
		pending := b.pending
		b.pending = nil
		if call == nil {
			break
		}
		pending = append(pending, b.orderRun(spec.Pos(), func(o *orderer) { call = o.expr(call) })...)
		tuple, ok := call.GoType().(*types.Tuple)
		if !ok || tuple.Len() != len(vars) {
			b.fail(spec.Pos(), "a package variable with multiple results needs a call")
			break
		}
		p := b.fset.Position(spec.Pos())
		hidden := &ir.Var{
			Name: fmt.Sprintf("InitVars$%d_%d", p.Line, p.Column),
			Type: tuple,
		}
		// The rank of the first named variable places the hidden variable,
		// and the stable sort of the init order keeps it in front.
		for _, v := range vars {
			if v.Name != "_" {
				hidden.Obj = v.Obj
				break
			}
		}
		if hidden.Obj == nil && len(vars) > 0 {
			// Each name is blank, and the rank of the first one still
			// places the call.
			hidden.Obj = vars[0].Obj
		}
		out = append(out, &ir.Global{Var: hidden, Init: call, Pending: pending})
		for i, v := range vars {
			field := &ir.RecordField{
				X:     &ir.Ident{Name: hidden.Name, Type: tuple},
				Index: i,
				Type:  tuple.At(i).Type(),
			}
			out = append(out, &ir.Global{Var: v, Init: b.copyPoint(b.convert(field, v.Type))})
		}
	default:
		b.fail(spec.Pos(), "a package variable with multiple results is not supported yet")
	}
	return out
}

// startInit gives the builder a fresh set of local names for the initializer of
// a package variable. A function literal inside such an initializer declares
// locals, and each declaration reads those maps.
func (b *builder) startInit(e ast.Expr) {
	b.names = map[types.Object]string{}
	b.used = map[string]bool{}
	b.gotoLabels = nil
	b.boxes = b.addressed(e)
	for v := range b.globalBoxes {
		b.boxes[v] = true
	}
	b.retFlag = ""
	b.retVars = nil
	b.branchFlags = nil
	b.fn = nil
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
	if d.Recv != nil && len(d.Recv.List) > 0 {
		if named := receiverName(d.Recv.List[0].Type); named == "_" {
			// The receiver type is a blank type, and no code calls such a
			// method.
			return nil
		}
	}
	f := &ir.Func{Name: obj.Name(), Obj: obj, Pos: d.Pos()}
	if d.Body != nil {
		f.End = d.Body.Rbrace
	}
	b.names = map[types.Object]string{}
	b.used = map[string]bool{}
	if d.Body == nil {
		// An assembly file holds the code of such a function, or a go:linkname
		// directive only exports the name. The Dart form panics, thus a call
		// of it gives a clear message and a declaration alone costs nothing.
		//
		// A //gart:dart directive names a Dart symbol, and the Dart form then
		// calls that symbol.
		f.NoBody = true
		if ref := ir.DartMarker(d.Doc); ref != nil {
			f.DartLib, f.DartName = ref.Lib, ref.Name
		}
		f.TypeParams = typeParams(sig.TypeParams())
		if d.Recv != nil && len(d.Recv.List) > 0 {
			f.Recv = b.recvVar(d.Recv.List[0], sig)
		}
		f.Params = b.varsOf(sig.Params())
		f.Results = b.results(sig)
		return f
	}
	f.Async = b.blocks.Blocks(obj)
	b.gotoLabels = gotoTargets(d.Body)
	b.boxes = b.addressed(d.Body)
	for v := range b.globalBoxes {
		b.boxes[v] = true
	}
	b.machineBoxes(d.Body, f.Async)
	b.retFlag = ""
	b.retVars = nil
	b.branchFlags = nil
	b.fn = f
	if d.Recv != nil && len(d.Recv.List) > 0 {
		f.Recv = b.recvVar(d.Recv.List[0], sig)
	}
	f.TypeParams = typeParams(sig.TypeParams())
	f.Params = b.varsOf(sig.Params())
	f.Results = b.results(sig)
	f.Body = b.block(d.Body)
	f.Body.List = append(b.returnState(), f.Body.List...)
	if f.Async || len(b.gotoLabels) > 0 {
		f.Machine = b.flatten(f.Body, f.Params, f.Recv, f.Async, f.Defers,
			f.DeferBlocks, f.Results, b.frameClass(d.Pos()), d.Pos())
		addResults(f.Machine, f.Results)
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
		// A blank result name still declares a result variable of Go, thus a
		// naked return of such a function gives the zero value of each result.
		out[i] = &ir.Var{Name: fmt.Sprintf("$r%d", i), Type: v.Type(), Named: v.Name() == "_"}
	}
	return out
}

// hasNamedResult reports whether the Go source gives a name to a result. A
// blank name counts, because Go accepts a naked return of such a function.
func hasNamedResult(f *ir.Func) bool {
	for _, r := range f.Results {
		if r.Named {
			return true
		}
	}
	return false
}
