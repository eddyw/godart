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

	// InitOrder lists each package variable that holds an initializer, in the
	// order that Go gives it a value. A variable comes after each variable that
	// its initializer reads, also through a call.
	InitOrder []*types.Var
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

	// End holds the closing brace of the body. Go gives a deferred call that
	// runs while the function falls off its end or panics the position of
	// that brace, and the source map must give the same line.
	End token.Pos

	// DeferBlocks marks a function whose defer stack can hold a call that
	// blocks. The return path and the panic path then run inside the machine.
	DeferBlocks bool

	// Machine holds the flat state machine of a function that can block.
	Machine *Machine

	// TypeParams holds the type parameters of a generic function or of a
	// generic method. The type parameters of the receiver type stay out.
	TypeParams []*types.TypeParam

	// NoBody marks a declaration with no body. An assembly file holds the code
	// of such a function, or a go:linkname directive only exports the name.
	// The Dart form panics.
	NoBody bool

	// DartLib and DartName name the Dart library and the Dart symbol that a
	// declaration with no body stands for. A //gart:dart directive writes the
	// 2, and the emitter then calls that symbol.
	DartLib  string
	DartName string
}

// Machine holds the flat state machine of a function that can block.
//
// Such a function gives up control with a plain return. It writes each local
// into a frame, and the scheduler runs the frame again later. Thus the body
// keeps no structured control flow, and stage 4 rewrites it into numbered
// states.
type Machine struct {
	// States holds the statements of each state, in order.
	States []*Block

	// Locals holds each variable that the frame keeps. The list holds each
	// parameter, each local of the body, and each temporary of a call that can
	// block.
	Locals []*Var

	// Class names the Dart class of the frame.
	Class string

	// Recv holds the receiver of a method, and it is nil for a function.
	Recv *Var

	// Defer marks a function that holds a defer statement. The frame keeps the
	// stack of the deferred calls.
	Defer bool

	// DeferBlocks marks a machine whose defer stack can hold a call that
	// blocks. The chain of the deferred calls then holds states of its own,
	// thus such a call gives up control like each other call.
	DeferBlocks bool

	// ResultVars holds 1 variable for each result of a machine that holds
	// DeferBlocks. A return statement writes them, and the end of the defer
	// chain reads them.
	ResultVars []*Var

	// DeferState names the first state of the defer chain. A return statement
	// and a panic each reach it.
	DeferState int

	// Blocking marks a machine of a function that can block. A machine of a
	// function that holds only a goto statement needs no frame.
	Blocking bool
}

// Global is a package level variable.
type Global struct {
	Var  *Var
	Init Expr
	// Pending holds the declaration of each function literal of Init that can
	// block. Such a literal becomes a local function, and the init routine
	// writes it in front of the assignment.
	Pending []Stmt
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

	// Boxed marks a variable whose address the code takes. Such a variable holds
	// a box, and a read of it reads the field of the box.
	Boxed bool

	// Dart holds the Dart type of a variable that no Go type describes, and
	// DartZero holds its first value. A select statement needs 1 such variable
	// for its case list.
	Dart     string
	DartZero string
}

// A Stmt is a statement.
type Stmt interface{ isStmt() }

// Node carries the position of the Go source of a statement. Stage 4 writes it,
// and stage 5 builds the source map of the output from it.
type Node struct{ Pos token.Pos }

func (n *Node) setPos(p token.Pos) {
	if n.Pos == token.NoPos {
		n.Pos = p
	}
}

func (n *Node) position() token.Pos { return n.Pos }

// SetPos gives a statement the position of its Go source. A statement that
// already holds one keeps it, thus an inner statement keeps its own position.
func SetPos(s Stmt, p token.Pos) {
	if n, ok := s.(interface{ setPos(token.Pos) }); ok {
		n.setPos(p)
	}
}

// PosOf gives the position of the Go source of a statement. A statement that
// stage 4 built alone carries none, and it gives token.NoPos.
func PosOf(s Stmt) token.Pos {
	if n, ok := s.(interface{ position() token.Pos }); ok {
		return n.position()
	}
	return token.NoPos
}

