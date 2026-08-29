// Package emit runs stage 5 of the compiler. It walks the intermediate tree and
// writes Dart source code.
package emit

import (
	"errors"
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
)

// Import gives the Dart URI of a Go package that the code imports.
type Import struct {
	Path string
	URI  string
}

// Result holds the Dart files of 1 package. Each key of Files is a path that is
// relative to the output directory of the package.
type Result struct {
	Library string
	Files   map[string]string

	// Shared holds the file of each unnamed struct type that the package uses.
	// Each key is a path that is relative to the output root, and each content
	// is a pure function of the shape. Thus 2 packages that use 1 shape write
	// 1 file with equal bytes, and Dart gives them 1 class.
	Shared map[string]string
}

// Config holds what the emitter needs beyond the tree of the package.
type Config struct {
	// Base names the library file of the package.
	Base string
	// Imports lists the Dart library of each Go dependency.
	Imports []Import
	// DartPackage names the pub package of the output.
	DartPackage string
	// URIs gives the Dart URI of each Go package of the build. A shared file
	// belongs to no package, thus it reads the URI of a field type here.
	URIs map[string]string
}

// Package writes the Dart source code of 1 package. Base names the library
// file, and imports lists the Dart libraries of the Go dependencies.
// partSuffix ends the name of each part file.
//
// A library file ends with `.dart` only. The 2 forms must differ, because a
// package may hold a file whose name is the name of a package inside it. The
// package math holds the file bits.go, and the package math/bits also exists.
const partSuffix = ".part.dart"

func Package(p *ir.Package, tpkg *types.Package, cfg Config) (*Result, error) {
	e := &emitter{pkg: tpkg, prefixes: map[string]string{}, base: cfg.Base,
		initNames: map[*ir.Func]string{},
		anon:      map[string]*types.Struct{}, localAnon: map[string]*types.Struct{},
		shadowed: map[string]map[string]bool{},
		nilChans: map[string]types.Type{}, cfg: cfg}
	e.setPrefixes(cfg.Imports)
	e.groupMethods(p)
	out := &Result{Library: cfg.Base + ".dart", Files: map[string]string{}}
	var parts []string
	for _, f := range p.Files {
		part := cfg.Base + "/" + strings.TrimSuffix(f.Name, ".go") + partSuffix
		out.Files[part] = e.partFile(cfg.Base, f)
		parts = append(parts, part)
	}
	sort.Strings(parts)
	out.Files[out.Library] = e.libraryFile(p, parts, cfg.Imports)
	out.Shared = e.sharedFiles()
	if len(e.errs) > 0 {
		return nil, errors.Join(e.errs...)
	}
	return out, nil
}

type emitter struct {
	pkg      *types.Package
	prefixes map[string]string
	base     string
	buf      strings.Builder
	errs     []error

	// methods groups each method by the name of its receiver type.
	methods map[*types.TypeName][]*ir.Func
	// inClass marks the methods that a class holds, thus a part file skips
	// them at top level.
	inClass map[*ir.Func]bool
	// globals lists each package variable in declaration order.
	globals []*ir.Global
	// inits lists each init function of the package.
	inits []*ir.Func
	// mainFunc holds the entry function of package main.
	mainFunc *ir.Func
	// initNames gives each init function a name that is unique in the package.
	initNames map[*ir.Func]string
	// breakables holds the label of each construct that a break statement can
	// leave. An empty label marks a loop, and a loop needs no label.
	breakables []string
	// labelSeq numbers the labels that a switch needs. The walk order is
	// stable, thus the numbers stay equal for equal input.
	labelSeq int
	// returnLabel holds the block that a return statement leaves. It is not
	// empty only in a function that holds a defer statement and a named result.
	returnLabel string
	returnVars  []*ir.Var
	// localAnon holds each unnamed struct type whose class stays inside the
	// library of the package. Read anonShared.
	localAnon map[string]*types.Struct
	// anon holds each unnamed struct type that the package uses, by the name of
	// its Dart class. The library file writes 1 class for each entry.
	anon map[string]*types.Struct
	// nilChans holds the element type of each nil channel that the package
	// needs, by the name of its variable. The library file writes 1 variable
	// for each entry.
	nilChans map[string]types.Type
	// shadowed holds the method names of each package that the build reads, by
	// the import path.
	shadowed map[string]map[string]bool
	// cfg holds the URI of each package and the name of the pub package.
	cfg Config
	// nilPrefix stands in front of the name of each nil channel. A shared file
	// holds its own channels, thus 2 files of 1 library give no collision.
	nilPrefix string
	// machine holds the state machine of the function that the emitter writes.
	machine *machineCtx
	// renames maps the local of a machine to the copy that a literal reads.
	// Read the field Captured of ir.FuncLit.
	renames map[string]string
	// captures holds the copy that the current state declares.
	captures []capture
	// frames holds the class of each frame that the file still needs. Dart
	// holds no class inside a function, thus each one waits for package level.
	frames []string
	// typeParams holds the type parameters of the function that the emitter
	// writes. The frame class of a literal of that function names them, because
	// the class stands at package level.
	typeParams []*types.TypeParam
}

