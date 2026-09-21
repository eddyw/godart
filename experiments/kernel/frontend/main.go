//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/eddyw/godart/internal/analyze"
	"github.com/eddyw/godart/internal/emit"
	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/normalize"
	"github.com/eddyw/godart/internal/verify"
	"golang.org/x/tools/go/packages"
)

type node = map[string]any

func kind(t types.Type) string {
	if t == nil {
		return "void"
	}
	if tuple, ok := t.(*types.Tuple); ok && tuple.Len() == 0 {
		return "void"
	}
	b, ok := t.(*types.Basic)
	if !ok {
		panic(fmt.Sprintf("unsupported type %s", t))
	}
	switch b.Kind() {
	case types.Int, types.Int64, types.UntypedInt:
		return "int"
	case types.Uint, types.Uint64:
		return "uint64"
	case types.Bool, types.UntypedBool:
		return "bool"
	case types.String, types.UntypedString:
		return "string"
	}
	panic(fmt.Sprintf("unsupported type %s", t))
}

func expr(e ir.Expr) node {
	if e == nil {
		return nil
	}
	switch x := e.(type) {
	case *ir.Const:
		switch x.Value.Kind() {
		case constant.Int:
			return node{"op": "int", "value": x.Value.ExactString(), "type": kind(x.Type)}
		case constant.Bool:
			return node{"op": "bool", "value": constant.BoolVal(x.Value)}
		case constant.String:
			s := constant.StringVal(x.Value)
			if !utf8.ValidString(s) {
				panic("prototype requires UTF-8 string literals")
			}
			return node{"op": "string", "value": s}
		}
	case *ir.Zero:
		if integer(kind(x.Type)) {
			return node{"op": "int", "value": "0", "type": kind(x.Type)}
		}
		if kind(x.Type) == "bool" {
			return node{"op": "bool", "value": false}
		}
	case *ir.Ident:
		if x.Boxed {
			panic("boxed variables are outside the prototype")
		}
		return node{"op": "get", "name": x.Name, "type": kind(x.Type)}
	case *ir.Binary:
		switch x.Op {
		case token.ADD, token.SUB, token.MUL, token.LSS, token.GTR, token.LEQ, token.GEQ, token.EQL, token.NEQ, token.AND, token.OR, token.XOR, token.SHL, token.SHR:
			if !integer(kind(x.X.GoType())) || !integer(kind(x.Y.GoType())) {
				panic("prototype binary operators require 64-bit integers")
			}
			return node{"op": "binary", "operator": x.Op.String(), "left": expr(x.X), "right": expr(x.Y), "type": kind(x.Type), "unsigned": kind(x.X.GoType()) == "uint64", "unsignedCount": kind(x.Y.GoType()) == "uint64"}
		}
	case *ir.Unary:
		if integer(kind(x.Type)) && (x.Op == token.SUB || x.Op == token.XOR) {
			return node{"op": "unary", "operator": x.Op.String(), "value": expr(x.X), "type": kind(x.Type)}
		}
	case *ir.Conv:
		if kind(x.From) == kind(x.Type) || (integer(kind(x.From)) && integer(kind(x.Type))) {
			return node{"op": "convert", "value": expr(x.X), "type": kind(x.Type)}
		}
	case *ir.Call:
		f, ok := x.Fun.(*ir.Ident)
		if !ok || x.Async || x.Spread || x.Variadic || len(x.TypeArgs) != 0 {
			panic("unsupported call")
		}
		args := []node{}
		for _, a := range x.Args {
			args = append(args, expr(a))
		}
		return node{"op": "call", "name": functionKey(f.Obj), "args": args, "type": kind(x.Type)}
	case *ir.Builtin:
		if x.Name != "println" || x.Spread || x.Results != nil {
			panic("unsupported builtin")
		}
		args := []node{}
		for _, a := range x.Args {
			if kind(a.GoType()) == "string" {
				if _, ok := a.(*ir.Const); !ok {
					panic("only string literals are supported")
				}
			}
			value := expr(a)
			value["type"] = kind(a.GoType())
			args = append(args, value)
		}
		return node{"op": "println", "args": args}
	}
	panic(fmt.Sprintf("unsupported expression %T", e))
}

