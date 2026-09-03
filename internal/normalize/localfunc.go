package normalize

import (
	"reflect"

	"github.com/eddyw/godart/internal/ir"
)

// This file places the declaration of each local function of a state machine.
//
// A function literal that can block becomes a local function, because the
// frame of the literal must name it. The declaration stands in front of the
// statement that reads it. A call that blocks then splits that statement over
// 2 states, and Dart holds a scope of its own for each case clause of a
// switch. Thus a declaration of 1 state reaches no other state, and the Dart
// compiler reports an undefined name.
//
// placeLocalFuncs moves such a declaration into each state that reads the
// name. A Go function value holds no equality, thus 2 declarations of 1
// literal hold 1 meaning.

// irPkgPath names the package of the tree. The walk reads a node of that
// package alone, thus it never enters the type graph of go/types.
var irPkgPath = reflect.TypeOf(ir.Ident{}).PkgPath()

// placeLocalFuncs moves each local function declaration of a machine into the
// states that read its name.
func placeLocalFuncs(m *ir.Machine) {
	// done holds each declaration that the pass already read. A move puts a
	// declaration into a later state, and the walk of the states reaches it a
	// second time. Without the set the pass would move it again and again.
	done := map[*ir.LocalFunc]bool{}
	for i, state := range m.States {
		for j := 0; j < len(state.List); j++ {
			decl, ok := state.List[j].(*ir.LocalFunc)
			if !ok || done[decl] {
				continue
			}
			done[decl] = true
			users := readerStates(m, decl)
			if len(users) == 0 || (len(users) == 1 && users[0] == i) {
				continue
			}
			state.List = append(state.List[:j:j], state.List[j+1:]...)
			j--
			for _, u := range users {
				insertDecl(m.States[u], decl)
			}
		}
	}
}

// readerStates gives the index of each state that reads the name of a
// declaration. The walk skips the declaration itself, thus a literal that
// calls itself needs no move.
func readerStates(m *ir.Machine, decl *ir.LocalFunc) []int {
	var out []int
	for i, state := range m.States {
		for _, s := range state.List {
			if readsName(reflect.ValueOf(s), decl.Name, decl) {
				out = append(out, i)
				break
			}
		}
	}
	return out
}

// insertDecl puts a declaration in front of the first statement of a block
// that reads its name.
func insertDecl(block *ir.Block, decl *ir.LocalFunc) {
	at := 0
	for k, s := range block.List {
		if readsName(reflect.ValueOf(s), decl.Name, decl) {
			at = k
			break
		}
	}
	list := make([]ir.Stmt, 0, len(block.List)+1)
	list = append(list, block.List[:at]...)
	list = append(list, decl)
	list = append(list, block.List[at:]...)
	block.List = list
}

// readsName reports whether a node of the tree holds an identifier of the
// name. The walk reads a struct of the tree package alone, thus it stops at
// the first node of go/types and holds no cycle.
func readsName(v reflect.Value, name string, skip *ir.LocalFunc) bool {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return false
		}
		if v.Type() == reflect.TypeOf(skip) &&
			v.UnsafePointer() == reflect.ValueOf(skip).UnsafePointer() {
			return false
		}
		return readsName(v.Elem(), name, skip)
	case reflect.Interface:
		if v.IsNil() {
			return false
		}
		return readsName(v.Elem(), name, skip)
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if readsName(v.Index(i), name, skip) {
				return true
			}
		}
		return false
	case reflect.Struct:
		t := v.Type()
		if t.PkgPath() != irPkgPath {
			return false
		}
		if t.Name() == "Ident" {
			return v.FieldByName("Name").String() == name
		}
		for i := 0; i < v.NumField(); i++ {
			if readsName(v.Field(i), name, skip) {
				return true
			}
		}
		return false
	}
	return false
}