// flushFrames writes the class of each frame that a declaration made.
func (e *emitter) flushFrames() {
	for _, text := range e.frames {
		e.line("%s", text)
	}
	e.frames = nil
}

// topName maps the name of a function, a variable, or a constant at package
// level.
//
// A class of the package may hold a method of the same name. Such a member hides
// the top level name inside the class body, thus the top level name takes a
// suffix. A type name always takes a suffix, thus a type never collides.
func (e *emitter) topName(pkg *types.Package, goName string) string {
	out := name.Package(goName)
	if e.shadowSet(pkg)[out] {
		return out + "$f"
	}
	return out
}

// shadowSet gives each Dart method name that a class of the package holds.
//
// The set comes from the type information, thus the emitter computes the same
// set for the package that it compiles and for a package that it imports. Only
// a method of a struct type reaches a class. A method of another named type
// becomes a top level function, and such a function shadows nothing.
func (e *emitter) shadowSet(pkg *types.Package) map[string]bool {
	if pkg == nil {
		return nil
	}
	if out, ok := e.shadowed[pkg.Path()]; ok {
		return out
	}
	out := map[string]bool{}
	scope := pkg.Scope()
	for _, n := range scope.Names() {
		obj, ok := scope.Lookup(n).(*types.TypeName)
		if !ok || obj.IsAlias() {
			continue
		}
		named, ok := types.Unalias(obj.Type()).(*types.Named)
		if !ok || !isStructType(named) {
			continue
		}
		for i := 0; i < named.NumMethods(); i++ {
			out[name.Package(named.Method(i).Name())] = true
		}
	}
	e.shadowed[pkg.Path()] = out
	return out
}

func (e *emitter) fail(format string, args ...any) {
	e.errs = append(e.errs, fmt.Errorf(format, args...))
}

func (e *emitter) setPrefixes(imports []Import) {
	sorted := make([]Import, len(imports))
	copy(sorted, imports)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	used := map[string]bool{}
	for i, im := range sorted {
		base := im.Path
		if k := strings.LastIndex(base, "/"); k >= 0 {
			base = base[k+1:]
		}
		prefix := "$" + sanitize(base)
		if used[prefix] {
			prefix = fmt.Sprintf("%s$%d", prefix, i)
		}
		used[prefix] = true
		e.prefixes[im.Path] = prefix
	}
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return b.String()
}

// groupMethods puts each method with its receiver type. A Go package declares
// a method in any of its files, but a Dart class holds its methods in 1 place.
func (e *emitter) groupMethods(p *ir.Package) {
	e.methods = map[*types.TypeName][]*ir.Func{}
	e.inClass = map[*ir.Func]bool{}
	for _, f := range p.Files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ir.Global:
				e.globals = append(e.globals, d)
			case *ir.Func:
				switch {
				case d.Recv != nil:
					obj := receiverTypeName(d.Recv.Type)
					if obj != nil && isStructType(obj.Type()) {
						e.methods[obj] = append(e.methods[obj], d)
						// A method with a pointer receiver becomes a top level
						// function. Go runs such a method with a nil receiver,
						// and Dart calls no member of null. A method of a
						// generic type stays a member of its class.
						e.inClass[d] = !isPointerType(d.Recv.Type) || genericType(obj)
					}
				case d.Name == "init":
					e.initNames[d] = fmt.Sprintf("_init$%d", len(e.inits))
					e.inits = append(e.inits, d)
				case d.Name == "main" && p.Name == "main":
					e.mainFunc = d
				}
			}
		}
	}
	e.globals = initOrder(e.globals, p.InitOrder)
}

// initOrder sorts the package variables into the order that Go gives them a
// value.
//
// A variable with no initializer takes its zero value first. Such a value reads
// nothing. Each other variable follows the order of the type checker, thus a
// variable comes after each variable that its initializer reads.
func initOrder(globals []*ir.Global, order []*types.Var) []*ir.Global {
	rank := make(map[*types.Var]int, len(order))
	for i, v := range order {
		rank[v] = i
	}
	out := make([]*ir.Global, 0, len(globals))
	rest := make([]*ir.Global, 0, len(globals))
	for _, g := range globals {
		if g.Init == nil {
			out = append(out, g)
			continue
		}
		rest = append(rest, g)
	}
	sort.SliceStable(rest, func(i, j int) bool {
		a, aok := rank[rest[i].Var.Obj]
		b, bok := rank[rest[j].Var.Obj]
		if aok != bok {
			return aok
		}
		return a < b
	})
	return append(out, rest...)
}

func receiverTypeName(t types.Type) *types.TypeName {
	u := types.Unalias(t)
	if p, ok := u.(*types.Pointer); ok {
		u = types.Unalias(p.Elem())
	}
	named, ok := u.(*types.Named)
	if !ok {
		return nil
	}
	return named.Obj()
}

const header = `// Generated by gart. Do not edit.
// ignore_for_file: unused_local_variable, unused_element, unused_field
// ignore_for_file: non_constant_identifier_names, camel_case_types
// ignore_for_file: constant_identifier_names, unnecessary_non_null_assertion
// ignore_for_file: unused_import, prefer_final_locals, curly_braces_in_flow_control_structures
`

