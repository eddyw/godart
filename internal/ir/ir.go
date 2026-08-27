// Package ir holds the intermediate tree of the compiler. Stage 4 builds the
// tree from the typed syntax tree, and stage 5 writes Dart source code from it.
//
// The tree keeps the statement structure of the Go source. It reuses the types
// of go/types, thus no second type model exists.
package ir

import (
	"go/constant"
	"go/token"
	"go/types"
)

// Package is 1 Go package. It becomes 1 Dart library.
type Package struct {
	Path  string
	Name  string
	Files []*File
}

// File is 1 Go file. It becomes 1 Dart part file.
type File struct {
	Name  string
	Decls []Decl
}

// A Decl is a declaration at package level.
type Decl interface{ isDecl() }

// Func is a function or a method.
type Func struct {
	Name    string
	Obj     *types.Func
	Recv    *Var
	Params  []*Var
	Results []*Var
	Body    *Block
	Async   bool
	Defers  bool
	Pos     token.Pos
}

// Global is a package level variable.
type Global struct {
	Var  *Var
	Init Expr
}

// ConstDecl is a package level constant.
type ConstDecl struct {
	Name  string
	Type  types.Type
	Value constant.Value
}

// TypeDecl is a named type. A struct becomes a Dart class.
type TypeDecl struct {
	Name string
	Obj  *types.TypeName
}

func (*Func) isDecl()      {}
func (*Global) isDecl()    {}
func (*ConstDecl) isDecl() {}
func (*TypeDecl) isDecl()  {}

// Var is a variable, a parameter, or a result.
type Var struct {
	Name string
	Obj  *types.Var
	Type types.Type

	// Named marks a result that the Go source gives a name.
	Named bool
}

// A Stmt is a statement.
type Stmt interface{ isStmt() }

type (
	// Block is a statement list with its own scope.
	Block struct{ List []Stmt }

	// ExprStmt evaluates an expression and drops the result.
	ExprStmt struct{ X Expr }

	// Declare introduces variables. Init is nil for the zero value.
	Declare struct {
		Vars []*Var
		Init Expr
	}

	// Assign writes 1 value to 1 location.
	Assign struct{ Lhs, Rhs Expr }

	// AssignMulti reads a record and writes the parts to the locations.
	AssignMulti struct {
		Lhs []Expr
		Rhs Expr
	}

	If struct {
		Cond Expr
		Then *Block
		Else Stmt
	}

	For struct {
		Init  Stmt
		Cond  Expr
		Post  Stmt
		Body  *Block
		Label string
	}

	Return struct{ Values []Expr }

	Break struct{ Label string }

	Continue struct{ Label string }

	Goto struct{ Label string }

	Labeled struct {
		Label string
		Stmt  Stmt

		// Goto marks a label that a goto statement reaches. Stage 4 turns such
		// a label into 1 block of a state machine.
		Goto bool
	}

	// Switch holds a tag switch. Normalize rewrites a switch with no tag into
	// a switch on true.
	Switch struct {
		Tag   Expr
		Cases []*Case
		Label string
	}

	Send struct{ Chan, Value Expr }

	Go struct{ Call Expr }

	Defer struct{ Call Expr }

	// Select runs 1 case of a select statement.
	Select struct {
		Cases []*SelectClause
		Label string
	}
)

// SelectKind names the form of 1 case of a select statement.
type SelectKind int

const (
	SelectRecv SelectKind = iota
	SelectSend
	SelectDefault
)

// SelectClause is 1 case of a select statement.
type SelectClause struct {
	Kind SelectKind
	Chan Expr
	// Value holds the value of a send case.
	Value Expr
	// Vars holds the variables that a receive case declares, and Lhs holds the
	// locations that a receive case writes.
	Vars []*Var
	Lhs  []Expr
	Elem types.Type
	Body *Block
}

// Case is 1 case of a switch. An empty Values list marks the default case.
type Case struct {
	Values      []Expr
	Body        *Block
	Fallthrough bool
}

func (*Block) isStmt()       {}
func (*ExprStmt) isStmt()    {}
func (*Declare) isStmt()     {}
func (*Assign) isStmt()      {}
func (*AssignMulti) isStmt() {}
func (*If) isStmt()          {}
func (*For) isStmt()         {}
func (*Return) isStmt()      {}
func (*Break) isStmt()       {}
func (*Continue) isStmt()    {}
func (*Goto) isStmt()        {}
func (*Labeled) isStmt()     {}
func (*Switch) isStmt()      {}
func (*Send) isStmt()        {}
func (*Go) isStmt()          {}
func (*Defer) isStmt()       {}
func (*Select) isStmt()      {}

// An Expr is an expression. Each expression carries its Go type.
type Expr interface {
	isExpr()
	GoType() types.Type
}

