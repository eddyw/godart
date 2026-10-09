package emit

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/typeset"
)

const deepSliceLevels = 32

// Keep large data literals out of a single Dart optimizer graph.
func (e *emitter) largeSlice(elem types.Type, parts []string) string {
	var text strings.Builder
	text.WriteString("(() {\n")
	var calls []string
	for start := 0; start < len(parts); start += 128 {
		name := fmt.Sprintf("$chunk%d", start/128)
		fmt.Fprintf(&text, "@pragma('dart2js:noInline')\nList<%s> %s() => <%s>[%s];\n",
			e.dartType(elem), name, e.dartType(elem), strings.Join(parts[start:min(start+128, len(parts))], ", "))
		calls = append(calls, "..."+name+"()")
	}
	fmt.Fprintf(&text, "return %s;\n})()", e.sliceLiteral(elem, calls))
	return text.String()
}

// Keep large array data out of a single Dart optimizer graph.
func (e *emitter) largeArray(elem types.Type, parts []string) string {
	var text strings.Builder
	text.WriteString("(() {\n")
	var calls []string
	for start := 0; start < len(parts); start += 128 {
		name := fmt.Sprintf("$chunk%d", start/128)
		fmt.Fprintf(&text, "@pragma('dart2js:noInline')\n@pragma('wasm:never-inline')\nList<%s> %s() => <%s>[%s];\n",
			e.dartType(elem), name, e.dartType(elem), strings.Join(parts[start:min(start+128, len(parts))], ", "))
		calls = append(calls, "..."+name+"()")
	}
	fmt.Fprintf(&text, "return <%s>[%s];\n})()", e.dartType(elem), strings.Join(calls, ", "))
	return e.inlineArray(text.String(), elem)
}

// A helper must not add a source frame for a Go call or a recoverable panic.
func literalData(x ir.Expr) bool {
	switch x := x.(type) {
	case *ir.Const, *ir.Zero:
		return true
	case *ir.Ident:
		return !x.Boxed
	case *ir.Unary:
		return x.Op == token.SUB && literalData(x.X)
	case *ir.Builtin:
		if x.Name != "complex" || x.Results != nil {
			return false
		}
		for _, arg := range x.Args {
			if !literalData(arg) {
				return false
			}
		}
		return true
	case *ir.Composite:
		if x.Results != nil {
			return false
		}
		switch typeset.Core(x.Type).(type) {
		case *types.Array, *types.Slice, *types.Struct:
		default:
			return false
		}
		for _, elem := range x.Elems {
			if !literalData(elem) {
				return false
			}
		}
		return true
	}
	return false
}

// Separate helpers bound Dart parser depth without a change to operand order.
func (e *emitter) deepSlice(x *ir.Composite) string {
	var chain []*ir.Composite
	for {
		if _, ok := x.Type.Underlying().(*types.Slice); !ok || x.Results != nil || x.Indices != nil || len(x.Elems) != 1 {
			break
		}
		chain = append(chain, x)
		inner, ok := x.Elems[0].(*ir.Composite)
		if !ok {
			break
		}
		x = inner
	}
	if len(chain) < deepSliceLevels {
		return ""
	}
	text := e.composite(chain[len(chain)-1])
	for count := len(chain) - 1; count > 0; {
		n := min(count, 16)
		text = e.sliceWrapper(n) + "(" + text + ")"
		count -= n
	}
	return text
}

func (e *emitter) sliceWrapper(n int) string {
	if e.lits.sliceWraps == nil {
		e.lits.sliceWraps = map[int]string{}
	}
	if name := e.lits.sliceWraps[n]; name != "" {
		return name
	}
	name := fmt.Sprintf("$sliceWrap%d_%d", e.lits.tag, n)
	e.lits.sliceWraps[n] = name
	result := strings.Repeat("Slice<", n) + "T" + strings.Repeat(">", n)
	var body strings.Builder
	previous := "value"
	for i := 0; i < n; i++ {
		next := fmt.Sprintf("v%d", i)
		fmt.Fprintf(&body, "final %s = Slice.of([%s]);\n", next, previous)
		previous = next
	}
	fmt.Fprintf(&body, "return %s;", previous)
	e.lits.decls = append(e.lits.decls, fmt.Sprintf("%s %s<T>(T value) {\n%s\n}", result, name, body.String()))
	return name
}

// Constant indexes permit a helper without an earlier read of another operand.
func (e *emitter) deepIndex(x *ir.Index) string {
	var chain []*ir.Index
	var base ir.Expr = x
	for {
		next, ok := base.(*ir.Index)
		if !ok {
			break
		}
		if _, ok := next.X.GoType().Underlying().(*types.Slice); !ok {
			break
		}
		if _, ok := next.Index.(*ir.Const); !ok || e.unsignedBound(next.Index) != "false" {
			break
		}
		chain = append(chain, next)
		base = next.X
	}
	if len(chain) < 16 {
		return ""
	}
	if e.lits.sliceRead == "" {
		e.lits.sliceRead = fmt.Sprintf("$sliceRead%d", e.lits.tag)
		e.lits.decls = append(e.lits.decls, fmt.Sprintf(`T %s<T>(Object? source, List<GoInt> indexes) {
    Object? value = source;
    for (final index in indexes) {
      value = (value as Slice).$at(index);
    }
    return value as T;
  }`, e.lits.sliceRead))
	}
	indexes := make([]string, len(chain))
	for i, node := range chain {
		indexes[len(chain)-1-i] = e.expr(node.Index)
	}
	return fmt.Sprintf("%s<%s>(%s, <GoInt>[%s])", e.lits.sliceRead, e.dartType(x.Type), e.expr(base), strings.Join(indexes, ", "))
}