func stmt(s ir.Stmt) node {
	if s == nil {
		return node{"op": "block", "items": []node{}}
	}
	switch x := s.(type) {
	case *ir.Block:
		list := []node{}
		for _, s := range x.List {
			list = append(list, stmt(s))
		}
		return node{"op": "block", "items": list}
	case *ir.Declare:
		if len(x.Vars) != 1 || x.Vars[0].Boxed {
			panic("unsupported declaration")
		}
		v := x.Vars[0]
		k := kind(v.Type)
		if !integer(k) && k != "bool" {
			panic("prototype variables require 64-bit integers or bool")
		}
		init := x.Init
		if init == nil {
			init = &ir.Zero{Type: v.Type}
		}
		return node{"op": "declare", "name": v.Name, "type": k, "value": expr(init)}
	case *ir.Assign:
		v, ok := x.Lhs.(*ir.Ident)
		if !ok || v.Boxed || x.Define {
			panic("unsupported assignment")
		}
		return node{"op": "set", "name": v.Name, "value": expr(x.Rhs)}
	case *ir.ExprStmt:
		return node{"op": "expression", "value": expr(x.X)}
	case *ir.Return:
		if len(x.Values) > 1 {
			panic("multiple results are outside the prototype")
		}
		var value node
		if len(x.Values) == 1 {
			value = expr(x.Values[0])
		}
		return node{"op": "return", "value": value}
	case *ir.If:
		return node{"op": "if", "condition": expr(x.Cond), "then": stmt(x.Then), "else": stmt(x.Else)}
	case *ir.For:
		if x.Label != "" || len(x.CondPre) != 0 {
			panic("unsupported loop")
		}
		return node{"op": "for", "init": stmt(x.Init), "condition": expr(x.Cond), "post": stmt(x.Post), "body": stmt(x.Body)}
	}
	panic(fmt.Sprintf("unsupported statement %T", s))
}

func integer(name string) bool { return name == "int" || name == "uint64" }

func functionKey(obj types.Object) string { return obj.Pkg().Path() + "::" + obj.Name() }

// Both backends receive the same original math/bits function bodies.
func selectedFiles(p *packages.Package) ([]*ast.File, error) {
	selected := map[string]bool{"OnesCount64": true, "Reverse64": true, "ReverseBytes64": true, "RotateLeft64": true}
	var files []*ast.File
	for _, original := range p.Syntax {
		copy := *original
		copy.Decls = nil
		if p.PkgPath == "math/bits" {
			copy.Imports = nil
		}
		for _, decl := range original.Decls {
			if p.PkgPath == "math/bits" {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || !selected[fn.Name.Name] {
					continue
				}
			} else if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok != token.CONST && gen.Tok != token.IMPORT {
				return nil, fmt.Errorf("prototype excludes package state and named types in %s", p.PkgPath)
			}
			copy.Decls = append(copy.Decls, decl)
		}
		files = append(files, &copy)
	}
	return files, nil
}