type (
	// Const is a literal value.
	Const struct {
		Value constant.Value
		Type  types.Type
	}

	// Ident reads a variable, a function, or a package level name.
	Ident struct {
		Name string
		Obj  types.Object
		Type types.Type
	}

	// Blank is the target `_` of an assignment.
	Blank struct{ Type types.Type }

	// Zero is the zero value of a type.
	Zero struct{ Type types.Type }

	// Field selects a struct field. X holds a struct or a pointer to a struct.
	Field struct {
		X    Expr
		Name string
		Type types.Type
	}

	// Index reads 1 element of a slice, an array, a string, or a map.
	Index struct {
		X, Index Expr
		Type     types.Type
	}

	// MapRead reads a map with the comma ok form.
	MapRead struct {
		X, Key Expr
		Type   types.Type
	}

	// Slice builds a slice from a slice, an array, or a string.
	Slice struct {
		X, Low, High, Max Expr
		Type              types.Type
	}

	// Call calls a function value, a method, or a builtin.
	Call struct {
		Fun      Expr
		Args     []Expr
		Type     types.Type
		Async    bool
		Spread   bool
		Variadic bool
	}

	// Method selects a method on a receiver. It appears as the Fun of a Call.
	Method struct {
		Recv Expr
		Obj  *types.Func
		Type types.Type
	}

	// Builtin calls a Go builtin, for example len or append.
	Builtin struct {
		Name   string
		Args   []Expr
		Type   types.Type
		Spread bool
	}

	Binary struct {
		Op   token.Token
		X, Y Expr
		Type types.Type
	}

	Unary struct {
		Op   token.Token
		X    Expr
		Type types.Type
	}

	// Conv converts a value between 2 Go types.
	Conv struct {
		X    Expr
		From types.Type
		Type types.Type
	}

	// Composite builds a struct, an array, a slice, or a map value. Keys is nil
	// for a positional literal.
	Composite struct {
		Keys  []Expr
		Elems []Expr
		Type  types.Type
	}

	// FuncLit is a closure.
	FuncLit struct {
		Params  []*Var
		Results []*Var
		Body    *Block
		Async   bool
		Defers  bool
		Type    types.Type
	}

	// TypeAssert reads the dynamic type of an interface value.
	TypeAssert struct {
		X      Expr
		Assert types.Type
		Comma  bool
		Type   types.Type
	}

	// TypeTest reports whether an interface value holds a type.
	TypeTest struct {
		X      Expr
		Assert types.Type
		Type   types.Type
	}

	// Recv reads 1 value from a channel.
	Recv struct {
		Chan  Expr
		Comma bool
		Type  types.Type
	}

	// Copy is the struct copy that stage 4 inserts at a copy point.
	Copy struct {
		X    Expr
		Type types.Type
	}

	// Record builds a Dart record for a function with multiple results.
	Record struct {
		Elems []Expr
		Type  types.Type
	}
)

func (e *Const) GoType() types.Type      { return e.Type }
func (e *Ident) GoType() types.Type      { return e.Type }
func (e *Blank) GoType() types.Type      { return e.Type }
func (e *Zero) GoType() types.Type       { return e.Type }
func (e *Field) GoType() types.Type      { return e.Type }
func (e *Index) GoType() types.Type      { return e.Type }
func (e *MapRead) GoType() types.Type    { return e.Type }
func (e *Slice) GoType() types.Type      { return e.Type }
func (e *Call) GoType() types.Type       { return e.Type }
func (e *Method) GoType() types.Type     { return e.Type }
func (e *Builtin) GoType() types.Type    { return e.Type }
func (e *Binary) GoType() types.Type     { return e.Type }
func (e *Unary) GoType() types.Type      { return e.Type }
func (e *Conv) GoType() types.Type       { return e.Type }
func (e *Composite) GoType() types.Type  { return e.Type }
func (e *FuncLit) GoType() types.Type    { return e.Type }
func (e *TypeAssert) GoType() types.Type { return e.Type }
func (e *TypeTest) GoType() types.Type   { return e.Type }
func (e *Recv) GoType() types.Type       { return e.Type }
func (e *Copy) GoType() types.Type       { return e.Type }
func (e *Record) GoType() types.Type     { return e.Type }

func (*Const) isExpr()      {}
func (*Ident) isExpr()      {}
func (*Blank) isExpr()      {}
func (*Zero) isExpr()       {}
func (*Field) isExpr()      {}
func (*Index) isExpr()      {}
func (*MapRead) isExpr()    {}
func (*Slice) isExpr()      {}
func (*Call) isExpr()       {}
func (*Method) isExpr()     {}
func (*Builtin) isExpr()    {}
func (*Binary) isExpr()     {}
func (*Unary) isExpr()      {}
func (*Conv) isExpr()       {}
func (*Composite) isExpr()  {}
func (*FuncLit) isExpr()    {}
func (*TypeAssert) isExpr() {}
func (*TypeTest) isExpr()   {}
func (*Recv) isExpr()       {}
func (*Copy) isExpr()       {}
func (*Record) isExpr()     {}