type (
	// Block is a statement list with its own scope.
	Block struct {
		Node
		List []Stmt
	}

	// ExprStmt evaluates an expression and drops the result.
	ExprStmt struct {
		Node
		X Expr
	}

	// Declare introduces variables. Init is nil for the zero value.
	Declare struct {
		Node
		Vars []*Var
		Init Expr
	}

	// Assign writes 1 value to 1 location.
	Assign struct {
		Node
		Lhs, Rhs Expr

		// Define marks the declaration of a variable that a machine lifted in
		// front of its loop. Go binds a new variable at such a point, thus the
		// write replaces the value and does not write through the old one.
		Define bool
	}

	// AssignMulti reads a record and writes the parts to the locations.
	AssignMulti struct {
		Node
		Lhs []Expr
		Rhs Expr

		// Define marks the declaration of a variable that a machine lifted.
		// Read the field Define of Assign.
		Define bool
	}

	If struct {
		Node
		Cond Expr
		Then *Block
		Else Stmt
	}

	For struct {
		Node
		Init  Stmt
		Cond  Expr
		Post  Stmt
		Body  *Block
		Label string
	}

	Return struct {
		Node
		Values []Expr
	}

	Break struct {
		Node
		Label string
	}

	Continue struct {
		Node
		Label string
	}

	Goto struct {
		Node
		Label string
	}

	Labeled struct {
		Node
		Label string
		Stmt  Stmt

		// Goto marks a label that a goto statement reaches. Stage 4 turns such
		// a label into 1 block of a state machine.
		Goto bool
	}

	// Switch holds a tag switch. Normalize rewrites a switch with no tag into
	// a switch on true.
	Switch struct {
		Node
		Tag   Expr
		Cases []*Case
		Label string
	}

	Send struct {
		Node
		Chan, Value Expr
	}

	Go struct {
		Node
		Call Expr
	}

	Defer struct {
		Node
		Call Expr
	}

	// Select runs 1 case of a select statement.
	Select struct {
		Node
		Cases []*SelectClause
		Label string
	}

	// Jump writes the number of the next state and starts the machine again.
	Jump struct{ State int }

	// Resume re-enters the call of the state before it, and it leaves the
	// machine when that call gave up control.
	//
	// Var holds the result of the call, and it is nil for a call with no
	// result. State names the state that holds this statement.
	Resume struct {
		Var   *Var
		State int
	}

	// DeferNext runs the next deferred call of a machine. The chain reaches
	// Done when the stack holds no call.
	DeferNext struct {
		Body int
		Done int
	}

	// DeferEnd stands at the end of the defer chain. A panic that the machine
	// caught reaches the caller here.
	DeferEnd struct{}

	// LocalFunc declares a function inside a function body. A function literal
	// that can block takes this form, because the frame of the literal must
	// name it.
	LocalFunc struct {
		Name string
		Lit  *FuncLit
	}

	// SelectBind writes the value of a receive case into its variables.
	SelectBind struct {
		List   Expr
		Index  int
		Clause *SelectClause
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
func (*Jump) isStmt()        {}
func (*Resume) isStmt()      {}
func (*DeferNext) isStmt()   {}
func (*DeferEnd) isStmt()    {}
func (*LocalFunc) isStmt()   {}
func (*SelectBind) isStmt()  {}

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

		// Boxed marks a read of a variable that holds a box. Read the field
		// Boxed of Var.
		Boxed bool
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

		// TypeArgs holds the type arguments of a call to a generic function or
		// to a generic method.
		TypeArgs []types.Type

		// Results holds a call whose results feed each argument. Go passes each
		// result of such a call as its own argument.
		//
		// The emitter writes a closure that takes the record of the results, and
		// each argument reads 1 field of it. ResultsVar names that closure
		// parameter.
		Results    Expr
		ResultsVar string
	}

	// Method selects a method on a receiver. It appears as the Fun of a Call.
	// MethodExpr is a method expression, for example `(*T).M`. The value is a
	// function whose first parameter is the receiver.
	MethodExpr struct {
		Obj  *types.Func
		Recv types.Type
		Type types.Type

		// TypeArgs holds the type arguments of an instance of a generic
		// method.
		TypeArgs []types.Type
	}

	// FuncInst names an instance of a generic function that reaches a value,
	// for example `equal[int]`. Type holds the instantiated signature.
	FuncInst struct {
		Fun      *Ident
		TypeArgs []types.Type
		Type     types.Type
	}

	Method struct {
		Recv Expr
		Obj  *types.Func
		Type types.Type

		// TypeArgs holds the type arguments of an instance of a generic
		// method.
		TypeArgs []types.Type
	}

	// Builtin calls a Go builtin, for example len or append.
	Builtin struct {
		Name   string
		Args   []Expr
		Type   types.Type
		Spread bool

		// Results holds a call whose results feed each argument, exactly as
		// the field of the same name on Call does.
		Results    Expr
		ResultsVar string
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
		// Results holds the record of the values in the order of the source,
		// and each element reads 1 field of it. ResultsVar names the record.
		Results    Expr
		ResultsVar string
		Keys       []Expr
		Elems      []Expr
		Type       types.Type
	}

	// FuncLit is a closure.
	FuncLit struct {
		Params  []*Var
		Results []*Var
		Body    *Block
		// End holds the closing brace of the body. Read the End field of Func.
		End    token.Pos
		Async  bool
		Defers bool
		Type   types.Type

		// Machine holds the flat state machine of a literal that can block.
		Machine *Machine

		// Captured holds each boxed variable of an outer scope that the body
		// reads. A machine keeps such a variable in 1 local and writes a new
		// box at each pass of a loop, thus the literal takes a copy of the box
		// of its own pass.
		Captured []*Var
	}

	// SelectList builds the case list of a select statement.
	SelectList struct {
		Cases []*SelectClause
		Type  types.Type
	}

	// SelectRun runs a select statement and gives the number of the case that
	// it takes.
	SelectRun struct {
		List       Expr
		HasDefault bool
		Type       types.Type
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

	// RecordField reads 1 field of a Dart record.
	RecordField struct {
		X     Expr
		Index int
		Type  types.Type
	}
)

func (e *Const) GoType() types.Type       { return e.Type }
func (e *Ident) GoType() types.Type       { return e.Type }
func (e *Blank) GoType() types.Type       { return e.Type }
func (e *Zero) GoType() types.Type        { return e.Type }
func (e *Field) GoType() types.Type       { return e.Type }
func (e *Index) GoType() types.Type       { return e.Type }
func (e *MapRead) GoType() types.Type     { return e.Type }
func (e *Slice) GoType() types.Type       { return e.Type }
func (e *Call) GoType() types.Type        { return e.Type }
func (e *Method) GoType() types.Type      { return e.Type }
func (e *MethodExpr) GoType() types.Type  { return e.Type }
func (e *FuncInst) GoType() types.Type    { return e.Type }
func (e *Builtin) GoType() types.Type     { return e.Type }
func (e *Binary) GoType() types.Type      { return e.Type }
func (e *Unary) GoType() types.Type       { return e.Type }
func (e *Conv) GoType() types.Type        { return e.Type }
func (e *Composite) GoType() types.Type   { return e.Type }
func (e *FuncLit) GoType() types.Type     { return e.Type }
func (e *TypeAssert) GoType() types.Type  { return e.Type }
func (e *TypeTest) GoType() types.Type    { return e.Type }
func (e *Recv) GoType() types.Type        { return e.Type }
func (e *Copy) GoType() types.Type        { return e.Type }
func (e *Record) GoType() types.Type      { return e.Type }
func (e *RecordField) GoType() types.Type { return e.Type }
func (e *SelectList) GoType() types.Type  { return e.Type }
func (e *SelectRun) GoType() types.Type   { return e.Type }

func (*Const) isExpr()       {}
func (*Ident) isExpr()       {}
func (*Blank) isExpr()       {}
func (*Zero) isExpr()        {}
func (*Field) isExpr()       {}
func (*Index) isExpr()       {}
func (*MapRead) isExpr()     {}
func (*Slice) isExpr()       {}
func (*Call) isExpr()        {}
func (*Method) isExpr()      {}
func (*MethodExpr) isExpr()  {}
func (*FuncInst) isExpr()    {}
func (*Builtin) isExpr()     {}
func (*Binary) isExpr()      {}
func (*Unary) isExpr()       {}
func (*Conv) isExpr()        {}
func (*Composite) isExpr()   {}
func (*FuncLit) isExpr()     {}
func (*TypeAssert) isExpr()  {}
func (*TypeTest) isExpr()    {}
func (*Recv) isExpr()        {}
func (*Copy) isExpr()        {}
func (*Record) isExpr()      {}
func (*RecordField) isExpr() {}
func (*SelectList) isExpr()  {}
func (*SelectRun) isExpr()   {}

// MapIterType types the iterator of a range loop over a map. Stage 4 builds
// the iterator with the builtin mapiter, and go/types holds no form of it.
// The type satisfies types.Type, thus a Var of the stage can carry it.
type MapIterType struct{ Map *types.Map }

func (t *MapIterType) Underlying() types.Type { return t }
func (t *MapIterType) String() string         { return "map iterator" }

// Fresh reports whether the expression builds a new value that no other
// expression names. A value semantics copy of such a value gives nothing.
//
// Stage 4 reads this rule at each copy point, and stage 5 reads it at the box
// of an interface value. The 2 must agree, because a copy that only 1 of them
// writes gives either a wasted object or an alias.
//
// A builtin is not fresh. iterkey and iterval each read the object that the
// map holds.
func Fresh(e Expr) bool {
	switch e.(type) {
	case *Composite, *Call, *Copy, *Zero, *Conv:
		return true
	}
	return false
}
