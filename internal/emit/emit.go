// Package emit runs stage 5 of the compiler. It walks the intermediate tree and
// writes Dart source code.
package emit

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"path/filepath"
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

	// MapFile names the part file that holds the source map of the package.
	// Stage 6 writes it, because the line numbers of the map come from the
	// output of dart format.
	MapFile string

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
	// Fset resolves the position of a statement. The emitter writes a marker
	// comment for each new source line, and stage 6 turns the markers into the
	// source map of the output.
	Fset *token.FileSet
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
	e := newEmitter(p, tpkg, cfg)
	// A shape can name its twin in a function that the emitter writes after
	// the twin. The first pass discovers every shape, channel, and
	// description in any order, and the second pass writes the output with
	// the full maps.
	e.generate(p, cfg)
	e.resetDiscovery()
	out := e.generate(p, cfg)
	if len(e.errs) > 0 {
		return nil, errors.Join(e.errs...)
	}
	return out, nil
}

func newEmitter(p *ir.Package, tpkg *types.Package, cfg Config) *emitter {
	e := &emitter{pkg: tpkg, prefixes: map[string]string{}, base: cfg.Base,
		initNames: map[*ir.Func]string{},
		anon:      map[string]*types.Struct{}, localAnon: map[string]*types.Struct{},
		shadowed: map[string]map[string]bool{},
		nilChans: map[string]types.Type{}, instDescs: map[string]types.Type{}, cfg: cfg,
		anonParams:  map[string][]*types.TypeParam{},
		groundTwins: map[string]*anonTwin{},
		funcIds:     map[any]int{},
		funcByObj:   map[types.Object]*ir.Func{},
		varLitId:    map[string]int{}}
	e.setPrefixes(cfg.Imports)
	e.groupMethods(p)
	return e
}

// funcId gives the token of a function, and it mints one on the first use.
func (e *emitter) funcId(key any) int {
	if id, ok := e.funcIds[key]; ok {
		return id
	}
	e.funcSeq++
	e.funcIds[key] = e.funcSeq
	return e.funcSeq
}

func (e *emitter) resetDiscovery() {
	e.prefixes = map[string]string{}
	e.dartLibs = map[string]string{}
	e.setPrefixes(e.cfg.Imports)
	e.anon = map[string]*types.Struct{}
	e.localAnon = map[string]*types.Struct{}
	e.anonParams = map[string][]*types.TypeParam{}
	e.groundTwins = map[string]*anonTwin{}
	e.nilChans = map[string]types.Type{}
	e.nilChansDone = nil
	e.instDescs = map[string]types.Type{}
	e.instDescsDone = nil
	e.descBusy = nil
	// The emitter runs 2 passes, and the second one writes the output. A frame
	// class that the first pass counted must reach the file of the second pass.
	e.frameSeen = nil
	e.anonDone = nil
	e.extra = nil
	e.errs = nil
	e.labelSeq = 0
	e.goLine = 0
}

func (e *emitter) generate(p *ir.Package, cfg Config) *Result {
	out := &Result{Library: cfg.Base + ".dart", Files: map[string]string{}}
	var parts []string
	for _, f := range p.Files {
		part := cfg.Base + "/" + strings.TrimSuffix(f.Name, ".go") + partSuffix
		out.Files[part] = e.partFile(cfg.Base, f)
		parts = append(parts, part)
	}
	sort.Strings(parts)
	out.MapFile = cfg.Base + ".map.dart"
	out.Files[out.Library] = e.libraryFile(p, parts, cfg.Imports)
	out.Shared = e.sharedFiles()
	return out
}