// libraryFile writes the library of the package.
//
// The body comes first, because it can name an unnamed struct type and the
// import of such a type stands at the top of the file.
func (e *emitter) libraryFile(p *ir.Package, parts []string, imports []Import) string {
	sorted := make([]Import, len(imports))
	copy(sorted, imports)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	e.buf.Reset()
	e.writeAnonClasses()
	e.writeNilChans()
	e.writeDescriptors()
	e.line("")
	e.writeInit(sorted)
	e.flushFrames()
	if p.Name == "main" {
		e.writeMain()
	}
	body := e.buf.String()

	e.buf.Reset()
	e.buf.WriteString(header)
	e.line("")
	e.line("import 'package:goruntime/goruntime.dart';")
	for _, im := range sorted {
		e.line("import '%s' as %s;", im.URI, e.prefixes[im.Path])
	}
	for _, uri := range e.anonURIs() {
		e.line("import '%s';", uri)
	}
	e.line("")
	for _, part := range parts {
		e.line("part '%s';", part)
	}
	e.line("")
	e.buf.WriteString(body)
	return e.buf.String()
}

// writeInit writes the function that gives each package variable its value.
// Dart holds a top level variable as late, thus the order stays explicit.
//
// Go gives a package its values after each package that it imports, thus the
// function starts with the imports. The flag stops a second run.
func (e *emitter) writeInit(imports []Import) {
	e.line("bool $initDone = false;")
	e.line("")
	e.line("void $init() {")
	e.line("if ($initDone) return;")
	e.line("$initDone = true;")
	for _, im := range imports {
		e.line("%s.$init();", e.prefixes[im.Path])
	}
	for _, g := range e.globals {
		if g.Var.Name == "_" {
			continue
		}
		// A function literal of the initializer that can block becomes a local
		// function of the init routine.
		e.list(g.Pending)
		value := e.zero(g.Var.Type)
		if g.Init != nil {
			value = e.expr(g.Init)
		}
		if g.Var.Boxed {
			e.line("%s = Box(%s);", e.topName(e.pkg, g.Var.Name), value)
			continue
		}
		e.line("%s = %s;", e.topName(e.pkg, g.Var.Name), value)
	}
	for _, f := range e.inits {
		e.line("%s();", e.initNames[f])
	}
	e.line("}")
	e.line("")
}

// writeMain writes the entry point. Package main runs as the first goroutine,
// and the scheduler ends when that goroutine returns.
func (e *emitter) writeMain() {
	e.line("void main(List<String> $arguments) {")
	e.line("$setArgs($arguments);")
	e.line("$goPanic = $fatal;")
	e.line("$init();")
	if e.mainFunc != nil {
		e.line("try {")
		e.line("$boot(() { _main(); });")
		e.line("} on GoPanic catch ($p) {")
		e.line("$fatal($p);")
		e.line("}")
	}
	e.line("}")
}

func (e *emitter) partFile(base string, f *ir.File) string {
	e.buf.Reset()
	e.buf.WriteString(header)
	e.line("")
	e.line("part of '../%s.dart';", base)
	e.line("")
	for _, d := range f.Decls {
		e.decl(d)
		e.flushFrames()
	}
	return e.buf.String()
}

func (e *emitter) decl(d ir.Decl) {
	switch d := d.(type) {
	case *ir.TypeDecl:
		e.typeDecl(d)
	case *ir.ConstDecl:
		e.constDecl(d)
	case *ir.Global:
		// A blank package variable holds a static check, for example
		// `var _ Reader = (*T)(nil)`. Go allows many of them in 1 package, thus
		// the output writes none.
		if d.Var.Name == "_" {
			return
		}
		if d.Var.Boxed {
			e.line("late Box<%s> %s;", e.dartType(d.Var.Type), e.topName(e.pkg, d.Var.Name))
			return
		}
		e.line("late %s %s;", e.dartType(d.Var.Type), e.topName(e.pkg, d.Var.Name))
	case *ir.Func:
		if e.inClass[d] {
			return
		}
		e.funcDecl(d)
	}
}

func (e *emitter) constDecl(d *ir.ConstDecl) {
	value := e.constant(d.Value, d.Type)
	keyword := "const"
	if isString(d.Type) {
		keyword = "final"
	}
	e.line("%s %s %s = %s;", keyword, e.dartType(d.Type), e.topName(e.pkg, d.Name), value)
}

func (e *emitter) typeDecl(d *ir.TypeDecl) {
	named, ok := types.Unalias(d.Obj.Type()).(*types.Named)
	if !ok {
		return
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		// A named type over a basic type keeps the Dart type of the basic
		// type. Its methods become top level functions.
		return
	}
	e.class(d.Obj, named, st)
}

func (e *emitter) line(format string, args ...any) {
	if len(args) == 0 {
		e.buf.WriteString(format)
	} else {
		fmt.Fprintf(&e.buf, format, args...)
	}
	e.buf.WriteByte('\n')
}
