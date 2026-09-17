//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
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
	case types.Int, types.UntypedInt:
		return "int"
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
			return node{"op": "int", "value": x.Value.ExactString()}
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
		if kind(x.Type) == "int" {
			return node{"op": "int", "value": "0"}
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
		case token.ADD, token.SUB, token.MUL, token.LSS, token.GTR, token.LEQ, token.GEQ, token.EQL, token.NEQ:
			if kind(x.X.GoType()) != "int" || kind(x.Y.GoType()) != "int" {
				panic("prototype binary operators require int")
			}
			return node{"op": "binary", "operator": x.Op.String(), "left": expr(x.X), "right": expr(x.Y), "type": kind(x.Type)}
		}
	case *ir.Conv:
		if kind(x.From) == kind(x.Type) {
			return expr(x.X)
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
		return node{"op": "call", "name": f.Name, "args": args, "type": kind(x.Type)}
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
			args = append(args, expr(a))
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
		if k != "int" && k != "bool" {
			panic("prototype variables require int or bool")
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

func run() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("usage: frontend input.go output-directory")
	}
	source, err := filepath.Abs(os.Args[1])
	if err != nil {
		return err
	}
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, source, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	if f.Name.Name != "main" || len(f.Imports) != 0 {
		return fmt.Errorf("prototype requires package main with no imports")
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}, Scopes: map[ast.Node]*types.Scope{}, Instances: map[*ast.Ident]types.Instance{}}
	conf := types.Config{GoVersion: "go1.27"}
	tp, err := conf.Check("example.com/kernelprobe", fs, []*ast.File{f}, info)
	if err != nil {
		return err
	}
	if err = verify.Package(fs, []*ast.File{f}, info, nil); err != nil {
		return err
	}
	pkg := &packages.Package{Fset: fs, Syntax: []*ast.File{f}, Types: tp, TypesInfo: info}
	blocks := analyze.Module([]*packages.Package{pkg})
	tree, err := normalize.Package(fs, pkg.Syntax, info, tp, blocks)
	if err != nil {
		return err
	}
	funcs := []node{}
	for _, file := range tree.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ir.Func)
			if !ok || fn.Async || fn.Defers || fn.Machine != nil || fn.Recv != nil || len(fn.TypeParams) != 0 || fn.NoBody || fn.Name == "init" {
				return fmt.Errorf("unsupported declaration %T", decl)
			}
			params := []node{}
			for _, p := range fn.Params {
				k := kind(p.Type)
				if k != "int" && k != "bool" {
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
				if result != "int" && result != "bool" {
					return fmt.Errorf("unsupported result")
				}
			}
			funcs = append(funcs, node{"name": fn.Name, "params": params, "result": result, "body": stmt(fn.Body), "offset": fs.Position(fn.Pos).Offset})
		}
	}
	output := os.Args[2]
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(node{"source": source, "functions": funcs})
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(output, "program.json"), data, 0644); err != nil {
		return err
	}
	emitted, err := emit.Package(tree, tp, emit.Config{Base: "main"})
	if err != nil {
		return err
	}
	for name, text := range emitted.Files {
		path := filepath.Join(output, name)
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err = os.WriteFile(path, []byte(text), 0644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(output, emitted.MapFile), []byte("part of 'main.dart';\nconst $srcmapData = <String>[];\n"), 0644)
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