type emitter struct {
	pkg      *types.Package
	prefixes map[string]string

	// dartLibs gives the import prefix of each Dart library that an external
	// declaration names, and dartOrder holds the order of the first read. The
	// order of the read is stable, thus the header stays stable.
	dartLibs  map[string]string
	dartOrder []string
	base      string
	buf       strings.Builder
	errs      []error

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
	// initNames gives a name to each function that holds no usable name of its
	// own: an init function, and a function that carries the blank name.
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
	// deferPlain marks the body of a function that holds a defer statement
	// and no state machine. A return statement then runs the deferred calls
	// itself, thus the frame of a deferred call names the line of the return.
	deferPlain bool
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
	// nilChansDone keeps each nil channel that writeNilChans wrote.
	nilChansDone map[string]bool
	// instDescs keys the variable of each instance description, and
	// instDescsDone keeps each one that writeInstDescs wrote. The text of 2
	// descriptions can reach each other, thus each one stands in a variable.
	instDescs     map[string]types.Type
	instDescsDone map[string]bool
	// descBusy keeps each instance description that the emitter writes now.
	descBusy map[string]bool
	// anonDone keeps each local shape class that writeAnonClasses wrote.
	anonDone map[string]bool
	// anonParams holds the type parameters of each shape class that reads a
	// parameter, by the name of the class.
	anonParams map[string][]*types.TypeParam
	// groundTwins holds the open twin of each ground shape that has one, by the
	// name of the class of the ground shape. A nil entry names no twin.
	groundTwins map[string]*anonTwin
	// typeCycle keeps each named type that reaches itself, and typeAcyclic
	// keeps each one that does not. Read cyclic. typeDepth caps a cycle
	// through a function type.
	typeCycle   map[*types.TypeName]bool
	typeAcyclic map[*types.TypeName]bool
	typeDepth   int
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
	// frameSeen holds the name of each frame class that the emitter already
	// wrote. placeLocalFuncs of stage 4 declares 1 literal in each state that
	// reads it, and the class of that literal stands at package level, thus a
	// second write would give Dart 2 declarations of 1 name.
	frameSeen map[string]bool
	// typeParams holds the type parameters of the function that the emitter
	// writes. The frame class of a literal of that function names them, because
	// the class stands at package level.
	typeParams []*types.TypeParam
	// funcIds holds a token for each function and each literal of the
	// package. The chain of the deferred calls gives the token of 1 function
	// to the runtime, and recover reads it, so a recover in a function that
	// the chain does not run gives nil.
	funcIds map[any]int
	funcSeq int
	// funcByObj gives the tree of each named function of the package.
	funcByObj map[types.Object]*ir.Func
	// curFuncId holds the token of the function that the emitter writes now.
	curFuncId int
	// varLitId holds the token of the literal that a hoisted variable holds,
	// by the name of the variable.
	varLitId map[string]int
	// goLine holds the source line of the last marker. A statement of the same
	// line needs no second marker.
	goLine int
	// goMarkFile holds the file of the last marker, in the form of the map.
	goMarkFile string
	// funcFile holds the map form of the file of the current function. A line
	// marker names its file only when a line directive changed it.
	funcFile string
	// curGoName holds the Go name of the current function. The frame class of
	// a local function reads it.
	curGoName string
	// litSeq numbers the literals of the current function, in the order of
	// the output. Go numbers them in the order of the source.
	litSeq int
	// extra holds each import that the emitter added. Read addImport.
	extra []Import
}

// The 2 markers of the source map. Each one stands on a line of its own, thus
// dart format keeps it in front of the code that it names. Stage 6 reads the
// markers of the formatted output and removes them.
const (
	fileMark = "//#f"
	lineMark = "//#g"
)

// markFunc opens the source map entry of a function.
func (e *emitter) markFunc(f *ir.Func) {
	if e.cfg.Fset == nil || !f.Pos.IsValid() {
		return
	}
	e.curGoName = e.goFuncName(f)
	e.funcFile = e.goFile(f.Pos)
	e.litSeq = 0
	e.line("%s%s|%s", fileMark, e.funcFile, e.curGoName)
	e.goLine = 0
	e.goMarkFile = e.funcFile
}

// goFile gives the map form of the file of a position. A file of the package
// takes the package path and the base name, thus 2 checkouts of 1 tree give
// equal output. The text of a line directive passes verbatim, exactly as the
// Go runtime reports it.
func (e *emitter) goFile(pos token.Pos) string {
	adjusted := e.cfg.Fset.Position(pos)
	raw := e.cfg.Fset.PositionFor(pos, false)
	if adjusted.Filename != raw.Filename {
		// A line directive with no file gives an empty name, and Go writes ??
		// for it. A directive that holds a column keeps the file of the last
		// 1, and go/token holds that rule already.
		if adjusted.Filename == "" {
			return "??"
		}
		return adjusted.Filename
	}
	base := filepath.Base(adjusted.Filename)
	if e.pkg != nil {
		return e.pkg.Path() + "/" + base
	}
	return base
}

// markDrop unmaps the output line that follows. The wrapper of a deferred
// call is such a line: Go holds no frame of it, thus the reader of the map
// must drop it from a stack.
func (e *emitter) markDrop() {
	if e.cfg.Fset == nil {
		return
	}
	e.line("%s0", lineMark)
	e.goLine = 0
	e.goMarkFile = ""
}

// mark opens the source map entry of a source line.
func (e *emitter) mark(pos token.Pos) {
	if e.cfg.Fset == nil || !pos.IsValid() {
		return
	}
	line := e.cfg.Fset.Position(pos).Line
	file := e.goFile(pos)
	if line == e.goLine && file == e.goMarkFile {
		return
	}
	e.goLine = line
	e.goMarkFile = file
	if file == e.funcFile {
		e.line("%s%d", lineMark, line)
		return
	}
	e.line("%s%d|%s", lineMark, line, file)
}