func run() error {
	if len(os.Args) != 4 {
		return fmt.Errorf("usage: frontend input.go-or-module output-directory direct|source|both")
	}
	mode := os.Args[3]
	if mode != "direct" && mode != "source" && mode != "both" {
		return fmt.Errorf("unsupported output mode %q", mode)
	}
	start := time.Now()
	timings := map[string]float64{}
	input, err := filepath.Abs(os.Args[1])
	if err != nil {
		return err
	}
	output, err := filepath.Abs(os.Args[2])
	if err != nil {
		return err
	}
	dir, pattern := input, "."
	if strings.HasSuffix(input, ".go") {
		dir, pattern = filepath.Dir(input), input
	}
	cfg := &packages.Config{Dir: dir, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes | packages.NeedModule}
	roots, err := packages.Load(cfg, pattern)
	if err != nil {
		return err
	}
	if packages.PrintErrors(roots) != 0 {
		return fmt.Errorf("package load failed")
	}
	if len(roots) != 1 || roots[0].Name != "main" {
		return fmt.Errorf("prototype requires 1 main package")
	}
	if roots[0].TypesSizes.Sizeof(types.Typ[types.Int]) != 8 {
		return fmt.Errorf("prototype requires 64-bit Go int")
	}
	var pkgs []*packages.Package
	packages.Visit(roots, nil, func(p *packages.Package) {
		if p.PkgPath != "unsafe" {
			pkgs = append(pkgs, p)
		}
	})
	for _, p := range pkgs {
		if p != roots[0] && p.PkgPath != "math/bits" && p.Module == nil {
			return fmt.Errorf("unsupported standard package %s", p.PkgPath)
		}
		files, err := selectedFiles(p)
		if err != nil {
			return err
		}
		p.Syntax = files
		if err := verify.Package(p.Fset, p.Syntax, p.TypesInfo, nil); err != nil {
			return err
		}
	}
	timings["load_verify"] = time.Since(start).Seconds()
	phase := time.Now()
	blocks := analyze.Module(pkgs)
	if err := blocks.SyncErrors(); err != nil {
		return err
	}
	timings["analyze"] = time.Since(phase).Seconds()
	bases := map[string]string{}
	dirs := map[string]string{}
	for i, p := range pkgs {
		base := fmt.Sprintf("p%d", i)
		out := filepath.Join(output, base)
		if p == roots[0] {
			base, out = "main", output
		}
		bases[p.PkgPath], dirs[p.PkgPath] = base, out
	}
	all := []node{}
	selectedFunctions := []string{}
	for _, p := range pkgs {
		phase = time.Now()
		tree, err := normalize.Package(p.Fset, p.Syntax, p.TypesInfo, p.Types, blocks, nil)
		if err != nil {
			return err
		}
		timings["normalize"] += time.Since(phase).Seconds()
		funcs := []node{}
		phase = time.Now()
		for _, file := range tree.Files {
			for _, decl := range file.Decls {
				if _, ok := decl.(*ir.ConstDecl); ok {
					continue
				}
				fn, ok := decl.(*ir.Func)
				if !ok || fn.Async || fn.Defers || fn.Machine != nil || fn.Recv != nil || len(fn.TypeParams) != 0 || fn.NoBody || fn.Name == "init" {
					return fmt.Errorf("unsupported declaration %T", decl)
				}
				selectedFunctions = append(selectedFunctions, functionKey(fn.Obj))
				if mode == "source" {
					continue
				}
				params := []node{}
				for _, p := range fn.Params {
					k := kind(p.Type)
					if !integer(k) && k != "bool" {
						return fmt.Errorf("unsupported parameter")
					}
					params = append(params, node{"name": p.Name, "type": k})
				}
				result := "void"
				if len(fn.Results) > 1 {
					return fmt.Errorf("multiple results are outside the prototype")
				}
				if len(fn.Results) == 1 {
					result = kind(fn.Results[0].Type)
					if !integer(result) && result != "bool" {
						return fmt.Errorf("unsupported result")
					}
				}
				pos := p.Fset.Position(fn.Pos)
				funcs = append(funcs, node{"name": functionKey(fn.Obj), "localName": fn.Name, "params": params, "result": result, "body": stmt(fn.Body), "source": pos.Filename, "offset": pos.Offset})
			}
		}
		deps := []string{}
		var imports []emit.Import
		for path, dep := range p.Imports {
			if base, ok := bases[dep.PkgPath]; ok {
				deps = append(deps, dep.PkgPath)
				imports = append(imports, emit.Import{Path: path, URI: filepath.ToSlash(filepath.Join(dirs[dep.PkgPath], base+".dart"))})
			}
		}
		sort.Strings(deps)
		all = append(all, node{"path": p.PkgPath, "functions": funcs, "imports": deps})
		timings["export_ir"] += time.Since(phase).Seconds()
		if mode == "direct" {
			continue
		}
		phase = time.Now()
		sort.Slice(imports, func(i, j int) bool { return imports[i].Path < imports[j].Path })
		base := bases[p.PkgPath]
		emitted, err := emit.Package(tree, p.Types, emit.Config{Base: base, Imports: imports})
		if err != nil {
			return err
		}
		for name, text := range emitted.Files {
			path := filepath.Join(dirs[p.PkgPath], name)
			if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err = os.WriteFile(path, []byte(text), 0644); err != nil {
				return err
			}
		}
		text := "part of '" + base + ".dart';\nconst $srcmapData = <String>[];\n"
		if err = os.WriteFile(filepath.Join(dirs[p.PkgPath], emitted.MapFile), []byte(text), 0644); err != nil {
			return err
		}
		timings["source_write"] += time.Since(phase).Seconds()
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	if mode != "source" {
		data, err := json.Marshal(node{"packages": all, "main": roots[0].PkgPath + "::main"})
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(output, "program.json"), data, 0644); err != nil {
			return err
		}
	}
	sort.Strings(selectedFunctions)
	timings["total"] = time.Since(start).Seconds()
	return json.NewEncoder(os.Stdout).Encode(node{"phases": timings, "functions": selectedFunctions})
}

func main() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
