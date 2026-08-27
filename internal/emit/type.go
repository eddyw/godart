package emit

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/name"
)

// dartType gives the Dart type that holds a value of the Go type.
func (e *emitter) dartType(t types.Type) string {
	t = types.Unalias(t)
	switch t := t.(type) {
	case *types.Basic:
		return basicType(t)
	case *types.Named:
		if _, ok := t.Underlying().(*types.Struct); ok {
			return e.className(t.Obj())
		}
		return e.dartType(t.Underlying())
	case *types.Pointer:
		return e.dartType(t.Elem()) + "?"
	case *types.Slice:
		return "Slice<" + e.dartType(t.Elem()) + ">"
	case *types.Array:
		return "List<" + e.dartType(t.Elem()) + ">"
	case *types.Map:
		return "GoMap<" + e.dartType(t.Key()) + ", " + e.dartType(t.Elem()) + ">"
	case *types.Signature:
		return e.funcType(t) + "?"
	case *types.Tuple:
		return e.recordType(t)
	case *types.Interface:
		return "Object?"
	case *types.Struct:
		e.fail("an unnamed struct type is outside the subset")
		return "Object?"
	}
	e.fail("the type %s is outside the subset", t)
	return "Object?"
}

func basicType(t *types.Basic) string {
	switch t.Kind() {
	case types.Bool, types.UntypedBool:
		return "bool"
	case types.String, types.UntypedString:
		return "GoString"
	case types.Float32, types.Float64, types.UntypedFloat:
		return "double"
	case types.UnsafePointer:
		return "Object?"
	}
	if t.Info()&types.IsInteger != 0 {
		return "int"
	}
	return "Object?"
}

// funcType gives the Dart type of a function value.
func (e *emitter) funcType(sig *types.Signature) string {
	params := make([]string, sig.Params().Len())
	for i := range params {
		params[i] = e.dartType(sig.Params().At(i).Type())
	}
	return e.resultType(sig) + " Function(" + strings.Join(params, ", ") + ")"
}

// resultType gives the Dart type that holds the results. A function with more
// than 1 result returns a record.
func (e *emitter) resultType(sig *types.Signature) string {
	rs := sig.Results()
	switch rs.Len() {
	case 0:
		return "void"
	case 1:
		return e.dartType(rs.At(0).Type())
	}
	return e.recordType(rs)
}

func (e *emitter) recordType(t *types.Tuple) string {
	parts := make([]string, t.Len())
	for i := range parts {
		parts[i] = e.dartType(t.At(i).Type())
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// zero gives the Dart text of the zero value of a Go type.
func (e *emitter) zero(t types.Type) string {
	u := types.Unalias(t)
	if named, ok := u.(*types.Named); ok {
		if _, isStruct := named.Underlying().(*types.Struct); isStruct {
			return e.className(named.Obj()) + ".$zero()"
		}
		u = named.Underlying()
	}
	switch u := u.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Kind() == types.Bool || u.Kind() == types.UntypedBool:
			return "false"
		case u.Kind() == types.String || u.Kind() == types.UntypedString:
			return "GoString.empty"
		case u.Info()&types.IsFloat != 0:
			return "0.0"
		case u.Info()&types.IsInteger != 0:
			return "0"
		}
		return "null"
	case *types.Slice:
		return "Slice<" + e.dartType(u.Elem()) + ">.nil()"
	case *types.Array:
		return e.makeArray(u)
	case *types.Map:
		return "GoMap<" + e.dartType(u.Key()) + ", " + e.dartType(u.Elem()) + ">.nil()"
	case *types.Pointer, *types.Signature, *types.Interface, *types.Chan:
		return "null"
	}
	e.fail("the type %s has no zero value in the subset", t)
	return "null"
}

func (e *emitter) makeArray(a *types.Array) string {
	return fmt.Sprintf("$makeArray<%s>(%d, () => %s)",
		e.dartType(a.Elem()), a.Len(), e.zero(a.Elem()))
}

// alloc gives a Dart function that builds the backing array of a slice. A
// numeric element type takes a typed data list.
func (e *emitter) alloc(elem types.Type) string {
	if list := typedList(elem); list != "" {
		return fmt.Sprintf("(n) => %s(n)", list)
	}
	return fmt.Sprintf("(n) => List<%s>.generate(n, (_) => %s)",
		e.dartType(elem), e.zero(elem))
}

// typedList gives the typed data list of a numeric element type. Such a list
// holds the values without a box, thus it saves memory and time.
func typedList(t types.Type) string {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	if !ok {
		return ""
	}
	switch b.Kind() {
	case types.Int8:
		return "Int8List"
	case types.Int16:
		return "Int16List"
	case types.Int32:
		return "Int32List"
	case types.Int, types.Int64:
		return "Int64List"
	case types.Uint8:
		return "Uint8List"
	case types.Uint16:
		return "Uint16List"
	case types.Uint32:
		return "Uint32List"
	case types.Uint, types.Uint64, types.Uintptr:
		return "Uint64List"
	case types.Float32:
		return "Float32List"
	case types.Float64:
		return "Float64List"
	}
	return ""
}

func (e *emitter) className(obj *types.TypeName) string {
	return e.qualify(obj, name.Package(obj.Name()))
}

// qualify adds the import prefix when the object belongs to another package.
func (e *emitter) qualify(obj types.Object, dartName string) string {
	if obj.Pkg() == nil || obj.Pkg() == e.pkg {
		return dartName
	}
	prefix, ok := e.prefixes[obj.Pkg().Path()]
	if !ok {
		e.fail("the package %s has no import prefix", obj.Pkg().Path())
		return dartName
	}
	return prefix + "." + dartName
}

// isUnsigned64 reports whether the type needs the unsigned helpers. A narrower
// unsigned type keeps a value that fits in a Dart int with no sign change.
func isUnsigned64(t types.Type) bool {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	if !ok {
		return false
	}
	switch b.Kind() {
	case types.Uint, types.Uint64, types.Uintptr:
		return true
	}
	return false
}

func isUnsigned(t types.Type) bool {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	return ok && b.Info()&types.IsUnsigned != 0
}

func isInteger(t types.Type) bool {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	return ok && b.Info()&types.IsInteger != 0
}

func isFloat(t types.Type) bool {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	return ok && b.Info()&types.IsFloat != 0
}

func isString(t types.Type) bool {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	return ok && b.Info()&types.IsString != 0
}

func isStructType(t types.Type) bool {
	_, ok := types.Unalias(t).Underlying().(*types.Struct)
	return ok
}

func isArrayType(t types.Type) bool {
	_, ok := types.Unalias(t).Underlying().(*types.Array)
	return ok
}

func isSliceType(t types.Type) bool {
	_, ok := types.Unalias(t).Underlying().(*types.Slice)
	return ok
}

func isPointerType(t types.Type) bool {
	_, ok := types.Unalias(t).Underlying().(*types.Pointer)
	return ok
}

// mask gives the runtime function that keeps a value inside a narrow type.
func mask(t types.Type) string {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	if !ok {
		return ""
	}
	switch b.Kind() {
	case types.Int8:
		return "$i8"
	case types.Int16:
		return "$i16"
	case types.Int32:
		return "$i32"
	case types.Uint8:
		return "$u8"
	case types.Uint16:
		return "$u16"
	case types.Uint32:
		return "$u32"
	case types.Float32:
		return "$f32"
	}
	return ""
}