// goFuncName writes the name of a function in the form of the Go runtime.
func (e *emitter) goFuncName(f *ir.Func) string {
	path := ""
	if e.pkg != nil {
		path = e.pkg.Path()
		if e.pkg.Name() == "main" {
			path = "main"
		}
	}
	if f.Recv == nil {
		return path + "." + f.Name
	}
	recv := "?"
	if obj := receiverTypeName(f.Recv.Type); obj != nil {
		recv = obj.Name()
		if isPointerType(f.Recv.Type) {
			recv = "(*" + recv + ")"
		}
	}
	return path + "." + recv + "." + f.Name
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

// shadowSet gives each Dart name that a class of the package holds: each
// method name and each field name.
//
// The set comes from the type information, thus the emitter computes the same
// set for the package that it compiles and for a package that it imports. Only
// a struct type reaches a class. A method of another named type becomes a top
// level function, and such a function shadows nothing.
//
// A field shadows a package variable of the same name inside each method of
// its class. `ccc[p.ccc]` of x/text/unicode/norm reads the package variable
// `ccc` and the field `ccc`, and Dart would read the field 2 times.
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
		st, ok := types.Unalias(named).Underlying().(*types.Struct)
		if !ok {
			continue
		}
		for i := 0; i < st.NumFields(); i++ {
			out[name.Package(st.Field(i).Name())] = true
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
			if fn, ok := d.(*ir.Func); ok && fn.Obj != nil {
				e.funcByObj[fn.Obj] = fn
			}
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
						// and Dart calls no member of null. The function of a
						// method of a generic type also takes the dictionary of
						// each type parameter of the receiver type.
						e.inClass[d] = !isPointerType(d.Recv.Type)
					}
				case d.Name == "init":
					e.initNames[d] = fmt.Sprintf("_init$%d", len(e.inits))
					e.inits = append(e.inits, d)
				case d.Name == "_":
					// A package may hold many functions of the blank name, and
					// stringer writes 1 for each type that it generates. Go
					// calls none of them, thus each 1 takes a name of its own
					// to keep the names of the Dart output distinct.
					e.initNames[d] = fmt.Sprintf("_blank$%d", len(e.initNames))
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

// header stands at the top of each generated file.
//
// The list holds each lint that a mechanical translation of Go raises. A user
// of the output runs `dart analyze` or `flutter analyze` over the whole
// directory, thus a clean run needs the list here and not in an analysis
// options file of the user.
const header = `// Generated by gart. Do not edit.
// ignore_for_file: unused_local_variable, unused_element, unused_field
// ignore_for_file: non_constant_identifier_names, camel_case_types
// ignore_for_file: constant_identifier_names, unnecessary_non_null_assertion
// ignore_for_file: unused_import, prefer_final_locals, curly_braces_in_flow_control_structures
// ignore_for_file: dead_code, unnecessary_null_comparison, avoid_init_to_null
// ignore_for_file: unnecessary_this, avoid_renaming_method_parameters
// ignore_for_file: library_private_types_in_public_api, prefer_final_fields
// ignore_for_file: no_leading_underscores_for_local_identifiers, unused_label
// ignore_for_file: prefer_conditional_assignment
// ignore_for_file: prefer_function_declarations_over_variables
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
	e.goLine = 0
	e.writeAnonClasses()
	e.writeNilChans()
	e.writeInstDescs()
	e.writeDescriptors()
	e.line("")
	e.writeInit(sorted)
	e.flushFrames()
	e.writeNilChans()
	e.writeInstDescs()
	e.writeAnonClasses()
	if p.Name == "main" {
		e.writeMain()
	}
	body := e.buf.String()

	// The body can add an import, thus the list of the file stands after it. An
	// added package needs no init call: the package that names its type already
	// runs it.
	all := append(append([]Import{}, sorted...), e.extra...)
	sort.Slice(all, func(i, j int) bool { return all[i].Path < all[j].Path })
	e.buf.Reset()
	e.buf.WriteString(header)
	e.line("")
	e.line("import 'package:goruntime/goruntime.dart';")
	for _, im := range all {
		e.line("import '%s' as %s;", im.URI, e.prefixes[im.Path])
	}
	for _, uri := range e.dartOrder {
		e.line("import '%s' as %s;", uri, e.dartLibs[uri])
	}
	for _, uri := range e.anonURIs() {
		e.line("import '%s';", uri)
	}
	e.line("")
	for _, part := range parts {
		e.line("part '%s';", part)
	}
	e.line("part '%s.map.dart';", e.base)
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
	e.line("$srcmap($srcmapData);")
	for _, im := range imports {
		e.line("%s.$init();", e.prefixes[im.Path])
	}
	blanks := 0
	for _, g := range e.globals {
		if g.Var.Name == "_" {
			// Go runs the initializer of a blank variable for its effect, in
			// the order of the init. Only the value falls away.
			if g.Init != nil {
				e.list(g.Pending)
				e.line("final $blank%d = %s;", blanks, e.expr(g.Init))
				blanks++
			}
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
		e.line("} on GoPanic catch ($p, $s) {")
		e.line("$fatal($p, $s);")
		e.line("}")
	}
	e.line("}")
}

func (e *emitter) partFile(base string, f *ir.File) string {
	e.buf.Reset()
	e.goLine = 0
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
	// A string builds its bytes, and a Go integer builds its form at the
	// first read, thus the 2 kinds take a lazy final.
	if isString(d.Type) || isInteger(d.Type) || isComplexType(d.Type) {
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
